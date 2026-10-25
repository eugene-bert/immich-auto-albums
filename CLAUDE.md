# immich-auto-albums

Go service with web UI that auto-organizes Immich photos into albums based on configurable rules (camera model, location, date, filename patterns). Docker container.

## Commands

```bash
go build ./...          # build
go vet ./...            # lint
go test -race ./...     # test
docker compose up -d    # run (production)
```

## CI/CD

- **CI** (`ci.yml`): `go build`, `go vet`, `go test -race` on push/PR to main
- **Release** (`release.yml`): triggered by `v*` tags, builds multi-arch Docker image (amd64+arm64), pushes to GHCR
- **GHCR tags**: `{{version}}` (semver **without** `v` prefix), `{{major}}.{{minor}}`, `latest`
- **Dependabot**: auto-merge patch/minor updates

## Key files

- `main.go` — entrypoint, HTTP server for web UI + sync engine
- `immich/` — Immich API client (albums, assets, search)
- `rules/` — album rule definitions and matching engine
- `sync/` — sync orchestrator (applies rules → creates/updates albums)
- `templates/` — Go HTML templates for web UI
- `static/` — CSS/JS assets
- `.env.example` — all env vars documented

## Important

- Web UI on port 8095 (configurable via `PORT`)
- Rules stored in SQLite (`DB_PATH`, default `/data/rules.db`)
- Optional basic auth via `UI_USER` / `UI_PASSWORD`
- Container runs as uid 1000
- Connects to Immich API via `IMMICH_URL` + `IMMICH_API_KEY`
- Do NOT use `:latest` in production — pin semver tags
