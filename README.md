# Tentserv Chat Server

A Go REST API + WebSocket server for the Tentserv Chat desktop application. Personal practice project.

## Stack

- **Go 1.26.1** · Gin (HTTP) · Gorilla WebSocket
- **PostgreSQL 18** · Redis 8
- **Swagger** (swaggo) · **BDD tests** (Godog/Gherkin)

## Features

- User authentication (JWT)
- Real-time chat via WebSocket
- LLM redirect to agent
- E2EE key distribution (Signal Protocol X3DH)
- Group chat with sender key re-keying

## Quick Start

### 1. Start dependencies

```shell
docker network create goat-net

# Redis (optional: add -v ~/data/redis:/data for persistence)
docker run -d --name redis --network goat-net \
  -p 6379:6379 \
  docker.io/library/redis:8 \
  redis-server --appendonly yes --requirepass "1234"

# PostgreSQL (optional: add -v ~/data/postgres:/data for persistence)
docker run -d --name postgres --network goat-net \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=goat \
  -p 5432:5432 \
  docker.io/library/postgres:18
```

### 2. Configure

Create `config/config.yaml`:

```yaml
auth_token:
  expiration: 3600
secrets:
  HMAC_SECRET: "00000000000000000000000000000000"
databases:
  postgres:
    driver: pgx
    dsn: "postgres://root:1234@localhost:5432/goat?sslmode=disable"
    config:
      max_open_conns: 30
      max_idle_conns: 15
      conn_max_lifetime: 3600
      conn_max_idle_time: 600
redis:
  addr: "localhost:6379"
  password: "1234"
  db: 0
```

Create `.env`:

```dotenv
APP_ENV=dev
SERVER_PORT=8080
```

Initialize the database schema:

```shell
psql -h localhost -U root -d goat -f config/init_postgres.sql
```

### 3. Run

```shell
make setup   # Install swag and godog CLIs (first time only)
make run     # Start server on :8080
```

## Commands

```shell
make build   # Compile to bin/goat-api
make run     # Start server on :8080
make test    # Run all tests (unit + BDD)
make unit    # Unit tests only: go test ./internal/... -v
make bdd     # BDD tests only: go test -v ./features
make swag    # Regenerate Swagger docs
make clean   # Remove build output
```

Run a single test:

```shell
go test ./internal/path/to/pkg/... -v -run TestFunctionName
```

## Deploy Locally (Docker)

```shell
# Build image
docker build -t goat-server:latest .

# Run container (uses goat-net network for DB/Redis access)
docker run -d --name goat-server --network goat-net \
  -p 8080:8080 \
  -v ./config:/app/config:ro \
  -v .env:/app/.env:ro \
  goat-server:latest
```
