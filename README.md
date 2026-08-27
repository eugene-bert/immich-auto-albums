# immich-auto-albums

[![CI](https://github.com/eugene-bert/immich-auto-albums/actions/workflows/ci.yml/badge.svg)](https://github.com/eugene-bert/immich-auto-albums/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eugene-bert/immich-auto-albums)](https://goreportcard.com/report/github.com/eugene-bert/immich-auto-albums)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Automatically organize your [Immich](https://immich.app) library into albums based on rules. Filter by camera, location, media type, date range, filename, and more.

This project is unofficial and is not affiliated with Immich.

## How it works

<img width="1427" height="603" alt="Screenshot 2026-08-27 at 16 47 44" src="https://github.com/user-attachments/assets/20abf200-b655-4b89-b7c5-4ee8ea33706f" />


1. You create a rule in the web UI (camera, city, date range, …)
2. On a timer — or when you click Sync — the app searches Immich for matching assets
3. It creates the album if needed and adds any new matches

Sync is **add-only**. Assets that stop matching a rule stay in the album. At least one filter is required so a rule cannot dump the whole library.

## Quick start

```bash
mkdir -p immich-auto-albums/data
cd immich-auto-albums
curl -o docker-compose.yml https://raw.githubusercontent.com/eugene-bert/immich-auto-albums/main/docker-compose.yml
curl -o .env https://raw.githubusercontent.com/eugene-bert/immich-auto-albums/main/.env.example
# Edit .env with your Immich URL and API key
docker compose up -d
```

No build step needed — pulls the pre-built image from `ghcr.io/eugene-bert/immich-auto-albums`.

Open `http://127.0.0.1:8095` to manage rules.

The container runs as uid 1000. Make sure `./data` is writable by that user (`mkdir -p data && chown 1000:1000 data` on Linux).

### Build from source

```bash
git clone https://github.com/eugene-bert/immich-auto-albums.git
cd immich-auto-albums
cp .env.example .env
# Edit .env with your Immich URL and API key
docker compose up -d --build
```

## Docker network

Compose attaches to an external network named `immich_default` so `IMMICH_URL=http://immich_server:2283` resolves. That is the default network created by the official Immich compose project named `immich`.

If `docker compose up` fails with `network immich_default not found`:

```bash
docker network ls | grep immich
```

Edit `docker-compose.yml` and replace `immich_default` with the network you found.

Alternatively, remove the `networks:` block and point Immich at a published port:

```env
IMMICH_URL=http://host.docker.internal:2283
```

`host.docker.internal` is mapped via `extra_hosts` / `host-gateway` so this also works on Linux.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `IMMICH_URL` | Yes | | Immich server URL |
| `IMMICH_API_KEY` | Yes | | Immich API key (Immich → Account Settings → API Keys) |
| `PORT` | No | `8095` | Web UI port |
| `DB_PATH` | No | `/data/rules.db` | SQLite database path |
| `UI_USER` | No | | Basic-auth username for the web UI |
| `UI_PASSWORD` | No | | Basic-auth password; must be set together with `UI_USER` |

## Available filters

| Filter | Example | Description |
|----------|---------|-------------|
| Camera Make | `FUJIFILM` | EXIF camera manufacturer |
| Camera Model | `X-E5` | EXIF camera model |
| Lens Model | `XF23mmF1.4` | EXIF lens model |
| Media Type | `IMAGE` / `VIDEO` | Asset type |
| City | `Krakow` | GPS-derived city |
| State | `Lesser Poland` | GPS-derived state |
| Country | `Poland` | GPS-derived country |
| Taken After | `2026-01-01` | Photos taken after date |
| Taken Before | `2026-12-31` | Photos taken before date |
| Original Filename | `bambu-a1` | Filename contains |
| Description | `3D print` | Description contains |

Filters are combined with AND logic.

## Security

- Keep `.env` private. It holds your Immich API key.
- The web UI can create albums and add assets. Compose publishes it on **127.0.0.1:8095** only. Do not map `8095:8095` to the world.
- Set `UI_USER` and `UI_PASSWORD` if you expose the UI on your LAN or behind anything other than localhost.
- Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). This project follows a [Code of Conduct](CODE_OF_CONDUCT.md).

## License

MIT
