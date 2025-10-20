# Go Gin Template

A batteries-included template for building REST APIs in Go using the Gin framework. It features a clean architecture,
dependency injection with Google Wire, database access with sqlc, PostgreSQL migrations with Goose, Redis, hot-reload
with Air, and a Docker-based local environment.

## Features

- Gin web framework
- Dependency Injection via Google Wire
- PostgreSQL with sqlc type-safe queries
- Database migrations with Goose
- Redis (optional) integration
- Docker Compose for app + Postgres + Redis
- Air for live reload during development
- Structured logging with slog (JSON or text)
- Makefile with common developer tasks

## Project Structure

```
.
├── cmd/api/main.go                # App entrypoint
├── internal/                      # Application code
│   ├── config/                    # Config loading (env + YAML)
│   ├── database/                  # DB setup and sqlc generated code
│   ├── handlers/                  # HTTP handlers (Gin)
│   ├── models/                    # Domain models and errors
│   ├── services/                  # Business logic
│   ├── utils/                     # Helpers
│   ├── wire.go, wire_gen.go       # DI wiring
├── sql/                           # SQL queries and migrations
│   ├── queries/                   # sqlc input queries
│   └── schema/                    # Goose migrations
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── sqlc.yaml
└── README.md
```

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Make (optional but recommended)

You can install necessary CLI tools via the Makefile:

- Air (live reload)
- Goose (migrations)
- sqlc (code generation)
- Wire (dependency injection)

Run: `make install-deps`

## Quick Start

### Option A: Run everything with Docker

1. Build and start containers:
    - `make docker`  (equivalent to clean + build + up)
    - or `make docker-build && make docker-up`
2. The API will be available at: http://localhost:8080

### Option B: Run locally on your machine

1. Start Postgres and Redis (recommended via Docker):
    - `docker compose up -d postgres redis`
2. Configure environment and config (see Configuration section).
3. Generate code and run migrations:
    - `make sqlc`
    - `make migrate-up`
4. Start the app with live reload:
    - `make dev`

## Configuration

Configuration is loaded from environment variables and optionally overridden by a YAML file passed with the
`-config.file` flag (default: `config.yaml`). A `.env` file is also supported and automatically loaded if present.

Environment variables (defaults in parentheses):

- APP_ENV (development)
- TZ (UTC)
- HOST (0.0.0.0)
- PORT (8080)
- DB_HOST (localhost)
- DB_PORT (5432)
- DB_USER (postgres)
- DB_PASSWORD (postgres)
- DB_NAME (postgres)
- DB_SSLMODE (disable)
- REDIS_HOST (localhost)
- REDIS_PORT (6379)
- REDIS_USER ("")
- REDIS_PASSWORD ("")
- REDIS_DB (0)
- LOG_LEVEL (info) [debug|info|warn|error]
- LOG_FORMAT (text) [text|json]

Example `.env`:

```
APP_ENV=development
HOST=0.0.0.0
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=postgres
REDIS_HOST=localhost
LOG_LEVEL=debug
LOG_FORMAT=text
TZ=UTC
```

Optional `config.yaml` overrides:

```
server:
  host: 0.0.0.0
  port: 8080

database:
  host: localhost
  port: 5432
  username: postgres
  password: postgres
  name: postgres
  sslmode: disable

redis:
  host: localhost
  port: 6379
  username: ""
  password: ""
  db: 0

logging:
  level: info
  format: text
```

## Database and Code Generation

- sqlc: generate type-safe Go code from SQL files in `sql/queries`
    - `make sqlc`
- Goose: run migrations in `sql/schema`
    - `make migrate-up`
    - `make migrate-down`
    - `make migrate-status`

The Makefile uses a default connection string for local development:
`postgres://postgres:postgres@localhost:5432/postgres`

When using Docker Compose, the app service is linked to `postgres` and `redis` services. Environment variables are
provided automatically by docker-compose.yml.

## Running and Building

- Development with live reload: `make dev`
- Build binary: `make build` (outputs to `./build/main`)
- Run tests: `make test`
- Test with race: `make test-race`
- Test coverage report: `make test-coverage`
- Clean artifacts: `make clean`

## API Endpoints

User routes are available under the `/users/v1` prefix.

- GET `/users/v1/` — List users with pagination
    - Query params: `page` (default 1), `page_size` (default 10)
- GET `/users/v1/:id` — Get user by ID (UUID)
- POST `/users/v1/` — Create user
- PUT `/users/v1/:id` — Update user by ID (UUID)

Example requests:

List users

```
curl "http://localhost:8080/users/v1/?page=1&page_size=10"
```

Create user

```
curl -X POST http://localhost:8080/users/v1/ \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Liddell"
  }'
```

Update user

```
curl -X PUT http://localhost:8080/users/v1/<UUID> \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Liddell"
  }'
```

Get user

```
curl http://localhost:8080/users/v1/<UUID>
```

## Docker

Common docker targets:

- `make docker-build` — build images
- `make docker-up` — start containers (app, postgres, redis)
- `make docker-down` — stop containers
- `make docker-restart` — restart
- `make docker-clean` — remove containers and local images

The API is exposed on `localhost:8080`. Postgres on `localhost:5432`. Redis on `localhost:6379`.

## Troubleshooting

- If Air is not found, run `make install-deps`.
- If migrations fail, ensure Postgres is running and the connection string/variables are correct.
- If sqlc code seems outdated, run `make sqlc` after editing SQL files.
- On Windows with WSL, ensure your Docker is accessible from WSL and paths are correct.

## License

This project is provided as a template. Add your preferred license file if you intend to distribute it.
