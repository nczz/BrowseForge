# BrowseForge Docker

[Traditional Chinese](README.zh-TW.md)

Deploy BrowseForge with the bundled KasmVNC remote desktop.

## Usage

For production deployments, pin a version tag instead of `latest` so a restart or pull does not upgrade the service unexpectedly:

```bash
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}

docker run -d --name browseforge \
  -p 19280:19280 -p 6901:6901 \
  -e VNC_PASSWORD=browseforge \
  -v "$PWD/browseforge/profiles:/app/profiles" \
  -v "$PWD/browseforge/data:/app/data" \
  -v "$PWD/browseforge/browsers:/app/browsers" \
  -v "$PWD/browseforge/logs:/app/logs" \
  -v "$PWD/browseforge/backups:/app/backups" \
  -e BROWSEFORGE_SEED_BROWSERS=1 \
  --restart unless-stopped \
  ghcr.io/nczz/browseforge:v1.10.7
```

Build from the local source tree:

```bash
cd docker
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}
docker compose up -d --build
```

The Compose file builds the `v1.10.7` release image by default. To test another version:

```bash
BROWSEFORGE_VERSION=v1.10.7 docker compose up -d --build
```

## First Startup

Current release images install the browser engines during the Docker build. On startup, BrowseForge seeds the host-mounted `/app/browsers` cache from the image when `/app/browsers/{engine}/.version` is missing or differs from the packaged version.

The first startup may still take 3-5 minutes and download browser engines when you use an older image, disable `BROWSEFORGE_PREINSTALL_BROWSERS`, or set `BROWSEFORGE_SEED_BROWSERS=0`. During that window, the dashboard may not be ready yet.

Use these commands to verify startup state:

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --json
docker exec browseforge /app/BrowseForge smoke rest --wait --timeout 5m --json
```

## Connections

| Service | URL |
|---------|-----|
| Dashboard + REST API + Playwright proxy | http://localhost:19280 |
| MCP Streamable HTTP | http://localhost:19280/mcp |
| Remote desktop (KasmVNC) | http://localhost:6901 |
| VNC login | `user` / `VNC_PASSWORD` environment variable, default `browseforge` |

## API Token

```bash
docker compose logs | grep "API Token"
# or
docker compose exec browseforge /app/BrowseForge token
```

For a `docker run` deployment:

```bash
docker logs browseforge | grep "API Token"
# or
docker exec browseforge /app/BrowseForge token
```

## Persistence, Upgrades, and Backups

Production deployments should use host bind mounts:

| Host path | Container path | Purpose |
|-----------|----------------|---------|
| `./browseforge/profiles` | `/app/profiles` | Profile metadata and browser user data. |
| `./browseforge/data` | `/app/data` | `.api-token` API token and fingerprint data. |
| `./browseforge/browsers` | `/app/browsers` | Downloaded or seeded Camoufox/CloakBrowser engines. |
| `./browseforge/logs` | `/app/logs` | Server logs. |
| `./browseforge/backups` | `/app/backups` | Filesystem and API backup output. |

When you pull a new image or recreate the container, reuse the same `-v "$PWD/browseforge/...:/app/..."` mounts. This keeps the API token, profiles, browser user data, browser cache, logs, and backups outside the container lifecycle.

`/app/browsers` is a BrowseForge-managed browser cache. The default `BROWSEFORGE_SEED_BROWSERS=1` updates it to the browser version packaged in the image. Set it to `0` only for debugging cases where you intentionally want to preserve a manually installed browser engine.

Upgrade example:

```bash
docker pull ghcr.io/nczz/browseforge:v1.10.7
docker stop browseforge
docker rm browseforge
docker run -d --name browseforge \
  -p 19280:19280 -p 6901:6901 \
  -e VNC_PASSWORD=browseforge \
  -v "$PWD/browseforge/profiles:/app/profiles" \
  -v "$PWD/browseforge/data:/app/data" \
  -v "$PWD/browseforge/browsers:/app/browsers" \
  -v "$PWD/browseforge/logs:/app/logs" \
  -v "$PWD/browseforge/backups:/app/backups" \
  -e BROWSEFORGE_SEED_BROWSERS=1 \
  --restart unless-stopped \
  ghcr.io/nczz/browseforge:v1.10.7
```

Full filesystem backup:

```bash
docker stop browseforge
tar -czf ./browseforge/backups/browseforge-runtime-$(date +%Y%m%d-%H%M%S).tgz ./browseforge/profiles ./browseforge/data ./browseforge/browsers ./browseforge/logs
docker start browseforge
```

Convenience backup command while the container is running:

```bash
docker exec browseforge /app/BrowseForge backup create --full --output /app/backups --json
```

The REST `/api/backup` endpoint creates a lighter profile metadata backup. Back up the host runtime directory when you need to preserve full browser user data and the API token.

## Features

- **KasmVNC**: better clipboard support than noVNC in Chrome, plus IME input support.
- **Complete WebGL spoofing**: GLX, software rendering, and full WebGL fingerprint control.
- **Docker auto-detection**: automatically enables `0.0.0.0` binding and `--no-sandbox`.
- **Playwright proxy**: dashboard, API, and Playwright WebSocket proxy all use port `19280`.
- **MCP HTTP**: Streamable HTTP MCP uses `19280/mcp` with the same Bearer token as the REST API.

## Notes

- The GHCR Docker image currently publishes `linux/amd64`. Apple Silicon and ARM servers run it through emulation.
- VNC is intended for watching browser state and basic remote operation.
- Browser engines, profiles, token data, logs, and backups are host-mounted under `./browseforge/` by default, so recreating the container does not delete them.

## Apple Silicon (M1/M2/M3)

`docker-compose.yml` sets `platform: linux/amd64` and runs through Rosetta/QEMU emulation.

Native `linux/arm64` images should only be enabled after KasmVNC, Camoufox, and CloakBrowser have all passed runtime validation inside an ARM container.
