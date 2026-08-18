# API Reference

Base URL: `http://localhost:8080`. All resource endpoints live under `/v1`.

## Envelopes

Every successful response is wrapped in `data`:

```json
{ "data": { "id": 1, "name": "Ada" } }
```

Every failure is wrapped in `error`:

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "validation failed",
    "details": { "email": "email" }
  }
}
```

`details` is optional and, for validation failures, maps the field's JSON name to
the validator tag it violated. Internal causes are logged, never serialized.

## Error codes

| Code | HTTP | When |
| --- | --- | --- |
| `INVALID_INPUT` | 400 | Body/query failed binding or validation |
| `UNAUTHORIZED` | 401 | Reserved — no auth in the template |
| `FORBIDDEN` | 403 | Reserved |
| `NOT_FOUND` | 404 | No row for the given id |
| `CONFLICT` | 409 | Unique constraint violated (e.g. duplicate email) |
| `RATE_LIMITED` | 429 | Reserved |
| `CANCELED` | 499 | Client disconnected; no body is written |
| `TIMEOUT` | 504 | Request context deadline exceeded |
| `INTERNAL` | 500 | Anything unclassified, including recovered panics |

Codes are defined in [internal/apperror/apperror.go](../internal/apperror/apperror.go)
and mapped to statuses in [internal/middleware/errors.go](../internal/middleware/errors.go).

## Pagination

List endpoints accept `page` and `page_size` as query parameters and echo them
back alongside `total`, the unpaginated row count.

| Parameter | Default | Clamp |
| --- | --- | --- |
| `page` | `1` | `< 1` becomes `1` |
| `page_size` | `10` | `> 100` becomes `100`; `<= 0` becomes `10` |

## Tracing

Send a W3C `traceparent` header and the service continues your trace; otherwise
it starts one. The active trace and span IDs appear on every log line for the
request.

---

## `GET /health`

Liveness check. Excluded from access logging and tracing, and not under `/v1`.

```json
{ "status": "ok" }
```

Note: this returns a bare object, not the `data` envelope.

---

## Users

The worked example. Delete or rename it when you build your own resource.

### `POST /v1/users`

Create a user. → `201 Created`

| Field | Type | Rules |
| --- | --- | --- |
| `name` | string | required |
| `email` | string\|null | optional; must be a valid email if present |

```bash
curl -X POST localhost:8080/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com"}'
```

```json
{
  "data": {
    "id": 1,
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "created_at": "2026-08-18T10:00:00Z",
    "updated_at": "2026-08-18T10:00:00Z"
  }
}
```

Errors: `INVALID_INPUT` (missing name, malformed email), `CONFLICT` (email taken).

Whitespace trimming of string fields is wired up via `util.TrimStructStr`, but
is currently inert — the handlers pass the request struct by value and the
helper only mutates through a pointer, so values are stored as sent. See the
note in the [development guide](development.guid.md#adding-a-resource).

### `GET /v1/users/:userID`

Fetch one user. → `200 OK`

`userID` must parse as an unsigned integer — anything else is `INVALID_INPUT`.
A missing (or soft-deleted) row is `NOT_FOUND`.

### `GET /v1/users`

List users. → `200 OK`

| Query | Match |
| --- | --- |
| `name` | substring, `LIKE %name%` |
| `email` | exact; must be a valid email |
| `page`, `page_size` | see [Pagination](#pagination) |

```bash
curl 'localhost:8080/v1/users?name=ada&page=1&page_size=20'
```

```json
{
  "data": {
    "users": [ { "id": 1, "name": "Ada Lovelace", "email": "ada@example.com" } ],
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

### `PUT /v1/users/:userID`

Partial update — only non-zero fields are written (GORM `Updates` semantics), so
omitting `name` leaves it unchanged. → `200 OK`

```bash
curl -X PUT localhost:8080/v1/users/1 \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@lovelace.dev"}'
```

The response echoes the fields you sent plus the id, not the full stored row.

Errors: `INVALID_INPUT`, `CONFLICT`.

### `DELETE /v1/users/:userID`

Soft delete — sets `deleted_at`; the row stays in the table and is filtered out
of every subsequent query. → `204 No Content`, empty body.

Deleting an id that does not exist also returns `204`.
