package merge

import (
	"fmt"

	"mihomoTui/internal/types"
)

// Engine performs smart YAML merging of ConfigDocuments.
type Engine struct {
	strategy types.MergeStrategy
}

// NewEngine creates a merge engine with the default strategy.
func NewEngine() *Engine {
	return &Engine{
		strategy: types.DefaultMergeStrategy(),
	}
}

// NewEngineWithStrategy creates a merge engine with a custom strategy.
func NewEngineWithStrategy(s types.MergeStrategy) *Engine {
	return &Engine{
		strategy: s,
	}
}

// Merge merges two config documents.
// base is the user's current config (authoritative for preserved fields).
// overlay is the incoming subscription config.
func (e *Engine) Merge(base, overlay *types.ConfigDocument) (*types.MergeResult, error) {
	if base == nil {
		return nil, fmt.Errorf("base config cannot be nil")
	}
	if overlay == nil {
		result := &types.MergeResult{
			Document: base.Clone(),
			Strategy: e.strategy,
		}
		return result, nil
	}

	result := base.Clone()
	var conflicts []types.ConflictEntry

	// 1. Proxies — replace from overlay
	if len(overlay.Proxies) > 0 {
		result.Proxies = mergeProxies(base.Proxies, overlay.Proxies, e.strategy.Proxies, &conflicts)
	}

	// 2. ProxyGroups — merge (preserve user groups, add new, union members)
	if len(overlay.ProxyGroups) > 0 {
		result.ProxyGroups = mergeProxyGroups(base.ProxyGroups, overlay.ProxyGroups, e.strategy.ProxyGroups, &conflicts)
	}

	// 3. Rules — append overlay to base
	if len(overlay.Rules) > 0 {
		result.Rules = mergeRules(base.Rules, overlay.Rules, e.strategy.Rules)
	}

	// 4. RuleProviders — merge by key
	if len(overlay.RuleProviders) > 0 {
		result.RuleProviders = mergeNamedMaps(base.RuleProviders, overlay.RuleProviders, e.strategy.RuleProviders, &conflicts)
	}

	// 5. ProxyProviders — replace from overlay
	if len(overlay.ProxyProviders) > 0 {
		result.ProxyProviders = mergeProxyProviders(base.ProxyProviders, overlay.ProxyProviders, e.strategy.ProxyProviders, &conflicts)
	}

	// 6. DNS — keep base (user config is authoritative)
	result.DNS = mergeDNS(base.DNS, overlay.DNS, e.strategy.DNS, &conflicts)

	// 7. Tun — keep base
	result.Tun = mergeTun(base.Tun, overlay.Tun, e.strategy.Tun, &conflicts)

	// 8. Scalar fields — replace from overlay only if not set in base
	mergeScalars(base, overlay, result)

	return &types.MergeResult{
		Document:  result,
		Conflicts: conflicts,
		Strategy:  e.strategy,
	}, nil
}

// Preview performs a dry-run merge, reporting what would change without modifying.
func (e *Engine) Preview(base, overlay *types.ConfigDocument) (*types.MergePreview, error) {
	if base == nil || overlay == nil {
		return &types.MergePreview{Strategy: e.strategy}, nil
	}

	preview := &types.MergePreview{
		Strategy: e.strategy,
	}

	// Proxy differences
	baseNames := make(map[string]bool)
	for _, p := range base.Proxies {
		if name, ok := p["name"].(string); ok {
			baseNames[name] = true
		}
	}
	for _, p := range overlay.Proxies {
		if name, ok := p["name"].(string); ok {
			if baseNames[name] {
				preview.Modified = append(preview.Modified, fmt.Sprintf("proxy:%s", name))
			} else {
				preview.Added = append(preview.Added, fmt.Sprintf("proxy:%s", name))
			}
		}
	}

	// Group differences
	baseGroups := make(map[string]bool)
	for _, g := range base.ProxyGroups {
		if name, ok := g["name"].(string); ok {
			baseGroups[name] = true
		}
	}
	for _, g := range overlay.ProxyGroups {
		if name, ok := g["name"].(string); ok {
			if baseGroups[name] {
				preview.Modified = append(preview.Modified, fmt.Sprintf("proxy-group:%s", name))
			} else {
				preview.Added = append(preview.Added, fmt.Sprintf("proxy-group:%s", name))
			}
		}
	}

	// Rule differences
	baseRuleSet := make(map[string]bool)
	for _, r := range base.Rules {
		baseRuleSet[r] = true
	}
	for _, r := range overlay.Rules {
		if !baseRuleSet[r] {
			preview.Added = append(preview.Added, fmt.Sprintf("rule:%s", truncate(r, 60)))
		}
	}

	return preview, nil
}

