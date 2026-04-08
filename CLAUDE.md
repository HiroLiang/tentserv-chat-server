# CLAUDE.md

Backend execution and style guide for `tentserv-chat-server/`. Business scope comes from the outer `../AGENTS.md`, the matching `../docs/agent-guides/*/SKILL.md`, and this repo's `AGENTS.md`.

## Commands

Run from `tentserv-chat-server/`:

```bash
make setup   # Install swag and godog CLIs before first use
make build   # Compile to bin/goat-api
make run     # Start server locally on :8080
make test    # Run unit + BDD tests
make unit    # go test ./internal/... -v
make bdd     # go test -v ./features
make swag    # Regenerate Swagger docs after API annotation changes
go test ./internal/path/to/pkg -count=1 -v -run TestFunctionName
```

Use `go test ./features -count=1 -v` whenever backend HTTP behavior changes.

## Architecture Rules

- Keep Clean Architecture direction: `domain -> application -> infrastructure/interface`.
- Handlers bind HTTP DTOs, build use case inputs with `adapter.BuildInput(c, data)` or `adapter.BuildEmptyInput(c)`, then delegate to use cases.
- Use cases accept `shared.UseCaseInput[T]`; business logic stays there, not in Gin handlers.
- Repository implementations use the existing `Masterminds/squirrel` plus `postgres.ScanOne[T]` / `postgres.ScanAll[T]` helpers. `BaseRepo.GetDB(ctx)` returns the transaction-bound DB when present.
- When package names clash, alias domain imports clearly, for example `domainuser`.

## BDD Rules

- BDD lives in `features/` using Godog/Gherkin and an `httptest.Server`.
- Keep the BDD server minimal for the flow under test; do not initialize full chat/ws/e2ee wiring unless that flow requires it.
- New or changed user-visible HTTP behavior must update a feature file and steps.
- Step logs must include `Given`, `Input`, `Action`, `Output`, `Mutation`, and `Duration`.
- Do not log raw passwords, verification tokens, private keys, session tokens, or E2EE key material.

## Unit Test Rules

- Use unit tests for internal contracts BDD should not own: SQL mapping, repo error translation, builders, transaction commit/rollback, cryptography, and narrow failure branches.
- Do not duplicate BDD happy paths unless the use case has internal behavior that is not visible over HTTP.
- Prefer focused commands first, then `go test ./internal/... -count=1 -v` before handoff when backend internals changed.

## Config

- Runtime config: `config/config.yaml`
- BDD config: `dev-doc/config/config.yaml`
- Schema initialization: `config/init_postgres.sql`
- Local PostgreSQL/Redis details are documented in the outer workspace `AGENTS.md` when needed for manual dev runs.
