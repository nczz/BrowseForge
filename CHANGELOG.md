# Changelog

All notable changes should be documented here. This project follows semantic version tags in the form `vX.Y.Z`.

## Unreleased

Nothing yet.

## v1.10.3 - 2026-08-04

### Fixed

- Validated Playwright Bind endpoints before registering browser sessions so external AutoPost-style clients no longer receive unusable WebSocket endpoints after a launch, crash, or handshake failure.
- Added structured endpoint health failure codes and diagnostic logs with session, profile, engine, endpoint, profile directory, user-data directory, executable path, timeout, and retry context.

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
