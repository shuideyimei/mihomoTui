package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mihomoTui/internal/events"
	"mihomoTui/internal/models"
)

type mockRuleRepo struct {
	rules []models.RuleDisplay
}

func (m *mockRuleRepo) GetRules() ([]models.RuleDisplay, error) {
	return m.rules, nil
}
func (m *mockRuleRepo) AddRule(ruleType, payload, proxy string) (string, error) {
	m.rules = append(m.rules, models.RuleDisplay{Type: ruleType, Payload: payload, Proxy: proxy})
	return "/mock/config.yaml", nil
}
func (m *mockRuleRepo) DeleteRule(index int) (string, error) {
	if index < 0 || index >= len(m.rules) {
		return "", errors.New("out of range")
	}
	m.rules = append(m.rules[:index], m.rules[index+1:]...)
	return "/mock/config.yaml", nil
}
func (m *mockRuleRepo) DeleteRuleByContent(ruleType, payload, proxy string) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockRuleRepo) MoveRule(fromIdx, toIdx int) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockRuleRepo) UpdateRule(index int, ruleType, payload, proxy string) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockRuleRepo) BatchAddRules(rules []models.RuleDisplay) (string, error) {
	m.rules = append(m.rules, rules...)
	return "/mock/config.yaml", nil
}

func TestRuleService(t *testing.T) {
	mockRepo := &mockRuleRepo{}
	svc := NewRuleService(mockRepo)

	if err := svc.AddRule("DOMAIN-SUFFIX", "google.com", "Proxy"); err != nil {
		t.Fatalf("unexpected error adding rule: %v", err)
	}

	rules, err := svc.GetRules()
	if err != nil || len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d, err: %v", len(rules), err)
	}

	if err := svc.DeleteRule(0); err != nil {
		t.Fatalf("unexpected error deleting rule: %v", err)
	}

	rules, _ = svc.GetRules()
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rules))
	}
}

type mockProfileRepo struct {
	profiles []string
	current  string
}

func (m *mockProfileRepo) FindAllProfiles() []string {
	return m.profiles
}
func (m *mockProfileRepo) GetCurrentProfilePath() string {
	return m.current
}
func (m *mockProfileRepo) ReadProfile(path string) (string, error) {
	return "mode: rule", nil
}
func (m *mockProfileRepo) WriteProfile(path, content string) error {
	return nil
}
func (m *mockProfileRepo) DeleteProfile(path string) error {
	return nil
}
func (m *mockProfileRepo) ValidateProfile(path string) error {
	return nil
}
func (m *mockProfileRepo) EnsureSafePath(cfgPath, path string) ([]string, error) {
	return nil, nil
}

func TestProfileService(t *testing.T) {
	bus := events.NewBus()
	repo := &mockProfileRepo{
		profiles: []string{"/path/1.yaml", "/path/2.yaml"},
		current:  "/path/1.yaml",
	}
	svc := NewProfileService(repo, bus)

	profiles := svc.ListProfiles()
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if svc.GetActiveProfile() != "/path/1.yaml" {
		t.Fatalf("expected /path/1.yaml, got %s", svc.GetActiveProfile())
	}
}

type mockGroupRepo struct {
	groups []models.ProxyGroupEntry
}

func (m *mockGroupRepo) GetGroups() ([]models.ProxyGroupEntry, error) {
	return m.groups, nil
}
func (m *mockGroupRepo) AddGroup(name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	m.groups = append(m.groups, models.ProxyGroupEntry{Name: name, Type: groupType, Proxies: proxies, Use: use})
	return "/mock/config.yaml", nil
}
func (m *mockGroupRepo) UpdateGroup(oldName, name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockGroupRepo) DeleteGroup(name string) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockGroupRepo) GetGroupNames() ([]string, error) {
	var names []string
	for _, g := range m.groups {
		names = append(names, g.Name)
	}
	return names, nil
}
func (m *mockGroupRepo) GetGroupDef(groupName string) ([]string, []string, error) {
	return nil, nil, nil
}

