# BrowseForge Platform Support Matrix

[繁體中文](platform-support.zh-TW.md)

This matrix defines the currently supported platforms and browser-runtime availability.

## Current Versions

| Component | Version | Updated |
|------|------|---------|
| BrowseForge | v2.1.13 | 2026-08-22 |
| Camoufox | v135.0.1-beta.24 | 2025-03-15 |
| CloakBrowser macOS | chromium-v145.0.7632.109.2 | 2026-03-04 |
| CloakBrowser Linux/Windows | chromium-v146.0.7680.177.4 | 2026-04-28 |
| BrowseForge Chromium | v0.1.6-alpha.0 | 2026-07-16 |

## Support Matrix

| Platform | BrowseForge | Camoufox | CloakBrowser | BrowseForge Chromium | Notes |
|------|:---:|:---:|:---:|:---:|------|
| macOS x64 (Intel) | Supported | v135 | v145 | Alpha artifact | Native binaries |
| macOS arm64 (Apple Silicon) | Supported | v135 | v145 | Alpha artifact | Native binaries |
| Linux x64 | Supported | v135 | v146 | Alpha artifact | Display server or Docker runtime required |
| Linux arm64 | Supported | v135 | v146 | Alpha artifact | Native binary; GHCR publishes a `linux/arm64` image |
| Windows x64 | Supported | v135 | v146 | Alpha artifact | Native binaries |
| Windows i686 (32-bit) | Not supported | Not supported | Not supported | Not packaged | BrowseForge does not publish 32-bit builds |
| Linux i686 (32-bit) | Not supported | Not supported | Not supported | Not packaged | BrowseForge does not publish 32-bit builds |

## Browser Runtime Selection

BrowseForge releases are independent from browser runtime releases. At startup and during `browsers install`, BrowseForge enables and downloads only the runtimes that support the current `GOOS/GOARCH`; unsupported runtimes are skipped and reported as disabled.

- Camoufox stays on `v135.0.1-beta.24` because that upstream release publishes complete browser binaries for macOS x64/arm64, Linux x64/arm64, and Windows x64. The newer `v152.0.2-alpha` public ZIPs currently contain the fingerprint/font payload but not a runnable browser binary, so BrowseForge does not select them for runtime downloads.
- CloakBrowser `chromium-v146.0.7680.177.5` is newer than `.4`, but upstream publishes it for Linux x64 and Windows x64 only; Linux arm64 remains on an earlier v146 build and macOS remains on v145. Keep BrowseForge on `chromium-v146.0.7680.177.4` for Linux/Windows and `chromium-v145.0.7632.109.2` for macOS unless a future release restores a cleaner cross-platform set.
- BrowseForge Chromium `v0.1.6-alpha.0` is the source-level Chromium alpha runtime from the `browseforge-runtime-chromium` release channel and is the fallback runtime on platforms where Camoufox is unsupported. GHCR defaults preinstall Camoufox plus BrowseForge Chromium for native `linux/amd64` and `linux/arm64` images; BrowseForge Chromium launch persona values are resolved separately for `Linux x86_64` and `Linux arm64` so UA, UA-CH, `navigator.platform`, and the native persona JSON remain architecture-coherent.

## Docker Platform Policy

The published GHCR Docker image is multi-arch for `linux/amd64` and `linux/arm64`.

Docker pulls the host-native image automatically on x64 servers, Apple Silicon, and ARM servers. Use `--platform linux/amd64` only for compatibility debugging.

Native architecture support removes amd64 emulation from the Docker path. Public fingerprint detector outcomes still depend on GPU/rendering, fonts, locale, proxy, and runtime profile coherence.

Native `linux/arm64` Docker support requires the matching BrowseForge Chromium `linux-arm64` runtime artifact. Release and Docker preinstall checks fail when the native artifact is missing instead of falling back to an x64/emulated runtime.

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

### BrowseForge Chromium v0.1.6-alpha.0

```text
Linux x64:      https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-linux-x64.zip
Linux arm64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-linux-arm64.zip
macOS arm64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-macos-arm64.zip
macOS x64:      https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-macos-x64.zip
Windows x64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-windows-x64.zip
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
