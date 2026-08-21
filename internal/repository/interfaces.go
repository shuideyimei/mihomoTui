package repository

import "mihomoTui/internal/models"

// RuleRepository defines methods for reading and modifying Mihomo rules.
type RuleRepository interface {
	GetRules() ([]models.RuleDisplay, error)
	AddRule(ruleType, payload, proxy string) (string, error)
	DeleteRule(index int) (string, error)
	DeleteRuleByContent(ruleType, payload, proxy string) (string, error)
	MoveRule(fromIdx, toIdx int) (string, error)
	UpdateRule(index int, ruleType, payload, proxy string) (string, error)
	BatchAddRules(rules []models.RuleDisplay) (string, error)
}

// ProxyGroupRepository defines methods for reading and modifying proxy groups.
type ProxyGroupRepository interface {
	GetGroups() ([]models.ProxyGroupEntry, error)
	AddGroup(name, groupType, filter, testURL string, use, proxies []string) (string, error)
	UpdateGroup(oldName, name, groupType, filter, testURL string, use, proxies []string) (string, error)
	DeleteGroup(name string) (string, error)
	GetGroupNames() ([]string, error)
	GetGroupDef(groupName string) (proxies []string, use []string, err error)
}

// ProviderRepository defines methods for reading and modifying proxy and rule providers.
type ProviderRepository interface {
	GetProxyProviders() ([]models.ProviderInfo, error)
	AddProxyProvider(name, url string) (string, error)
	AddLocalProxyProvider(name, localPath string) (string, error)
	RemoveProxyProvider(name string) error

	GetRuleProviders() ([]models.RuleProviderEntry, error)
	AddRuleProvider(name, url, behavior string, interval int) (string, error)
	UpdateRuleProvider(name, url, behavior string, interval int) (string, error)
	DeleteRuleProvider(name string) (string, error)
	BatchAddRuleProviders(providers []models.RuleProviderEntry) (string, error)
}

// ProfileRepository defines methods for discovering, loading, and modifying config files.
type ProfileRepository interface {
	FindAllProfiles() []string
	GetCurrentProfilePath() string
	ReadProfile(path string) (string, error)
	WriteProfile(path, content string) error
	DeleteProfile(path string) error
	ValidateProfile(path string) error
	EnsureSafePath(cfgPath, path string) ([]string, error)
}
