# BrowseForge Platform Support Matrix

[繁體中文](platform-support.zh-TW.md)

This matrix defines the currently supported platforms and browser-runtime availability.

## Current Versions

| Component | Version | Updated |
|------|------|---------|
| BrowseForge | v1.10.5 | 2026-11-12 |
| Camoufox | v135.0.1-beta.24 | 2025-03-15 |
| CloakBrowser macOS | chromium-v145.0.7632.109.2 | 2026-03-04 |
| CloakBrowser Linux/Windows | chromium-v146.0.7680.177.4 | 2026-04-28 |

## Support Matrix

| Platform | BrowseForge | Camoufox | CloakBrowser | Notes |
|------|:---:|:---:|:---:|------|
| macOS x64 (Intel) | Supported | v135 | v145 | Native binary |
| macOS arm64 (Apple Silicon) | Supported | v135 | v145 | Native binary |
| Linux x64 | Supported | v135 | v146 | Display server or Docker runtime required |
| Linux arm64 | Binary supported | v135 | v146 | CloakBrowser requires the v146 Linux arm64 build |
| Windows x64 | Supported | v135 | v146 | Native binary |
| Windows i686 (32-bit) | Not supported | v135 available | Not supported | BrowseForge does not publish 32-bit builds |
| Linux i686 (32-bit) | Not supported | v135 available | Not supported | BrowseForge does not publish 32-bit builds |

## Browser Runtime Selection

The default browser runtime versions prioritize the newest build that is still usable across BrowseForge's supported platforms.

- Camoufox `v150.0.2-beta.25` is newer than `v135.0.1-beta.24`, but its upstream release does not provide the full platform set used by BrowseForge. Keep `v135.0.1-beta.24` until a newer Camoufox release has macOS x64/arm64, Linux x64/arm64, and Windows x64 assets that pass runtime validation.
- CloakBrowser `chromium-v146.0.7680.177.5` is newer than `.4`, but upstream publishes it for Linux x64 and Windows x64 only; Linux arm64 remains on an earlier v146 build and macOS remains on v145. Keep BrowseForge on `chromium-v146.0.7680.177.4` for Linux/Windows and `chromium-v145.0.7632.109.2` for macOS unless a future release restores a cleaner cross-platform set.

## Docker Platform Policy

The published GHCR Docker image is currently `linux/amd64`.

Apple Silicon and ARM servers can run the Docker image through emulation. Native `linux/arm64` Docker images should only be enabled after KasmVNC, Camoufox, and CloakBrowser runtime checks pass inside an ARM container.

## Download URL Reference

### Camoufox v135.0.1-beta.24

```text
macOS x64:    https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-mac.x86_64.zip
macOS arm64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-mac.arm64.zip
Linux x64:    https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-lin.x86_64.zip
Linux arm64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-lin.arm64.zip
Windows x64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-win.x86_64.zip
```

### CloakBrowser macOS chromium-v145.0.7632.109.2

```text
macOS x64:    https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v145.0.7632.109.2/cloakbrowser-darwin-x64.tar.gz
macOS arm64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v145.0.7632.109.2/cloakbrowser-darwin-arm64.tar.gz
```

### CloakBrowser Linux/Windows chromium-v146.0.7680.177.4

```text
Linux x64:    https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-linux-x64.tar.gz
Linux arm64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-linux-arm64.tar.gz
Windows x64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-windows-x64.zip
```

## Upgrade Checklist

When updating browser runtime versions:

- [ ] Confirm that each supported platform has a compatible browser binary.
- [ ] Update versions and URLs in `internal/browser/download.go`.
- [ ] Update this matrix.
- [ ] Test browser auto-download.
- [ ] Test anti-detection behavior with BrowserLeaks and SannySoft.
- [ ] Run `scripts/release-preflight.sh vX.Y.Z`.
- [ ] Run `scripts/release-push.sh vX.Y.Z`.
