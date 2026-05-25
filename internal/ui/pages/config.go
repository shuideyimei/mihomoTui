package pages

import (
	"fmt"
	"log"
	"strconv"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/models"
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
	labels     []string
	portLabels []string

	// Current config values
	currentConfig *config.AppConfig
	mihomoConfig  *models.Config
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
		portLabels:    []string{"HTTP端口", "SOCKS端口", "混合端口"},
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

	for _, label := range c.portLabels {
		c.form.AddInputField(label, "", 10, func(textToCheck string, lastChar rune) bool {
			if lastChar == 0 {
				return true
			}
			return lastChar >= '0' && lastChar <= '9'
		}, nil)
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

	go func() {
		config, err := api.Client.GetConfig()
		if err != nil {
			log.Printf("failed to fetch mihomo config for ports: %v", err)
			return
		}
		c.mihomoConfig = config
		ui.Updater.UpdateUi(func() {
			c.updatePortFields()
		})
	}()
}

// Deactivate deactivates the config page
func (c *ConfigPage) Deactivate() {
}

// updateConfigForm loads current configuration into form fields
func (c *ConfigPage) updateConfigForm() {
	for _, label := range c.labels {
		c.form.GetFormItemByLabel(label).(*tview.InputField).SetText(c.currentConfig.GetValue(label))
	}
	c.updatePortFields()
	c.statusText.SetText("[green]配置加载完毕[white]")
}

func (c *ConfigPage) updatePortFields() {
	if c.mihomoConfig == nil {
		return
	}
	setField := func(label string, value int) {
		if value != 0 {
			c.form.GetFormItemByLabel(label).(*tview.InputField).SetText(strconv.Itoa(value))
		} else {
			c.form.GetFormItemByLabel(label).(*tview.InputField).SetText("")
		}
	}
	setField("HTTP端口", c.mihomoConfig.Port)
	setField("SOCKS端口", c.mihomoConfig.SocksPort)
	setField("混合端口", c.mihomoConfig.MixedPort)
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

	// Validate (use a temporary manager to check config without writing)
	tempManager := &config.Manager{}
	tempManager.SetInMemory(&newConfig)
	if err := tempManager.Validate(); err != nil {
		c.showStatus(fmt.Sprintf("[red]配置无效: %v[white]", err))
		return
	}

	// Save
	if err := c.configManager.Set(&newConfig); err != nil {
		c.showStatus(fmt.Sprintf("[red]保存失败: %v[white]", err))
		return
	}

	// Update API
	api.UpdateClient(newConfig.API.BaseURL, newConfig.API.Secret)

	c.currentConfig = &newConfig

	portConfig := &models.Config{
		Port:      c.getPortFieldValue("HTTP端口"),
		SocksPort: c.getPortFieldValue("SOCKS端口"),
		MixedPort: c.getPortFieldValue("混合端口"),
	}
	if err := api.Client.UpdateConfig(portConfig); err != nil {
		c.showStatus(fmt.Sprintf("[yellow]配置已保存，但端口更新失败: %v[white]", err))
		return
	}

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

// testConnection tests the API connection using form values
func (c *ConfigPage) testConnection() {
	url := c.form.GetFormItemByLabel("API地址").(*tview.InputField).GetText()
	secret := c.form.GetFormItemByLabel("API密钥").(*tview.InputField).GetText()

	if url == "" {
		c.showStatus("[red]API地址不能为空[white]")
		return
	}

	testBtn := c.form.GetButton(2)
	testBtn.SetDisabled(true)
	c.showStatus("[yellow]正在测试连接...[white]")

	origURL := c.currentConfig.API.BaseURL
	origSecret := c.currentConfig.API.Secret

	go func() {
		api.UpdateClient(url, secret)
		defer api.UpdateClient(origURL, origSecret)
		err := api.Client.HealthCheck()

		ui.Updater.UpdateUi(func() {
			testBtn.SetDisabled(false)
			if err != nil {
				c.showStatus(fmt.Sprintf("[red]连接失败: %v[white]", err))
			} else {
				c.showStatus("[green]连接成功![white]")
			}
		})
	}()
}

func (c *ConfigPage) getPortFieldValue(label string) int {
	text := c.form.GetFormItemByLabel(label).(*tview.InputField).GetText()
	if text == "" {
		return 0
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return value
}

// showStatus displays a status message
func (c *ConfigPage) showStatus(message string) {
	c.statusText.SetText(message)
}
