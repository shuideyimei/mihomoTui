package proxygroup

import (
	"testing"

	"mihomoTui/internal/types"
)

func TestExtractGroups(t *testing.T) {
	mgr := NewManager()

	doc := &types.ConfigDocument{
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"hk-01", "DIRECT"}},
			{"name": "Auto", "type": "url-test", "proxies": []interface{}{"jp-01"}, "url": "http://www.gstatic.com/generate_204", "interval": 300},
		},
	}

	groups := mgr.ExtractGroups(doc)
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	if groups[0].Name != "Proxy" {
		t.Errorf("expected first group 'Proxy', got %q", groups[0].Name)
	}
	if groups[1].Type != "url-test" {
		t.Errorf("expected second group type 'url-test', got %q", groups[1].Type)
	}

	proxies := getStringList(doc.ProxyGroups[0], "proxies")
	if len(proxies) != 2 {
		t.Errorf("expected 2 proxies in Proxy group, got %d", len(proxies))
	}
}

func TestExtractProxies(t *testing.T) {
	mgr := NewManager()

	doc := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
			{"name": "jp-01", "type": "trojan"},
			{"name": "sg-01", "type": "ss"},
		},
	}

	names := mgr.ExtractProxies(doc)
	if len(names) != 3 {
		t.Errorf("expected 3 proxies, got %d", len(names))
	}

	// Should be sorted
	if names[0] != "hk-01" {
		t.Errorf("expected 'hk-01' first, got %q", names[0])
	}
}

func TestAnalyzeDependencies_AllPresent(t *testing.T) {
	mgr := NewManager()

	groups := []types.ProxyGroupDef{
		{Name: "Proxy", Type: "select", Proxies: []string{"hk-01", "jp-01", "DIRECT"}},
		{Name: "Auto", Type: "url-test", Proxies: []string{"sg-01"}, Use: []string{"my-provider"}},
	}

	availProxies := []string{"hk-01", "jp-01", "sg-01", "DIRECT", "REJECT"}
	availProviders := []string{"my-provider"}

	report := mgr.AnalyzeDependencies(groups, availProxies, availProviders)

	if len(report.Missing) != 0 {
		t.Errorf("expected 0 missing, got %v", report.Missing)
	}
	if !report.CanAutoFill {
		t.Error("expected CanAutoFill to be true")
	}
	if len(report.GroupRefs["Proxy"]) != 3 {
		t.Errorf("expected 3 refs for Proxy, got %v", report.GroupRefs["Proxy"])
	}
}

func TestAnalyzeDependencies_Missing(t *testing.T) {
	mgr := NewManager()

	groups := []types.ProxyGroupDef{
		{Name: "Download", Type: "select", Proxies: []string{"us-01", "eu-01"}},
	}

	availProxies := []string{"hk-01"}

	report := mgr.AnalyzeDependencies(groups, availProxies, nil)

	if len(report.Missing) == 0 {
		t.Error("expected missing proxies to be detected")
	}
	if report.CanAutoFill {
		t.Error("expected CanAutoFill to be false when dependencies are missing")
	}
}

func TestAnalyzeDependencies_ProviderReferences(t *testing.T) {
	mgr := NewManager()

	groups := []types.ProxyGroupDef{
		{Name: "Auto", Type: "url-test", Use: []string{"sub-provider", "my-provider"}},
	}

	availProviders := []string{"sub-provider"}
	availProxies := []string{}

	report := mgr.AnalyzeDependencies(groups, availProxies, availProviders)

	if len(report.Missing) != 1 {
		t.Errorf("expected 1 missing provider, got %v", report.Missing)
	}
}

func TestResolveConflict_Merge(t *testing.T) {
	mgr := NewManager()

	local := types.ProxyGroupDef{Name: "Proxy", Type: "select", Proxies: []string{"hk-01", "jp-01"}}
	imported := types.ProxyGroupDef{Name: "Proxy", Type: "select", Proxies: []string{"sg-01", "us-01"}}

	merged, resolution := mgr.ResolveConflict(local, imported)

	if resolution != types.ResolutionMerge {
		t.Errorf("expected ResolutionMerge, got %s", resolution)
	}
	// Union of proxies
	if len(merged.Proxies) != 4 {
		t.Errorf("expected 4 proxies (union), got %d: %v", len(merged.Proxies), merged.Proxies)
	}
}

