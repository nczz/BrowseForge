# Changelog

All notable changes should be documented here. This project follows semantic version tags in the form `vX.Y.Z`.

## Unreleased

## v2.1.14 - 2026-09-02

### Fixed

- Fixed Camoufox Playwright Bind endpoint health checks by disabling Playwright viewport initialization during Camoufox probe pages, allowing Camoufox v135 sessions to start without `Browser.setDefaultViewport` errors.

## v2.1.13 - 2026-08-22

### Fixed

- Added Dashboard, REST, and MCP proxy-region controls and validation so BrowseForge Chromium profiles can configure required proxy persona metadata before launch.
- Made the release preflight asset server avoid blocking host-name lookups before Docker build verification.

## v2.1.12 - 2026-08-04

### Fixed

- Validated Playwright Bind endpoints before registering browser sessions so external clients no longer receive unusable WebSocket endpoints after a launch, crash, or handshake failure.
- Added structured endpoint health failure codes and diagnostic logs with session, profile, runtime, endpoint, profile directory, user-data directory, executable path, timeout, and retry context.

## v2.1.11 - 2026-07-27

### Fixed

- Made MCP `screenshot` URL delivery work from stdio when `public_base_url` is configured by sharing temporary screenshot artifacts through the main HTTP service.

## v2.1.10 - 2026-07-27

### Fixed

- Added temporary unauthenticated MCP screenshot URLs so remote HTTP agents can fetch image bytes directly instead of parsing base64 image blocks.
- Documented `BROWSEFORGE_PUBLIC_BASE_URL` Docker configuration and the screenshot URL delivery contract for remote agents.

## v2.1.9 - 2026-07-20

### Fixed

- Added a validated MCP default web profile so `web_search` and `web_explore` can start without an explicit profile ID while preserving agent-capable Chromium runtime checks.
- Bumped Docker, Linux server, release, and platform documentation references for the `v2.1.9` release tag.

## v2.1.8 - 2026-07-19

### Fixed

- Pointed BrowseForge to BrowseForge Chromium runtime `v0.1.6-alpha.0` for the BrowserLeaks Fonts crash-fix runtime release.
- Bumped Docker, Linux server, release, and platform documentation references for the `v2.1.8` release tag.

## v2.1.7 - 2026-07-16

### Fixed

- Pointed BrowseForge to BrowseForge Chromium runtime `v0.1.5-alpha.0` after the detector evidence hardening release.
- Bumped Docker, Linux server, release, and platform documentation references for the `v2.1.7` release tag.

## v2.1.6 - 2026-07-16

### Fixed

- Bumped Docker, Linux server, release, and platform documentation references for the `v2.1.6` release tag.
- Aligned BrowseForge Chromium Docker/native headed sessions with the native persona screen contract, including DPR switches, coherent KasmVNC geometry defaults, and smoke-test assertions for screen/window, locale, timezone, WebDriver, and WebGL surfaces.
- Captured sanitized Pixelscan screen/window evidence so detector reports can distinguish true fingerprint mismatches from missing viewport evidence.

## v2.1.5 - 2026-07-16

### Fixed

- Preserved `DISPLAY`, `HOME`, and `LIBGL_ALWAYS_SOFTWARE` when launching BrowseForge Chromium so GHCR Docker sessions can use the KasmVNC X display instead of failing with a missing-X-server error.
- Bumped Docker and Linux server references for the `v2.1.5` release tag.

## v2.1.4 - 2026-07-16

### Changed

- Published GHCR release images as native `linux/amd64` and `linux/arm64` manifests backed by BrowseForge Chromium runtime `v0.1.4-alpha.0`.
- Updated Docker and Linux server references for the `v2.1.4` release tag.
- Kept the runtime release gate strict: Docker preinstall verifies the matching linux runtime asset before image build.

### Fixed

- Added native BrowseForge Chromium `linux/arm64` download mapping and runtime asset checks so ARM containers no longer fall back to x64/emulation assets.
- Preserved coherent native persona metadata for BrowseForge Chromium launch on Linux, macOS, and Windows runtime packages.

## v2.1.2 - 2026-07-13

### Changed

