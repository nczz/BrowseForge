# Linux Server 部署指南

[English](linux-server.md)

## 推薦：Docker 部署

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

正式部署建議 pin 版本 tag，例如 `v2.1.13`。`latest` 可用於快速試用，但重啟或重新拉取時可能非預期升級。

快速指令會讓容器內的 BrowseForge 對外 bind，同時預設 `BROWSEFORGE_PUBLIC_BASE_URL` 為同主機 agent 可抓取的 `http://localhost:19280`。遠端 agent 正式使用前，請先 export `BROWSEFORGE_PUBLIC_BASE_URL` 為外部可連到的 server origin。

> GHCR Docker image 發佈原生 `linux/amd64` 與 `linux/arm64` manifests。x64 server、Apple Silicon 與 ARM server 會自動拉取對應架構；只有相容性 debug 才需要指定 `--platform linux/amd64`。

Docker 預設使用軟體 GL，以維持容器可攜性與 WebGL 一致性。除非 host 已明確提供真實 GPU path，否則保留 `BROWSEFORGE_DOCKER_GPU_MODE=software`。已配置 GPU 環境且要保留 browser-default GPU evidence 時使用 `native`，明確 host GPU passthrough 時使用 `passthrough`；不支援的值會 fail closed。

## 首次啟動與 Browser Engines

BrowseForge 需要 Camoufox 與 BrowseForge Chromium engines 準備完成後，Dashboard 才會真正 ready。目前 Docker image 會在 image build 階段預先安裝 engines，並在啟動時如果 host-mounted `/app/browsers` 缺少該 engine，或 `.version` 與 image 內建版本不同，就用 image 版本更新 `/app/browsers`。這讓 browser engines 跟著 BrowseForge image 版本對齊。若 image 是用舊版 binary 建置、關閉 browser preinstall，或設定 `BROWSEFORGE_SEED_BROWSERS=0`，啟動時仍可能花數分鐘下載 engines，這段期間 `19280` 還不會回應。

請用 logs 與 wait-aware smoke checks 判斷，不要直接認定 container 壞掉：

```bash
docker logs -f browseforge
docker exec browseforge /app/BrowseForge browsers status --json
docker exec browseforge /app/BrowseForge smoke rest --wait --timeout 5m --json
```

自製 image 時，建議保留預設的 `BROWSEFORGE_PREINSTALL_BROWSERS=1`。特殊 debug 情境可設定 `BROWSEFORGE_SEED_BROWSERS=0`，避免容器用 image 內建版本更新 `/app/browsers`。

| 服務 | URL |
|------|-----|
| Dashboard + API | http://YOUR_SERVER:19280 |
| MCP Streamable HTTP | http://YOUR_SERVER:19280/mcp |
| 遠端桌面 (KasmVNC) | http://YOUR_SERVER:6901 |
| VNC 帳號 | `user` / 環境變數 `VNC_PASSWORD` |

### 取得 API Token

```bash
docker logs browseforge | grep "API Token"
# 或
docker exec browseforge /app/BrowseForge token
```

### 持久化資料

正式部署建議把 runtime data mount 到 host 目錄。這樣 profiles、API token、下載的 browser engines、logs 都在容器外；執行 `docker pull`、`docker stop`、`docker rm`、重新 `docker run` 時，不會刪掉使用者內容。

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

持久化路徑：

| Host path | Container path | 用途 |
|-----------|----------------|------|
| `./browseforge/profiles` | `/app/profiles` | Profile metadata 與 browser user data。 |
| `./browseforge/data` | `/app/data` | `.api-token` API token 與 fingerprint data。 |
| `./browseforge/browsers` | `/app/browsers` | 已下載或 seed 的 Camoufox、BrowseForge Chromium 與可選 CloakBrowser engines。 |
| `./browseforge/logs` | `/app/logs` | Server logs。 |
| `./browseforge/backups` | `/app/backups` | Filesystem 與 API backup 輸出。 |

Docker named volumes 也能持久化，但 host bind mounts 比較容易檢查、複製、snapshot 與備份。

### Docker Compose

```yaml
services:
  browseforge:
    image: ghcr.io/nczz/browseforge:v2.1.13
    ports:
      - "19280:19280"
      - "6901:6901"
    volumes:
      - ./browseforge/profiles:/app/profiles
      - ./browseforge/data:/app/data
      - ./browseforge/browsers:/app/browsers
      - ./browseforge/logs:/app/logs
      - ./browseforge/backups:/app/backups
    environment:
      - VNC_PASSWORD=browseforge
      - BROWSEFORGE_SEED_BROWSERS=1
    restart: unless-stopped
```

## 防火牆設定

| 端口 | 用途 | 是否必要 |
|------|------|---------|
| 19280 | Dashboard + REST API + MCP Streamable HTTP + Playwright WebSocket proxy | 必要 |
| 6901 | KasmVNC 遠端桌面 | 選用（需要看畫面時） |

```bash
sudo ufw allow 19280/tcp
sudo ufw allow 6901/tcp
```

## 安全建議

- **不要**把 19280 和 6901 直接暴露到公網，用 SSH tunnel 或 VPN 存取
- KasmVNC 有 Basic Auth 保護（user/password）
- API/MCP Token 存在 `data/.api-token`，不要外洩
- 建議用 nginx reverse proxy + HTTPS 包裝

```bash
# SSH tunnel 方式（最安全）
ssh -L 19280:localhost:19280 -L 6901:localhost:6901 user@server
# 然後本機開 http://localhost:19280、http://localhost:19280/mcp 和 http://localhost:6901
```

## 升級

```bash
docker pull ghcr.io/nczz/browseforge:v2.1.13
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
  ghcr.io/nczz/browseforge:v2.1.13
```

Profiles、Token、下載的 browser engines、logs 都保留在 host 的 `./browseforge/` 目錄。Pull 新 image 並重建容器時，必須沿用同一組 bind mounts。

## 備份

最穩妥的完整備份包含 browser user data；建議停止容器後直接封存 host runtime 目錄：

```bash
docker stop browseforge
tar -czf ./browseforge/backups/browseforge-runtime-$(date +%Y%m%d-%H%M%S).tgz ./browseforge/profiles ./browseforge/data ./browseforge/browsers ./browseforge/logs
docker start browseforge
```

如果需要在容器執行中用較方便的 operator command：

```bash
docker exec browseforge /app/BrowseForge backup create --full --output /app/backups --json
```

重大升級或搬遷前，優先使用 stopped-container archive。

若只需要透過 REST API 做較輕量的 profile metadata backup：

```bash
TOKEN=$(docker exec browseforge /app/BrowseForge token)
curl -fsS -X POST http://127.0.0.1:19280/api/backup \
  -H "Authorization: Bearer $TOKEN" \
  -o ./browseforge/backups/browseforge-api-backup-$(date +%Y%m%d).zip
```

REST backup 適合 profile metadata import/export。若要保存完整 browser user data 與 API token，請使用 filesystem backup。

## 特性

- **KasmVNC** — Chrome 上 seamless 剪貼簿、IME 中文輸入
- **WebGL 完整偽裝** — GLX + 軟體渲染 + 完整 WebGL 指紋
- **Docker 自動偵測** — 自動 `0.0.0.0` + `--no-sandbox`
- **Playwright WebSocket proxy** — 外部腳本透過 19280 port 連入操作瀏覽器
- **MCP Streamable HTTP** — 遠端 MCP client 透過 `19280/mcp` 連入，使用 Bearer Token
