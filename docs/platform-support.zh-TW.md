# BrowseForge 平台支援矩陣

[English](platform-support.md)

> 維護此表以確保每個平台都有對應的瀏覽器引擎可用。
> 更新時機：瀏覽器引擎發佈新版本時。

## 當前版本

| 元件 | 版本 | 更新日期 |
|------|------|---------|
| BrowseForge | v1.10.8 | 2026-11-13 |
| Camoufox | v135.0.1-beta.24 | 2025-03-15 |
| CloakBrowser macOS | chromium-v145.0.7632.109.2 | 2026-03-04 |
| CloakBrowser Linux/Windows | chromium-v146.0.7680.177.4 | 2026-04-28 |

## 平台支援矩陣

| 平台 | BrowseForge | 🦊 Camoufox | 🌐 CloakBrowser | 備註 |
|------|:---:|:---:|:---:|------|
| macOS x64 (Intel) | ✅ | ✅ v135 | ✅ v145 | |
| macOS arm64 (Apple Silicon) | ✅ | ✅ v135 | ✅ v145 | |
| Linux x64 | ✅ | ✅ v135 | ✅ v146 | 需要 xvfb 或 Docker runtime |
| Linux arm64 | ✅ | ✅ v135 | ✅ v146 | CloakBrowser 需用 v146 版本 |
| Windows x64 | ✅ | ✅ v135 | ✅ v146 | |
| Windows i686 (32-bit) | ❌ | ✅ v135 | ❌ | BrowseForge 不提供 32-bit build |
| Linux i686 (32-bit) | ❌ | ✅ v135 | ❌ | 同上 |

## 瀏覽器 Runtime 選版原則

預設瀏覽器 runtime 版本以「能通過 BrowseForge 支援平台的最新可用組合」為準，而不是只看上游最新 tag。

- Camoufox `v150.0.2-beta.25` 比 `v135.0.1-beta.24` 新，但上游 release 目前沒有 BrowseForge 需要的完整平台資產；在有 macOS x64/arm64、Linux x64/arm64、Windows x64 且通過 runtime 驗證的新版本前，維持 `v135.0.1-beta.24`。
- CloakBrowser `chromium-v146.0.7680.177.5` 比 `.4` 新，但上游只提供 Linux x64 與 Windows x64；Linux arm64 與 macOS 仍停在不同版本。因此 BrowseForge 維持 Linux/Windows 使用 `chromium-v146.0.7680.177.4`、macOS 使用 `chromium-v145.0.7632.109.2`，直到上游恢復更乾淨的跨平台組合。

## 下載 URL 對照表

### Camoufox (v135.0.1-beta.24)

```
macOS x64:    https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-mac.x86_64.zip
macOS arm64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-mac.arm64.zip
Linux x64:    https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-lin.x86_64.zip
Linux arm64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-lin.arm64.zip
Windows x64:  https://github.com/daijro/camoufox/releases/download/v135.0.1-beta.24/camoufox-135.0.1-beta.24-win.x86_64.zip
```

### CloakBrowser macOS (chromium-v145.0.7632.109.2)

```
macOS x64:    https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v145.0.7632.109.2/cloakbrowser-darwin-x64.tar.gz
macOS arm64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v145.0.7632.109.2/cloakbrowser-darwin-arm64.tar.gz
```

### CloakBrowser Linux/Windows (chromium-v146.0.7680.177.4)

```
Linux x64:    https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-linux-x64.tar.gz
Linux arm64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-linux-arm64.tar.gz
Windows x64:  https://github.com/CloakHQ/CloakBrowser/releases/download/chromium-v146.0.7680.177.4/cloakbrowser-windows-x64.zip
```

## 升級檢查清單

更新瀏覽器版本時：

- [ ] 確認新版本在所有支援平台都有 binary
- [ ] 更新 `internal/browser/download.go` 中的版本號和 URL
- [ ] 更新此表
- [ ] 測試自動下載功能
- [ ] 測試反偵測能力（browserleaks、Sannysoft）
- [ ] 依照 [Release Process](release.md) 執行 `scripts/release-preflight.sh vX.Y.Z`
- [ ] 依照 [Release Process](release.md) 執行 `scripts/release-push.sh vX.Y.Z`
