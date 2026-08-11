# immich-auto-albums

Automatically organize your [Immich](https://immich.app) library into albums based on rules. Filter by camera, location, media type, date range, and more.

## Features

- Web UI (HTMX) to create and manage rules
- Auto-sync on a configurable interval per rule
- Manual sync with one click
- Filter by camera make/model, lens, city/state/country, media type, date range
- Albums created automatically if they don't exist
- Runs as a single Docker container alongside Immich

## Quick start

```bash
git clone https://github.com/eugene-bert/immich-auto-albums.git
cd immich-auto-albums
cp .env.example .env
# Edit .env with your Immich URL and API key
docker compose up -d
```

Open `http://your-server:8095` to manage rules.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `IMMICH_URL` | Yes | | Immich server URL |
| `IMMICH_API_KEY` | Yes | | Immich API key |
| `PORT` | No | `8095` | Web UI port |
| `DB_PATH` | No | `/data/rules.db` | SQLite database path |

## Available filters

| Filter | Example | Description |
|--------|---------|-------------|
| Camera Make | `FUJIFILM` | EXIF camera manufacturer |
| Camera Model | `X-E5` | EXIF camera model |
| Lens Model | `XF23mmF1.4` | EXIF lens model |
| Media Type | `IMAGE` / `VIDEO` | Asset type |
| City | `Krakow` | GPS-derived city |
| State | `Lesser Poland` | GPS-derived state |
| Country | `Poland` | GPS-derived country |
| Taken After | `2026-01-01` | Photos taken after date |
| Taken Before | `2026-12-31` | Photos taken before date |

Filters are combined with AND logic.

## License

MIT
