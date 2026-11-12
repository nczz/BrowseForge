# BrowseForge API Reference

[繁體中文](API.zh-TW.md)

## Connection Information

| Service | URL | Purpose |
|------|-----|---------|
| REST API | `http://127.0.0.1:19280/api` | Profile and browser automation |
| Dashboard | `http://127.0.0.1:19280` | Web management UI |
| MCP Streamable HTTP | `http://127.0.0.1:19280/mcp` | AI agent integration on the main service port |

## Authentication

All REST API endpoints except `/api/status` require a Bearer token:

```http
Authorization: Bearer <token>
```

The token is generated on first start and stored in `data/.api-token`. The Dashboard asks for the token on first use.

Error responses use stable, language-neutral `code` values:

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "invalid or missing token"
  }
}
```

## Response Shape

Successful responses wrap data in `data`:

```json
{
  "data": {}
}
```

List endpoints may include `total`.

## System

### GET `/api/status`

Public health/status endpoint.

```bash
curl http://127.0.0.1:19280/api/status
```

```json
{
  "version": "1.10.4",
  "status": "ok",
  "active_sessions": 1,
  "endpoint_health_policy": {
    "timeout_ms": 15000,
    "attempts": 2,
    "retry_delay_ms": 500,
    "probe_page": false,
    "history_limit": 256
  }
}
```

### POST `/api/shutdown`

Closes browser sessions and stops the server.

```bash
curl -X POST http://127.0.0.1:19280/api/shutdown \
  -H "Authorization: Bearer $TOKEN"
```

## Profiles

### POST `/api/profiles`

Creates a browser profile.

```bash
curl -X POST http://127.0.0.1:19280/api/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Brand Account 1",
    "engine": "firefox",
    "group": "Client A",
    "tags": ["facebook", "brand"],
    "proxy": {
      "type": "socks5",
      "host": "proxy.example.com",
      "port": 1080,
      "username": "user",
      "password": "pass"
    }
  }'
```

Fields:

| Field | Description |
|------|-------------|
| `name` | Required profile name |
| `engine` | `firefox` for Camoufox or `chromium` for CloakBrowser |
| `group` | Optional grouping label |
| `tags` | Optional string tags |
| `proxy` | Optional SOCKS5 or HTTP proxy configuration |
| `fingerprint` | Optional explicit fingerprint; auto-assigned when omitted |

### GET `/api/profiles`

Lists profiles.

```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/profiles
curl -H "Authorization: Bearer $TOKEN" "http://127.0.0.1:19280/api/profiles?group=Client%20A&tag=facebook"
```

### GET `/api/profiles/{id}`

Gets a single profile.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6
```

### PUT `/api/profiles/{id}`

Updates profile metadata, proxy settings, or other mutable fields.

```bash
curl -X PUT http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Renamed Profile", "group": "Client B"}'
```

### DELETE `/api/profiles/{id}`

Deletes a profile.

```bash
curl -X DELETE http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6 \
  -H "Authorization: Bearer $TOKEN"
```

### POST `/api/profiles/{id}/duplicate`

Creates a copy with a new profile ID and fingerprint while retaining group/proxy settings.

```bash
curl -X POST http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6/duplicate \
  -H "Authorization: Bearer $TOKEN"
```

### POST `/api/profiles/{id}/export`

Exports one profile as a ZIP archive.

```bash
curl -X POST http://127.0.0.1:19280/api/profiles/prof_a1b2c3d4e5f6/export \
  -H "Authorization: Bearer $TOKEN" \
  -o profile.zip
```

## Groups

Groups are profile labels plus optional proxy policy. A group proxy is resolved at browser launch time:

| Mode | Effective proxy order |
|------|-----------------------|
| `default` | Profile proxy override, then group proxy, then no proxy |
| `enforced` | Group proxy override, then profile proxy, then no proxy |

Proxy changes affect newly opened browser sessions. Close and reopen active profile browsers in that group to apply a changed group proxy policy.

