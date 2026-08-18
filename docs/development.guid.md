# Development Guide

## Setup

Requires Go 1.25+ and PostgreSQL.

```bash
cp .env.example .env       # then edit DB_*
go mod download
make dev
```

For local work, `LOG_FORMAT=text` and `SERVER_MODE=debug` are easier to read than
the defaults.

### Live reload

```bash
make install-deps          # installs air
air
```

[.air.toml](../.air.toml) rebuilds with optimizations off and runs the binary
under `dlv` headless on `:2345`, so you can attach a debugger to the reloading
process. [.vscode/launch.json](../.vscode/launch.json) has the attach config.

### Database schema

There is no migration tooling in the template, and `AutoMigrate` is not called —
pick your own (goose, atlas, plain SQL files) and wire it in. The example
`users` table DDL is in the [README](../README.md#quick-start).

## Make targets

| Target | Does |
| --- | --- |
| `make help` | Lists targets (default) |
| `make dev` | `go run ./cmd/server` |
| `make build` | Stripped binary at `./build/main` |
| `make test` | `go test -v ./...` |
| `make test-coverage` | Writes `coverage.out` + `coverage.html` |
| `make test-race` | Race detector |
| `make clean` | Removes build artifacts and the Go build cache |

## Adding a resource

Five files, bottom up. Copy the `users` example and rename.

**1. `internal/model/thing.go`** — the domain struct plus a list filter. Three tag
families do three jobs: `gorm` maps columns, `json` shapes the response *and*
names fields in validation errors, `validate` states the domain rules.

```go
type Thing struct {
    ID        uint64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
    Name      string         `json:"name" gorm:"column:name;not null" validate:"required"`
    CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
    UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
    DeletedAt gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
}
```

`gorm.DeletedAt` is what makes deletes soft and filters deleted rows from reads.

**2. `internal/repository/thing.go`** — interface + GORM implementation. Two rules:

- Start every query from `database.ExtractTx(ctx, r.db).WithContext(ctx)`, never
  from `r.db` directly, or the method will silently escape an open transaction.
- Translate sentinel driver errors into `AppError`s; wrap everything else with
  `fmt.Errorf` and let `apperror.From` classify it as `INTERNAL`.

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, e.Wrap(err, e.CodeNotFound, "thing not found")
}
return nil, fmt.Errorf("get thing %d: %w", id, err)
```

`TranslateError: true` is set on the GORM config, so `gorm.ErrDuplicatedKey` and
friends are portable across drivers.

**3. `internal/service/thing.go`** — interface + implementation over the repository.
Business rules and orchestration only; input arrives already validated by the
handler, so the service does not re-check it. Wrap repository errors with the
operation name and return:

```go
func (s *service) Create(ctx context.Context, thing *model.Thing) (*model.Thing, error) {
    if err := s.repo.Create(ctx, thing); err != nil {
        return nil, fmt.Errorf("service.Create: %w", err)
    }
    return thing, nil
}
```

Need several repository calls to be atomic? Take a `database.TxManager` and wrap
them; the context carries the transaction down.

```go
return s.tx.WithTx(ctx, func(ctx context.Context) error {
    if err := s.repo.Create(ctx, thing); err != nil {
        return err
    }
    return s.other.Touch(ctx, thing.ID)
})
```

**4. `internal/handler/thing.go`** — request DTOs carrying `validate` tags, plus
thin handlers. A handler binds, validates, calls the service, and writes. On any
error it calls `c.Error(err)` and returns — **never** `c.JSON` with an error
body, or the response escapes the envelope.

```go
type createThingRequest struct {
    Name string `json:"name" validate:"required"`
}

