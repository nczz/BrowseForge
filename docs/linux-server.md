# Linux Server Deployment

[繁體中文](linux-server.zh-TW.md)

## Recommended: Docker

```bash
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}

docker run -d --name browseforge \
  -p 19280:19280 -p 6901:6901 \
  -e VNC_PASSWORD=browseforge \
  -e BROWSEFORGE_PUBLIC_BASE_URL=${BROWSEFORGE_PUBLIC_BASE_URL:-http://localhost:19280} \
  -v "$PWD/browseforge/profiles:/app/profiles" \
  -v "$PWD/browseforge/data:/app/data" \
  -v "$PWD/browseforge/browsers:/app/browsers" \
  -v "$PWD/browseforge/logs:/app/logs" \
  -v "$PWD/browseforge/backups:/app/backups" \
  -e BROWSEFORGE_SEED_BROWSERS=1 \
  --restart unless-stopped \
  ghcr.io/nczz/browseforge:v2.1.13
```

Pin a version tag such as `v2.1.13` for production deployments. Use `latest` only for short trials, because pulling or restarting later may upgrade unexpectedly.

The quick command binds BrowseForge inside the container while defaulting `BROWSEFORGE_PUBLIC_BASE_URL` to `http://localhost:19280`, a fetchable origin for same-host agents. For remote agents, export `BROWSEFORGE_PUBLIC_BASE_URL` to the externally reachable server origin before running the command.

> The GHCR Docker image publishes native `linux/amd64` and `linux/arm64` manifests. Docker pulls the host-native image automatically on x64 servers, Apple Silicon, and ARM servers. Use `--platform linux/amd64` only for compatibility debugging.

Docker defaults to software GL for container portability and WebGL coherence. Leave `BROWSEFORGE_DOCKER_GPU_MODE=software` unless the host deliberately exposes a real GPU path. Use `native` for browser-default GPU evidence with an explicitly configured GPU environment, or `passthrough` for deliberate host GPU passthrough; unsupported values fail closed.

## First Startup and Browser Engines

BrowseForge needs Camoufox and BrowseForge Chromium engines before the dashboard can become ready. Current Docker images preinstall these engines during image build and seed `/app/browsers` on startup when the host-mounted engine is missing or its `.version` differs from the image. This keeps browser engines aligned with the BrowseForge image version. If the image was built from an older binary, browser preinstall was disabled, or `BROWSEFORGE_SEED_BROWSERS=0` is set, startup may spend several minutes downloading engines before `19280` responds.

Use logs and wait-aware smoke checks instead of assuming the container is broken:

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --json
docker exec browseforge /app/BrowseForge smoke rest --wait --timeout 5m --json
```

For custom images, keep browser preinstall enabled with the default `BROWSEFORGE_PREINSTALL_BROWSERS=1`. For special debugging cases, set `BROWSEFORGE_SEED_BROWSERS=0` to stop the container from updating `/app/browsers` from the image.

| Service | URL |
|------|-----|
| Dashboard + REST API + Playwright proxy | `http://YOUR_SERVER:19280` |
| MCP Streamable HTTP | `http://YOUR_SERVER:19280/mcp` |
| KasmVNC remote desktop | `http://YOUR_SERVER:6901` |
| VNC credentials | `user` / `VNC_PASSWORD` |

## Get the API Token

```bash
docker logs browseforge | grep "API Token"
docker exec browseforge /app/BrowseForge token
```

## Persistent Data

Production deployments should mount runtime data to host directories. This keeps profile data, the API token, downloaded browser engines, and logs outside the container so `docker pull`, `docker stop`, `docker rm`, and `docker run` do not delete user content.

```bash
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}

docker run -d --name browseforge \
  -p 19280:19280 -p 6901:6901 \
  -e VNC_PASSWORD=browseforge \
  -e BROWSEFORGE_PUBLIC_BASE_URL=${BROWSEFORGE_PUBLIC_BASE_URL:-http://localhost:19280} \
  -v "$PWD/browseforge/profiles:/app/profiles" \
  -v "$PWD/browseforge/data:/app/data" \
  -v "$PWD/browseforge/browsers:/app/browsers" \
  -v "$PWD/browseforge/logs:/app/logs" \
  -v "$PWD/browseforge/backups:/app/backups" \
  -e BROWSEFORGE_SEED_BROWSERS=1 \
  --restart unless-stopped \
  ghcr.io/nczz/browseforge:v2.1.13
```