- Updated BrowseForge to the maintained `github.com/mxschmitt/playwright-go` driver package.
- Preinstalled the Playwright control driver in GHCR Docker images so container startup does not depend on runtime downloads from npm or nodejs.org.
- Bumped Docker, Linux server, release, and platform documentation references to the `v2.1.2` release tag.
- Decoupled BrowseForge release support from browser runtime release support so unsupported runtime/platform pairs are skipped and disabled at runtime instead of blocking BF startup.
- Kept Camoufox runtime selection on `v135.0.1-beta.24` because it publishes complete browser binaries for the supported native platforms, and updated BrowseForge Chromium runtime selection to `v0.1.2-alpha.0`.
- Added a shared BrowseForge Chromium release-asset checker plus release-preflight, workflow, and Docker build override support for staged runtime asset roots, and fail release gates before Docker build when the selected runtime version lacks `linux-x64` or `linux-arm64` release assets.

### Fixed

- Kept BrowseForge Chromium launch identity coherent on native Linux ARM64 by deriving UA, UA-CH architecture, `navigator.platform`, and `Accept-Language` from the resolved runtime platform/locale instead of blindly reusing incompatible pool values.
- Populated BrowseForge Chromium native persona JSON from the same canonical launch persona used for command-line fingerprint switches, including browser, platform, locale, hardware, screen, GPU, WebRTC, and storage fields.
- Rejected invalid BrowseForge Chromium proxy region labels before launch and clamped screen available dimensions so emitted switches and native JSON stay coherent.

## v2.1.1 - 2026-07-13

### Changed

- Prepared GHCR Docker publishing for native `linux/amd64` and `linux/arm64` manifests.
- Updated Docker runtime builds to select BrowseForge release ZIPs and KasmVNC packages from the BuildKit target architecture.
- Bumped Docker, Linux server, release, and platform documentation references to the `v2.1.1` release tag.

### Fixed

- Added BrowseForge Chromium downloader support for `linux/arm64` so ARM64 containers install the native `linux-arm64` runtime asset instead of failing during browser preinstall.

## v2.1.0 - 2026-07-12

### Changed

- Updated release and Docker references for the `v2.1.0` release tag.
- Kept Camoufox plus BrowseForge Chromium as the GHCR/Docker preinstall default while retaining CloakBrowser as a manual/custom runtime.

### Added

- Verified BrowseForge can download and install the released BrowseForge Chromium `v0.1.0-alpha.0` runtime asset from GitHub Releases.

## v2.0.0 - 2026-07-07

### Changed

- Replaced the public profile browser-family contract with explicit runtime providers. Profile create/update APIs, MCP profile tools, workflows, dashboard forms, and profile storage now use `runtime_id` values such as `camoufox` and `cloakbrowser`.
- Moved runtime binary paths and CloakBrowser launch/fingerprint settings under `runtimes.<id>` configuration, with `default_runtime_id` for UI defaults and `/api/runtimes` metadata.
- Updated Docker, Linux server, release, and platform documentation for the `v2.0.0` release tag.

### Added

- Added `BrowseForge migrate profiles --from v1 --to v2 [--apply]` to migrate legacy profile JSON safely, including original-file `.v1.bak` backups for every rewritten profile.
- Added runtime capability metadata so API, MCP web sessions, dashboard, and browser manager code gate behavior on provider capabilities instead of hard-coded engine strings.

### Fixed

- Rejected deprecated `engine` profile create/update inputs at REST and MCP boundaries, rejected non-string or disabled `runtime_id` updates, and prevented dashboard selection of disabled runtimes before metadata loads.
- Preserved anti-detection ownership of managed CloakBrowser fingerprint flags, Camoufox WebGL normalization, and large `CAMOU_CONFIG` chunking after the runtime-provider refactor.

## v1.10.2 - 2026-07-05

### Fixed

- Strengthened CloakBrowser safe GPU fallback for Windows VM environments by adding GPU sandbox bypass and in-process GPU launch flags. This covers hosts where `--disable-gpu` alone still fails with `GPU process isn't usable`.

## v1.10.1 - 2026-07-05

### Added

- Added opt-in CloakBrowser launch compatibility settings for Windows VM environments, including safe GPU launch flags, isolated runtime cache, transient cache repair, and sanitized extra Chromium args.

### Fixed

