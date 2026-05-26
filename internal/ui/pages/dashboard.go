package pages

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"
	"mihomoTui/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// DashboardPage represents the main dashboard page
type DashboardPage struct {
	*tview.Flex

	// Components
	connectionsBox  *tview.TextView
	systemInfoBox   *tview.TextView
	controlButtons  *tview.Flex
	allowLanBtn     *tview.Button
	tunBtn          *tview.Button
	statusText      *tview.TextView
	operationStatus *tview.TextView // Status bar for operation results

	// Button navigation
	buttonNav *components.FocusNavigator

	// Data
	connectionsData []models.Connection
	memoryData      *models.MemoryUsage
	configData      *models.Config

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	// Update frequency
	updateInterval time.Duration
}

// NewDashboardPage creates a new dashboard page
func NewDashboardPage() *DashboardPage {
	dashboard := &DashboardPage{
		Flex:           tview.NewFlex(),
		updateInterval: 2 * time.Second,
	}

	dashboard.setupLayout()

	return dashboard
}

// setupLayout sets up the dashboard layout
func (d *DashboardPage) setupLayout() {
	// Create main containers
	d.createConnectionsBox()
	d.createSystemInfoBox()
	d.createControlButtons()
	d.createOperationStatus()

	// Create left column (connections + system info)
	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(d.connectionsBox, 0, 1, false)
	leftCol.AddItem(d.systemInfoBox, 0, 1, false)

	// Create right column (control panel + operation status)
	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(d.controlButtons, 0, 1, true)
	rightCol.AddItem(d.operationStatus, 3, 0, false) // Fixed height for status

	// Main layout
	d.SetDirection(tview.FlexColumn)
	d.AddItem(leftCol, 0, 3, false)
	d.AddItem(rightCol, 0, 2, true)

	d.SetBorder(true)
	d.SetTitle(fmt.Sprintf(" %s ", i18n.T("dash.title")))
}

// createConnectionsBox creates the connections statistics component
func (d *DashboardPage) createConnectionsBox() {
	d.connectionsBox = tview.NewTextView()
	d.connectionsBox.SetBorder(true)
	d.connectionsBox.SetTitle(fmt.Sprintf(" %s ", i18n.T("dash.conn_stats")))
	d.connectionsBox.SetDynamicColors(true)
	d.connectionsBox.SetText(i18n.T("dash.fetching_conn"))
}

// createSystemInfoBox creates the system resource monitoring component
func (d *DashboardPage) createSystemInfoBox() {
	d.systemInfoBox = tview.NewTextView()
	d.systemInfoBox.SetBorder(true)
	d.systemInfoBox.SetTitle(fmt.Sprintf(" %s ", i18n.T("dash.sys_info")))
	d.systemInfoBox.SetDynamicColors(true)
	d.systemInfoBox.SetText(i18n.T("dash.fetching_sys"))
}

// createControlButtons creates the control buttons component
func (d *DashboardPage) createControlButtons() {
	d.controlButtons = tview.NewFlex()
	d.controlButtons.SetBorder(true)
	d.controlButtons.SetTitle(fmt.Sprintf(" %s ", i18n.T("dash.control_panel")))
	d.controlButtons.SetDirection(tview.FlexRow)

	// Create buttons row
	buttonsRow := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Create AllowLAN button
	d.allowLanBtn = tview.NewButton("○ AllowLAN")
	d.allowLanBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorGray))
	d.allowLanBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorGray))
	d.allowLanBtn.SetSelectedFunc(d.toggleAllowLan)

	// Create TUN button
	d.tunBtn = tview.NewButton("○ TUN")
	d.tunBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorGray))
	d.tunBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorGray))
	d.tunBtn.SetSelectedFunc(d.toggleTun)

	// Initialize button navigation
	d.buttonNav = components.NewFocusNavigator(d.allowLanBtn, d.tunBtn)

	// Add buttons to buttons row
	buttonsRow.AddItem(d.allowLanBtn, 0, 2, true)
	buttonsRow.AddItem(nil, 0, 1, false)
	buttonsRow.AddItem(d.tunBtn, 0, 2, false)

	// Create status text view for mode and port info
	d.statusText = tview.NewTextView()
	d.statusText.SetDynamicColors(true)
	d.statusText.SetText(i18n.T("dash.fetching_status"))

	// Add to main control panel
	d.controlButtons.AddItem(buttonsRow, 1, 0, true)
	d.controlButtons.AddItem(d.statusText, 0, 1, false)

	// Set up keyboard navigation for buttons
	d.controlButtons.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB, tcell.KeyRight:
			d.buttonNav.Next()
			return nil
		case tcell.KeyBacktab, tcell.KeyLeft:
			d.buttonNav.Prev()
			return nil
		}
		return event
	})
}

