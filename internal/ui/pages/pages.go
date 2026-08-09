package pages

import (
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"
	proxygroupspage "mihomoTui/internal/ui/pages/proxygroups"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

// NewEditor creates a new unified editor page
func NewEditor(proxyGroups *proxygroupspage.Page, rules *RulesPage, providers *rproviders.Page) *EditorPage {
	return NewEditorPage(proxyGroups, rules, providers)
}
func NewProxyGroups() *proxygroupspage.Page {
	return proxygroupspage.NewPage()
}

// NewRules creates a new rules management page
func NewRules() *RulesPage {
	return NewRulesPage()
}

// UnifiedSettingsPage wraps the API Config and UI Settings side by side
type UnifiedSettingsPage struct {
	*tview.Flex
	configPage *ConfigPage
	settings   *Settings
}

func NewUnifiedSettings(configManager *config.Manager, appName, appVersion string) *UnifiedSettingsPage {
	u := &UnifiedSettingsPage{
		Flex:       tview.NewFlex().SetDirection(tview.FlexColumn),
		configPage: NewConfigPage(configManager),
		settings:   newSettingsPage(configManager, appName, appVersion),
	}

	u.AddItem(u.configPage, 0, 1, true)
	u.AddItem(u.settings, 0, 1, false)

	// Set input capture to jump between them using Tab
	u.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Toggle focus
			if u.configPage.HasFocus() {
				ui.Updater.GetApp().SetFocus(u.settings)
			} else {
				ui.Updater.GetApp().SetFocus(u.configPage)
			}
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			if u.configPage.HasFocus() {
				ui.Updater.GetApp().SetFocus(u.settings)
			} else {
				ui.Updater.GetApp().SetFocus(u.configPage)
			}
			return nil
		}
		return event
	})

	return u
}

func (u *UnifiedSettingsPage) Activate() {
	u.configPage.Activate()
	// u.settings doesn't have Activate currently, if it does, call it
}

func (u *UnifiedSettingsPage) Deactivate() {
	u.configPage.Deactivate()
}

func NewConfigManager() *ConfigManager {
	return newConfigManagerPage()
}