### GET `/api/groups`

Lists configured group proxy policies.

```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/groups
```

### GET `/api/groups/{name}`

Gets one group proxy policy.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/groups/Client%20A
```

### PUT `/api/groups/{name}`

Creates or updates a group proxy policy.

```bash
curl -X PUT http://127.0.0.1:19280/api/groups/Client%20A \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "proxy_mode": "default",
    "proxy": {
      "type": "socks5",
      "host": "proxy.example.com",
      "port": 1080,
      "username": "user",
      "password": "pass"
    }
  }'
```

Response includes `active_sessions` and `restart_required` so callers can warn operators when existing browsers need to be reopened.

### DELETE `/api/groups/{name}`

Deletes the group label and its group proxy setting without deleting profiles. Profiles in the group become ungrouped. If the group has active browser sessions, the request returns `409 GROUP_HAS_ACTIVE_SESSIONS`; close those browsers first so runtime proxy state cannot become ambiguous.

```bash
curl -X DELETE http://127.0.0.1:19280/api/groups/Client%20A \
  -H "Authorization: Bearer $TOKEN"
```

### DELETE `/api/groups/{name}/proxy`

Clears the group proxy policy without deleting profiles in the group.

```bash
curl -X DELETE http://127.0.0.1:19280/api/groups/Client%20A/proxy \
  -H "Authorization: Bearer $TOKEN"
```

### POST `/api/profiles/import`

Imports a profile ZIP.

```bash
curl -X POST http://127.0.0.1:19280/api/profiles/import \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@profile.zip"
```

## Sessions

### POST `/api/sessions`

Starts a browser session for a profile.

```bash
curl -X POST http://127.0.0.1:19280/api/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"profile_id": "prof_a1b2c3d4e5f6"}'
```

```json
{
  "data": {
    "session_id": "sess_prof_a1b2c3d4e5f6",
    "launch_id": "sess_prof_a1b2c3d4e5f6-20261112T120000.000000000Z-1",
    "profile_id": "prof_a1b2c3d4e5f6",
    "engine": "firefox",
    "health_status": "healthy",
    "health_attempts": 1,
    "last_health_latency_ms": 42
  }
}
```

### GET `/api/sessions`

Lists active sessions.

```bash
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:19280/api/sessions
```

Session responses include a unique `launch_id`, creation time, and endpoint health summary. A failed `POST /api/sessions` response includes `diagnostics.endpoint_health_policy` and the latest lifecycle events for that profile.

### GET `/api/diagnostics/sessions`

Returns newest-first in-memory lifecycle events, including endpoint retries, browser disconnects, Playwright restarts, and explicit cleanup reasons. Requires authentication. Optional filters: `session_id`, `launch_id`, `profile_id`, and `limit`.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/diagnostics/sessions?profile_id=prof_a1b2c3d4e5f6&limit=50"
```

The history is bounded by `endpoint_health.history_limit` and resets when BrowseForge restarts. Use the configured JSON log for cross-restart diagnosis.

### DELETE `/api/sessions/{id}`

Closes a browser session.

```bash
curl -X DELETE http://127.0.0.1:19280/api/sessions/sess_prof_a1b2c3d4e5f6 \
  -H "Authorization: Bearer $TOKEN"
```

## Browser Automation

All browser automation endpoints operate through Playwright and work across supported engines unless a runtime-specific limitation is documented.

### POST `/api/sessions/{id}/navigate`

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/navigate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "wait_until": "load"}'
```

`wait_until` accepts `load`, `domcontentloaded`, or `networkidle`.

### POST `/api/sessions/{id}/click`

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/click \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "button#login"}'
```

### POST `/api/sessions/{id}/type`

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/type \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "input[name=email]", "text": "user@example.com", "delay": 50}'
```

### POST `/api/sessions/{id}/eval`

Evaluates JavaScript in the page context.

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/eval \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"script": "document.title"}'
```

### GET `/api/sessions/{id}/screenshot`

