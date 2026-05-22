# Rules & Rule-Provider Feature Implementation Plan

## Discovered Assets

### Already Built (Orphaned)
- `internal/ui/pages/rules.go` — fully functional `RulesPage` with table, search, keyboard shortcuts, showing Type/Payload/Proxy from `GET /rules` API. Never registered in app/sidebar.
- `internal/models/clash.go` — `Rule` model (Type, Payload, Proxy)
- `internal/types/config.go` — `ConfigDocument.Rules` ([]string), `.RuleProviders` ([]map)
- `internal/types/proxygroup.go` — `RuleProviderDef` struct
- `internal/merge/engine.go` — `mergeRules()`, `mergeNamedMaps()` for rule-providers
- `internal/api/client.go` — `GetRules()` (GET /rules)

### Missing
- Rule-provider API methods (GET /providers/rules, PUT /providers/rules/:name)
- Rule-provider YAML editing (mirror of proxy-provider pattern)
- Rule CRUD: add/delete/reorder inline rules via YAML
- Rule-provider status model
- Quick-import from Loyalsoldier/clash-rules

### External Reference
- Loyalsoldier/clash-rules: 14 rule files (reject, proxy, direct, private, icloud, apple, google, gfw, tld-not-cn, telegramcidr, cncidr, lancidr, applications) across 3 behaviors (domain, ipcidr, classical)

---

## Phase 1: Register Existing RulesPage (3 files, ~10 lines total)

### 1.1 Add "rules" to pageNames (app.go:42)
```go
pageNames: []string{"dashboard", "proxies", "connections", "config", "logs", "subscriptions", "proxygroups", "rules"},
```

### 1.2 Register RulesPage in setupPages() (app.go)
```go
rulesPage := pages.NewRules()
a.pages.AddPage("rules", rulesPage, true, false)
```

### 1.3 Add factory in pages/pages.go
```go
type Rules struct { *RulesPage }
func NewRules() *Rules { return &Rules{RulesPage: NewRulesPage()} }
```

### 1.4 Add "规则" to sidebar (sidebar.go:29)
```go
{Label: "规则", Icon: "📋", Shortcut: "R"},
```
Note: shortcut "R" may conflict with Connections' shortcut - change Connections to "C" or Rules to "L" to avoid collision. Check existing shortcuts: D(dashboard), P(proxies), R(connections), C(config), L(logs), S(subscriptions), G(proxygroups). Use "U" or "T" for rules.

---

## Phase 2: Rule-Provider Backend (API + YAML)

### 2.1 Add RuleProvider model (models/clash.go)
```go
type RuleProvider struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    VehicleType string   `json:"vehicleType"`
    Behavior    string   `json:"behavior"`
    RuleCount   int      `json:"ruleCount"`
    UpdatedAt   string   `json:"updatedAt"`
}

type RuleProvidersResponse struct {
    Providers map[string]*RuleProvider `json:"providers"`
}
```

### 2.2 Add API methods (api/client.go)
```go
func (c *HttpClient) GetRuleProviders() (*models.RuleProvidersResponse, error) {
    // GET /providers/rules
}

func (c *HttpClient) RefreshRuleProvider(name string) error {
    // PUT /providers/rules/{name}
}
```

### 2.3 Add YAML editing for rule-providers (config/mihomo_config.go)
Mirror the existing proxy-provider pattern:

```go
// GetRuleProvidersFromConfig returns rule-provider definitions from config file
func GetRuleProvidersFromConfig() ([]RuleProviderEntry, error)

// AddRuleProviderToConfig adds a new rule-provider to the config
func AddRuleProviderToConfig(name, behavior, url, path string, interval int) (string, error)

// DeleteRuleProviderFromConfig removes a rule-provider by name
func DeleteRuleProviderFromConfig(name string) (string, error)

type RuleProviderEntry struct {
    Name     string `yaml:"-"`
    Type     string `yaml:"type"`     // "http"
    Behavior string `yaml:"behavior"` // domain|ipcidr|classical
    URL      string `yaml:"url"`
    Path     string `yaml:"path"`
    Interval int    `yaml:"interval"`
}
```

