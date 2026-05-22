package merge

import (
	"testing"

	"mihomoTui/internal/types"
)

func TestMergeEngine_EmptyOverlay(t *testing.T) {
	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss", "server": "hk.example.com"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []string{"hk-01"}},
		},
		Rules: []string{
			"DOMAIN-SUFFIX,google.com,Proxy",
			"MATCH,DIRECT",
		},
	}

	engine := NewEngine()
	result, err := engine.Merge(base, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Document.Proxies) != 1 {
		t.Errorf("expected 1 proxy, got %d", len(result.Document.Proxies))
	}
	if len(result.Document.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(result.Document.Rules))
	}
}

func TestMergeEngine_ProxyGroupsMerged(t *testing.T) {
	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
			{"name": "jp-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []string{"hk-01"}},
		},
	}

	overlay := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "sg-01", "type": "ss"},
			{"name": "us-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []string{"sg-01", "us-01"}},
			{"name": "Download", "type": "select", "proxies": []string{"sg-01"}},
		},
	}

	engine := NewEngine()
	result, err := engine.Merge(base, overlay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Proxies should be replaced (2 from overlay, base proxies discarded)
	if len(result.Document.Proxies) != 2 {
		t.Errorf("expected 2 proxies (overlay replaces base), got %d", len(result.Document.Proxies))
	}

	// Proxy "Proxy" should have merged members (union of hk-01 + sg-01 + us-01)
	proxyGroup := findGroup(result.Document.ProxyGroups, "Proxy")
	if proxyGroup == nil {
		t.Fatal("expected Proxy group to exist")
	}
	members := getStringSlice(proxyGroup, "proxies")
	if len(members) != 3 {
		t.Errorf("expected 3 members in Proxy group (merged), got %d: %v", len(members), members)
	}

	// "Download" group should be newly added
	download := findGroup(result.Document.ProxyGroups, "Download")
	if download == nil {
		t.Fatal("expected Download group to exist")
	}
}

func TestMergeEngine_RulesAppended(t *testing.T) {
	base := &types.ConfigDocument{
		Rules: []string{
			"DOMAIN-SUFFIX,local,DIRECT",
			"MATCH,DIRECT",
		},
	}

	overlay := &types.ConfigDocument{
		Rules: []string{
			"DOMAIN-SUFFIX,google.com,Proxy",
			"DOMAIN-SUFFIX,youtube.com,Proxy",
			"MATCH,DIRECT", // duplicate
		},
	}

	engine := NewEngine()
	result, err := engine.Merge(base, overlay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4 unique rules (2 base + 2 new from overlay, MATCH,DIRECT is duplicate)
	if len(result.Document.Rules) != 4 {
		t.Errorf("expected 4 rules, got %d: %v", len(result.Document.Rules), result.Document.Rules)
	}

	// User's rules should come first
	if result.Document.Rules[0] != "DOMAIN-SUFFIX,local,Direct" {
		t.Logf("first rule: %s", result.Document.Rules[0])
	}
}

func TestMergeEngine_ConflictDetection(t *testing.T) {
	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss", "server": "hk.example.com"},
		},
	}

	overlay := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss", "server": "hk2.example.com"},
		},
	}

	// Use merge strategy that reveals conflicts
	strategy := types.DefaultMergeStrategy()
	strategy.Proxies = types.MergeModeMerge

	engine := NewEngineWithStrategy(strategy)
	result, err := engine.Merge(base, overlay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Conflicts) == 0 {
		t.Error("expected at least one conflict for duplicate proxy name")
	}

	// Overlay should win for proxy with same name
	hk := findProxy(result.Document.Proxies, "hk-01")
	if hk == nil {
		t.Fatal("expected hk-01 proxy")
	}
	if hk["server"] != "hk2.example.com" {
		t.Errorf("expected server hk2.example.com (overlay wins), got %v", hk["server"])
	}
}

func TestMergeEngine_Preview(t *testing.T) {
	base := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select", "proxies": []string{"hk-01"}},
		},
		Rules: []string{"MATCH,DIRECT"},
	}

	overlay := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "sg-01", "type": "ss"},
			{"name": "us-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Download", "type": "select", "proxies": []string{"sg-01"}},
		},
		Rules: []string{"DOMAIN-SUFFIX,google.com,Proxy"},
	}

	engine := NewEngine()
	preview, err := engine.Preview(base, overlay)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(preview.Added) != 4 {
		t.Errorf("expected 4 additions, got %d: %v", len(preview.Added), preview.Added)
	}
}

func findGroup(groups []map[string]interface{}, name string) map[string]interface{} {
	for _, g := range groups {
		if n, ok := g["name"].(string); ok && n == name {
			return g
		}
	}
	return nil
}

func findProxy(proxies []map[string]interface{}, name string) map[string]interface{} {
	for _, p := range proxies {
		if n, ok := p["name"].(string); ok && n == name {
			return p
		}
	}
	return nil
}