Returns a PNG screenshot.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/sessions/$SID/screenshot?full_page=true" \
  -o screenshot.png
```

### GET `/api/sessions/{id}/content`

Returns full HTML or selected element text.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/$SID/content

curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:19280/api/sessions/$SID/content?selector=h1"
```

### POST `/api/sessions/{id}/wait`

Waits for a selector.

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/wait \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selector": "#result", "timeout": 10000}'
```

### GET `/api/sessions/{id}/cookies`

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/sessions/$SID/cookies
```

### POST `/api/sessions/{id}/cookies`

Imports cookies into the session context.

```bash
curl -X POST http://127.0.0.1:19280/api/sessions/$SID/cookies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '[{"name":"session","value":"abc123","domain":".example.com","path":"/"}]'
```

## Backup and Restore

### POST `/api/backup`

Exports all profiles and group proxy policies as a ZIP archive.

```bash
curl -X POST http://127.0.0.1:19280/api/backup \
  -H "Authorization: Bearer $TOKEN" \
  -o browseforge-backup.zip
```

### POST `/api/restore`

Restores profiles and group proxy policies from a backup ZIP. Existing profiles are not overwritten; group proxy policies in the backup update matching group names.

```bash
curl -X POST http://127.0.0.1:19280/api/restore \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@browseforge-backup.zip"
```

## Playwright Connect

### GET `/api/playwright/endpoint`

Lists Playwright-compatible endpoints for active sessions.

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:19280/api/playwright/endpoint
```

### WebSocket `/api/playwright/ws/{session_id}`

Proxy endpoint for external Playwright clients.

```javascript
import { firefox } from 'playwright';

const browser = await firefox.connect(
  'ws://YOUR_SERVER:19280/api/playwright/ws/sess_prof_xxx',
  { headers: { Authorization: 'Bearer YOUR_TOKEN' } }
);
```

Compatibility notes:

- Use Playwright client `1.60.x`.
- Proxy mode requires only port `19280`.
- Bearer token authentication is required.

## Workflow

### POST `/api/workflow/run`

Runs a YAML workflow.

```bash
curl -X POST http://127.0.0.1:19280/api/workflow/run \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/yaml" \
  -d @examples/multi-login.yaml
```

Common actions:

- `create_profile`
- `open_browser`
- `close_browser`
- `navigate`
- `click`
- `type`
- `eval`
- `wait`
- `screenshot`
- `sleep`

## MCP Tools

BrowseForge exposes MCP tools at `http://127.0.0.1:19280/mcp` (Streamable HTTP transport, JSON-RPC 2.0). MCP is mounted on the main BrowseForge HTTP service port rather than a separate listener.

Migration note: older clients configured for a separate `:19281` MCP listener should update to the main service port plus `/mcp`.

For recommended agent tool-use prompting, see [docs/agent-prompt-guide.md](docs/agent-prompt-guide.md).

### Authentication

All MCP requests require Bearer token authentication:

```http
Authorization: Bearer <token>
```

### Tool List

