package pages

import (
	"fmt"
	"log"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

// Config represents the config page
type Config struct {
	*ConfigPage
}

// Activate activates the config page
func (c *Config) Activate() {
	log.Printf("Activating config page")
	c.ConfigPage.Activate()
}

// Deactivate deactivates the config page
func (c *Config) Deactivate() {
	log.Printf("Deactivating config page")
	c.ConfigPage.Deactivate()
}

// ConfigPage represents the configuration page
type ConfigPage struct {
	*tview.Flex
	configManager *config.Manager

	// Form components
	form       *tview.Form
	statusText *tview.TextView

	// Labels
	labels []string

	// Current config values
	currentConfig *config.AppConfig
}

// NewConfigPage creates a new configuration page
func NewConfigPage(configManager *config.Manager) *ConfigPage {
	page := &ConfigPage{
		Flex:          tview.NewFlex(),
		configManager: configManager,
		form:          tview.NewForm(),
		statusText:    tview.NewTextView(),
		currentConfig: configManager.Get(),
		labels:        []string{"API地址", "API密钥"},
	}

	page.setupUI()
	return page
}

// setupUI initializes the configuration page UI
func (c *ConfigPage) setupUI() {
	// Configure form
	c.setUpFormItems()
	c.form.SetBorder(true)
	c.form.SetTitle(" 应用配置 ")
	c.form.SetButtonsAlign(tview.AlignCenter)

	// Configure status text
	c.statusText.SetBorder(true)
	c.statusText.SetTitle(" 状态 ")
	c.statusText.SetDynamicColors(true)
	c.statusText.SetText("[green]加载配置中...[white]")

	// Layout - status bar now has fixed height of 1 line + borders (3 total)
	c.SetDirection(tview.FlexColumn)
	c.AddItem(c.form, 0, 3, true)
	c.AddItem(c.statusText, 0, 1, false)
}

// setUpFormItems initializes the form items
func (c *ConfigPage) setUpFormItems() {
	for _, label := range c.labels {
		c.form.AddInputField(label, "", 50, nil, nil)
	}

	// Action buttons
	c.form.AddButton("保存", c.saveConfig)
	c.form.AddButton("重置", c.resetConfig)
	c.form.AddButton("测试连接", c.testConnection)
}

// Activate activates the config page
func (c *ConfigPage) Activate() {
	ui.Updater.UpdateUi(
		func() {
			c.updateConfigForm()
		})
}

// Deactivate deactivates the config page
func (c *ConfigPage) Deactivate() {
}

// updateConfigForm loads current configuration into form fields
func (c *ConfigPage) updateConfigForm() {
	for _, label := range c.labels {
		c.form.GetFormItemByLabel(label).(*tview.InputField).SetText(c.currentConfig.GetValue(label))
	}
	c.statusText.SetText("[green]配置加载完毕[white]")
}

// saveConfig saves the configuration from form
func (c *ConfigPage) saveConfig() {
	// Create backup first
	if err := c.configManager.Backup(); err != nil {
		c.showStatus(fmt.Sprintf("[red]备份失败: %v[white]", err))
		return
	}

	// Get current config and only update API settings
	newConfig := *c.currentConfig // Copy current config

	// Update API settings from form
	for _, value := range c.labels {
		newConfig.SetValue(value, c.form.GetFormItemByLabel(value).(*tview.InputField).GetText())
	}

	// Validate
	tempManager := &config.Manager{}
	tempManager.Set(&newConfig)
	if err := tempManager.Validate(); err != nil {
		c.showStatus(fmt.Sprintf("[red]配置无效: %v[white]", err))
		return
	}

	// Save
	c.configManager.Set(&newConfig)
	if err := c.configManager.Save(); err != nil {
		c.showStatus(fmt.Sprintf("[red]保存失败: %v[white]", err))
		return
	}

	// Update API
	api.UpdateClient(newConfig.API.BaseURL, newConfig.API.Secret)

	c.currentConfig = &newConfig
	c.showStatus("[green]配置已保存[white]")
}

// resetConfig resets configuration to defaults
func (c *ConfigPage) resetConfig() {
	if err := c.configManager.Reset(); err != nil {
		c.showStatus(fmt.Sprintf("[red]重置失败: %v[white]", err))
		return
	}
	c.currentConfig = c.configManager.Get()

	api.UpdateClient(c.currentConfig.API.BaseURL, c.currentConfig.API.Secret)
	go ui.Updater.UpdateUi(
		func() {
			// Update API
			c.updateConfigForm()
			c.showStatus("[yellow]配置已重置为默认值[white]")
		})
}

// testConnection tests the API connection
func (c *ConfigPage) testConnection() {
	c.showStatus("[yellow]正在测试连接...[white]")

	if err := api.Client.HealthCheck(); err != nil {
		c.showStatus(fmt.Sprintf("[red]连接失败: %v[white]", err))
	} else {
		c.showStatus("[green]连接成功![white]")
	}
}

// showStatus displays a status message
func (c *ConfigPage) showStatus(message string) {
	c.statusText.SetText(message)
}
