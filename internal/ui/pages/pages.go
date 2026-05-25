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

// NewDashboard creates a new dashboard page
func NewDashboard() *Dashboard {
	return &Dashboard{
		DashboardPage: NewDashboardPage(),
	}
}

// NewProxies creates a new proxies page
func NewProxies() *Proxies {
	return &Proxies{
		ProxiesPage: NewProxiesPage(),
	}
}

// NewConnections creates a new connections page
func NewConnections() *Connections {
	return &Connections{
		ConnectionsPage: NewConnectionsPage(),
	}
}

// NewConfig creates a new config page
func NewConfig(configManager *config.Manager) *Config {
	return &Config{
		ConfigPage: NewConfigPage(configManager),
	}
}

// NewLogs creates a new logs page
func NewLogs() *Logs {
	return &Logs{
		LogsPage: NewLogsPage(),
	}
}

// ProxyGroups wraps proxygroups.Page to implement ActivatablePage
type ProxyGroups struct {
	*proxygroupspage.Page
}

// NewProxyGroups creates a new proxy groups management page
func NewProxyGroups() *ProxyGroups {
	return &ProxyGroups{
		Page: proxygroupspage.NewPage(),
	}
}

// Activate activates the proxy groups page
func (p *ProxyGroups) Activate() {
	p.Page.Activate()
}

// Deactivate deactivates the proxy groups page
func (p *ProxyGroups) Deactivate() {
	p.Page.Deactivate()
}

// Rules wraps RulesPage to implement ActivatablePage
type Rules struct {
	*RulesPage
}

// NewRules creates a new rules management page
func NewRules() *Rules {
	return &Rules{
		RulesPage: NewRulesPage(),
	}
}

// Activate activates the rules page
func (p *Rules) Activate() {
	p.RulesPage.Activate()
}

// Deactivate deactivates the rules page
func (p *Rules) Deactivate() {
	p.RulesPage.Deactivate()
}

// NewSettings creates a new settings page
func NewSettings(configManager *config.Manager, appName, appVersion string) *Settings {
	return newSettingsPage(configManager, appName, appVersion)
}

func NewConfigManager() *ConfigManager {
	return newConfigManagerPage()
}
