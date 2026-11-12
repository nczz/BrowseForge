# BrowseForge API 文件

[English](API.md)

## 連線資訊

| 服務 | URL | 用途 |
|------|-----|------|
| REST API | `http://127.0.0.1:19280/api` | Profile 管理 + 瀏覽器操作 |
| Dashboard | `http://127.0.0.1:19280` | Web 管理介面 |
| MCP Server | `http://127.0.0.1:19280/mcp` | AI Agent 整合（JSON-RPC） |

## 認證

所有 REST API（除 `/api/status` 與臨時 `/api/screenshots/:id` 連結）需要 Bearer Token：

```
Authorization: Bearer {token}
```

Token 位於 `data/.api-token`。Dashboard 首次開啟時會要求輸入。

---

## REST API

### 系統

#### GET /api/status
```bash
curl http://127.0.0.1:19280/api/status
```
```json
{"version": "2.1.16", "status": "ok", "active_sessions": 1, "endpoint_health_policy": {"timeout_ms": 15000, "attempts": 2, "retry_delay_ms": 500, "probe_page": false, "history_limit": 256}}
```

#### POST /api/shutdown
關閉所有瀏覽器並停止 server。

---

### Profile 管理

#### POST /api/profiles — 建立 Profile
```bash
curl -X POST http://127.0.0.1:19280/api/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "FB Brand #1",
    "runtime_id": "camoufox",
    "group": "客戶A",
    "tags": ["facebook", "品牌"],
    "proxy": {
      "type": "socks5",
      "host": "proxy.example.com",
      "port": 1080,
      "username": "user",
      "password": "pass"
    }
  }'
```

回傳：
```json
{
  "data": {
    "id": "prof_a1b2c3d4e5f6",
    "name": "FB Brand #1",
    "runtime_id": "camoufox",
    "group": "客戶A",
    "fingerprint": {
      "navigator.userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) ...",
      "navigator.platform": "Win32",
      "screen.width": 1920,
      "screen.height": 1080,
      "canvas:seed": 3948271650,
      "..."
    },
    "proxy": {"type": "socks5", "host": "proxy.example.com", "port": 1080, "region": "us-ny"}
  }
}
```

- `runtime_id`：runtime provider，例如 `"camoufox"`、`"cloakbrowser"` 或 opt-in alpha `"browseforge-chromium"`
- `fingerprint`：未提供時自動從指紋池分配
- `proxy`：選填，支援 `socks5` 和 `http`；`region` 是選填的已遮罩地理/區域標籤，會保留給 BrowseForge Chromium native WebRTC persona metadata，請勿放入憑證或原始 IP
- 使用 BrowseForge Chromium alpha profile 前，需先在 `config.json` 啟用 `runtimes.browseforge-chromium`，並將 `binary_path` 指向 runtime artifact 解壓後的瀏覽器 binary。`proxy.region` 僅填去識別化地理標籤（例如 `us-ny`、`tw-taipei`），不可填原始 IP、憑證或客戶識別資訊。

#### GET /api/profiles — 列出 Profile
```bash
# 全部
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/profiles

# 篩選
curl -H "Authorization: Bearer $TOKEN" "http://127.0.0.1:19280/api/profiles?group=客戶A&tag=facebook"
```

#### GET /api/profiles/:id — 取得單一 Profile
```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6
```

#### PUT /api/profiles/:id — 更新 Profile
```bash
curl -X PUT http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "新名稱", "group": "客戶B"}'
```

#### DELETE /api/profiles/:id — 刪除 Profile
```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6
```

#### POST /api/profiles/:id/duplicate — 複製 Profile
新 ID、新指紋，保留 proxy 和分組。

---

### Groups（Group Proxy Policy）

Group 是 Profile 的分組標籤，也可以設定 group-scoped proxy policy。實際 proxy 會在瀏覽器啟動時決定：

| 模式 | Effective proxy 順序 |
|------|----------------------|
| `default` | Profile proxy 優先，接著 group proxy，最後無 proxy |
| `enforced` | Group proxy 優先，接著 profile proxy，最後無 proxy |

Group proxy 變更只會套用到新開啟的瀏覽器；已開啟的 profile browser 需要關閉後重新開啟才會套用。

#### GET /api/groups — 列出 Group Proxy Policy
```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/groups
```

#### PUT /api/groups/:name — 建立或更新 Group Proxy Policy
```bash
curl -X PUT http://127.0.0.1:19280/api/groups/%E5%AE%A2%E6%88%B6A \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"proxy_mode":"default","proxy":{"type":"socks5","host":"proxy.example.com","port":1080,"region":"us-ny"}}'
```

