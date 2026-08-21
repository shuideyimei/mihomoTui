package pages

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ConfigPage represents the configuration page
type ConfigPage struct {
	*tview.Flex
	configManager *config.Manager

	// Form components
	form *tview.Form

	// Direct field references (avoids label-based lookup)
	apiAddrField   *tview.InputField
	apiSecretField *tview.InputField
	httpPortField  *tview.InputField
	socksPortField *tview.InputField
	mixedPortField *tview.InputField
	statusField    *tview.TextView

	// Current config values
	currentConfig *config.AppConfig
	mihomoConfig  *models.Config

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

// NewConfigPage creates a new configuration page
func NewConfigPage(configManager *config.Manager) *ConfigPage {
	page := &ConfigPage{
		Flex:          tview.NewFlex(),
		configManager: configManager,
		form:          tview.NewForm(),
		currentConfig: configManager.Get(),
	}

	page.setupUI()
	return page
}

// setupUI initializes the configuration page UI
func (c *ConfigPage) setupUI() {
	// Configure form
	c.setUpFormItems()
	c.form.SetBorder(true)
	c.form.SetTitle(fmt.Sprintf(" %s ", i18n.T("cfg.title")))
	components.StyleFocusBorder(c.form)
	c.form.SetButtonsAlign(tview.AlignCenter)
	c.form.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.form.SetButtonStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	c.form.SetButtonActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorBlack))

	// Add inline status as a read-only field
	statusView := tview.NewTextView()
	statusView.SetDynamicColors(true)
	statusView.SetText(i18n.T("cfg.saving"))
	c.form.AddFormItem(statusView)
	c.statusField = statusView

	// Layout: just the form
	c.SetDirection(tview.FlexColumn)
	c.AddItem(c.form, 0, 1, true)
}

// setUpFormItems initializes the form items
func (c *ConfigPage) setUpFormItems() {
	c.apiAddrField = tview.NewInputField()
	c.apiAddrField.SetLabel(i18n.T("cfg.api_address"))
	c.apiAddrField.SetFieldWidth(0)
	c.apiAddrField.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.form.AddFormItem(c.apiAddrField)

	c.apiSecretField = tview.NewInputField()
	c.apiSecretField.SetLabel(i18n.T("cfg.api_secret"))
	c.apiSecretField.SetFieldWidth(0)
	c.apiSecretField.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.form.AddFormItem(c.apiSecretField)

	c.httpPortField = tview.NewInputField()
	c.httpPortField.SetLabel(i18n.T("cfg.http_port"))
	c.httpPortField.SetFieldWidth(0)
	c.httpPortField.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.httpPortField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if lastChar == 0 {
			return true
		}
		return lastChar >= '0' && lastChar <= '9'
	})
	c.form.AddFormItem(c.httpPortField)

	c.socksPortField = tview.NewInputField()
	c.socksPortField.SetLabel(i18n.T("cfg.socks_port"))
	c.socksPortField.SetFieldWidth(0)
	c.socksPortField.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.socksPortField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if lastChar == 0 {
			return true
		}
		return lastChar >= '0' && lastChar <= '9'
	})
	c.form.AddFormItem(c.socksPortField)

	c.mixedPortField = tview.NewInputField()
	c.mixedPortField.SetLabel(i18n.T("cfg.mixed_port"))
	c.mixedPortField.SetFieldWidth(0)
	c.mixedPortField.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.mixedPortField.SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
		if lastChar == 0 {
			return true
		}
		return lastChar >= '0' && lastChar <= '9'
	})
	c.form.AddFormItem(c.mixedPortField)

	// Action buttons
	c.form.AddButton(i18n.T("cfg.save_btn"), c.saveConfig)
	c.form.AddButton(i18n.T("cfg.reset_btn"), c.confirmReset)
	c.form.AddButton(i18n.T("cfg.test_btn"), c.testConnection)
}

// Activate activates the config page
// Runs on the page-lifecycle goroutine; all widget mutations must be
// marshalled onto the tview UI goroutine via PostUi.
func (c *ConfigPage) Activate() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel
	c.mu.Unlock()

	ui.Updater.PostUi(func() {
		c.setStatus(i18n.T("cfg.saving"))
		// Load local config immediately
		c.updateConfigForm()
	})

	// Fetch remote port config in background
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		config, err := api.Client.GetConfigWithContext(ctx)
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err != nil {
			log.Printf("failed to fetch mihomo config for ports: %v", err)
			ui.Updater.PostUi(func() {
				c.setStatus(i18n.T("cfg.loaded"))
			})
			return
		}
		c.mu.Lock()
		c.mihomoConfig = config
		c.mu.Unlock()
		ui.Updater.PostUi(func() {
			c.updatePortFields()
			c.setStatus(i18n.T("cfg.loaded"))
		})
	}()
}

// Deactivate deactivates the config page
func (c *ConfigPage) Deactivate() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.mu.Unlock()
}

// updateConfigForm loads current configuration into form fields
func (c *ConfigPage) updateConfigForm() {
	c.apiAddrField.SetText(c.currentConfig.API.BaseURL)
	c.apiSecretField.SetText(c.currentConfig.API.Secret)
	c.updatePortFields()
}