---

## Phase 3: Rule-Provider Management Page (new file + wiring)

### 3.1 Create internal/ui/pages/ruleproviders/page.go
New page showing:
- **Left panel**: List of rule-providers (name, behavior, rule count, status)
- **Right panel**: Detail view + action bar
- **Action buttons**: [R] Refresh selected, [D] Delete, [I] Import from URL, [Q] Quick import

Behaviors:
- `Activate()`: loads rule-providers from both API (/providers/rules) and config
- Refresh button: calls `PUT /providers/rules/{name}` then re-queries
- Delete: calls `DeleteRuleProviderFromConfig()`, then reload via API
- Import form: name, URL, behavior dropdown, interval input
- Quick-import: pre-built list of Loyalsoldier/clash-rules entries

### 3.2 Register in app infrastructure
- Add "ruleproviders" to pageNames
- Register in setupPages()
- Add factory in pages/pages.go
- Add "规则提供者" to sidebar with unique shortcut

### 3.3 Quick Import — Loyalsoldier presets
Hardcoded table of the 14 rule-providers with pre-filled name, URL, behavior:

| Name | URL | Behavior |
|------|-----|----------|
| reject | cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt | domain |
| proxy | .../proxy.txt | domain |
| direct | .../direct.txt | domain |
| private | .../private.txt | domain |
| icloud | .../icloud.txt | domain |
| apple | .../apple.txt | domain |
| google | .../google.txt | domain |
| gfw | .../gfw.txt | domain |
| tld-not-cn | .../tld-not-cn.txt | domain |
| telegramcidr | .../telegramcidr.txt | ipcidr |
| cncidr | .../cncidr.txt | ipcidr |
| lancidr | .../lancidr.txt | ipcidr |
| applications | .../applications.txt | classical |

---

## Phase 4: Enhance RulesPage with CRUD

### 4.1 Add YAML editing for inline rules (config/mihomo_config.go)
```go
func AddRuleToConfig(ruleType, payload, policy string) (string, error)
func DeleteRuleFromConfig(payload string) (string, error)  // match by payload
func GetRulesFromConfig() ([]string, error)
```

### 4.2 Add keyboard/button commands to RulesPage
- `d` Delete: remove selected rule from config
- `a` Add: show form to add new rule (type dropdown + payload input + policy input)
- `e` Edit: edit selected rule
- Refresh: reload after CRUD

---

## File Modification Summary

| File | Action | Lines |
|------|--------|-------|
| `internal/app.go` | Add pageNames entries + setupPages registrations | ~8 |
| `internal/ui/components/sidebar.go` | Add nav items (2 new) | ~4 |
| `internal/ui/pages/pages.go` | Add factory functions (2 new) | ~12 |
| `internal/models/clash.go` | Add RuleProvider model | ~15 |
| `internal/api/client.go` | Add GetRuleProviders + RefreshRuleProvider | ~35 |
| `internal/config/mihomo_config.go` | Add RuleProviderEntry + YAML CRUD (5 functions) | ~180 |
| `internal/ui/pages/ruleproviders/page.go` | NEW: rule-provider management page | ~400 |
| `internal/ui/pages/rules.go` | Add rule CRUD buttons + handlers | ~80 |

Total: ~734 lines across 8 files (7 modified, 1 new)

---

## Validation Criteria

- [ ] Rules page accessible from sidebar, shows current rules
- [ ] Rule-providers page shows providers with status
- [ ] Can import a rule-provider from URL → appears in config
- [ ] Can refresh a rule-provider → updates rule count
- [ ] Can delete a rule-provider → removed from config
- [ ] Quick-import adds a Loyalsoldier rule-provider
- [ ] Can add/delete inline rules from RulesPage
- [ ] All operations: go build + go vet pass
- [ ] No QueueUpdateDraw calls from UI goroutine (apply same fix pattern as proxygroups)