// mergeProxies merges proxy lists by name.
func mergeProxies(base, overlay []map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) []map[string]interface{} {
	switch mode {
	case types.MergeModeReplace:
		return cloneSlice(overlay)
	case types.MergeModeSkip:
		return cloneSlice(base)
	case types.MergeModeMerge:
		// Merge by name: overlay wins for matching names
		result := cloneSlice(base)
		baseIdx := make(map[string]int)
		for i, p := range base {
			if name, ok := p["name"].(string); ok {
				baseIdx[name] = i
			}
		}
		for _, op := range overlay {
			if name, ok := op["name"].(string); ok {
				if idx, exists := baseIdx[name]; exists {
					// Conflict — overlay wins by default
					result[idx] = cloneMap(op)
					*conflicts = append(*conflicts, types.ConflictEntry{
						Field:      fmt.Sprintf("proxies.%s", name),
						Local:      base[idx],
						Imported:   op,
						Resolution: string(types.ResolutionUseImported),
					})
				} else {
					result = append(result, cloneMap(op))
				}
			}
		}
		return result
	default:
		return cloneSlice(overlay)
	}
}

// mergeProxyGroups merges proxy-group lists.
// Preserves user's groups. For matching names, merges proxies arrays (union).
func mergeProxyGroups(base, overlay []map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) []map[string]interface{} {
	switch mode {
	case types.MergeModeReplace:
		return cloneSlice(overlay)
	case types.MergeModeSkip:
		return cloneSlice(base)
	case types.MergeModeMerge:
		result := cloneSlice(base)
		baseIdx := make(map[string]int)
		for i, g := range base {
			if name, ok := g["name"].(string); ok {
				baseIdx[name] = i
			}
		}
		for _, og := range overlay {
			name, _ := og["name"].(string)
			if name == "" {
				continue
			}
			if idx, exists := baseIdx[name]; exists {
				// Merge members: union of proxies
				existing := result[idx]
				baseProxies := getStringSlice(existing, "proxies")
				importedProxies := getStringSlice(og, "proxies")

				proxySet := make(map[string]bool)
				for _, p := range baseProxies {
					proxySet[p] = true
				}
				for _, p := range importedProxies {
					if !proxySet[p] {
						baseProxies = append(baseProxies, p)
					}
				}
				existing["proxies"] = baseProxies

				// Union of "use" (providers)
				baseUse := getStringSlice(existing, "use")
				importedUse := getStringSlice(og, "use")
				useSet := make(map[string]bool)
				for _, u := range baseUse {
					useSet[u] = true
				}
				for _, u := range importedUse {
					if !useSet[u] {
						baseUse = append(baseUse, u)
					}
				}
				existing["use"] = baseUse

				*conflicts = append(*conflicts, types.ConflictEntry{
					Field:      fmt.Sprintf("proxy-groups.%s", name),
					Local:      "preserved with merged members",
					Imported:   "members merged",
					Resolution: string(types.ResolutionMerge),
				})
			} else {
				// New group — add it
				result = append(result, cloneMap(og))
			}
		}
		return result
	default:
		return cloneSlice(overlay)
	}
}

// mergeRules handles rule merging.
// "append" = user rules first, then subscription rules (deduped).
func mergeRules(base, overlay []string, mode types.MergeMode) []string {
	switch mode {
	case types.MergeModeReplace:
		result := make([]string, len(overlay))
		copy(result, overlay)
		return result
	case types.MergeModeSkip:
		result := make([]string, len(base))
		copy(result, base)
		return result
	case types.MergeModeAppend:
		ruleSet := make(map[string]bool)
		for _, r := range base {
			ruleSet[r] = true
		}
		result := make([]string, len(base))
		copy(result, base)
		for _, r := range overlay {
			if !ruleSet[r] {
				result = append(result, r)
			}
		}
		return result
	case types.MergeModePrepend:
		result := make([]string, len(overlay))
		copy(result, overlay)
		ruleSet := make(map[string]bool)
		for _, r := range overlay {
			ruleSet[r] = true
		}
		for _, r := range base {
			if !ruleSet[r] {
				result = append(result, r)
			}
		}
		return result
	default:
		result := make([]string, len(overlay))
		copy(result, overlay)
		return result
	}
}

