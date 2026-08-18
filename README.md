# Go Gin Template

A production-shaped starting point for a Go HTTP service: Gin router, GORM over
PostgreSQL, structured logging, OpenTelemetry tracing, and a consistent JSON
error envelope — wired end to end through one worked example (`users` CRUD) you
delete or rename.

## Quick Start

Requires Go 1.25+ and a reachable PostgreSQL.

```bash
git clone https://github.com/aarondever/go-gin-template.git
cd go-gin-template
cp .env.example .env      # edit DB_* to point at your database
go mod download
make dev                  # http://localhost:8080
```

Create the example table (there is no migration step — see
[Configuration](#configuration)):

```sql
CREATE TABLE users (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT UNIQUE,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
```

Common tasks:

```bash
make help           # list all targets
make dev            # go run ./cmd/server
make build          # ./build/main
make test           # go test -v ./...
make test-coverage  # writes coverage.html
make test-race      # race detector
make install-deps   # installs air for live reload
air                 # live reload + delve on :2345 (see .air.toml)
```

Docker:

```bash
docker build -t go-gin-template .
docker run --rm -p 8080:8080 --env-file .env go-gin-template
```

## Tech Stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.25 |
| HTTP | [gin-gonic/gin](https://github.com/gin-gonic/gin) |
| ORM / DB | [gorm](https://gorm.io) + `gorm.io/driver/postgres` (pgx) |
| Config | [caarlos0/env](https://github.com/caarlos0/env) + [joho/godotenv](https://github.com/joho/godotenv) |
| Validation | [go-playground/validator/v10](https://github.com/go-playground/validator) |
| Logging | stdlib `log/slog` (JSON or text) |
| Tracing | OpenTelemetry SDK, OTLP/HTTP exporter, `otelgin` + GORM tracing plugin |
| IDs | [google/uuid](https://github.com/google/uuid) (UUIDv7, time-ordered) |

## Architecture

```text
request
   │
   ▼
otelgin ──► Logger ──► ErrorHandler ──► CustomRecovery ──► handler   bind + validate
(span)      (access)   (envelope)       (panic → c.Error)     │
                            ▲                                 ▼
                            │                              service   business rules
                            │                                 │
                            │                                 ▼
                            └───────── error ──────────── repository   GORM / SQL
                                                              │
                                                              ▼
                                                          PostgreSQL
```

## Project Structure

```text
cmd/server/main.go        composition root: config → logger → telemetry → db → repo → svc → handler → server
config/                   env-tagged config structs, .env loading
internal/
  apperror/               error codes, *AppError, From() normalization
  database/               connection + pool, tracing plugin, tx manager, Paginate scope
  handler/                HTTP binding + validation, request/response DTOs
  logger/                 slog setup, context handler, trace-id extractor
  middleware/             CORS, access logger, error handler
  model/                  domain structs (GORM + json + validate tags)
  pagination/             Page/PageSize/Total with clamped limits
  repository/             GORM queries, driver-error → AppError mapping
  response/               success envelope: {"data": …}
  router/                 middleware chain + route table
  service/                business rules and orchestration
  telemetry/              OpenTelemetry tracer provider
  util/                   UUIDv7 IDs, recursive struct-string trimming
  validation/             shared validator instance + field naming
docs/                     API reference, development guide
```

## Configuration

All configuration is environment variables, loaded from `.env` if present (a
missing `.env` is fine — real environment variables win in deployment). Parsed
once in [config/config.go](config/config.go); `required` variables fail the boot
rather than defaulting to something wrong.

| Variable | Default | Notes |
| --- | --- | --- |
| `SERVER_PORT` | `8080` | |
| `SERVER_MODE` | `release` | `debug` or `release`; sets Gin's mode |
| `SERVER_READ_TIMEOUT` | `30s` | |
| `SERVER_WRITE_TIMEOUT` | `30s` | |
| `DB_HOST` | — | **required** |
| `DB_PORT` | `5432` | |
| `DB_USER` | — | **required** |
| `DB_PASSWORD` | — | **required**; unset from the environment after reading |
| `DB_NAME` | — | **required** |
| `DB_SSLMODE` | `disable` | use `require` or stricter in production |
| `DB_MAX_OPEN_CONNS` | `25` | |
| `DB_MAX_IDLE_CONNS` | `5` | |
| `DB_CONN_MAX_LIFETIME` | `5m` | |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` (`text` is nicer locally) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(empty)* | empty = trace IDs generated, nothing exported |
| `OTEL_SERVICE_NAME` | `go-gin-service` | shown as the service in your tracing backend |
| `OTEL_TRACES_SAMPLER_ARG` | `1` | ratio 0–1, parent-based |

## Documentation

- [docs/API.md](docs/API.md) — endpoints, envelopes, error codes, pagination.
- [docs/development.guid.md](docs/development.guid.md) — local setup, adding a resource, testing, conventions.
