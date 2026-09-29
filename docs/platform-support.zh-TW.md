# BrowseForge 平台支援矩陣

[English](platform-support.md)

> 維護此表以確保每個平台都有對應的瀏覽器引擎可用。
> 更新時機：瀏覽器引擎發佈新版本時。

## 當前版本

| 元件 | 版本 | 更新日期 |
|------|------|---------|
| BrowseForge | v2.1.15 | 2026-09-29 |
| Camoufox | v135.0.1-beta.24 | 2025-03-15 |
| CloakBrowser macOS | chromium-v145.0.7632.109.2 | 2026-03-04 |
| CloakBrowser Linux/Windows | chromium-v146.0.7680.177.4 | 2026-04-28 |
| BrowseForge Chromium | v0.1.6-alpha.0 | 2026-07-16 |

## 平台支援矩陣

| 平台 | BrowseForge | Camoufox | CloakBrowser | BrowseForge Chromium | 備註 |
|------|:---:|:---:|:---:|:---:|------|
| macOS x64 (Intel) | 支援 | v135 | v145 | Alpha artifact | 原生 binary |
| macOS arm64 (Apple Silicon) | 支援 | v135 | v145 | Alpha artifact | 原生 binary |
| Linux x64 | 支援 | v135 | v146 | Alpha artifact | 需要 display server 或 Docker runtime |
| Linux arm64 | 支援 | v135 | v146 | Alpha artifact | 原生 binary；GHCR 發佈 `linux/arm64` image |
| Windows x64 | 支援 | v135 | v146 | Alpha artifact | 原生 binary |
| Windows i686 (32-bit) | 不支援 | 不支援 | 不支援 | Not packaged | BrowseForge 不提供 32-bit build |
| Linux i686 (32-bit) | 不支援 | 不支援 | 不支援 | Not packaged | 同上 |

## 瀏覽器 Runtime 選版原則

BrowseForge 版本與瀏覽器 runtime 版本分開。啟動與 `browsers install` 時，BrowseForge 依目前 `GOOS/GOARCH` 只啟用並下載支援的平台 runtime；不支援的 runtime 會被略過並標示為停用。

- Camoufox 目前 BrowseForge release 維持使用 `v135.0.1-beta.24`，因為該 pinned 上游 release 有完整瀏覽器 binary：macOS x64/arm64、Linux x64/arm64、Windows x64。較新的上游 Camoufox release 也已提供這些平台的 browser ZIP：GitHub latest stable marker 是 `v152.0.4-beta.30`，release feed 最新項目是 prerelease `v156.0.1-beta.32`。升級 pinned runtime 版本屬於另一個相容性驗證任務。
- CloakBrowser 維持 Linux/Windows 使用 `chromium-v146.0.7680.177.4`、macOS 使用 `chromium-v145.0.7632.109.2`。較新的上游 release 目前沒有提供 BrowseForge 使用的乾淨跨平台 asset set；最新 `chromium-v152.0.7977.82.1-pro` 推進 Linux x64/arm64 與 Windows x64，但 macOS 仍停在另一個較舊 build，且 GitHub release attachments 暴露的是 checksum/signature metadata 而非直接 browser archives。
- BrowseForge Chromium `v0.1.6-alpha.0` 是 `browseforge-runtime-chromium` release channel 的 source-level Chromium alpha runtime，也是 Camoufox 不支援平台上的 fallback runtime。GHCR 預設會在原生 `linux/amd64` 與 `linux/arm64` image 內預載 Camoufox 與 BrowseForge Chromium；BrowseForge Chromium launch persona 會分別解析 `Linux x86_64` 與 `Linux arm64`，確保 UA、UA-CH、`navigator.platform` 與 native persona JSON 的架構一致。

## Docker 平台政策

GHCR Docker image 發佈 `linux/amd64` 與 `linux/arm64` multi-arch manifests。

Docker 會在 x64 server、Apple Silicon 與 ARM server 自動拉取 host-native image。只有相容性 debug 才需要指定 `--platform linux/amd64`。

原生架構支援會從 Docker 路徑移除 amd64 emulation。Public fingerprint detector 的結果仍取決於 GPU/rendering、fonts、locale、proxy 與 runtime profile coherence。

原生 `linux/arm64` Docker 支援需要對應的 BrowseForge Chromium `linux-arm64` runtime artifact。Release 與 Docker preinstall 檢查在缺少原生 artifact 時會失敗，不會 fallback 到 x64/emulated runtime。

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

### BrowseForge Chromium v0.1.6-alpha.0

```
Linux x64:      https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-linux-x64.zip
Linux arm64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-linux-arm64.zip
macOS arm64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-macos-arm64.zip
macOS x64:      https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-macos-x64.zip
Windows x64:    https://github.com/nczz/browseforge-runtime-chromium/releases/download/v0.1.6-alpha.0/browseforge-runtime-chromium-v0.1.6-alpha.0-windows-x64.zip
```

## 升級檢查清單

更新瀏覽器版本時：

- [ ] 確認新版本在所有支援平台都有 binary
- [ ] 更新 `internal/runtime/support.go` 中的版本號、平台檔名與支援判斷
- [ ] 更新此表
- [ ] 測試自動下載功能
- [ ] 測試反偵測能力（browserleaks、Sannysoft）
- [ ] 依照 [Release Process](release.md) 執行 `scripts/release-preflight.sh vX.Y.Z`
- [ ] 依照 [Release Process](release.md) 執行 `scripts/release-push.sh vX.Y.Z`