| Tool | Description |
|------|-------------|
| `list_profiles` | List all browser profiles |
| `create_profile` | Create a new browser profile |
| `delete_profile` | Delete a profile |
| `update_profile` | Update profile settings |
| `list_groups` | List group proxy policies |
| `get_group` | Read one group proxy policy |
| `update_group_proxy` | Set a group-scoped proxy policy |
| `clear_group_proxy` | Clear a group proxy policy |
| `delete_group` | Delete a group label and proxy policy without deleting profiles |
| `open_browser` | Open a browser session for a profile |
| `close_browser` | Close a browser session |
| `navigate` | Navigate to a URL |
| `click` | Click an element |
| `type_text` | Type text into an element |
| `screenshot` | Take a screenshot |
| `get_content` | Get page content |
| `evaluate` | Execute JavaScript |
| `new_tab` | Open a new tab |
| `list_tabs` | List all tabs |
| `switch_tab` | Switch to a tab |
| `close_tab` | Close a tab |
| `web_search` | Search the web using a profile-bound agent session |
| `web_explore` | Explore a webpage using a profile-bound agent session |
| `create_session` | Create an agent web session for a Chromium profile |
| `destroy_session` | Destroy an agent web session and close its page |
| `list_sessions` | List active agent web sessions |
| `gc_sessions` | Trigger web session garbage collection |
| `wait_for` | Wait for a page selector state |
| `get_page_state` | Get URL/title/text/focus/tab state for the current page |
| `get_cookies` | Read browser context cookies |
| `set_cookies` | Add browser context cookies |
| `run_workflow` | Execute a BrowseForge workflow through the workflow engine |
| `form_fill` | Fill multiple fields with humanized typing |
| `select_option` | Select `<select>` options |
| `check` | Check or uncheck checkbox/radio elements |
| `press_key` | Press a keyboard key or shortcut |
| `list_downloads` | List files in a profile downloads directory |
| `read_download` | Read a small file from a profile downloads directory |
| `delete_download` | Delete a file from a profile downloads directory |
| `web_extract` | Extract structured fields from the current page with selector schema |
| `doctor_profile` | Diagnose profile/browser/session readiness |

### Agent Web Sessions

`web_search` and `web_explore` run through profile-bound agent sessions:

- Only Chromium/CloakBrowser profiles are accepted for these tools.
- One profile has one persistent browser instance owned by `browser.Manager`.
- `SessionPool` connects to that browser through the profile's Playwright Bind endpoint.
- Each agent session opens one independent `Page` via `connectedBrowser.NewPage()`; it does not create a separate browser or `BrowserContext`.
- `session_id` pins later calls to the same page.
- GC/destroy/shutdown close idle agent pages and metadata only; they do **not** close the profile browser.

Defaults:

| Setting | Default |
|---------|---------|
| Idle TTL | 5 minutes |
| GC sweep interval | 1 minute |
| Max sessions per profile | 10 |

MCP response-shape compatibility note:

- BrowseForge keeps the standard MCP `content` text block as the primary human/client-compatible payload.
- For session-aware web tools, machine-readable session metadata is also exposed as top-level result fields: `session_id`, `profile_id`, and, when applicable, `session_created`.
- `web_search` also exposes top-level `extraction_mode`, `results`, and, when structured extraction has no results, `raw_fallback` for LLM-friendly SERP interpretation.
- This avoids breaking clients that only read `content` while still allowing agents to reuse pages reliably without parsing the text payload.

MCP error codes used by these tools:

| Code | Meaning |
|------|---------|
| `-32602` | Missing required argument, e.g. `query`, `url`, `profile_id`, or `session_id` |
| `-32000` | Runtime/session failure, e.g. session pool unavailable, profile not found, non-Chromium profile, browser launch/connect failure, navigation/search failure |

### `web_search`

Search the web with a provider-backed search engine and return structured results with title, URL, and snippet. Supported engines are `google`, `bing`, and `duckduckgo` (`ddg` is accepted as an alias for `duckduckgo`). If an engine's DOM shape changes and structured extraction returns no results, BrowseForge returns an LLM-friendly raw SERP fallback containing page text and candidate links while preserving explicit captcha/consent/unusual-traffic errors.

**Parameters:**

| Parameter | Type | Required | Description |
|------|------|------|-------------|
| `query` | string | Yes | Search query |
| `engine` | string | No | Search engine: `google`, `bing`, or `duckduckgo`. Default `google` |
| `profile_id` | string | Required if `session_id` is omitted | Chromium/CloakBrowser profile to use |
| `session_id` | string | No | Existing agent session to reuse; when omitted, a new session/page is created |
| `max_results` | number | No | Maximum results. Default `10`; values above `30` are clamped by `WebSearch` |

