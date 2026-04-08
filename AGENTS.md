# AGENTS.md

Backend repo scope handoff for `tentserv-chat-server/`.

## Scope

This repo owns the Go REST API, WebSocket server, application use cases, domain models, PostgreSQL repositories, Redis integrations, email builders, and Godog BDD tests.

## Read Order

1. Read `../AGENTS.md` for workspace-wide business routing.
2. Read `../docs/README.md` only when routing docs work or UI/UX operation records.
3. Read only the relevant `../docs/agent-guides/*/SKILL.md` for the business flow.
4. Read this file to confirm the work belongs in the backend repo.
5. Read `CLAUDE.md` for backend commands, architecture style, and test execution rules.

## Boundaries

- Do not edit `../tentserv-chat/` unless the selected business guide or API contract requires frontend or Rust changes.
- Do not read unrelated business guides unless a real dependency is discovered.
- Follow `../AGENTS.md` for guide update checks; keep full business-guide policy there, not in this file.
- Backend user-visible HTTP behavior changes must update BDD coverage under `features/`.