回應會包含 `active_sessions` 與 `restart_required`，方便提醒操作者是否需要重新開啟既有瀏覽器。

Proxy 物件支援 `type`、`host`、`port`、選填的 `username`、`password`、`region`。`region` 會 trim 後保存於 profile/group 設定，讓 Chromium native persona 的 WebRTC proxy-region metadata 能和實際選用的 proxy 對齊。

#### DELETE /api/groups/:name/proxy — 清除 Group Proxy
```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/groups/%E5%AE%A2%E6%88%B6A/proxy
```

#### DELETE /api/groups/:name — 刪除 Group
刪除 group label 與 group proxy 設定，但不會刪除 Profile；該 group 內的 Profile 會改為未分組。若 group 內還有已開啟的 browser session，API 會回傳 `409 GROUP_HAS_ACTIVE_SESSIONS`，需要先關閉那些瀏覽器，避免 runtime proxy 狀態不明確。

```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/groups/%E5%AE%A2%E6%88%B6A
```

---

### Session（瀏覽器控制）

#### POST /api/sessions — 開啟瀏覽器
```bash
curl -X POST http://127.0.0.1:19280/api/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"profile_id": "prof_a1b2c3d4e5f6"}'
```
```json
{"data": {"session_id": "sess_prof_a1b2c3d4e5f6", "launch_id": "sess_prof_a1b2c3d4e5f6-20261112T120000.000000000Z-1", "profile_id": "prof_a1b2c3d4e5f6", "runtime_id": "camoufox", "health_status": "healthy", "health_attempts": 1, "last_health_latency_ms": 42}}
```

開啟一個帶有該 Profile 指紋和 Proxy 的瀏覽器視窗。

#### GET /api/sessions — 列出活躍 Session
```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/sessions
```

Session response 會包含唯一的 `launch_id`、建立時間與 endpoint health 摘要。`POST /api/sessions` 失敗時，response 會附上 `diagnostics.endpoint_health_policy` 與該 profile 最新的 lifecycle events。

#### GET /api/diagnostics/sessions — Session lifecycle 診斷

依新到舊回傳 endpoint retry、browser disconnect、Playwright restart、runtime ID 與明確 cleanup reason。需要 Bearer token；可使用 `session_id`、`launch_id`、`profile_id` 與 `limit` 篩選。

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/diagnostics/sessions?profile_id=prof_a1b2c3d4e5f6&limit=50"
```

記憶體內歷史受 `endpoint_health.history_limit` 限制，BrowseForge 重啟後會清空；跨重啟診斷請查設定的 JSON log。

#### DELETE /api/sessions/:id — 關閉瀏覽器
```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/sess_prof_a1b2c3d4e5f6
```

---

### 瀏覽器操作

所有操作透過 Playwright，引擎無關（Firefox/Chromium 同一 API）。

#### POST /api/sessions/:id/navigate
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/navigate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://facebook.com", "wait_until": "load"}'
```
`wait_until`：`"load"` | `"domcontentloaded"` | `"networkidle"`

#### POST /api/sessions/:id/click
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/click \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "button#login"}'
```

#### POST /api/sessions/:id/type
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/type \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "input[name=email]", "text": "user@example.com", "delay": 50}'
```
`delay`：每個字元的間隔（毫秒），模擬打字速度。

#### POST /api/sessions/:id/eval — 執行 JavaScript
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/eval \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"script": "document.title"}'
```
```json
{"data": "Facebook - log in or sign up"}
```

#### GET /api/sessions/:id/screenshot
```bash
# 回傳 PNG binary
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/$SID/screenshot \
  -o screenshot.png

# 全頁截圖
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/sessions/$SID/screenshot?full_page=true" \
  -o full.png
```

#### GET /api/screenshots/:id
```bash
# 回傳 MCP screenshot URL delivery 建立的臨時截圖；不需要 Bearer token
curl "http://127.0.0.1:19280/api/screenshots/$ARTIFACT_ID" -o screenshot.png
```

此 URL 供外部 agent 直接下載圖片 bytes。ID 為隨機值，會在 `expires_at` 後失效，過期回 `404`。

#### GET /api/profiles/:id/artifacts/:path
```bash
# 回傳明確用 save_path 儲存的 profile artifact
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/profiles/$PROFILE_ID/artifacts/manual-evidence.png" \
  -o screenshot.png
