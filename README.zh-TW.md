# BrowseForge

[English](README.md)

🦊🌐 BrowseForge 是 automation-ready 的反偵測瀏覽器工作區，適合需要在桌面、伺服器、Docker、REST API、MCP、Playwright workflow 中管理可重複、可隔離瀏覽器身份的團隊。

它整合 Firefox/Camoufox 與 Chromium/CloakBrowser runtime，提供 Profile 隔離、指紋管理、遠端控制、備份/還原，以及 AI agent 友善的自動化介面。

## 為什麼選 BrowseForge

- 用一致的方式管理隔離瀏覽器身份，不需要為每個 browser engine 重寫自動化流程。
- 可透過 Web UI、REST API、MCP tools、YAML workflow 或 Playwright client 控制 profiles。
- 可在本機開發，也可用 Docker + KasmVNC 部署到 Linux server。
- 透過支援矩陣、防呆 release scripts、CI checks，讓發布與平台行為可追溯。
- 以國際化為基礎：英文主文件、繁中文件、Dashboard/extension i18n。

## 適合誰

- 需要驗證多帳號、多地區瀏覽器流程的 QA / automation 團隊。
- 研究 fingerprint 與反偵測行為的 browser-runtime 研究者。
- 需要 MCP 控制瀏覽器 session 的 AI agent builder。
- 需要穩定 profile 儲存、備份、還原、遠端瀏覽器操作的使用者。

## 信任與安全

- MIT license。
- 公開 security policy 與 support process。
- REST API、MCP HTTP、Playwright proxy 都使用 token 驗證。
- Docker 正式部署建議 pin 版本 tag。
- Release preflight 會檢查 tests、i18n、Docker build、Camoufox Bind runtime、release artifact 一致性。

## 專案資源

- [貢獻指南](CONTRIBUTING.md)
- [安全政策](SECURITY.md)
- [支援說明](SUPPORT.md)
- [行為準則](CODE_OF_CONDUCT.md)
- [發布流程](docs/release.md)
- [平台支援](docs/platform-support.md)
- [國際化](docs/i18n.md)
- [更新紀錄](CHANGELOG.md)
- [API 文件](API.md)

## 功能

- **雙引擎**：Firefox (Camoufox) + Chromium (CloakBrowser)。
- **獨立 Profile**：Cookie、localStorage、Proxy、指紋設定互不干擾。
- **指紋池**：每個 Profile 可自動分配指紋，並依照本機或 Proxy 調整 Timezone/Language。
- **Web Dashboard**：瀏覽器開啟 `http://127.0.0.1:19280` 管理 Profile。
- **REST API**：支援 Profile、Session、備份、還原、Workflow 控制。
- **MCP Server**：支援 stdio 與 Streamable HTTP。
- **Playwright Connect**：外部 Playwright client 可連入既有 session。
- **Docker**：包含 KasmVNC 遠端桌面、Dashboard/API、MCP HTTP 與瀏覽器 runtime 依賴。
- **Portable**：單一執行檔，首次啟動自動下載瀏覽器引擎。

## 快速開始

