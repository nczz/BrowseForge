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
  -v "$PWD/browseforge/profiles:/app/profiles" \
  -v "$PWD/browseforge/data:/app/data" \
  -v "$PWD/browseforge/browsers:/app/browsers" \
  -v "$PWD/browseforge/logs:/app/logs" \
  -v "$PWD/browseforge/backups:/app/backups" \
  -e BROWSEFORGE_SEED_BROWSERS=1 \
  --restart unless-stopped \
  ghcr.io/nczz/browseforge:v1.10.8
```

本地從原始碼 build：

```bash
cd docker
mkdir -p ./browseforge/{profiles,data,browsers,logs,backups}
docker compose up -d --build
```

compose 預設會建置 `v1.10.8` release image。要測其他版本：

```bash
BROWSEFORGE_VERSION=v1.10.8 docker compose up -d --build
```

若 image 沒有預載 browser engines，首次啟動需要 3-5 分鐘下載瀏覽器引擎（~440MB）。

新版 image 會在 build 階段預先安裝 browser engines，並在啟動時如果 `/app/browsers/{engine}/.version` 缺失或不同，就用 image 內建版本更新 host mount。若使用舊版 image、關閉 `BROWSEFORGE_PREINSTALL_BROWSERS`，或設定 `BROWSEFORGE_SEED_BROWSERS=0`，第一次啟動仍可能需要下載；這段期間 dashboard 還不會 ready。可用以下方式確認：

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
| `./browseforge/browsers` | `/app/browsers` | 已下載的 Camoufox/CloakBrowser engines。 |
| `./browseforge/logs` | `/app/logs` | Server logs。 |
| `./browseforge/backups` | `/app/backups` | Filesystem 與 API backup 輸出。 |

Pull 新 image 或重建容器時，必須沿用同一組 `-v "$PWD/browseforge/...:/app/..."` mounts。這樣 token 與使用者產生的 profile/browser data 不會因 container 被刪除而消失。

`/app/browsers` 是 BF-managed browser cache。預設 `BROWSEFORGE_SEED_BROWSERS=1` 會讓它跟著 image 內建 browser version 更新；若特殊 debug 需要保留手動放置的 browser，可設為 `0`。

升級範例：

```bash
docker pull ghcr.io/nczz/browseforge:v1.10.8
docker stop browseforge
docker rm browseforge
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
  ghcr.io/nczz/browseforge:v1.10.8
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

- GHCR Docker image 目前發佈 `linux/amd64`。Apple Silicon 或 ARM server 會透過 emulation 執行。
- VNC 用於觀看瀏覽器畫面和基本操作
- 中文輸入和剪貼簿在 Chrome 瀏覽器上 seamless 支援
- 瀏覽器引擎、Profile 資料、Token、logs 預設 mount 到 host `./browseforge/` 目錄，重建容器不會遺失

## Apple Silicon (M1/M2/M3)

docker-compose.yml 已設定 `platform: linux/amd64`，透過 Rosetta/QEMU 模擬執行。
原生 `linux/arm64` image 會在 KasmVNC、Camoufox、CloakBrowser runtime 都驗證完成後再開啟。