```

此 endpoint 使用與 REST/MCP 相同的 Bearer token，並拒絕絕對路徑與 path traversal。

#### GET /api/sessions/:id/content
```bash
# 整頁 HTML
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/$SID/content

# 指定元素的文字
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/sessions/$SID/content?selector=h1"
```

#### POST /api/sessions/:id/wait
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/wait \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "#result", "timeout": 10000}'
```

#### GET /api/sessions/:id/cookies
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/$SID/cookies
```

#### POST /api/sessions/:id/cookies — 匯入 Cookies
```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/cookies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '[{"name":"session","value":"abc123","domain":".facebook.com","path":"/"}]'
```

---

### 備份/還原

#### POST /api/profiles/:id/export — 匯出單一 Profile
回傳 .zip 檔案。

#### POST /api/profiles/import — 匯入 Profile
```bash
curl -X POST http://127.0.0.1:19280/api/profiles/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@profile.zip"
```

#### POST /api/backup — 全量備份
匯出所有 profiles 與 group proxy policies。
回傳包含所有 Profile metadata 與 `groups.json` 的 .zip。

#### POST /api/restore — 全量還原
從備份 ZIP 還原 profiles 與 group proxy policies。既有 Profile 不會被覆蓋；備份內同名 group proxy policy 會更新。

```bash
curl -X POST http://127.0.0.1:19280/api/restore \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@backup.zip"
```

---

### Workflow

#### POST /api/workflow/run — 執行 Workflow
```bash
curl -X POST http://127.0.0.1:19280/api/workflow/run \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "自動登入",
    "steps": [
      {"name": "建立", "action": "create_profile", "params": {"name": "Auto", "runtime_id": "camoufox", "var": "p1"}},
      {"name": "開啟", "action": "open_browser", "profile_id": "$p1"},
      {"name": "導航", "action": "navigate", "profile_id": "$p1", "params": {"url": "https://facebook.com"}},
      {"name": "等待", "action": "sleep", "params": {"seconds": 5}},
      {"name": "關閉", "action": "close_browser", "profile_id": "$p1"}
    ]
  }'
```

支援的 action：
| Action | 說明 | 參數 |
|--------|------|------|
| `create_profile` | 建立 Profile | `name`, `runtime_id`, `var`（變數名） |
| `open_browser` | 開啟瀏覽器 | `profile_id` |
| `close_browser` | 關閉瀏覽器 | `profile_id` |
| `navigate` | 導航 | `profile_id`, `url` |
| `click` | 點擊 | `profile_id`, `selector` |
| `type` | 輸入 | `profile_id`, `selector`, `text` |
| `eval` | 執行 JS | `profile_id`, `script` |
| `wait` | 等待元素 | `profile_id`, `selector` |
| `screenshot` | 截圖 | `profile_id` |
| `sleep` | 等待 | `seconds` |

變數：`create_profile` 的 `var` 參數定義變數名，後續步驟用 `$變數名` 引用 profile_id。

---

## MCP Server（AI Agent 整合）

端點：`http://127.0.0.1:19280/mcp`
協議：JSON-RPC 2.0 over HTTP POST
規格：MCP 2025-11-25

遷移提醒：舊版 client 若使用獨立的 `:19281` MCP listener，請改成主服務 port 加上 `/mcp`。

### 連線

```python
# Python (mcp 套件)
from mcp import ClientSession
async with ClientSession("http://127.0.0.1:19280/mcp") as session:
    await session.initialize()
    tools = await session.list_tools()
```

### 可用 Tools

