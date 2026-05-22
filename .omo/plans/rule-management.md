# Rule & Rule-Provider Management Implementation Plan

## Current State

| Component | Status |
|-----------|--------|
| `internal/ui/pages/rules.go` | ✅ Fully built (table, search, filter) — but **orphaned** |
| `internal/models/clash.go` Rule struct | ✅ Exists (Type, Payload, Proxy) |
| `internal/api/client.go` GetRules() | ✅ Exists (GET /rules) |
| `internal/types/config.go` Rules/RuleProviders fields | ✅ Exists |
| `internal/types/proxygroup.go` RuleProviderDef | ✅ Exists |
| `internal/merge/engine.go` rule merging | ✅ Exists |
| App registration (sidebar, pageNames, setupPages) | ❌ RulesPage not registered |
| API: rule modification (add/delete/reorder) | ❌ Not implemented |
| API: rule-provider CRUD (GET/PUT /providers/rules) | ❌ Not implemented |
| YAML: rule-provider config editing | ❌ Not implemented |
| UI: RuleProvidersPage | ❌ Not implemented |
| UI: Quick-import from Loyalsoldier/clash-rules | ❌ Not implemented |

---

## Phase 1: Activate Existing RulesPage (3 files, ~15 lines)

Register the already-built RulesPage so it's accessible from the sidebar.

**Files:**
1. `internal/app.go` — Add "rules" to pageNames, add RulesPage in setupPages()
2. `internal/ui/pages/pages.go` — Add Rules wrapper + NewRules() factory
3. `internal/ui/components/sidebar.go` — Add "规则" nav item

**Expected result:** Navigate to rules tab → see live rule table with search/filter.

---

## Phase 2: RuleProvider Model & API (2 files, ~60 lines)

Add the backend for rule-provider operations.

**Files:**
4. `internal/models/clash.go` — Add `RuleProvider` struct (name, type, behavior, url, path, interval, updatedAt, ruleCount)
5. `internal/api/client.go` — Add:
   - `GetRuleProviders() (*RuleProvidersResponse, error)` → GET `/providers/rules`
   - `RefreshRuleProvider(name string) error` → PUT `/providers/rules/{name}`

**Expected result:** Can fetch and refresh rule-providers programmatically.

---

## Phase 3: YAML Config Editing for Rule-Providers (1 file, ~120 lines)

Mirror the proxy-provider YAML manipulation pattern for rule-providers.

**File:**
6. `internal/config/mihomo_config.go` — Add:
   - `GetRuleProvidersFromConfig() ([]RuleProviderEntry, error)`
   - `AddRuleProviderToConfig(name, url, behavior string, interval int) (configPath string, err error)`
   - `DeleteRuleProviderFromConfig(name string) (configPath string, err error)`
   - `RuleProviderEntry` struct (name, type, behavior, url, path, interval)
   - Wire new provider into rules section (add `RULE-SET,{name},DIRECT` for DIRECT-behavior providers)

**Expected result:** Can add/delete rule-providers in mihomo config.yaml with hot-reload.

---

## Phase 4: RulesPage Enhancement — CRUD (1 file, ~200 lines)

Add rule creation, deletion, and reordering to the existing RulesPage.

**File:**
7. `internal/ui/pages/rules.go` — Add:
   - Action bar with buttons: [A] 添加规则, [D] 删除规则, [↑↓] 移动规则
   - `AddRuleDialog` — modal form: type dropdown (DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD, IP-CIDR, GEOIP, MATCH, RULE-SET), payload input, proxy input
   - `DeleteSelectedRule()` — removes the selected rule from config
   - `MoveRuleUp/MoveRuleDown()` — reorder rules
   - Keyboard shortcuts: 'a' add, 'd' delete, 'j' move down, 'k' move up
   - YAML writing via `config.AddRuleToConfig()` / `config.DeleteRuleFromConfig()` / `config.ReorderRules()`
   - Fix `Refresh()` to use `go` for `UpdateUi` calls (prevent deadlock)

**Expected result:** Full CRUD on inline rules from the UI.

---

## Phase 5: RuleProvidersPage — New UI Page (1 file, ~350 lines)

Build a dedicated page for managing rule-providers.

