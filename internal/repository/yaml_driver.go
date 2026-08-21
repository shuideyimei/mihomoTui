package repository

import (
	"fmt"
	"os"

	"mihomoTui/internal/config"
	"mihomoTui/internal/models"
)

// LocalYAMLDriver implements RuleRepository, ProxyGroupRepository, ProviderRepository, and ProfileRepository
// using local YAML file manipulation and atomic file writes.
type LocalYAMLDriver struct{}

// NewLocalYAMLDriver creates a new LocalYAMLDriver instance.
func NewLocalYAMLDriver() *LocalYAMLDriver {
	return &LocalYAMLDriver{}
}

// ── RuleRepository Implementation ──

func (d *LocalYAMLDriver) GetRules() ([]models.RuleDisplay, error) {
	rawRules, err := config.GetRulesFromConfig()
	if err != nil {
		return nil, err
	}
	var rules []models.RuleDisplay
	for _, raw := range rawRules {
		rules = append(rules, config.ParseRuleLine(raw))
	}
	return rules, nil
}

func (d *LocalYAMLDriver) AddRule(ruleType, payload, proxy string) (string, error) {
	return config.AddRuleToConfig(ruleType, payload, proxy)
}

func (d *LocalYAMLDriver) DeleteRule(index int) (string, error) {
	return config.DeleteRuleFromConfig(index)
}

func (d *LocalYAMLDriver) DeleteRuleByContent(ruleType, payload, proxy string) (string, error) {
	return config.DeleteRuleByContent(ruleType, payload, proxy)
}

func (d *LocalYAMLDriver) MoveRule(fromIdx, toIdx int) (string, error) {
	return config.MoveRuleToPosition(fromIdx, toIdx)
}

func (d *LocalYAMLDriver) UpdateRule(index int, ruleType, payload, proxy string) (string, error) {
	return config.UpdateRuleInConfig(index, ruleType, payload, proxy)
}

func (d *LocalYAMLDriver) BatchAddRules(rules []models.RuleDisplay) (string, error) {
	return config.BatchAddRulesToConfig(rules)
}

// ── ProxyGroupRepository Implementation ──

func (d *LocalYAMLDriver) GetGroups() ([]models.ProxyGroupEntry, error) {
	return config.GetAllProxyGroupsFromConfig()
}

func (d *LocalYAMLDriver) AddGroup(name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	return config.AddGroupToConfig(name, groupType, filter, testURL, use, proxies)
}

func (d *LocalYAMLDriver) UpdateGroup(oldName, name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	return config.UpdateGroupInConfig(oldName, name, groupType, filter, testURL, use, proxies)
}

func (d *LocalYAMLDriver) DeleteGroup(name string) (string, error) {
	return config.DeleteGroupFromConfig(name)
}

func (d *LocalYAMLDriver) GetGroupNames() ([]string, error) {
	return config.GetProxyGroupsFromConfig()
}

func (d *LocalYAMLDriver) GetGroupDef(groupName string) ([]string, []string, error) {
	return config.GetGroupDef(groupName)
}

// ── ProviderRepository Implementation ──

func (d *LocalYAMLDriver) GetProxyProviders() ([]models.ProviderInfo, error) {
	return config.GetMihomoProxyProviders()
}

func (d *LocalYAMLDriver) AddProxyProvider(name, url string) (string, error) {
	return config.AddProviderToConfig(name, url)
}

func (d *LocalYAMLDriver) AddLocalProxyProvider(name, localPath string) (string, error) {
	return config.AddLocalProviderToConfig(name, localPath)
}

func (d *LocalYAMLDriver) RemoveProxyProvider(name string) error {
	return config.RemoveProviderFromConfig(name)
}

func (d *LocalYAMLDriver) GetRuleProviders() ([]models.RuleProviderEntry, error) {
	return config.GetRuleProvidersFromConfig()
}

func (d *LocalYAMLDriver) AddRuleProvider(name, url, behavior string, interval int) (string, error) {
	return config.AddRuleProviderToConfig(name, url, behavior, interval)
}

func (d *LocalYAMLDriver) UpdateRuleProvider(name, url, behavior string, interval int) (string, error) {
	return config.UpdateRuleProviderInConfig(name, url, behavior, interval)
}

func (d *LocalYAMLDriver) DeleteRuleProvider(name string) (string, error) {
	return config.DeleteRuleProviderFromConfig(name)
}

func (d *LocalYAMLDriver) BatchAddRuleProviders(providers []models.RuleProviderEntry) (string, error) {
	var batch []config.RuleProviderBatchEntry
	for _, p := range providers {
		batch = append(batch, config.RuleProviderBatchEntry{
			Name:     p.Name,
			URL:      p.URL,
			Behavior: p.Behavior,
			Interval: p.Interval,
		})
	}
	return config.BatchAddRuleProvidersToConfig(batch)
}

// ── ProfileRepository Implementation ──

func (d *LocalYAMLDriver) FindAllProfiles() []string {
	return config.FindAllConfigFiles()
}

func (d *LocalYAMLDriver) GetCurrentProfilePath() string {
	return config.FindCurrentConfigPath()
}

func (d *LocalYAMLDriver) ReadProfile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (d *LocalYAMLDriver) WriteProfile(path, content string) error {
	if content == "" {
		return fmt.Errorf("config content cannot be empty")
	}
	return config.WriteFileAtomic(path, []byte(content), 0644)
}

func (d *LocalYAMLDriver) DeleteProfile(path string) error {
	if config.SameConfigFile(path, config.FindCurrentConfigPath()) {
		return fmt.Errorf("cannot delete active profile")
	}
	return os.Remove(path)
}

func (d *LocalYAMLDriver) ValidateProfile(path string) error {
	return config.ValidateConfigFile(path)
}

func (d *LocalYAMLDriver) EnsureSafePath(cfgPath, path string) ([]string, error) {
	return config.EnsureSafePathInConfig(cfgPath, path)
}