func (h *Handler) Create(c *gin.Context) {
    var req createThingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(err)
        return
    }
    if err := validation.ValidateStruct(util.TrimStructStr(&req)); err != nil {
        c.Error(err)
        return
    }
    thing, err := h.svc.Create(c.Request.Context(), &model.Thing{Name: req.Name})
    if err != nil {
        c.Error(err)
        return
    }
    response.JSON(c, http.StatusCreated, thing)
}
```

Pass `&req`, not `req` — `TrimStructStr` takes a pointer and silently does
nothing when handed a value. For partial updates, exclude fields that are
legally absent, named by **Go field name**, not JSON name:

```go
validation.ValidateStruct(util.TrimStructStr(&req), "Name")
```

Pass `c.Request.Context()`, not `c` — that is what carries the span and the
cancellation.

For list endpoints, embed `pagination.Pagination` in both the query struct and
the response struct; `database.Paginate(page)` fills in `Total` as a scope.

**5. Wire it up** — construct in [cmd/server/main.go](../cmd/server/main.go) and
register the route group in [internal/router/router.go](../internal/router/router.go).

## Conventions

**Errors.** `apperror.New`/`Wrap` at the boundary where you know the meaning;
`fmt.Errorf("%w")` everywhere else. Prefix wrap messages with the operation
(`"service.Create: %w"`) so the log line reads as a trail.

**Logging.** Use `logger.*Context(ctx, …)` so the trace IDs land on the line.
`logger.Err(err)` is the standard error attribute. The access log and error log
are already emitted by middleware — don't log the same request again in a
handler.

**Validation.** Request DTOs carry `validate` tags and are checked in the handler
with `validation.ValidateStruct`; Gin's binding validator still catches JSON
shape and type errors during `ShouldBind*`. Both name fields identically via
`validation.FieldName`, so error `details` keys are stable regardless of which
one fired. The service trusts its input — validate before calling one from a
job or CLI.

**Config.** Add a field with an `env` tag to the right struct in
[config/config.go](../config/config.go), and add it to `.env.example`. Mark it
`required` if there is no safe default. Nothing else reads `os.Getenv`.

**Middleware order** in [router.go](../internal/router/router.go) is load-bearing:
`otelgin` outermost (so everything is inside a span), then `Logger`, then
`ErrorHandler`, then `CustomRecovery` innermost — a panic must unwind *into*
`ErrorHandler` to get the envelope.

## Testing

```bash
make test
make test-coverage      # open coverage.html
make test-race
```

Table-driven tests with the stdlib `testing` package; no assertion library. See
[internal/apperror/apperror_test.go](../internal/apperror/apperror_test.go) and
[internal/middleware/errors_test.go](../internal/middleware/errors_test.go) for
the shape.

Because each layer depends on the interface below it, a service test needs no
database — a struct implementing `repository.Repository` with function fields is
enough:

```go
type fakeRepo struct {
    repository.Repository
    create func(context.Context, *model.User) error
}

func (f fakeRepo) Create(ctx context.Context, u *model.User) error {
    return f.create(ctx, u)
}
```

Embedding the interface means you only implement the methods your test touches.

For handler tests, use `gin.CreateTestContext` with `httptest.NewRecorder` and
mount the same middleware you care about.

## Deployment

```bash
docker build -t go-gin-template .
docker run --rm -p 8080:8080 --env-file .env go-gin-template
```

Multi-stage build, static binary, alpine runtime. Before shipping: set
`SERVER_MODE=release`, `LOG_FORMAT=json`, a non-`disable` `DB_SSLMODE`, and point
`OTEL_EXPORTER_OTLP_ENDPOINT` at a collector (lower `OTEL_TRACES_SAMPLER_ARG`
below `1` under real traffic).

The server handles SIGINT/SIGTERM: it stops accepting connections, drains for up
to 10s, closes the pool, and flushes pending spans (5s). Give your orchestrator a
`terminationGracePeriodSeconds` above that.

The template ships no authentication, rate limiting, or request-size limit, and
[`middleware.CORS`](../internal/middleware/cors.go) exists but is **not**
registered — it allows every origin, so tighten it before you enable it.