到 [Releases](https://github.com/nczz/BrowseForge/releases) 下載對應平台的 ZIP：

| 平台 | 檔案 |
|------|------|
| macOS Intel | `BrowseForge-vX.X.X-lite-macos-x64.zip` |
| macOS Apple Silicon | `BrowseForge-vX.X.X-lite-macos-arm64.zip` |
| Linux x64 | `BrowseForge-vX.X.X-lite-linux-x64.zip` |
| Linux arm64 | `BrowseForge-vX.X.X-lite-linux-arm64.zip` |
| Windows x64 | `BrowseForge-vX.X.X-lite-windows-x64.zip` |

```bash
unzip BrowseForge-vX.X.X-lite-macos-arm64.zip
cd BrowseForge-lite
./BrowseForge
```

Dashboard 會開在 `http://127.0.0.1:19280`。首次啟動會在需要時下載瀏覽器引擎。

## Docker

正式部署建議 pin 版本 tag，不要依賴 `latest`：

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
  ghcr.io/nczz/browseforge:v1.10.7
```

`./browseforge/` host 目錄就是持久化 runtime。之後 pull 新 image 或重建 container 時沿用這組 mounts，profiles、token、browser data、logs、backups 都會保留。

| 服務 | URL |
|------|-----|
| Dashboard + REST API + Playwright proxy | `http://localhost:19280` |
| MCP Streamable HTTP | `http://localhost:19280/mcp` |
| KasmVNC 遠端桌面 | `http://localhost:6901` |
| VNC 帳號 | `user` / `VNC_PASSWORD` |
| API Token | `docker exec browseforge /app/BrowseForge token` |

從原始碼 build：

```bash
cd docker
docker compose up -d --build
```

目前 GHCR Docker image 是 `linux/amd64`。Apple Silicon 可以透過 Docker emulation 執行；原生 `linux/arm64` Docker image 會等 KasmVNC、Camoufox、CloakBrowser 都在 ARM container 驗證後再開啟。

Linux server 部署細節見 [docs/linux-server.md](docs/linux-server.md)。

## CLI

```bash
# 顯示指令與版本
./BrowseForge --help
./BrowseForge version

# 初始化 config.json、profiles/、data/、logs/
./BrowseForge init

# 啟動 server
./BrowseForge serve
./BrowseForge serve --host 0.0.0.0 --no-sandbox --no-open

# 顯示 API Token
./BrowseForge token
./BrowseForge token --json

# 環境檢查
./BrowseForge doctor
./BrowseForge doctor --strict --json
./BrowseForge status --json
./BrowseForge browsers status --json
./BrowseForge open

# 顯示整合能力並驗證 server transport
./BrowseForge capabilities --json
./BrowseForge smoke rest --base-url http://127.0.0.1:19280 --wait --json
./BrowseForge smoke mcp --base-url http://127.0.0.1:19280 --token "$TOKEN" --wait --json
./BrowseForge mcp-config stdio --json

# 備份與 browser 準備
./BrowseForge browsers install
./BrowseForge backup create --full --output ./backups --json

# 執行 server-backed workflow 並檢視 API resources
./BrowseForge workflow run examples/multi-login.yaml --token "$TOKEN" --json
./BrowseForge profiles list --token "$TOKEN" --json
./BrowseForge sessions list --token "$TOKEN" --json

# MCP stdio 模式
./BrowseForge --mcp
./BrowseForge mcp-stdio
```

全域 CLI flags：

| Flag | 說明 |
|------|------|
| `--base-dir DIR` | runtime 目錄，包含 config、profiles、data、logs、browser engines；預設為 binary 所在目錄。 |
| `--config PATH` | config 檔路徑；相對路徑會以 `--base-dir` 解析，預設為 `config.json`。 |

Agent 與 CI 整合建議對 `token`、`doctor`、`capabilities`、`smoke` 使用 `--json`，輸出才穩定可解析。未知指令會回傳非 0 exit code 並顯示 usage，不會意外啟動 server。

完整 command reference、exit codes、JSON schema 與 agent integration checklist 見 [docs/cli.md](docs/cli.md)。

## 設定檔

首次啟動時會建立 `config.json`：

```json
{
  "host": "127.0.0.1",
  "port": "19280",
  "profiles_dir": "profiles",
  "data_dir": "data",
  "log_file": "logs/server.log",
  "camoufox_path": "/path/to/browsers/camoufox/...",
  "cloakbrowser_path": "/path/to/browsers/cloakbrowser/...",
  "cloakbrowser": {
    "safe_gpu": false,
    "auto_safe_gpu_fallback": false,
    "isolated_runtime_cache": false,
    "repair_transient_cache_on_launch_failure": false,
    "extra_args": []
  },
  "endpoint_health": {
    "timeout_ms": 15000,
    "attempts": 2,
    "retry_delay_ms": 500,
    "bind_grace_ms": 100,
    "probe_page": false,
    "history_limit": 256
  },
  "fingerprint_dir": "data"
}
```

| 欄位 | 說明 | 預設 |
|------|------|------|
| `host` | 監聽地址 | `127.0.0.1`；Docker 自動改為 `0.0.0.0` |
| `port` | REST API + Dashboard 端口 | `19280` |
| `no_sandbox` | 停用 Chromium sandbox | `false`；Docker 自動啟用 |
| `profiles_dir` | Profile 資料目錄 | `profiles` |
| `data_dir` | token、指紋池資料目錄 | `data` |
| `log_file` | 日誌檔案 | `logs/server.log` |
| `endpoint_health.timeout_ms` | Playwright proxy backend 連線與 handshake 的 timeout，上限 120000 | `15000` |
| `endpoint_health.attempts` | 清理或重啟前，對同一個 browser child 執行不建立額外連線的健康檢查次數，上限 5 | `2` |
| `endpoint_health.retry_delay_ms` | endpoint 檢查之間的等待時間，上限 30000 | `500` |
| `endpoint_health.bind_grace_ms` | Bind 後的穩定視窗：manager 會在 grace 前後各驗證一次 browser，Proxy 或 MCP 也會等 grace 到期後才連入；上限 5000 | `100` |
| `endpoint_health.probe_page` | 額外讀取既有 session page 的 title；不建立新頁面或 client connection | `false` |
| `endpoint_health.history_limit` | diagnostics API 保留的記憶體內 session lifecycle events，上限 2000 | `256` |
| `camoufox_path` | Camoufox 執行檔路徑 | 自動偵測 |
| `cloakbrowser_path` | CloakBrowser 執行檔路徑 | 自動偵測 |
| `cloakbrowser.safe_gpu` | 為 Windows VM/headful 相容性加入 Chromium GPU-safe 啟動參數 | `false` |
| `cloakbrowser.auto_safe_gpu_fallback` | 第一次啟動維持原始行為，只在 GPU/cache 啟動失敗後才自動用 safe GPU 與 isolated runtime cache 重試一次 | `false` |
| `cloakbrowser.isolated_runtime_cache` | 將 Chromium disk cache 導到 profile 底下的每次啟動 runtime cache 目錄 | `false` |
| `cloakbrowser.repair_transient_cache_on_launch_failure` | GPU/cache 或 endpoint 穩定性失敗後，在自動重試前清除包含根目錄 `BrowserMetrics` 的可重建 Chromium cache | `false` |
| `cloakbrowser.extra_args` | 額外 CloakBrowser/Chromium args；BrowseForge 會忽略會覆蓋 profile、proxy、cache 或 debugging ownership 的 args | `[]` |
| `fingerprint_dir` | 指紋池 JSON 目錄 | `data` |

Endpoint health check 預設只測試 connect。BrowseForge 會在 Bind grace 前後各驗證一次 manager 持有的 browser，再對同一個 Bind endpoint 重試後才決定是否關閉 child process；browser disconnect 與 cleanup reason 會記錄於需驗證的 `GET /api/diagnostics/sessions`。Child 在 grace 期間退出時會回傳結構化 `BROWSER_PROCESS_EXITED`，讓 consumer 可輪替不相容的 profile metadata，而不是取得已失效 endpoint。Server 重啟後仍以持久化 JSON log 為準。

`cloakbrowser` 區塊是主機 runtime policy，不是 profile identity。它只影響 Chromium/CloakBrowser 啟動；Firefox/Camoufox profile 不會讀取這些設定。若 Windows VM 環境出現 `GPU process isn't usable` 或 `Unable to create cache` 這類 Chromium 啟動錯誤，可使用：

```json
{
  "cloakbrowser": {
    "auto_safe_gpu_fallback": true,
    "repair_transient_cache_on_launch_failure": true
  }
}
```

## MCP

BrowseForge 支援兩種 MCP 模式。

### Streamable HTTP

BrowseForge server 啟動後，MCP HTTP 會在主服務 port 的 `http://127.0.0.1:19280/mcp` 提供服務。

遷移提醒：舊版 client 若使用獨立的 `:19281` MCP listener，請改成主服務 port 加上 `/mcp`。

HTTP MCP 使用與 REST API 相同的 Bearer Token：

```http
Authorization: Bearer <token>
```

`web_search` MCP tool 支援 provider-backed search；預設 engine 是 `google`，也可指定 `bing` 或 `duckduckgo`。

遠端 MCP 設定範例：

```json
{
  "mcpServers": {
    "browseforge": {
      "url": "http://YOUR_SERVER:19280/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN"
      }
    }
  }
}
```

### stdio

```bash
BrowseForge --mcp
```

Kiro / Claude 設定範例：

```json
{
  "browseforge": {
    "command": "/path/to/BrowseForge",
    "args": ["--mcp"]
  }
}
```

## REST API

Base URL:

```text
http://127.0.0.1:19280/api
```

除了 `/api/status` 之外，所有 API 都需要 Bearer Token：

```http
Authorization: Bearer <token>
```

Token 會在首次啟動時自動建立，存放於 `data/.api-token`。

完整 API 文件見 [API.md](API.md)。

## Playwright Connect

BrowseForge 會為每個執行中的 session 暴露經 Bearer token 驗證且有 Bind handoff gate 的 Playwright WebSocket proxy。所有本機與遠端整合都必須使用回傳的 `proxy` URL；raw Bind endpoint 僅保留於內部 diagnostics。

```bash
curl http://127.0.0.1:19280/api/playwright/endpoint \
  -H "Authorization: Bearer $TOKEN"
```

Node.js 範例：

```javascript
import { firefox } from 'playwright';

const browser = await firefox.connect(
  'ws://YOUR_SERVER:19280/api/playwright/ws/sess_prof_xxx',
  { headers: { Authorization: 'Bearer YOUR_TOKEN' } }
);

const page = browser.contexts()[0].pages()[0];
await page.goto('https://example.com');
console.log(await page.title());
await browser.close();
```

相容性：

| 項目 | 說明 |
|------|------|
| Client 版本 | 需使用與 BrowseForge driver 相容的 Playwright client；目前目標是 `1.60.x` |
| 連線 URL | 使用 `/api/playwright/endpoint` 回傳的 `proxy`；raw Bind endpoint 僅供內部診斷 |
| Docker | Proxy 模式只需暴露 `19280` |
| 驗證 | Proxy 模式必須使用 Bearer Token |
| 反偵測 runtime | 使用 Playwright 內部協議，不暴露 CDP |

## YAML Workflow

可以透過 REST API 執行 workflow YAML：

```yaml
name: Multi-account login
steps:
  - name: Create profile
    action: create_profile
    params: { name: "FB Account", engine: firefox, var: p1 }

  - name: Open browser
    action: open_browser
    profile_id: $p1

  - name: Navigate
    action: navigate
    profile_id: $p1
    params: { url: "https://facebook.com" }

  - name: Wait
    action: sleep
    params: { seconds: 30 }

  - name: Close
    action: close_browser
    profile_id: $p1
```

```bash
TOKEN=$(cat data/.api-token)
curl -X POST http://127.0.0.1:19280/api/workflow/run \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @examples/multi-login.yaml
```

支援 action：`create_profile`、`open_browser`、`close_browser`、`navigate`、`click`、`type`、`eval`、`wait`、`screenshot`、`sleep`。

## Profile 資料

每個 Profile 存放於 `profiles/`：

```text
profiles/
  prof_abc123/
    profile.json
    browser-data/
```

請把 profiles、備份 ZIP、匯出的 profiles、cookies、`data/.api-token` 都視為敏感資料。

## 平台支援

| 平台 | BrowseForge | Camoufox | CloakBrowser |
|------|:---:|:---:|:---:|
| macOS x64 | 支援 | 支援 | 支援 |
| macOS arm64 | 支援 | 支援 | 支援 |
| Linux x64 | 支援 | 支援 | 支援 |
| Linux arm64 | Binary 支援 | 支援 | 支援 |
| Windows x64 | 支援 | 支援 | 支援 |

詳細說明見 [docs/platform-support.md](docs/platform-support.md)。

現行雙瀏覽器反偵測架構請見 [docs/dual-browser-architecture.zh-TW.md](docs/dual-browser-architecture.zh-TW.md)。

## 從原始碼 Build

```bash
git clone https://github.com/nczz/BrowseForge.git
cd BrowseForge

npm install
node scripts/generate-fingerprints.js --browser firefox --os windows --count 500
node scripts/generate-fingerprints.js --browser firefox --os macos --count 500
node scripts/generate-fingerprints.js --browser chrome --os windows --count 500
node scripts/generate-fingerprints.js --browser chrome --os macos --count 500

go build -ldflags="-s -w" -o BrowseForge ./cmd/server
./BrowseForge
```

## 開發檢查

```bash
go test -count=1 ./...
go vet ./...
bash -n scripts/release-preflight.sh scripts/release-push.sh
docker compose -f docker/docker-compose.yml config
```

發布請使用 [docs/release.md](docs/release.md) 的防呆流程，不要手動建立或推送 release tag。

## 架構

```text
BrowseForge
  Main HTTP service (:19280)
    REST API (/api)
    MCP Streamable HTTP (/mcp)
    Web Dashboard (/)
    Playwright-compatible browser control
  Browser runtime
    Camoufox profiles
    CloakBrowser profiles
  Profile store
  Fingerprint data
```

## 合理使用

BrowseForge 目標用途是合法 QA、自動化、隱私研究、相容性測試與受控瀏覽器操作。請勿用於未授權存取、憑證濫用、垃圾訊息、詐欺，或規避你沒有擁有或沒有授權測試的系統。

## License

MIT. See [LICENSE](LICENSE).
