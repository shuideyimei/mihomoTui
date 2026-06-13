package pages

import (
	"mihomoTui/internal/config"
	proxygroupspage "mihomoTui/internal/ui/pages/proxygroups"
)

// ActivatablePage interface for pages that need activation/deactivation control
type ActivatablePage interface {
	Activate()
	Deactivate()
}

// NavigationGuard lets a page defer navigation while it confirms unsaved work.
type NavigationGuard interface {
	RequestDeactivate(continueNavigation func()) bool
}

// NewDashboard creates a new dashboard page
func NewDashboard() *DashboardPage {
	return NewDashboardPage()
}

// NewProxies creates a new proxies page
func NewProxies() *ProxiesPage {
	return NewProxiesPage()
}

// NewConnections creates a new connections page
func NewConnections() *ConnectionsPage {
	return NewConnectionsPage()
}

// NewConfig creates a new config page
func NewConfig(configManager *config.Manager) *ConfigPage {
	return NewConfigPage(configManager)
}

// NewLogs creates a new logs page
func NewLogs() *LogsPage {
	return NewLogsPage()
}

// NewProxyGroups creates a new proxy groups management page
func NewProxyGroups() *proxygroupspage.Page {
	return proxygroupspage.NewPage()
}

// NewRules creates a new rules management page
func NewRules() *RulesPage {
	return NewRulesPage()
}

// NewSettings creates a new settings page
func NewSettings(configManager *config.Manager, appName, appVersion string) *Settings {
	return newSettingsPage(configManager, appName, appVersion)
}

func NewConfigManager() *ConfigManager {
	return newConfigManagerPage()
}
