# Go Gin Template

A batteries-included template for building REST APIs in Go using the Gin framework. It features a clean architecture,
PostgreSQL with GORM, database migrations with GORM AutoMigrate, Redis, hot-reload with Air, and a Docker-based local environment.

## Features

- Gin web framework
- PostgreSQL with GORM ORM
- Database migrations with GORM AutoMigrate
- Redis integration
- Docker Compose for app + Postgres + Redis
- Air for live reload during development
- Structured logging with slog (JSON or text)
- Makefile with common developer tasks

## Project Structure

```text
.
├── cmd/api/main.go                # App entrypoint
├── internal/                      # Application code
│   ├── domain/                    # Domain logic (handlers, services, repositories)
│   ├── shared/                    # Shared components
│   │   ├── config/                # Config loading (env variables)
│   │   ├── database/              # DB and Redis setup
│   │   ├── errors/                # Error definitions
│   │   ├── middleware/            # HTTP middleware
│   │   └── model/                 # Base models
├── pkg/                           # Public packages
│   ├── logger/                    # Logger utilities
│   ├── pagination/                # Pagination helpers
│   └── response/                  # Response helpers
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Make (optional but recommended)

You can install necessary CLI tools via the Makefile:

- Air (live reload)

Run: `make install-deps`

## Quick Start

### Option A: Run everything with Docker

1. Build and start containers:
    - `make docker`  (equivalent to clean + build + up)
    - or `make docker-build && make docker-up`
2. The API will be available at: <http://localhost:8080>

### Option B: Run locally on your machine

1. Start Postgres and Redis (recommended via Docker):
   - `docker compose up -d postgres redis`
2. Configure environment and config (see Configuration section).
3. Start the app with live reload:
   - `make dev`

   Note: Database migrations are handled automatically via GORM AutoMigrate when the application starts.

## Configuration

Configuration is loaded from environment variables. A `.env` file is also supported and automatically loaded if present.

Copy `.env.example` to `.env` and update the values as needed:

```bash
cp .env.example .env
```

See `.env.example` for all available configuration options and their defaults.

## Database and Migrations

This project uses GORM for database operations and migrations. Database migrations are handled automatically via GORM's `AutoMigrate` feature, which runs when repositories are initialized.

### How Migrations Work

- Migrations are defined in your domain models (e.g., `internal/domain/user/user.go`)
- When a repository is created, it automatically calls `AutoMigrate` on the model
- GORM will create or update tables based on your model definitions
- No manual migration commands are needed

### Example

```go
// In repository.go
func NewRepository(db *database.Database) Repository {
    db.DB.AutoMigrate(&User{})  // Automatically creates/updates the users table
    return &repository{db: db}
}
```

When using Docker Compose, the app service is linked to `postgres` and `redis` services. Environment variables are
provided automatically by docker-compose.yml. Note that docker-compose.yml uses `DB_USER` and `DB_NAME` which map to
`DB_USERNAME` and `DB_DATABASE` in the application config.

## Running and Building

- Development with live reload: `make dev`
- Build binary: `make build` (outputs to `./build/main`)
- Run tests: `make test`
- Test with race: `make test-race`
- Test coverage report: `make test-coverage`
- Clean artifacts: `make clean`

## API Endpoints

**Health Check:**

- GET `/health` — Health check endpoint

**User routes** (available under `/api/v1/users`):

- GET `/api/v1/users` — List users with pagination
  - Query params: `page` (default 1), `page_size` (default 10)
- GET `/api/v1/users/:id` — Get user by ID
- POST `/api/v1/users` — Create user
- PUT `/api/v1/users/:id` — Update user by ID
- DELETE `/api/v1/users/:id` — Delete user by ID

Example requests:

**Health check:**

```bash
curl http://localhost:8080/health
```

**List users:**

```bash
curl "http://localhost:8080/api/v1/users?page=1&page_size=10"
```

**Create user:**

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Liddell"
  }'
```

**Get user:**

```bash
curl http://localhost:8080/api/v1/users/1
```

**Update user:**

```bash
curl -X PUT http://localhost:8080/api/v1/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Liddell"
  }'
```

**Delete user:**

```bash
curl -X DELETE http://localhost:8080/api/v1/users/1
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
- If database connection fails, ensure Postgres is running and the connection variables are correct.
- If migrations fail, check that your GORM models are correctly defined and that the database user has proper permissions.
- On Windows with WSL, ensure your Docker is accessible from WSL and paths are correct.

## License

This project is provided as a template. Add your preferred license file if you intend to distribute it.