func (c *ConfigPage) updatePortFields() {
	if c.mihomoConfig == nil {
		return
	}
	setField := func(field *tview.InputField, value int) {
		if value != 0 {
			field.SetText(strconv.Itoa(value))
		} else {
			field.SetText("")
		}
	}
	setField(c.httpPortField, c.mihomoConfig.Port)
	setField(c.socksPortField, c.mihomoConfig.SocksPort)
	setField(c.mixedPortField, c.mihomoConfig.MixedPort)
}

// saveConfig saves the configuration from form
func (c *ConfigPage) saveConfig() {
	// Create backup first
	if err := c.configManager.Backup(); err != nil {
		c.showStatus(fmt.Sprintf(i18n.T("cfg.save_fail"), err))
		return
	}

	// Get current config and only update API settings
	newConfig := *c.currentConfig // Copy current config

	// Update API settings from form
	newConfig.API.BaseURL = c.apiAddrField.GetText()
	newConfig.API.Secret = c.apiSecretField.GetText()

	// Validate (use a temporary manager to check config without writing)
	tempManager := &config.Manager{}
	tempManager.SetInMemory(&newConfig)
	if err := tempManager.Validate(); err != nil {
		c.showStatus(fmt.Sprintf(i18n.T("cfg.invalid_fail"), err))
		return
	}

	// Save
	if err := c.configManager.Set(&newConfig); err != nil {
		c.showStatus(fmt.Sprintf(i18n.T("cfg.save_fail"), err))
		return
	}

	// Update API
	api.UpdateClient(newConfig.API.BaseURL, newConfig.API.Secret)

	c.currentConfig = &newConfig

	portConfig := map[string]interface{}{
		"port":       c.getPortFieldValue(c.httpPortField),
		"socks-port": c.getPortFieldValue(c.socksPortField),
		"mixed-port": c.getPortFieldValue(c.mixedPortField),
	}
	if err := api.Client.PatchConfig(portConfig); err != nil {
		c.showStatus(fmt.Sprintf(i18n.T("cfg.saved_port_fail"), err))
		return
	}

	c.showStatus(i18n.T("cfg.saved_ok"))
}

func (c *ConfigPage) confirmReset() {
	ui.Updater.ShowConfirm(i18n.T("cfg.reset_btn"), i18n.T("cfg.reset_confirm"), c.resetConfig)
}

// resetConfig resets configuration to defaults
func (c *ConfigPage) resetConfig() {
	if err := c.configManager.Reset(); err != nil {
		c.showStatus(fmt.Sprintf(i18n.T("cfg.reset_fail"), err))
		return
	}
	c.currentConfig = c.configManager.Get()

	api.UpdateClient(c.currentConfig.API.BaseURL, c.currentConfig.API.Secret)
	ui.Updater.PostUi(func() {
		// Update API
		c.updateConfigForm()
		c.showStatus(i18n.T("cfg.reset_ok"))
	})
}

// testConnection tests the API connection using form values
func (c *ConfigPage) testConnection() {
	url := c.apiAddrField.GetText()
	secret := c.apiSecretField.GetText()

	if url == "" {
		c.showStatus(i18n.T("cfg.api_empty"))
		return
	}

	testBtn := c.form.GetButton(2)
	testBtn.SetDisabled(true)
	c.showStatus(i18n.T("cfg.testing"))

	go func() {
		testClient := api.NewClient(url, secret)
		err := testClient.HealthCheck()

		ui.Updater.PostUi(func() {
			testBtn.SetDisabled(false)
			if err != nil {
				c.showStatus(fmt.Sprintf(i18n.T("cfg.test_fail"), err))
			} else {
				c.showStatus(i18n.T("cfg.test_ok"))
			}
		})
	}()
}

func (c *ConfigPage) getPortFieldValue(field *tview.InputField) int {
	text := field.GetText()
	if text == "" {
		return 0
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return value
}

// setStatus updates the inline status text
func (c *ConfigPage) setStatus(message string) {
	if c.statusField != nil {
		c.statusField.SetText(message)
	}
}

// showStatus displays a status message (alias for setStatus)
func (c *ConfigPage) showStatus(message string) {
	c.setStatus(message)
}

// UpdateTexts refreshes localized labels on the app config form
func (c *ConfigPage) UpdateTexts() {
	ui.Updater.PostUi(func() {
		if c.form != nil {
			c.form.SetTitle(fmt.Sprintf(" %s ", i18n.T("cfg.title")))
		}
		if c.apiAddrField != nil {
			c.apiAddrField.SetLabel(i18n.T("cfg.api_address"))
		}
		if c.apiSecretField != nil {
			c.apiSecretField.SetLabel(i18n.T("cfg.api_secret"))
		}
		if c.httpPortField != nil {
			c.httpPortField.SetLabel(i18n.T("cfg.http_port"))
		}
		if c.socksPortField != nil {
			c.socksPortField.SetLabel(i18n.T("cfg.socks_port"))
		}
		if c.mixedPortField != nil {
			c.mixedPortField.SetLabel(i18n.T("cfg.mixed_port"))
		}
		if saveBtn := c.form.GetButton(0); saveBtn != nil {
			saveBtn.SetLabel(i18n.T("cfg.save"))
		}
		if resetBtn := c.form.GetButton(1); resetBtn != nil {
			resetBtn.SetLabel(i18n.T("cfg.reset"))
		}
		if testBtn := c.form.GetButton(2); testBtn != nil {
			testBtn.SetLabel(i18n.T("cfg.test_btn"))
		}
	})
}
