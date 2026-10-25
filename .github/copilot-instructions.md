# Copilot Instructions

Go service with web UI: auto-organize Immich photos into albums based on rules. Docker container.

## Build & Test

```bash
go build ./...
go vet ./...
go test -race ./...
```

## CI/CD

- CI on push/PR: build, vet, test
- Release on `v*` tags → GHCR multi-arch image
- GHCR tags: `{{version}}` (**no** `v` prefix), `{{major}}.{{minor}}`, `latest`

## Architecture

- `main.go` — HTTP server + sync engine
- `immich/` — Immich API client
- `rules/` — rule definitions and matching
- `sync/` — sync orchestrator
- `templates/` + `static/` — web UI
- Rules in SQLite, web UI on port 8095