**File:**
8. `internal/ui/pages/ruleproviders/page.go` — New file:
   - List of rule-providers (name, behavior, URL, status, rule count)
   - Detail view showing provider metadata
   - Action bar: [R] 刷新, [A] 添加, [U] 更新, [D] 删除
   - `AddProviderDialog` — form: name, URL, behavior (auto-detect from URL), interval
   - Quick-import section: pre-built list of Loyalsoldier/clash-rules providers with checkboxes
   - Keyboard shortcuts: 'a' add, 'd' delete, 'u' update selected, 'r' refresh
   - Async refresh with status indicator
   - Follow same goroutine-safe patterns as fixed proxygroups/subscriptions pages:
     - Form data captured on UI goroutine before launching background worker
     - Single `go ui.Updater.UpdateUi(...)` for post-I/O UI updates
     - Direct tview ops on UI goroutine for button/keyboard handlers
     - `showError/showSuccess/showStatus` use `go ui.Updater.UpdateUi(...)`

**Integration:**
- Register in `app.go` (pageNames, setupPages)
- Add to `sidebar.go`
- Add `NewRuleProviders()` factory in `pages/pages.go`

**Expected result:** Full rule-provider management with quick import from clash-rules.

---

## Phase 6: YAML Rule Editing Functions (1 file, ~100 lines)

Add functions for editing inline rules in the YAML config.

**File:**
9. `internal/config/mihomo_config.go` — Add:
   - `AddRuleToConfig(ruleType, payload, proxy string) (configPath string, err error)`
   - `DeleteRuleFromConfig(index int) (configPath string, err error)`
   - `UpdateRuleInConfig(index int, ruleType, payload, proxy string) (configPath string, err error)`
   - `ReorderRules(newOrder []int) (configPath string, err error)`
   - All use the same yaml.Node manipulation pattern as proxy-group functions

**Expected result:** Programmatic rule CRUD on config files.

---

## Pre-built Rule-Provider Quick-Import List

Based on Loyalsoldier/clash-rules analysis:

```go
var BuiltinRuleProviders = []struct{ Name, URL, Behavior, Description string }{
    {"reject",       "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt",       "domain",    "广告/恶意域名 → REJECT"},
    {"proxy",        "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt",        "domain",    "需要代理的境外域名 → PROXY"},
    {"direct",       "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/direct.txt",       "domain",    "国内直连域名 → DIRECT"},
    {"private",      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/private.txt",      "domain",    "私有网络/路由器 → DIRECT"},
    {"icloud",       "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/icloud.txt",       "domain",    "iCloud 服务 → DIRECT"},
    {"apple",        "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/apple.txt",        "domain",    "Apple 国内服务 → DIRECT"},
    {"google",       "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/google.txt",       "domain",    "Google 国内服务 → PROXY"},
    {"gfw",          "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/gfw.txt",          "domain",    "GFW 屏蔽域名 → PROXY"},
    {"tld-not-cn",   "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/tld-not-cn.txt",   "domain",    "非中国 TLD → PROXY"},
    {"telegramcidr", "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/telegramcidr.txt", "ipcidr",    "Telegram IP 段 → PROXY"},
    {"cncidr",       "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/cncidr.txt",       "ipcidr",    "中国大陆 IP 段 → DIRECT"},
    {"lancidr",      "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/lancidr.txt",      "ipcidr",    "内网 IP 段 (RFC1918) → DIRECT"},
    {"applications", "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/applications.txt", "classical", "应用进程名规则 → DIRECT"},
}
```

---

## Execution Order & Dependencies

```
Phase 1 (registers existing page)     ← no deps, start immediately
    ↓
Phase 2 (models + API)               ← needed by Phase 5
    ↓
Phase 3 (YAML editing for providers) ← needed by Phase 5
    ↓
Phase 6 (YAML editing for rules)     ← needed by Phase 4
    ↓
Phase 5 (RuleProvidersPage)          ← parallel with Phase 4
Phase 4 (RulesPage CRUD enhancement) ← parallel with Phase 5
```

## Verification for Each Phase

- `go build ./...` must pass
- `go vet` must pass
- All `UpdateUi` calls from UI goroutine must use `go` prefix
- All tview widget reads must be on UI goroutine (capture before `go`)
