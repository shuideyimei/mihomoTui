package proxygroup

import (
	"fmt"
	"sort"
	"strings"

	"mihomoTui/internal/types"
)

// Manager handles proxy group import, extraction, and dependency analysis.
type Manager struct{}

// NewManager creates a new proxy group manager.
func NewManager() *Manager {
	return &Manager{}
}

// ExtractGroups extracts proxy-group definitions from a ConfigDocument.
func (m *Manager) ExtractGroups(doc *types.ConfigDocument) []types.ProxyGroupDef {
	if doc == nil || len(doc.ProxyGroups) == 0 {
		return nil
	}

	groups := make([]types.ProxyGroupDef, 0, len(doc.ProxyGroups))
	for _, g := range doc.ProxyGroups {
		group := m.toGroupDef(g)
		if group.Name != "" {
			groups = append(groups, group)
		}
	}
	return groups
}

// ExtractProxies extracts proxy node names from a ConfigDocument.
func (m *Manager) ExtractProxies(doc *types.ConfigDocument) []string {
	if doc == nil {
		return nil
	}
	names := make([]string, 0, len(doc.Proxies))
	for _, p := range doc.Proxies {
		if name, ok := p["name"].(string); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ExtractProxyProviders extracts proxy-provider names from a ConfigDocument.
func (m *Manager) ExtractProxyProviders(doc *types.ConfigDocument) []string {
	if doc == nil {
		return nil
	}
	names := make([]string, 0, len(doc.ProxyProviders))
	for _, p := range doc.ProxyProviders {
		if name, ok := p["name"].(string); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// AnalyzeDependencies analyzes what proxies each proxy-group references
// and detects which ones are missing from the available proxy set.
func (m *Manager) AnalyzeDependencies(groups []types.ProxyGroupDef, availableProxies []string, availableProviders []string) *types.DependencyReport {
	report := &types.DependencyReport{
		Groups:    groups,
		GroupRefs: make(map[string][]string),
	}

	availSet := make(map[string]bool)
	for _, p := range availableProxies {
		availSet[p] = true
		report.Available = append(report.Available, p)
	}
	provSet := make(map[string]bool)
	for _, p := range availableProviders {
		provSet[p] = true
		report.Providers = append(report.Providers, p)
	}

	for _, group := range groups {
		refs := make([]string, 0)
		for _, proxy := range group.Proxies {
			refs = append(refs, proxy)
			if !availSet[proxy] && !provSet[proxy] {
				report.Missing = append(report.Missing, fmt.Sprintf("%s (referenced by %s)", proxy, group.Name))
			}
		}
		for _, use := range group.Use {
			refs = append(refs, fmt.Sprintf("provider:%s", use))
			if !provSet[use] {
				report.Missing = append(report.Missing, fmt.Sprintf("provider:%s (referenced by %s)", use, group.Name))
			}
		}
		report.GroupRefs[group.Name] = refs
	}

	report.CanAutoFill = len(report.Missing) == 0

	return report
}

// ResolveConflict determines how to handle a conflict between local and imported groups.
func (m *Manager) ResolveConflict(local, imported types.ProxyGroupDef) (*types.ProxyGroupDef, types.ConflictResolution) {
	if local.Type != imported.Type {
		// Different types — keep local, note conflict
		return &local, types.ResolutionKeepLocal
	}

	// Same type — merge members
	merged := local
	proxySet := make(map[string]bool)
	for _, p := range local.Proxies {
		proxySet[p] = true
	}
	for _, p := range imported.Proxies {
		if !proxySet[p] {
			merged.Proxies = append(merged.Proxies, p)
		}
	}

	useSet := make(map[string]bool)
	for _, u := range local.Use {
		useSet[u] = true
	}
	for _, u := range imported.Use {
		if !useSet[u] {
			merged.Use = append(merged.Use, u)
		}
	}

	return &merged, types.ResolutionMerge
}

// ImportGroups imports proxy groups from a source document into a base document.
func (m *Manager) ImportGroups(base, source *types.ConfigDocument, mode types.ProxyGroupImportMode) (*types.ImportResult, error) {
	result := &types.ImportResult{}

	if base == nil {
		return nil, fmt.Errorf("base config is required")
	}
	if source == nil {
		return result, nil
	}

	imported := m.ExtractGroups(source)
	existing := m.ExtractGroups(base)
	availProxies := m.ExtractProxies(base)
	availProviders := m.ExtractProxyProviders(base)

	// Build lookup of existing groups
	existingMap := make(map[string]types.ProxyGroupDef)
	for _, g := range existing {
		existingMap[g.Name] = g
	}

	for _, ig := range imported {
		if existingGroup, exists := existingMap[ig.Name]; exists {
			switch mode {
			case types.ImportModeSkip:
				result.GroupsSkipped = append(result.GroupsSkipped, ig.Name)
				continue
			case types.ImportModeReplace:
				// Replace the existing group
				result.GroupsUpdated = append(result.GroupsUpdated, ig.Name)
				m.updateGroupInDoc(base, ig)
			case types.ImportModeMerge:
				merged, resolution := m.ResolveConflict(existingGroup, ig)
				result.Conflicts = append(result.Conflicts, types.ConflictEntry{
					Field:      fmt.Sprintf("proxy-groups.%s", ig.Name),
					Local:      existingGroup,
					Imported:   ig,
					Resolution: string(resolution),
				})
				m.updateGroupInDoc(base, *merged)
				result.GroupsUpdated = append(result.GroupsUpdated, ig.Name)
			case types.ImportModeRename:
				// Rename imported group
				newName := fmt.Sprintf("%s (imported)", ig.Name)
				ig.Name = newName
				m.addGroupToDoc(base, ig)
				result.GroupsAdded = append(result.GroupsAdded, newName)
			}
		} else {
			m.addGroupToDoc(base, ig)
			result.GroupsAdded = append(result.GroupsAdded, ig.Name)
		}
	}

	// Analyze dependencies
	report := m.AnalyzeDependencies(imported, availProxies, availProviders)
	result.Warnings = m.buildWarnings(report)

	return result, nil
}

// MergeProxies adds proxies from source into base (deduped by name).
func (m *Manager) MergeProxies(base, source *types.ConfigDocument) int {
	if source == nil || len(source.Proxies) == 0 {
		return 0
	}

	existing := make(map[string]bool)
	for _, p := range base.Proxies {
		if name, ok := p["name"].(string); ok {
			existing[name] = true
		}
	}

	added := 0
	for _, sp := range source.Proxies {
		name, _ := sp["name"].(string)
		if name != "" && !existing[name] {
			base.Proxies = append(base.Proxies, sp)
			existing[name] = true
			added++
		}
	}

	return added
}

// FindOrphanedProxies returns proxy names that exist but are not referenced by any group.
func (m *Manager) FindOrphanedProxies(doc *types.ConfigDocument) []string {
	if doc == nil {
		return nil
	}

	referenced := make(map[string]bool)
	for _, g := range doc.ProxyGroups {
		for _, p := range getStringList(g, "proxies") {
			referenced[p] = true
		}
		for _, u := range getStringList(g, "use") {
			referenced[fmt.Sprintf("provider:%s", u)] = true
		}
	}

	var orphaned []string
	for _, p := range doc.Proxies {
		if name, ok := p["name"].(string); ok && !referenced[name] {
			orphaned = append(orphaned, name)
		}
	}
	return orphaned
}

// --- helpers ---

func (m *Manager) toGroupDef(raw map[string]interface{}) types.ProxyGroupDef {
	def := types.ProxyGroupDef{}
	if name, ok := raw["name"].(string); ok {
		def.Name = name
	}
	if t, ok := raw["type"].(string); ok {
		def.Type = t
	}
	def.Proxies = getStringList(raw, "proxies")
	def.Use = getStringList(raw, "use")
	if url, ok := raw["url"].(string); ok {
		def.URL = url
	}
	if interval, ok := raw["interval"].(int); ok {
		def.Interval = interval
	}
	if filter, ok := raw["filter"].(string); ok {
		def.Filter = filter
	}
	return def
}

func (m *Manager) updateGroupInDoc(doc *types.ConfigDocument, group types.ProxyGroupDef) {
	for i, g := range doc.ProxyGroups {
		if name, ok := g["name"].(string); ok && name == group.Name {
			doc.ProxyGroups[i] = groupDefToMap(group)
			return
		}
	}
	// Not found — add
	m.addGroupToDoc(doc, group)
}

func (m *Manager) addGroupToDoc(doc *types.ConfigDocument, group types.ProxyGroupDef) {
	doc.ProxyGroups = append(doc.ProxyGroups, groupDefToMap(group))
}

func groupDefToMap(g types.ProxyGroupDef) map[string]interface{} {
	m := map[string]interface{}{
		"name": g.Name,
		"type": g.Type,
	}
	if len(g.Proxies) > 0 {
		m["proxies"] = g.Proxies
	}
	if len(g.Use) > 0 {
		m["use"] = g.Use
	}
	if g.URL != "" {
		m["url"] = g.URL
	}
	if g.Interval > 0 {
		m["interval"] = g.Interval
	}
	if g.Filter != "" {
		m["filter"] = g.Filter
	}
	return m
}

func (m *Manager) buildWarnings(report *types.DependencyReport) []string {
	var warnings []string

	// Group unused proxies warning
	if len(report.Missing) > 0 {
		missingStr := make([]string, len(report.Missing))
		copy(missingStr, report.Missing)
		if len(missingStr) > 5 {
			warnings = append(warnings, fmt.Sprintf(
				"%d proxy/providers are referenced but not defined (showing first 5): %s",
				len(missingStr),
				strings.Join(missingStr[:5], ", "),
			))
		} else {
			warnings = append(warnings, fmt.Sprintf(
				"Missing proxies/providers: %s",
				strings.Join(missingStr, ", "),
			))
		}
	}

	return warnings
}

func getStringList(m map[string]interface{}, key string) []string {
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