// createOperationStatus creates the operation status bar
func (d *DashboardPage) createOperationStatus() {
	d.operationStatus = tview.NewTextView()
	d.operationStatus.SetBorder(true)
	d.operationStatus.SetTitle(fmt.Sprintf(" %s ", i18n.T("dash.control_panel")))
	d.operationStatus.SetDynamicColors(true)
	d.operationStatus.SetText(i18n.T("dash.status_ready"))
}

// Activate activates the dashboard page
func (d *DashboardPage) Activate() {
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.startDataUpdates()
}

// Deactivate deactivates the dashboard page
func (d *DashboardPage) Deactivate() {
	if d.ctx != nil {
		d.cancel()
	}
}

// startDataUpdates starts the real-time data update mechanism
func (d *DashboardPage) startDataUpdates() {
	d.updateSystemInfo()

	go d.startStreamMemoryUsage()
	// Start periodic updates for other data
	go d.periodicUpdates()
}

// periodicUpdates handles periodic updates for non-streaming data
func (d *DashboardPage) periodicUpdates() {
	ticker := time.NewTicker(d.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.updateConnectionsData()
			d.updateProxyStatusData()
			d.updateSystemInfo()
		}
	}
}

// updateConnectionsData updates connections data
func (d *DashboardPage) updateConnectionsData() {
	connections, err := api.Client.GetConnections()
	if err != nil {
		ui.Updater.PostUi(func() {
			d.connectionsBox.SetText(fmt.Sprintf(i18n.T("dash.conn_fetch_fail"), err))
		})
		return
	}

	d.mutex.Lock()
	d.connectionsData = connections
	d.mutex.Unlock()

	d.updateConnectionsDisplay()
}

// updateConnectionsDisplay updates the connections display
func (d *DashboardPage) updateConnectionsDisplay() {
	d.mutex.RLock()
	connections := d.connectionsData
	d.mutex.RUnlock()

	if connections == nil {
		return
	}

	totalConnections := len(connections)
	content := fmt.Sprintf(i18n.T("dash.total_conn"), totalConnections)
	ui.Updater.PostUi(func() {
		d.connectionsBox.SetText(content)
	})
}

// updateProxyStatusData updates proxy status data
func (d *DashboardPage) updateProxyStatusData() {
	// Get current config
	config, err := api.Client.GetConfig()
	if err != nil {
		log.Printf("Failed to get config: %v", err)
		return
	}

	d.mutex.Lock()
	d.configData = config
	d.mutex.Unlock()

	d.updateControlButtons()
}

// updateControlButtons updates the control buttons based on current config
func (d *DashboardPage) updateControlButtons() {
	d.mutex.RLock()
	config := d.configData
	d.mutex.RUnlock()

	if config == nil {
		return
	}

	// Compute button states (outside PostUi to avoid blocking UI goroutine)
	on := config.AllowLan

	tunOn := false
	if config.Tun != nil {
		if enable, ok := config.Tun["enable"].(bool); ok && enable {
			tunOn = true
		}
	}

	// Build status text outside PostUi
	statusContent := d.buildStatusText(config)

	ui.Updater.PostUi(func() {
		// AllowLAN toggle: ● ON / ○ OFF with brightness
		if on {
			d.allowLanBtn.SetLabel("● AllowLAN")
			d.allowLanBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
			d.allowLanBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
		} else {
			d.allowLanBtn.SetLabel("○ AllowLAN")
			d.allowLanBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorGray))
			d.allowLanBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorGray))
		}
		// TUN toggle: ● ON / ○ OFF with brightness
		if tunOn {
			d.tunBtn.SetLabel("● TUN")
			d.tunBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
			d.tunBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
		} else {
			d.tunBtn.SetLabel("○ TUN")
			d.tunBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorGray))
			d.tunBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorGray))
		}
		d.statusText.SetText(statusContent)
	})
}

// buildStatusText builds the status text content string (goroutine-safe, no UI calls)
func (d *DashboardPage) buildStatusText(config *models.Config) string {
	if config == nil {
		return ""
	}

	var contentBuilder strings.Builder

	var modeColor string
	switch config.Mode {
	case "rule":
		modeColor = ui.ThemeTagModeRule
	case "global":
		modeColor = ui.ThemeTagModeGlobal
	case "direct":
		modeColor = ui.ThemeTagModeDirect
	default:
		modeColor = "[white]"
	}

	fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.mode_label"), modeColor, config.Mode))

	fmt.Fprint(&contentBuilder, i18n.T("dash.port_status"))
	hasPort := false
	if config.Port != 0 {
		fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.http_port"), config.Port))
		hasPort = true
	}
	if config.SocksPort != 0 {
		fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.socks_port"), config.SocksPort))
		hasPort = true
	}
	if config.MixedPort != 0 {
		fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.mixed_port"), config.MixedPort))
		hasPort = true
	}
	if config.RedirPort != 0 {
		fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.redir_port"), config.RedirPort))
		hasPort = true
	}
	if config.TProxyPort != 0 {
		fmt.Fprint(&contentBuilder, fmt.Sprintf(i18n.T("dash.tproxy_port"), config.TProxyPort))
		hasPort = true
	}
	if !hasPort {
		contentBuilder.WriteString(i18n.T("dash.no_active_port"))
	}

	return contentBuilder.String()
}