**Result shape:** `content[0].text` is a text prefix followed by pretty-printed JSON. Top-level result fields include `session_id`, `profile_id`, `session_created`, `engine`, `extraction_mode`, `results`, and optional `raw_fallback`.

```json
{
  "content": [{
    "type": "text",
    "text": "Found 5 results for \"Go programming language tutorial\" (mode: structured):\n{...}"
  }],
  "session_id": "sess_search_0123abcd",
  "profile_id": "prof_abc123",
  "session_created": true,
  "engine": "google",
  "extraction_mode": "structured",
  "results": [
    {"title": "Result title", "url": "https://example.com", "snippet": "Result snippet"}
  ]
}
```

When structured extraction is empty, `extraction_mode` is `raw_fallback` and `raw_fallback` contains:

```json
{
  "page_title": "Search Results",
  "text": "visible SERP text for LLM interpretation...",
  "candidate_links": [{"text": "Candidate", "url": "https://example.com"}]
}
```

### `web_explore`

Navigate to a URL and extract structured content: URL, title, optional meta description, visible text, and links.

**Parameters:**

| Parameter | Type | Required | Description |
|------|------|------|-------------|
| `url` | string | Yes | URL to explore; `https://` is prepended when no `http://` or `https://` prefix is provided |
| `profile_id` | string | Required if `session_id` is omitted | Chromium/CloakBrowser profile to use |
| `session_id` | string | No | Existing agent session to reuse; when omitted, a new session/page is created |
| `max_text_length` | number | No | Maximum text length. Default `3000`; extraction clamps to `10000` |
| `max_links` | number | No | Maximum links to extract. Default `50`; extraction clamps to `200` |

**Result shape:** `content[0].text` is pretty-printed JSON with `url`, `title`, `text`, `links`, and optional `description`. Top-level result fields also include `session_id`, `profile_id`, and `session_created`.

```json
{
  "content": [{
    "type": "text",
    "text": "{\n  \"url\": \"https://example.com\",\n  \"title\": \"Example Domain\",\n  \"text\": \"Example Domain...\",\n  \"links\": []\n}"
  }],
  "session_id": "sess_search_0123abcd",
  "profile_id": "prof_abc123",
  "session_created": true
}
```

### `create_session`

Create an agent web session for a Chromium/CloakBrowser profile without performing a search or page exploration.

**Parameters:** `profile_id` string, required.

**Returns:** text confirmation plus top-level `session_id` and `profile_id`.

```json
{
  "content": [{
    "type": "text",
    "text": "Session created: sess_search_0123abcd (profile: prof_abc123, browser: sess_prof_abc123)"
  }],
  "session_id": "sess_search_0123abcd",
  "profile_id": "prof_abc123"
}
```

### `destroy_session`

Destroy a session and close its agent page.

**Parameters:** `session_id` string, required.

**Returns:** text confirmation plus top-level `session_id`.

```json
{
  "content": [{
    "type": "text",
    "text": "Session destroyed: sess_search_0123abcd"
  }],
  "session_id": "sess_search_0123abcd"
}
```

### `list_sessions`

List active agent web sessions. Optional `profile_id` filters the result.

**Parameters:** `profile_id` string, optional.

**Returns:** `content[0].text` as pretty-printed JSON array of session info objects: `id`, `profile_id`, `browser_id`, `created_at`, `last_accessed`, and `idle_seconds`.

### `gc_sessions`

Run session GC immediately.

**Parameters:** none.

**Returns:** text confirmation such as `GC completed: closed 2 sessions`.

### Agent Page Utilities

These tools accept either `profile_id` for the active profile browser page or `session_id` for an agent web session page.

#### `wait_for`

Wait for a selector instead of using fixed sleeps.

Parameters: `selector` required; optional `profile_id`, `session_id`, `state` (`attached`, `visible`, `hidden`, `detached`; default `visible`), and `timeout` in milliseconds.

Returns top-level `matched`, `selector`, `state`, `url`, `title`, `elapsed_ms`, and session/profile metadata.

#### `get_page_state`