- CloakBrowser launches can now retry once with a safe GPU fallback after GPU/cache startup failures such as `GPU process isn't usable` or `Unable to create cache`, without changing the first-launch anti-detection behavior by default.
- Safe GPU fallback restarts the Playwright driver cleanly, drops stale sessions after the restart, and avoids repeated manager-level retries once the fallback path has already been exhausted.

## v1.9.0 - 2026-07-01

### Added

- Added agent-ready CLI commands for runtime status, dashboard opening, MCP client config generation, browser engine status/install, full filesystem backups, metadata backups, and restore workflows.
- Added wait-aware CLI smoke checks so local and container deployments can block until REST or MCP endpoints are ready.
- Added local quickstart, cloud deployment, agent integration, and developer integration guides.

### Changed

- Docker release images now preinstall BrowseForge-managed browser engines during image build by default.
- Container startup now seeds or updates `/app/browsers` from the image when the mounted browser cache is missing or its engine version differs, while keeping tokens, profiles, data, logs, and backups on host-mounted paths.
- Docker and Linux server documentation now include persistent `/app/backups` mounts and explicit browser-cache upgrade behavior.

## v1.8.1 - 2026-06-30

### Changed

- Playwright Go now uses the upstream community `v0.6000.0` release instead of the temporary `nczz/playwright-go` integration fork.

## v1.7.7 - 2026-05-15

### Fixed

- Playwright driver compatibility patching is now format tolerant, so Firefox/Camoufox page errors without source locations no longer crash the driver in Docker builds.
- BrowseForge startup now fails fast if Playwright driver installation or patching fails instead of silently running an unpatched driver.

## v1.7.6 - 2026-05-15

### Fixed

- Playwright 1.60 driver crashes caused by Firefox/Camoufox page errors without source locations no longer poison later browser launches.
- Chromium/CloakBrowser launches now recover from stale Playwright driver protocol failures even when dead sessions remain in memory.

## v1.7.5 - 2026-05-15

### Fixed

- Firefox/Camoufox profile launches now clean stale profile locks before startup and automatically restart the Playwright driver once after recoverable protocol EOF errors.
- Spike tests no longer attempt to download Playwright-managed browsers during `go test ./...`.

## v1.7.4 - 2026-05-15

### Fixed

- Dashboard version status now remains visible after language initialization or locale changes instead of being overwritten by the translated connecting label.

## v1.7.3 - 2026-05-15

### Fixed

- Release binaries now report the tag version through the REST API, doctor output, and MCP initialize response.
- Release workflow verifies the packaged Linux binary version before publishing assets.

## v1.7.2 - 2026-05-15

### Added

- Guarded release scripts and release workflow asset checks.
- Docker release build hardening with pinned release artifact selection and KasmVNC checksum verification.
- Community governance, support, security, and contribution documentation.
- Initial application i18n policy and locale structure.
- English-first README and API reference with Traditional Chinese counterparts.
- English public docs for platform support, Linux server deployment, and Playwright patch status.
- i18n coverage checker for Dashboard and WebExtension locale key parity.
- Marketing-oriented product positioning, audience, trust, and deployment messaging in README.
- Dual-browser anti-detection architecture documentation in English and Traditional Chinese.
- Opt-in CloakBrowser runtime spike harness for the Playwright Bind endpoint path.

### Changed

- Docker documentation recommends pinning version tags for production deployments.
- Replaced remaining early Camoufox-only tool naming in local scripts and clarified current dual-browser fingerprint behavior.
- Release preflight runs the CloakBrowser Bind spike when a local binary is available and can enforce it with `REQUIRE_CLOAKBROWSER=1`.
- Release preflight keeps base Go tests separate from explicit browser runtime spike gates.

## v1.7.0 - 2026-05-15

### Added

- Playwright 1.60 integration through the project fork.
- MCP Streamable HTTP authentication with Bearer tokens.
- Camoufox runtime spike coverage for the Playwright Bind endpoint path.

### Changed

- Removed the previous Playwright 1.59.1 hotfix path and now uses the Playwright 1.60 `browser.Bind()` endpoint directly.
- Improved startup, token, browser-download, profile-store, backup/restore, and session request error handling.

### Upgrade Notes

- External Playwright clients should use Playwright 1.60.x for `browserType.connect()`.
- Existing `config.json`, `data/.api-token`, and `profiles/` remain compatible.
- MCP HTTP clients must send `Authorization: Bearer <token>`.
