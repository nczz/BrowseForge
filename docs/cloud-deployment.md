# Cloud Deployment

Use Docker with host bind mounts for production. The host directory is the durable runtime; the container is replaceable.

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
  ghcr.io/nczz/browseforge:v2.1.13
```

## First Startup

The Docker image is designed to preinstall Camoufox and BrowseForge Chromium at image build time and update `/app/browsers` from the image when the mounted engine is missing or its `.version` differs. This makes browser engines part of the BrowseForge image contract while keeping CloakBrowser available for manual/custom installs. If the image was built from an older BrowseForge binary, browser preinstall was disabled, `BROWSEFORGE_PREINSTALL_RUNTIMES` names a runtime not present in the image, or `BROWSEFORGE_SEED_BROWSERS=0` is set, the first startup may spend several minutes downloading browsers before the dashboard responds.

Check readiness:

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --runtimes camoufox,browseforge-chromium --json
docker exec browseforge /app/BrowseForge smoke rest --wait --json
```

## Upgrade

```bash
docker pull ghcr.io/nczz/browseforge:v2.1.13
docker stop browseforge
docker rm browseforge
# Re-run docker run with the same -v "$PWD/browseforge/...:/app/..." mounts.
```

Profiles, token, browser user data, browser engines, and logs remain in `./browseforge/`.

## Backup

Full backup:

```bash
docker stop browseforge
tar -czf ./browseforge/backups/browseforge-runtime-$(date +%Y%m%d-%H%M%S).tgz ./browseforge/profiles ./browseforge/data ./browseforge/browsers ./browseforge/logs
docker start browseforge
```

Convenience backup while the container is running:

```bash
docker exec browseforge /app/BrowseForge backup create --full --output /app/backups --json
```

`/app/backups` maps to the host `./browseforge/backups` directory in the recommended Docker command.

Metadata backup:

```bash
TOKEN=$(docker exec browseforge /app/BrowseForge token)
curl -fsS -X POST http://127.0.0.1:19280/api/backup \
  -H "Authorization: Bearer $TOKEN" \
  -o ./browseforge/backups/browseforge-api-backup-$(date +%Y%m%d).zip
```

Use full backups when you need complete browser user data and token preservation.