// mergeNamedMaps merges two slices of named maps (e.g., rule-providers).
func mergeNamedMaps(base, overlay []map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) []map[string]interface{} {
	switch mode {
	case types.MergeModeReplace:
		return cloneSlice(overlay)
	case types.MergeModeSkip:
		return cloneSlice(base)
	case types.MergeModeMerge:
		result := cloneSlice(base)
		baseIdx := make(map[string]int)
		for i, item := range base {
			if name, ok := item["name"].(string); ok {
				baseIdx[name] = i
			}
		}
		for _, oi := range overlay {
			name, _ := oi["name"].(string)
			if name == "" {
				continue
			}
			if _, exists := baseIdx[name]; !exists {
				result = append(result, cloneMap(oi))
			}
		}
		return result
	default:
		return cloneSlice(overlay)
	}
}

// mergeProxyProviders handles proxy-provider merging.
func mergeProxyProviders(base, overlay []map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) []map[string]interface{} {
	switch mode {
	case types.MergeModeReplace:
		return cloneSlice(overlay)
	case types.MergeModeSkip:
		return cloneSlice(base)
	case types.MergeModeMerge:
		// Merge by name, overlay wins for matching names
		result := cloneSlice(base)
		baseIdx := make(map[string]int)
		for i, p := range base {
			if name, ok := p["name"].(string); ok {
				baseIdx[name] = i
			}
		}
		for _, op := range overlay {
			name, _ := op["name"].(string)
			if name == "" {
				continue
			}
			if idx, exists := baseIdx[name]; exists {
				result[idx] = cloneMap(op)
				*conflicts = append(*conflicts, types.ConflictEntry{
					Field:      fmt.Sprintf("proxy-providers.%s", name),
					Resolution: string(types.ResolutionUseImported),
				})
			} else {
				result = append(result, cloneMap(op))
			}
		}
		return result
	default:
		return cloneSlice(overlay)
	}
}

func mergeDNS(base, overlay map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) map[string]interface{} {
	switch mode {
	case types.MergeModeMerge:
		if overlay == nil {
			return base
		}
		if base == nil {
			return overlay
		}
		merged := cloneMap(base)
		for k, v := range overlay {
			merged[k] = v
		}
		return merged
	default:
		if base != nil {
			return cloneMap(base)
		}
		return cloneMap(overlay)
	}
}

func mergeTun(base, overlay map[string]interface{}, mode types.MergeMode, conflicts *[]types.ConflictEntry) map[string]interface{} {
	return mergeDNS(base, overlay, mode, conflicts) // same logic
}

// mergeScalars fills in scalar fields from overlay only if not set in base.
func mergeScalars(base, overlay, result *types.ConfigDocument) {
	if result.Port == 0 && overlay.Port != 0 {
		result.Port = overlay.Port
	}
	if result.SocksPort == 0 && overlay.SocksPort != 0 {
		result.SocksPort = overlay.SocksPort
	}
	if result.MixedPort == 0 && overlay.MixedPort != 0 {
		result.MixedPort = overlay.MixedPort
	}
	if result.ExternalUI == "" && overlay.ExternalUI != "" {
		result.ExternalUI = overlay.ExternalUI
	}
	if result.Mode == "" && overlay.Mode != "" {
		result.Mode = overlay.Mode
	}
	if result.LogLevel == "" && overlay.LogLevel != "" {
		result.LogLevel = overlay.LogLevel
	}
	if result.Secret == "" && overlay.Secret != "" {
		result.Secret = overlay.Secret
	}
}

// --- helpers ---

func cloneSlice(src []map[string]interface{}) []map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make([]map[string]interface{}, len(src))
	for i, m := range src {
		dst[i] = cloneMap(m)
	}
	return dst
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