// showOperationStatus displays operation result status
func (d *DashboardPage) showOperationStatus(message string) {
	d.operationStatus.SetText(message)

	// Auto-clear status after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		ui.Updater.PostUi(func() {
			d.operationStatus.SetText(i18n.T("dash.status_ready"))
		})
	}()
}

// updateSystemInfo updates system information
func (d *DashboardPage) updateSystemInfo() {
	content := strings.Builder{}
	var memUsage string
	d.mutex.RLock()
	if d.memoryData != nil {
		memUsage = utils.FormatBytes(d.memoryData.Inuse)
	} else {
		memUsage = "-"
	}
	d.mutex.RUnlock()
	fmt.Fprint(&content, fmt.Sprintf(i18n.T("dash.memory_usage"), memUsage))

	ui.Updater.PostUi(func() {
		d.systemInfoBox.SetText(content.String())
	})
}

// Stop stops all background updates
func (d *DashboardPage) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
}

// Refresh manually refreshes all data
func (d *DashboardPage) Refresh() {
	go func() {
		d.updateConnectionsData()
		d.updateProxyStatusData()
		d.updateSystemInfo()
	}()
}

// SetUpdateInterval sets the update interval for periodic updates
func (d *DashboardPage) SetUpdateInterval(interval time.Duration) {
	d.updateInterval = interval
}

// GetInputCapture returns the input capture function for keyboard shortcuts
func (d *DashboardPage) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			d.Refresh()
			return nil
		}
		return event
	}
}

// startStreamMemoryUsage starts streaming memory data
func (d *DashboardPage) startStreamMemoryUsage() {
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			err := api.StreamClient.StreamMemoryUsage(d.ctx, func(memory *models.MemoryUsage) {
				d.mutex.Lock()
				d.memoryData = memory
				d.mutex.Unlock()
			})

			if err != nil && err != context.Canceled {
				d.mutex.Lock()
				d.memoryData = nil
				d.mutex.Unlock()
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// toggleAllowLan toggles the Allow LAN setting
func (d *DashboardPage) toggleAllowLan() {
	d.mutex.RLock()
	config := d.configData
	d.mutex.RUnlock()

	if config == nil {
		d.showOperationStatus(i18n.T("dash.no_config_err"))
		return
	}

	// Show operation in progress
	d.showOperationStatus(i18n.T("dash.toggling_lan"))

	// Create a new config with toggled Allow LAN setting
	newConfig := *config // Copy existing config
	newConfig.AllowLan = !config.AllowLan

	// Update the setting via API
	err := api.Client.UpdateConfig(&newConfig)

	if err != nil {
		d.showOperationStatus(fmt.Sprintf(i18n.T("dash.lan_toggle_fail"), err))
		log.Printf("Failed to toggle Allow LAN: %v", err)
		return
	}

	status := i18n.T("dash.lan_off_label")
	if newConfig.AllowLan {
		status = i18n.T("dash.lan_on_label")
	}
	d.showOperationStatus(fmt.Sprintf(i18n.T("dash.lan_toggled"), status))
	log.Printf("Allow LAN toggled to: %v", newConfig.AllowLan)

	// Refresh data to show updated status
	go d.updateProxyStatusData()
}

// toggleTun toggles the TUN mode setting
func (d *DashboardPage) toggleTun() {
	d.mutex.RLock()
	config := d.configData
	d.mutex.RUnlock()

	if config == nil {
		d.showOperationStatus(i18n.T("dash.no_config_err"))
		return
	}

	// Show operation in progress
	d.showOperationStatus(i18n.T("dash.toggling_tun"))

	// Create a new config with toggled TUN setting
	newConfig := *config // Copy existing config

	// Initialize Tun map if it's nil
	if newConfig.Tun == nil {
		newConfig.Tun = make(map[string]interface{})
	}

	// Get current TUN state
	currentTunEnabled := false
	if enable, ok := newConfig.Tun["enable"].(bool); ok {
		currentTunEnabled = enable
	}

	// Toggle TUN state
	newTunEnabled := !currentTunEnabled
	newConfig.Tun["enable"] = newTunEnabled

	// Update the setting via API
	err := api.Client.UpdateConfig(&newConfig)

	if err != nil {
		d.showOperationStatus(fmt.Sprintf(i18n.T("dash.tun_toggle_fail"), err))
		log.Printf("Failed to toggle TUN: %v", err)
		return
	}

	status := i18n.T("dash.tun_off_label")
	if newTunEnabled {
		status = i18n.T("dash.tun_on_label")
	}
	d.showOperationStatus(fmt.Sprintf(i18n.T("dash.tun_toggled"), status))
	log.Printf("TUN toggled to: %v", newTunEnabled)

	// Refresh data to show updated status
	go d.updateProxyStatusData()
}
