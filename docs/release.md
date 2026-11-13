# Release Process

BrowseForge releases are tag driven. Do not create or push release tags by hand.

## Prepare

1. Update the version references:

   - `docker/docker-compose.yml`
   - `docker/README.md`
   - `docs/linux-server.md`
   - `docs/linux-server.zh-TW.md`
   - `docs/platform-support.md`
   - `docs/platform-support.zh-TW.md`
   - `README.md`
   - `README.zh-TW.md`

2. Commit and push `v1`.

3. Run preflight from a clean `v1` checkout:

   ```bash
   scripts/release-preflight.sh v1.10.8
   ```

The preflight checks the working tree, version references, Go tests, vet, UI i18n syntax, public documentation language, workflow YAML, Docker Compose config, the Camoufox Bind spike, the CloakBrowser Bind spike when a local binary is available, and a real `linux/amd64` Docker build.

For release machines that must prove both browser engines before publishing, enforce the CloakBrowser spike:

```bash
REQUIRE_CLOAKBROWSER=1 \
CLOAKBROWSER_PATH=/path/to/Chromium \
scripts/release-preflight.sh v1.10.8
```

## Publish

After preflight passes:

```bash
scripts/release-push.sh v1.10.8
gh run watch
```

The pushed tag starts the GitHub release workflow:

1. Verify
2. Cross-platform binary packages
3. GitHub release asset verification
4. GitHub release
5. GHCR Docker image

## Platform Policy

The release workflow publishes binary zip assets for:

- `linux-x64`
- `linux-arm64`
- `macos-x64`
- `macos-arm64`
- `windows-x64`

The Docker image is currently published as `linux/amd64` only. Apple Silicon runs it through Docker emulation. Native `linux/arm64` Docker images should only be enabled after KasmVNC, Camoufox, and CloakBrowser runtime checks pass inside an ARM container.
