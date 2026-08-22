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

Build from the local source tree:

```bash
cd docker
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}
docker compose up -d --build
```

The Compose file builds the `v2.1.13` release image by default. To test another version:

```bash
BROWSEFORGE_VERSION=v2.1.13 docker compose up -d --build
```

When validating an unreleased BrowseForge Chromium runtime, point the image build at a staged runtime asset root. The root must contain `checksums.txt`, `runtime.manifest.json`, and `<runtime-version>/browseforge-runtime-chromium-<runtime-version>-linux-{x64,arm64}.zip`:

```bash
BROWSEFORGE_CHROMIUM_RELEASE_BASE_URL=https://host/runtime/releases \
BROWSEFORGE_VERSION=v2.1.13 docker compose up -d --build
```

## First Startup

Current release images install Camoufox and BrowseForge Chromium during the Docker build. On startup, BrowseForge seeds the host-mounted `/app/browsers` cache from the image when `/app/browsers/{engine}/.version` is missing or differs from the packaged version. CloakBrowser remains available for manual/custom installs but is no longer part of the default GHCR preinstall set.

The first startup may still take 3-5 minutes and download browser engines when you use an older image, disable `BROWSEFORGE_PREINSTALL_BROWSERS`, set `BROWSEFORGE_PREINSTALL_RUNTIMES` to a runtime not already present in the image, or set `BROWSEFORGE_SEED_BROWSERS=0`. During that window, the dashboard may not be ready yet.

Use these commands to verify startup state:

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --runtimes camoufox,browseforge-chromium --json
docker exec browseforge /app/BrowseForge smoke rest --wait --timeout 5m --json
```

## Connections

| Service | URL |
|---------|-----|
| Dashboard + REST API + Playwright proxy | http://localhost:19280 |
| MCP Streamable HTTP | http://localhost:19280/mcp |
| Remote desktop (KasmVNC) | http://localhost:6901 |
| VNC login | `user` / `VNC_PASSWORD` environment variable, default `browseforge` |

The quick Docker command binds BrowseForge inside the container while defaulting `BROWSEFORGE_PUBLIC_BASE_URL` to `http://localhost:19280`, a fetchable origin for same-host agents. For remote servers or reverse proxies, export `BROWSEFORGE_PUBLIC_BASE_URL` with the externally reachable origin, including any path prefix.

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
| `./browseforge/browsers` | `/app/browsers` | Downloaded or seeded Camoufox, BrowseForge Chromium, and optional CloakBrowser engines. |
| `./browseforge/logs` | `/app/logs` | Server logs. |
| `./browseforge/backups` | `/app/backups` | Filesystem and API backup output. |

When you pull a new image or recreate the container, reuse the same `-v "$PWD/browseforge/...:/app/..."` mounts. This keeps the API token, profiles, browser user data, browser cache, logs, and backups outside the container lifecycle.

`/app/browsers` is a BrowseForge-managed browser cache. The default `BROWSEFORGE_SEED_BROWSERS=1` updates it to the browser versions packaged in the image. `BROWSEFORGE_PREINSTALL_RUNTIMES` controls the build-time set and defaults to `camoufox,browseforge-chromium`. Set seeding to `0` only for debugging cases where you intentionally want to preserve a manually installed browser engine.

Upgrade example:

```bash
docker pull ghcr.io/nczz/browseforge:v2.1.13
docker stop browseforge
docker rm browseforge
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
- **Auditable GPU policy**: software mode uses SwiftShader-aligned WebGL metadata; native/passthrough keep browser-default GPU evidence only when the profile does not explicitly provide WebGL vendor/renderer, and they do not claim fixed WebGL2/extensions/limits.
- **Docker auto-detection**: automatically enables `0.0.0.0` binding and `--no-sandbox`.
- **Playwright proxy**: dashboard, API, and Playwright WebSocket proxy all use port `19280`.
- **MCP HTTP**: Streamable HTTP MCP uses `19280/mcp` with the same Bearer token as the REST API.
- **Agent-ready screenshot URLs**: set `BROWSEFORGE_PUBLIC_BASE_URL` so MCP `screenshot` can return temporary, unauthenticated, externally reachable URLs instead of base64 image blocks.

## Notes

- The GHCR Docker image publishes native `linux/amd64` and `linux/arm64` manifests. Docker automatically pulls the host-native image on x64 servers, Apple Silicon, and ARM servers.
- Native architecture support removes amd64 emulation from the Docker path. Public fingerprint detector outcomes still depend on GPU/rendering, fonts, locale, proxy, and runtime profile coherence.
- Native `linux/arm64` Docker support requires the matching BrowseForge Chromium `linux-arm64` runtime artifact. Release and Docker preinstall checks fail when the native artifact is missing instead of falling back to an x64/emulated runtime.
- Default GHCR browser preinstall is Camoufox plus BrowseForge Chromium. Use `BROWSEFORGE_PREINSTALL_RUNTIMES` at image build time if you need a different set.
- Docker defaults to coherent software GL (`BROWSEFORGE_DOCKER_GPU_MODE=software`): BrowseForge Chromium reports SwiftShader-aligned WebGL strings instead of pretending a host GPU exists. `native` keeps browser-default GPU evidence when no profile WebGL vendor/renderer is provided, `passthrough` is reserved for deliberate host GPU passthrough, and both modes omit fixed WebGL2/extensions/limits. Any other value fails closed instead of silently drifting.
- KasmVNC defaults to `1920x1080`, matching the BrowseForge Chromium native screen persona. Override both `BROWSEFORGE_DOCKER_DISPLAY_WIDTH` and `BROWSEFORGE_DOCKER_DISPLAY_HEIGHT` only when you also want headed browser window/screen evidence to follow that custom desktop size.
- VNC is intended for watching browser state and basic remote operation.
- Browser engines, profiles, token data, logs, and backups are host-mounted under `./browseforge/` by default, so recreating the container does not delete them.

## Apple Silicon (M1/M2/M3)

Docker pulls the native `linux/arm64` image on Apple Silicon and ARM servers by default. Use `docker run --platform linux/amd64 ...` or `DOCKER_DEFAULT_PLATFORM=linux/amd64 docker compose up -d --build` only when you intentionally need x64 compatibility debugging.