Persisted paths:

| Host path | Container path | Purpose |
|-----------|----------------|---------|
| `./browseforge/profiles` | `/app/profiles` | Profile metadata and browser user data. |
| `./browseforge/data` | `/app/data` | API token at `.api-token` and fingerprint data. |
| `./browseforge/browsers` | `/app/browsers` | Downloaded or seeded Camoufox, BrowseForge Chromium, and optional CloakBrowser engines. |
| `./browseforge/logs` | `/app/logs` | Server logs. |
| `./browseforge/backups` | `/app/backups` | Filesystem and API backup output. |

Named Docker volumes also work, but host bind mounts are easier to inspect, copy, snapshot, and back up.

## Docker Compose

```yaml
services:
  browseforge:
    image: ghcr.io/nczz/browseforge:v2.1.13
    ports:
      - "19280:19280"
      - "6901:6901"
    volumes:
      - ./browseforge/profiles:/app/profiles
      - ./browseforge/data:/app/data
      - ./browseforge/browsers:/app/browsers
      - ./browseforge/logs:/app/logs
      - ./browseforge/backups:/app/backups
    environment:
      - VNC_PASSWORD=browseforge
      - BROWSEFORGE_SEED_BROWSERS=1
    restart: unless-stopped
```

## Firewall

| Port | Purpose | Required |
|------|---------|----------|
| 19280 | Dashboard + REST API + MCP Streamable HTTP + Playwright WebSocket proxy | Required |
| 6901 | KasmVNC remote desktop | Optional, only when visual access is needed |

```bash
sudo ufw allow 19280/tcp
sudo ufw allow 6901/tcp
```

## Security

- Do not expose `19280` or `6901` directly to the public internet.
- Use SSH tunnels, VPN, or a hardened HTTPS reverse proxy.
- KasmVNC uses Basic Auth through `user` and `VNC_PASSWORD`.
- API and MCP share the Bearer token stored in `data/.api-token`.
- Treat profiles, backup ZIPs, exported profiles, cookies, and tokens as sensitive.

Recommended SSH tunnel:

```bash
ssh -L 19280:localhost:19280 -L 6901:localhost:6901 user@server
```

Then open:

- `http://localhost:19280`
- `http://localhost:19280/mcp`
- `http://localhost:6901`

## Upgrade

```bash
docker pull ghcr.io/nczz/browseforge:v2.1.13
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
  ghcr.io/nczz/browseforge:v2.1.13
```

Profiles, tokens, downloaded browser engines, and logs remain in the host `./browseforge/` directory. Pulling a new image and recreating the container must reuse the same bind mounts.

## Backup

For the safest full backup that includes browser user data, stop the container and archive the host runtime directory:

```bash
docker stop browseforge
tar -czf ./browseforge/backups/browseforge-runtime-$(date +%Y%m%d-%H%M%S).tgz ./browseforge/profiles ./browseforge/data ./browseforge/browsers ./browseforge/logs
docker start browseforge
```

For an operator-friendly full backup command while the container is running:

```bash
docker exec browseforge /app/BrowseForge backup create --full --output /app/backups --json
```

Prefer the stopped-container archive before major upgrades or migrations.

For a lighter metadata-level backup through the REST API:

```bash
TOKEN=$(docker exec browseforge /app/BrowseForge token)
curl -fsS -X POST http://127.0.0.1:19280/api/backup \
  -H "Authorization: Bearer $TOKEN" \
  -o ./browseforge/backups/browseforge-api-backup-$(date +%Y%m%d).zip
```

The REST backup is useful for profile metadata import/export. Use the filesystem backup when you need to preserve complete browser user data and the API token.

## Runtime Features

- KasmVNC remote desktop with browser viewing/control.
- GLX and software rendering for WebGL behavior in containerized environments.
- Docker auto-detection for `0.0.0.0` binding and `--no-sandbox`.
- Playwright WebSocket proxy through port `19280`.
- MCP Streamable HTTP through `19280/mcp` with Bearer token authentication.
