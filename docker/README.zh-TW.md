# BrowseForge Docker

[English](README.md)

一鍵部署 BrowseForge + KasmVNC 遠端桌面。

## 使用方式

建議正式部署 pin 版本 tag，避免 `latest` 在重啟或重新拉取時非預期升級：

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

本地從原始碼 build：

```bash
cd docker
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}
docker compose up -d --build
```

compose 預設會建置 `v2.1.13` release image。要測其他版本：

```bash
BROWSEFORGE_VERSION=v2.1.13 docker compose up -d --build
```

驗證尚未發布的 BrowseForge Chromium runtime 時，請把 image build 指向暫存的 runtime asset root。該 root 必須包含 `checksums.txt`、`runtime.manifest.json`，以及 `<runtime-version>/browseforge-runtime-chromium-<runtime-version>-linux-{x64,arm64}.zip`：

```bash
BROWSEFORGE_CHROMIUM_RELEASE_BASE_URL=https://host/runtime/releases \
BROWSEFORGE_VERSION=v2.1.13 docker compose up -d --build
```

## 首次啟動

目前 release image 會在 Docker build 階段預先安裝 Camoufox 與 BrowseForge Chromium。啟動時如果 `/app/browsers/{engine}/.version` 缺失或不同，就用 image 內建版本更新 host mount。若使用舊版 image、關閉 `BROWSEFORGE_PREINSTALL_BROWSERS`、將 `BROWSEFORGE_PREINSTALL_RUNTIMES` 設為 image 未預載的 runtime，或設定 `BROWSEFORGE_SEED_BROWSERS=0`，第一次啟動仍可能需要下載；這段期間 dashboard 還不會 ready。可用以下方式確認：

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --json
docker exec browseforge /app/BrowseForge smoke rest --wait --timeout 5m --json
```

## 連線

| 服務 | URL |
|------|-----|
| Dashboard + REST API + Playwright proxy | http://localhost:19280 |
| MCP Streamable HTTP | http://localhost:19280/mcp |
| 遠端桌面 (KasmVNC) | http://localhost:6901 |
| VNC 帳號 | `user` / 環境變數 `VNC_PASSWORD`（預設 `browseforge`） |

快速 Docker 指令會讓容器內的 BrowseForge 對外 bind，同時預設 `BROWSEFORGE_PUBLIC_BASE_URL` 為同主機 agent 可抓取的 `http://localhost:19280`。正式遠端 server 或 reverse proxy 請先 export `BROWSEFORGE_PUBLIC_BASE_URL` 為外部 agent 實際可連到的 origin，若有 path prefix 也要包含。

## 取得 API Token

```bash
docker compose logs | grep "API Token"
# 或
docker compose exec browseforge /app/BrowseForge token
```

## 持久化、升級與備份

正式部署預設使用 host bind mounts：

| Host path | Container path | 用途 |
|-----------|----------------|------|
| `./browseforge/profiles` | `/app/profiles` | Profile metadata 與 browser user data。 |
| `./browseforge/data` | `/app/data` | `.api-token` API token 與 fingerprint data。 |
| `./browseforge/browsers` | `/app/browsers` | 已下載或 seed 的 Camoufox、BrowseForge Chromium 與可選 CloakBrowser engines。 |
| `./browseforge/logs` | `/app/logs` | Server logs。 |
| `./browseforge/backups` | `/app/backups` | Filesystem 與 API backup 輸出。 |

Pull 新 image 或重建容器時，必須沿用同一組 `-v "$PWD/browseforge/...:/app/..."` mounts。這樣 token 與使用者產生的 profile/browser data 不會因 container 被刪除而消失。

`/app/browsers` 是 BF-managed browser cache。預設 `BROWSEFORGE_SEED_BROWSERS=1` 會讓它跟著 image 內建 browser version 更新；`BROWSEFORGE_PREINSTALL_RUNTIMES` 控制 build-time 預載清單，預設為 `camoufox,browseforge-chromium`。若特殊 debug 需要保留手動放置的 browser，可設為 `0`。

升級範例：

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

完整 filesystem 備份：

```bash
docker stop browseforge
tar -czf ./browseforge/backups/browseforge-runtime-$(date +%Y%m%d-%H%M%S).tgz ./browseforge/profiles ./browseforge/data ./browseforge/browsers ./browseforge/logs
docker start browseforge
```

容器執行中的便利備份指令：

```bash
docker exec browseforge /app/BrowseForge backup create --full --output /app/backups --json
```

REST API 的 `/api/backup` 是較輕量的 profile metadata backup；若要保留完整 browser user data 與 token，請備份 host runtime 目錄。

## 特性

- **KasmVNC** — 比 noVNC 更好的剪貼簿支援（Chrome 上 seamless）、IME 中文輸入
- **WebGL 完整偽裝** — GLX + 軟體渲染 + 完整 WebGL 指紋
- **Docker 自動偵測** — 自動啟用 `0.0.0.0` 綁定和 `--no-sandbox`
- **Playwright proxy** — Dashboard + API + Playwright WebSocket proxy 都走 19280
- **MCP HTTP** — Streamable HTTP MCP 走 `19280/mcp`，使用與 REST API 相同的 Bearer Token

## 注意事項

- GHCR Docker image 發佈原生 `linux/amd64` 與 `linux/arm64` manifests。x64 server、Apple Silicon 與 ARM server 會自動拉取對應架構。
- 原生架構支援會從 Docker 路徑移除 amd64 emulation。Public fingerprint detector 的結果仍取決於 GPU/rendering、fonts、locale、proxy 與 runtime profile coherence。
- 原生 `linux/arm64` Docker 支援需要對應的 BrowseForge Chromium `linux-arm64` runtime artifact。Release 與 Docker preinstall 檢查在缺少原生 artifact 時會失敗，不會 fallback 到 x64/emulated runtime。
- Docker 預設使用一致的軟體 GL（`BROWSEFORGE_DOCKER_GPU_MODE=software`）：BrowseForge Chromium 會回報與 SwiftShader 對齊的 WebGL 字串，不假裝有 host GPU。`native` 保留 browser-default GPU evidence 給已配置真實 GPU path 的環境，`passthrough` 只用於明確 host GPU passthrough；其他值會 fail closed，不會靜默漂移。
- KasmVNC 預設為 `1920x1080`，與 BrowseForge Chromium native screen persona 對齊。只有在你希望 headed browser 視窗與 Screen API evidence 同步採用自訂桌面大小時，才同時覆寫 `BROWSEFORGE_DOCKER_DISPLAY_WIDTH` 與 `BROWSEFORGE_DOCKER_DISPLAY_HEIGHT`。
- VNC 用於觀看瀏覽器畫面和基本操作
- 中文輸入和剪貼簿在 Chrome 瀏覽器上 seamless 支援
- 瀏覽器引擎、Profile 資料、Token、logs 預設 mount 到 host `./browseforge/` 目錄，重建容器不會遺失
- MCP `screenshot` 對遠端 agent 會以臨時、免驗證的 `screenshot_url` artifact link 為主；請設定 `BROWSEFORGE_PUBLIC_BASE_URL`，避免 agent 只收到 base64 image block。

## Apple Silicon (M1/M2/M3)

Apple Silicon 與 ARM server 預設會拉取原生 `linux/arm64` image。只有相容性 debug 需要 x64 時，才使用 `docker run --platform linux/amd64 ...` 或 `DOCKER_DEFAULT_PLATFORM=linux/amd64 docker compose up -d --build`。
