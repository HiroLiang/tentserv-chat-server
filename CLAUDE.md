# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make setup   # Install swag and godog CLIs (required before first run)
make build   # Compile to bin/goat-api
make run     # Start server on :8080
make test    # Run all tests (unit + BDD)
make unit    # Run unit tests: go test ./internal/... -v
make bdd     # Run BDD tests: go test -v ./features
make swag    # Regenerate Swagger docs (after changing API annotations)
```

To run a single unit test:
```bash
go test ./internal/path/to/pkg/... -v -run TestFunctionName
```

## Architecture

Clean Architecture with strict unidirectional dependency: `domain → application → infrastructure/interface`

**`internal/domain/`** — Entities and repository interfaces. No framework dependencies.

**`internal/application/`** — Use cases. All accept `shared.UseCaseInput[T]` which wraps:
- `Base.Auth` — AccountID, UserID, Roles, AccessToken
- `Base.Request` — IP, TraceID, DeviceID
- `Data T` — typed request payload

**`internal/infrastructure/`** — Implementations: PostgreSQL repos (via pgx/sqlx + Masterminds/squirrel), Redis session, local file storage.

**`internal/interface/http/`** — Gin handlers, middleware, DTOs, and error translators. Each feature group has:
- `*_handler.go` — route handler
- `*_dto.go` — request/response structs
- `*_error_translator.go` — maps domain errors → HTTP status codes

**`internal/bootstrap/`** — DI wiring: `BuildDeps()` constructs all repos/services; `usecases.go` wires use cases.

**`features/`** — BDD tests using Godog/Gherkin. `suite.go` sets up mock deps and an `httptest.Server`; `context.go` contains step definitions.

## Key Patterns

**Gin → UseCase bridge** (`internal/interface/http/adapter/`):
```go
adapter.BuildInput(c, data)      // with typed request data
adapter.BuildEmptyInput(c)       // no request body
```

**Import naming:** When a handler package name matches an application package name (e.g., both named `user`), the imported application package is used unqualified. If domain and application packages clash in the same file, alias domain as `domainuser`.

**Repository queries:** Use `Masterminds/squirrel` — wrap conditions with `squirrel.Eq{...}` and pass to a shared `findOneBy()` helper.

**Mocking in BDD tests:** `bootstrap.MockDeps()` with option functions. `FileStorage` mock must be set explicitly: `deps.FileStorage = mockShared.MockFileStorage()`.

## Configuration

- Production config: `config/config.yaml`
- BDD test config: `dev-doc/config/config.yaml` (loaded via `config.LoadConfig("../dev-doc/config")`)
- Runtime env vars: `APP_ENV`, `SERVER_PORT`, `CONFIG_PATH`
- YAML values support `${VAR:default}` expansion

## Infrastructure Requirements (local dev)

```bash
# PostgreSQL
docker run -d --name postgres --network goat-net \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=goat \
  -p 5432:5432 postgres:18

# Redis
docker run -d --name redis --network goat-net \
  -p 6379:6379 redis:8 redis-server --requirepass "1234"
```

Initialize schema with `config/init_postgres.sql`.