func TestResolveConflict_DifferentTypes(t *testing.T) {
	mgr := NewManager()

	local := types.ProxyGroupDef{Name: "Proxy", Type: "select", Proxies: []string{"hk-01"}}
	imported := types.ProxyGroupDef{Name: "Proxy", Type: "url-test", Proxies: []string{"sg-01"}, URL: "http://test.com"}

	merged, resolution := mgr.ResolveConflict(local, imported)

	if resolution != types.ResolutionKeepLocal {
		t.Errorf("expected ResolutionKeepLocal for different types, got %s", resolution)
	}
	if merged.Type != "select" {
		t.Errorf("expected local type preserved, got %s", merged.Type)
	}
}

func TestImportGroups(t *testing.T) {
	mgr := NewManager()

	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"hk-01"}},
		},
	}

	source := &types.ConfigDocument{
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"sg-01"}},
			{"name": "Download", "type": "select", "proxies": []interface{}{"sg-01"}},
		},
	}

	result, err := mgr.ImportGroups(base, source, types.ImportModeMerge)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.GroupsAdded) != 1 || result.GroupsAdded[0] != "Download" {
		t.Errorf("expected Download added, got %v", result.GroupsAdded)
	}
	if len(result.GroupsUpdated) != 1 || result.GroupsUpdated[0] != "Proxy" {
		t.Errorf("expected Proxy updated, got %v", result.GroupsUpdated)
	}

	// Verify Proxy group members are merged
	proxyGroup := findGroupInDoc(base, "Proxy")
	if proxyGroup == nil {
		t.Fatal("Proxy group not found")
	}
	members := getStringList(proxyGroup, "proxies")
	if len(members) != 2 {
		t.Errorf("expected 2 members after merge, got %d: %v", len(members), members)
	}
}

func TestImportGroups_SkipMode(t *testing.T) {
	mgr := NewManager()

	base := &types.ConfigDocument{
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"hk-01"}},
		},
	}

	source := &types.ConfigDocument{
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"sg-01"}},
			{"name": "Download", "type": "select", "proxies": []interface{}{"sg-01"}},
		},
	}

	result, _ := mgr.ImportGroups(base, source, types.ImportModeSkip)

	if len(result.GroupsSkipped) != 1 || result.GroupsSkipped[0] != "Proxy" {
		t.Errorf("expected Proxy skipped, got %v", result.GroupsSkipped)
	}
	if len(result.GroupsAdded) != 1 || result.GroupsAdded[0] != "Download" {
		t.Errorf("expected Download added, got %v", result.GroupsAdded)
	}
}

func TestFindOrphanedProxies(t *testing.T) {
	mgr := NewManager()

	doc := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
			{"name": "orphan-01", "type": "ss"},
			{"name": "jp-01", "type": "trojan"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []interface{}{"hk-01", "jp-01"}},
		},
	}

	orphaned := mgr.FindOrphanedProxies(doc)
	if len(orphaned) != 1 || orphaned[0] != "orphan-01" {
		t.Errorf("expected orphan-01, got %v", orphaned)
	}
}

func TestMergeProxies(t *testing.T) {
	mgr := NewManager()

	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
		},
	}

	source := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "jp-01", "type": "ss"},
			{"name": "sg-01", "type": "ss"},
			{"name": "hk-01", "type": "ss"}, // duplicate
		},
	}

	added := mgr.MergeProxies(base, source)
	if added != 2 {
		t.Errorf("expected 2 new proxies, got %d", added)
	}
	if len(base.Proxies) != 3 {
		t.Errorf("expected 3 total proxies, got %d", len(base.Proxies))
	}
}

func findGroupInDoc(doc *types.ConfigDocument, name string) map[string]interface{} {
	for _, g := range doc.ProxyGroups {
		if n, ok := g["name"].(string); ok && n == name {
			return g
		}
	}
	return nil
}
