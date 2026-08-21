package repository

import (
	"os"
	"path/filepath"
	"testing"

	"mihomoTui/internal/models"
)

var (
	_ RuleRepository       = (*LocalYAMLDriver)(nil)
	_ ProxyGroupRepository = (*LocalYAMLDriver)(nil)
	_ ProviderRepository   = (*LocalYAMLDriver)(nil)
	_ ProfileRepository    = (*LocalYAMLDriver)(nil)
)

func TestLocalYAMLDriver_ModelIntegrity(t *testing.T) {
	driver := NewLocalYAMLDriver()
	if driver == nil {
		t.Fatal("expected non-nil driver")
	}

	var _ RuleRepository = driver
	var _ ProxyGroupRepository = driver
	var _ ProviderRepository = driver
	var _ ProfileRepository = driver
}

func TestLocalYAMLDriver_ProfileOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repo_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	driver := NewLocalYAMLDriver()

	testFile := filepath.Join(tmpDir, "test_config.yaml")
	content := "port: 7890\nmode: rule\n"

	// Test WriteProfile
	if err := driver.WriteProfile(testFile, content); err != nil {
		t.Fatalf("WriteProfile failed: %v", err)
	}

	// Test ReadProfile
	readBack, err := driver.ReadProfile(testFile)
	if err != nil {
		t.Fatalf("ReadProfile failed: %v", err)
	}
	if readBack != content {
		t.Errorf("ReadProfile mismatch. got %q, want %q", readBack, content)
	}

	// Test WriteProfile with empty content (should fail)
	if err := driver.WriteProfile(testFile, ""); err == nil {
		t.Errorf("WriteProfile with empty content should return error")
	}

	// Test ValidateProfile
	if err := driver.ValidateProfile(testFile); err != nil {
		t.Errorf("ValidateProfile failed for valid config: %v", err)
	}

	// Test DeleteProfile
	if err := driver.DeleteProfile(testFile); err != nil {
		t.Errorf("DeleteProfile failed: %v", err)
	}
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("testFile still exists after DeleteProfile")
	}
}

func TestLocalYAMLDriver_RulesAndBatchAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repo_rules_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "config.yaml")
	initialContent := `port: 7890
mode: rule
rules:
  - DOMAIN-SUFFIX,google.com,PROXY
  - MATCH,DIRECT
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	t.Setenv("MIHOMO_CONFIG_PATH", testFile)
	driver := NewLocalYAMLDriver()

	rules, err := driver.GetRules()
	if err != nil {
		t.Fatalf("GetRules failed: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Type != "DOMAIN-SUFFIX" || rules[0].Payload != "google.com" || rules[0].Proxy != "PROXY" {
		t.Errorf("rule 0 mismatch: %+v", rules[0])
	}
	if rules[1].Type != "MATCH" || rules[1].Proxy != "DIRECT" {
		t.Errorf("rule 1 mismatch: %+v", rules[1])
	}

	// Test BatchAddRules with unified models.RuleDisplay
	newRules := []models.RuleDisplay{
		{Type: "DOMAIN-KEYWORD", Payload: "github", Proxy: "PROXY"},
		{Type: "GEOIP", Payload: "CN", Proxy: "DIRECT"},
	}
	if _, err := driver.BatchAddRules(newRules); err != nil {
		t.Fatalf("BatchAddRules failed: %v", err)
	}

	updatedRules, err := driver.GetRules()
	if err != nil {
		t.Fatalf("GetRules after batch add failed: %v", err)
	}
	if len(updatedRules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(updatedRules))
	}
	if updatedRules[2].Payload != "github" || updatedRules[3].Payload != "CN" {
		t.Errorf("updated rules mismatch: %+v", updatedRules)
	}
}

func TestLocalYAMLDriver_ProxyGroupsAndProviders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repo_groups_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "config.yaml")
	initialContent := `port: 7890
mode: rule
proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - node1
      - node2
    use:
      - provider1
proxy-providers:
  provider1:
    type: http
    url: https://example.com/sub
    interval: 3600
rule-providers:
  rule1:
    type: http
    behavior: domain
    url: https://example.com/rules.yaml
    path: ./ruleset/rule1.yaml
    interval: 86400
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	t.Setenv("MIHOMO_CONFIG_PATH", testFile)
	driver := NewLocalYAMLDriver()

	// 1. Test GetGroups
	groups, err := driver.GetGroups()
	if err != nil {
		t.Fatalf("GetGroups failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	g := groups[0]
	if g.Name != "PROXY" || g.Type != "select" || len(g.Proxies) != 2 || len(g.Use) != 1 {
		t.Errorf("group mismatch: %+v", g)
	}

	// 2. Test GetProxyProviders
	providers, err := driver.GetProxyProviders()
	if err != nil {
		t.Fatalf("GetProxyProviders failed: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 proxy provider, got %d", len(providers))
	}
	p := providers[0]
	if p.Name != "provider1" || p.URL != "https://example.com/sub" || p.Interval != 3600 {
		t.Errorf("provider mismatch: %+v", p)
	}

	// 3. Test GetRuleProviders
	ruleProviders, err := driver.GetRuleProviders()
	if err != nil {
		t.Fatalf("GetRuleProviders failed: %v", err)
	}
	if len(ruleProviders) != 1 {
		t.Fatalf("expected 1 rule provider, got %d", len(ruleProviders))
	}
	rp := ruleProviders[0]
	if rp.Name != "rule1" || rp.Type != "http" || rp.Behavior != "domain" || rp.URL != "https://example.com/rules.yaml" {
		t.Errorf("rule provider mismatch: %+v", rp)
	}

	// 4. Test BatchAddRuleProviders
	batch := []models.RuleProviderEntry{
		{Name: "rule2", URL: "https://example.com/rules2.yaml", Behavior: "domain", Interval: 3600},
		{Name: "rule3", URL: "https://example.com/rules3.yaml", Behavior: "ipcidr", Interval: 7200},
	}
	if _, err := driver.BatchAddRuleProviders(batch); err != nil {
		t.Fatalf("BatchAddRuleProviders failed: %v", err)
	}

	updatedRuleProviders, err := driver.GetRuleProviders()
	if err != nil {
		t.Fatalf("GetRuleProviders after batch add failed: %v", err)
	}
	if len(updatedRuleProviders) != 3 {
		t.Fatalf("expected 3 rule providers, got %d", len(updatedRuleProviders))
	}
}