func TestGroupService(t *testing.T) {
	repo := &mockGroupRepo{}
	svc := NewProxyGroupService(repo)

	if err := svc.AddGroup("ProxyGroup1", "select", "", "", nil, []string{"Node1", "Node2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names, err := svc.GetGroupNames()
	if err != nil || len(names) != 1 || names[0] != "ProxyGroup1" {
		t.Fatalf("unexpected group names: %v", names)
	}
}

type mockProviderRepo struct {
	providers     []models.ProviderInfo
	ruleProviders []models.RuleProviderEntry
}

func (m *mockProviderRepo) GetProxyProviders() ([]models.ProviderInfo, error) {
	return m.providers, nil
}
func (m *mockProviderRepo) AddProxyProvider(name, url string) (string, error) {
	m.providers = append(m.providers, models.ProviderInfo{Name: name, URL: url})
	return "/mock/config.yaml", nil
}
func (m *mockProviderRepo) AddLocalProxyProvider(name, localPath string) (string, error) {
	m.providers = append(m.providers, models.ProviderInfo{Name: name, URL: localPath})
	return "/mock/config.yaml", nil
}
func (m *mockProviderRepo) RemoveProxyProvider(name string) error {
	return nil
}
func (m *mockProviderRepo) GetRuleProviders() ([]models.RuleProviderEntry, error) {
	return m.ruleProviders, nil
}
func (m *mockProviderRepo) AddRuleProvider(name, url, behavior string, interval int) (string, error) {
	m.ruleProviders = append(m.ruleProviders, models.RuleProviderEntry{Name: name, URL: url, Behavior: behavior, Interval: interval})
	return "/mock/config.yaml", nil
}
func (m *mockProviderRepo) UpdateRuleProvider(name, url, behavior string, interval int) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockProviderRepo) DeleteRuleProvider(name string) (string, error) {
	return "/mock/config.yaml", nil
}
func (m *mockProviderRepo) BatchAddRuleProviders(providers []models.RuleProviderEntry) (string, error) {
	m.ruleProviders = append(m.ruleProviders, providers...)
	return "/mock/config.yaml", nil
}

func TestProviderService(t *testing.T) {
	repo := &mockProviderRepo{}
	svc := NewProviderService(repo, nil)

	if err := svc.AddProxyProvider("Sub1", "https://example.com/sub"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	provs, err := svc.GetProxyProviders()
	if err != nil || len(provs) != 1 || provs[0].Name != "Sub1" {
		t.Fatalf("unexpected providers: %v", provs)
	}

	if err := svc.AddRuleProvider("Rules1", "https://example.com/rules", "classical", 86400); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rps, err := svc.GetRuleProviders()
	if err != nil || len(rps) != 1 || rps[0].Name != "Rules1" {
		t.Fatalf("unexpected rule providers: %v", rps)
	}

	batch := []models.RuleProviderEntry{
		{Name: "Rules2", URL: "https://example.com/rules2", Behavior: "domain", Interval: 3600},
		{Name: "Rules3", URL: "https://example.com/rules3", Behavior: "ipcidr", Interval: 7200},
	}
	count, err := svc.BatchAddRuleProviders(batch)
	if err != nil || count != 2 {
		t.Fatalf("unexpected error batch adding rule providers: %v, count: %d", err, count)
	}

	rps, err = svc.GetRuleProviders()
	if err != nil || len(rps) != 3 {
		t.Fatalf("expected 3 rule providers, got %d", len(rps))
	}

	// Test ImportLocal and ImportURL with nil subMgr (safety check)
	if _, err := svc.ImportLocal("Test", "/path/to/sub.yaml"); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected error about subscription manager not initialized, got: %v", err)
	}
	if _, err := svc.ImportURL(context.Background(), "Test", "https://example.com/sub.yaml"); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected error about subscription manager not initialized, got: %v", err)
	}

	// Test ImportLocal parameter validation
	if _, err := svc.ImportLocal("", "/path/to/sub.yaml"); err == nil || !strings.Contains(err.Error(), "provider name cannot be empty") {
		t.Fatalf("expected provider name validation error, got: %v", err)
	}
	if _, err := svc.ImportLocal("Test", ""); err == nil || !strings.Contains(err.Error(), "file path cannot be empty") {
		t.Fatalf("expected file path validation error, got: %v", err)
	}

	// Test ImportURL parameter validation
	if _, err := svc.ImportURL(context.Background(), "", "https://example.com"); err == nil || !strings.Contains(err.Error(), "provider name cannot be empty") {
		t.Fatalf("expected provider name validation error, got: %v", err)
	}
	if _, err := svc.ImportURL(context.Background(), "Test", ""); err == nil || !strings.Contains(err.Error(), "provider URL cannot be empty") {
		t.Fatalf("expected provider URL validation error, got: %v", err)
	}
}
