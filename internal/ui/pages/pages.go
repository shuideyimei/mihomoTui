package pages

import (
	"mihomoTui/internal/config"
	"mihomoTui/internal/events"
	"mihomoTui/internal/service"
	"mihomoTui/internal/ui"

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

// UnifiedSettingsPage wraps the API Config and UI Settings side by side
type UnifiedSettingsPage struct {
	*tview.Flex
	configPage *ConfigPage
	settings   *Settings
}

func NewUnifiedSettings(configManager *config.Manager, appName, appVersion string, bus *events.Bus) *UnifiedSettingsPage {
	u := &UnifiedSettingsPage{
		Flex:       tview.NewFlex().SetDirection(tview.FlexColumn),
		configPage: NewConfigPage(configManager),
		settings:   newSettingsPage(configManager, appName, appVersion, bus),
	}

	u.AddItem(u.configPage, 0, 1, true)
	u.AddItem(u.settings, 0, 1, false)

	// Set input capture to jump between them using Tab / Shift+Tab
	u.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		isBacktab := event.Key() == tcell.KeyBacktab || (event.Key() == tcell.KeyTab && event.Modifiers()&tcell.ModShift != 0)
		if isBacktab || event.Key() == tcell.KeyTab {
			// Toggle focus
			if u.configPage.HasFocus() {
				ui.Updater.SetFocus(u.settings)
			} else {
				ui.Updater.SetFocus(u.configPage)
			}
			return nil
		}
		return event
	})

	return u
}

func (u *UnifiedSettingsPage) Activate() {
	if u.configPage != nil {
		u.configPage.Activate()
	}
	// u.settings doesn't have Activate currently, if it does, call it
}

func (u *UnifiedSettingsPage) Deactivate() {
	if u.configPage != nil {
		u.configPage.Deactivate()
	}
}

func (u *UnifiedSettingsPage) UpdateTexts() {
	if u.settings != nil {
		u.settings.refreshTexts()
	}
	if u.configPage != nil {
		u.configPage.UpdateTexts()
	}
}

func NewConfigManager(profileService *service.ProfileService, bus *events.Bus) *ConfigManager {
	return newConfigManagerPage(profileService, bus)
}
