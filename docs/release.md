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

2. Commit and push `main`.

3. Run preflight from a clean `main` checkout:

   ```bash
   scripts/release-preflight.sh v2.1.13
   ```

The preflight checks the working tree, version references, Go tests, vet, UI i18n syntax, public documentation language, workflow YAML, Docker Compose config, the Camoufox Bind spike, the CloakBrowser Bind spike when a local binary is available, BrowseForge Chromium runtime release asset availability for `linux-x64` and `linux-arm64`, and real `linux/amd64` plus `linux/arm64` Docker builds.

For release machines that must prove both browser engines before publishing, enforce the CloakBrowser spike:

```bash
REQUIRE_CLOAKBROWSER=1 \
CLOAKBROWSER_PATH=/path/to/Chromium \
scripts/release-preflight.sh v2.1.13
```

If the runtime release assets are staged outside GitHub while validating a coordinated BrowseForge/runtime release, point the preflight at that release-asset root. The URL must contain `checksums.txt`, `runtime.manifest.json`, and `<runtime-version>/browseforge-runtime-chromium-<runtime-version>-linux-{x64,arm64}.zip`:

```bash
BROWSEFORGE_CHROMIUM_RELEASE_BASE_URL=https://host/runtime/releases \
scripts/release-preflight.sh v2.1.13
```

To check only the BrowseForge Chromium release assets without building Docker images:

```bash
BROWSEFORGE_CHROMIUM_RELEASE_BASE_URL=https://host/runtime/releases \
scripts/check-browseforge-chromium-assets.sh
```

## Publish

After preflight passes:

```bash
scripts/release-push.sh v2.1.13
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

The Docker image is published as a multi-arch manifest for `linux/amd64` and `linux/arm64`. Docker pulls the host-native image automatically on x64 servers, Apple Silicon, and ARM servers; use `--platform linux/amd64` only for compatibility debugging.