| Tool | 說明 | 參數 |
|------|------|------|
| `list_profiles` | 列出所有 Profile | — |
| `create_profile` | 建立 Profile | `name`, `runtime_id`, `group` |
| `delete_profile` | 刪除 Profile | `profile_id` |
| `update_profile` | 更新 Profile 設定 | `profile_id` |
| `list_groups` | 列出 group proxy policies | — |
| `get_group` | 讀取單一 group proxy policy | `group` |
| `update_group_proxy` | 設定 group-scoped proxy policy | `group`, `proxy`, `proxy_mode`（選填） |
| `clear_group_proxy` | 清除 group proxy policy | `group` |
| `delete_group` | 刪除 group label 與 group proxy policy，不刪除 Profile | `group` |
| `open_browser` | 開啟瀏覽器 | `profile_id` |
| `close_browser` | 關閉瀏覽器 | `profile_id` |
| `navigate` | 導航到 URL | `profile_id`, `url` |
| `click` | 點擊元素 | `profile_id`, `selector` |
| `type_text` | 輸入文字 | `profile_id`, `selector`, `text` |
| `screenshot` | 截圖 | `profile_id` |
| `get_content` | 取得頁面內容 | `profile_id`, `selector`（選填） |
| `evaluate` | 執行 JavaScript | `profile_id`, `script` |
| `new_tab` | 開啟新分頁 | `profile_id`, `url`（選填） |
| `list_tabs` | 列出分頁 | `profile_id` |
| `switch_tab` | 切換分頁 | `profile_id`, `index` |
| `close_tab` | 關閉分頁 | `profile_id`, `index` |
| `web_search` | 使用支援 agent web session 的 runtime profile 執行 provider-backed web search | `query`, `engine`（選填，`google`/`bing`/`duckduckgo`）, `profile_id` 或 `session_id`, `max_results`（選填） |
| `web_explore` | 使用支援 agent web session 的 runtime profile 探索網頁內容 | `url`, `profile_id` 或 `session_id`, `max_text_length`（選填）, `max_links`（選填） |
| `create_session` | 建立 agent web session | `profile_id` |
| `destroy_session` | 銷毀 agent web session | `session_id` |
| `list_sessions` | 列出 agent web sessions | `profile_id`（選填） |
| `gc_sessions` | 立即執行 agent web session GC | — |

只要 BrowseForge 能判斷可抓取的 base URL，`screenshot` 就會預設回傳臨時、免驗證的 `screenshot_url`；HTTP MCP 會從 request 或 `public_base_url` 推導，stdio MCP 可使用 `public_base_url` / `BROWSEFORGE_PUBLIC_BASE_URL`。URL ID 是隨機值，會在 `expires_at` 後失效；可用 `url_ttl_seconds` 設定存活秒數（預設 `600`，限制 `30-3600`）。可用 `delivery` 控制回傳：`url`、`image` 或 `both`；`include_image=false` 會省略 MCP image base64 block。URL screenshot artifacts 會存放在設定的 data 目錄，所以 stdio 建立的 URL 也能由主要 HTTP service 提供。請設定 `public_base_url` 或 `BROWSEFORGE_PUBLIC_BASE_URL` 來決定外部可見 URL，包含 reverse proxy path prefix。

### 範例：用 Claude 操作

```
User: 幫我建立一個 Firefox profile，然後開啟瀏覽器到 facebook.com

Claude: [呼叫 create_profile] → 建立了 prof_xxx
        [呼叫 open_browser] → 瀏覽器已開啟
        [呼叫 navigate] → 已導航到 facebook.com
```

一般瀏覽器控制 tools 可操作 Firefox 或 Chromium profile；`web_search`、`web_explore` 與 agent web session tools 需要 Chromium/CloakBrowser profile。

---

## 錯誤格式

所有錯誤回傳統一格式：
```json
{
  "error": {
    "code": "PROFILE_NOT_FOUND",
    "message": "profile not found: prof_xxx"
  }
}
```

常見錯誤碼：
| Code | HTTP | 說明 |
|------|------|------|
| `UNAUTHORIZED` | 401 | Token 無效或缺失 |
| `NOT_FOUND` | 404 | Profile 或 Session 不存在 |
| `MISSING_NAME` | 400 | 建立 Profile 時缺少 name |
| `LAUNCH_FAILED` | 500 | 瀏覽器啟動失敗 |
| `NAVIGATE_FAILED` | 500 | 導航失敗 |

## Playwright Connect

### GET /api/playwright/endpoint

取得所有 session 的 Playwright Connect endpoint。BrowseForge 目前使用 Playwright 1.60 整合版，回傳的 endpoint 來自 Playwright driver 的 `browser.Bind()`。

**Response:**
```json
{
  "data": [
    {
      "session_id": "sess_prof_abc123",
      "profile_id": "prof_abc123",
      "runtime_id": "cloakbrowser",
      "endpoint": "ws://127.0.0.1:54321/abcdef...",
      "proxy": "ws://your-host:19280/api/playwright/ws/sess_prof_abc123"
    }
  ]
}
```

外部 Playwright client 可以使用 `browserType.connect(endpoint)` 直連；遠端或 Docker 場景建議使用 `proxy` URL，並在連線時帶 Bearer Token。

**限制：** Client 必須使用與 BrowseForge driver 相容的 Playwright 版本；目前應使用 Playwright **1.60.x**。