Return a compact page observation for agent planning: `url`, `title`, visible text excerpt, active element metadata, tab index, tab count, and session/profile metadata.

Optional `text_max_length` controls the visible text excerpt length.

#### `form_fill`

Fill multiple fields using BrowseForge's existing humanized typing path:

```json
{
  "profile_id": "prof_abc123",
  "fields": [
    {"selector": "#email", "text": "user@example.com", "clear": true},
    {"selector": "#password", "text": "secret", "clear": true}
  ]
}
```

#### `select_option`, `check`, and `press_key`

Use Playwright-native page APIs for common input actions:

- `select_option`: `selector` plus one of `values`, `labels`, or `indexes`.
- `check`: `selector`, optional `checked` boolean (`true` by default).
- `press_key`: `key`, optional `delay` milliseconds.

#### `web_extract`

Extract structured page data using a deterministic selector schema. This tool does not call an LLM.

```json
{
  "session_id": "sess_search_0123abcd",
  "schema": {
    "headline": {"selector": "h1", "attr": "text"},
    "canonical": {"selector": "link[rel=canonical]", "attr": "href"},
    "links": {"selector": "main a", "attr": "href", "all": true}
  }
}
```

Returns `url`, `title`, extracted `data`, and selector `evidence`.

### Cookies, Downloads, And Artifacts

#### `get_cookies` / `set_cookies`

Read or add Playwright browser-context cookies for a profile. `set_cookies` accepts a Playwright `OptionalCookie` array in `cookies`.

#### `screenshot` artifact saving

`screenshot` still returns an MCP image block. It also accepts:

| Parameter | Type | Description |
|------|------|-------------|
| `format` | string | `jpeg` or `png`; default `jpeg` |
| `save_path` | string | Optional path under the profile `artifacts` directory |

Absolute paths and path traversal are rejected; saved files stay under the profile artifacts directory.

#### `list_downloads`, `read_download`, `delete_download`

Manage files in a profile's `downloads` directory:

- `list_downloads`: optional `limit`, default `50`.
- `read_download`: `name` plus optional `max_bytes`, default `1048576`. Text-like files return `text`; binary files return `base64`.
- `delete_download`: removes one file by `name`.

Only direct file names inside `downloads` are accepted.

### Workflow And Diagnostics

#### `run_workflow`

Execute the existing BrowseForge workflow engine from MCP. HTTP MCP supports this when the server injects the workflow engine; stdio MCP reports unavailable because it does not own an HTTP API base.

Input can be a workflow object:

```json
{
  "workflow": {
    "name": "login-smoke",
    "steps": [
      {"name": "open", "action": "open_browser", "profile_id": "prof_abc123"},
      {"name": "go", "action": "navigate", "profile_id": "prof_abc123", "params": {"url": "https://example.com"}}
    ]
  }
}
```

or a `yaml` string with the same workflow shape.

Workflow `screenshot` actions now call the REST screenshot endpoint and save the image to `params.path` or a generated `artifacts/workflow-*.png` path.

#### `doctor_profile`

Return profile readiness data: engine, group, profile/download directories, browser running status, Playwright Bind endpoint presence, active URL, tab count, effective proxy source/mode, and active agent web sessions.

### Common Patterns

**Search and explore first result with the same agent session:**

```json
// Step 1: Search. Save returned session_id.
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "web_search",
    "arguments": {
      "query": "latest Go release notes",
      "engine": "google",
      "profile_id": "prof_abc123"
    }
  }
}

// Step 2: Explore the first result URL using the same page/session.
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "web_explore",
    "arguments": {
      "url": "<url_from_step_1>",
      "session_id": "<session_id_from_step_1>"
    }
  }
}
```

## Security Notes

- Do not expose `19280` or `6901` directly to the public internet.
- Treat tokens, profiles, cookies, backups, and exported profile ZIPs as sensitive.
- Use SSH tunnels, VPN, or a hardened HTTPS reverse proxy for remote access.
