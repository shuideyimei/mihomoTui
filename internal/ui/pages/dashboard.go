package pages

import (
	"context"
	"fmt"
	"log"
	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"
	"mihomoTui/internal/utils"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Dashboard represents the dashboard page
type Dashboard struct {
	*DashboardPage
}

// Activate activates the dashboard page
func (d *Dashboard) Activate() {
	log.Printf("Activating dashboard page")
	d.DashboardPage.Activate()
}

// Deactivate deactivates the dashboard page
func (d *Dashboard) Deactivate() {
	log.Printf("Deactivating dashboard page")
	d.DashboardPage.Deactivate()
}

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
	buttonNav *components.ButtonNavigator

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
	d.AddItem(leftCol, 0, 1, false)
	d.AddItem(rightCol, 0, 1, true)

	d.SetBorder(true)
	d.SetTitle(" 仪表板 ")
}

// createConnectionsBox creates the connections statistics component
func (d *DashboardPage) createConnectionsBox() {
	d.connectionsBox = tview.NewTextView()
	d.connectionsBox.SetBorder(true)
	d.connectionsBox.SetTitle(" 连接统计 ")
	d.connectionsBox.SetDynamicColors(true)
	d.connectionsBox.SetText("正在获取连接数据...")
}

// createSystemInfoBox creates the system resource monitoring component
func (d *DashboardPage) createSystemInfoBox() {
	d.systemInfoBox = tview.NewTextView()
	d.systemInfoBox.SetBorder(true)
	d.systemInfoBox.SetTitle(" 系统信息 ")
	d.systemInfoBox.SetDynamicColors(true)
	d.systemInfoBox.SetText("正在获取系统信息...")
}

// createControlButtons creates the control buttons component
func (d *DashboardPage) createControlButtons() {
	d.controlButtons = tview.NewFlex()
	d.controlButtons.SetBorder(true)
	d.controlButtons.SetTitle(" 控制面板 ")
	d.controlButtons.SetDirection(tview.FlexRow)

	// Create buttons row
	buttonsRow := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Create AllowLAN button
	d.allowLanBtn = tview.NewButton("AllowLAN: ?")
	d.allowLanBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorGreen))
	d.allowLanBtn.SetSelectedFunc(d.toggleAllowLan)

	// Create TUN button
	d.tunBtn = tview.NewButton("TUN: ?")
	d.tunBtn.SetStyle(tcell.StyleDefault.Background(tcell.ColorDeepSkyBlue))
	d.tunBtn.SetSelectedFunc(d.toggleTun)

	// Initialize button navigation
	d.buttonNav = components.NewButtonNavigator()
	d.buttonNav.SetButtons([]*tview.Button{d.allowLanBtn, d.tunBtn})

	// Add buttons to buttons row
	buttonsRow.AddItem(d.allowLanBtn, 0, 2, true)
	buttonsRow.AddItem(nil, 0, 1, false)
	buttonsRow.AddItem(d.tunBtn, 0, 2, false)

	// Create status text view for mode and port info
	d.statusText = tview.NewTextView()
	d.statusText.SetDynamicColors(true)
	d.statusText.SetText("正在获取状态信息...")

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
	d.operationStatus.SetTitle(" 操作状态 ")
	d.operationStatus.SetDynamicColors(true)
	d.operationStatus.SetText("[white]就绪[white]")
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
			d.connectionsBox.SetText(fmt.Sprintf("[red]获取连接数据失败: %v[white]", err))
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
	content := fmt.Sprintf("[green]🔗 总连接数[white] %d", totalConnections)
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
	allowLanStatus := "OFF"
	allowLanBg := tcell.ColorRed
	if config.AllowLan {
		allowLanStatus = "ON"
		allowLanBg = tcell.ColorGreen
	}

	tunStatus := "OFF"
	tunColor := tcell.ColorRed
	if config.Tun != nil {
		if enable, ok := config.Tun["enable"].(bool); ok && enable {
			tunStatus = "ON"
			tunColor = tcell.ColorGreen
		}
	}

	// Build status text outside PostUi
	statusContent := d.buildStatusText(config)

	ui.Updater.PostUi(func() {
		d.allowLanBtn.SetLabel(fmt.Sprintf("AllowLAN: %s", allowLanStatus))
		d.allowLanBtn.SetBackgroundColor(allowLanBg)
		d.tunBtn.SetLabel(fmt.Sprintf("TUN: %s", tunStatus))
		d.tunBtn.SetBackgroundColor(tunColor)
		d.statusText.SetText(statusContent)
	})
}

// buildStatusText builds the status text content string (goroutine-safe, no UI calls)
func (d *DashboardPage) buildStatusText(config *models.Config) string {
	if config == nil {
		return ""
	}

	var contentBuilder strings.Builder

	modeColor := "white"
	switch config.Mode {
	case "global":
		modeColor = "red"
	case "rule":
		modeColor = "green"
	case "direct":
		modeColor = "yellow"
	}

	fmt.Fprintf(&contentBuilder, "🎯 模式 [%s]%s[white]\n", modeColor, config.Mode)

	fmt.Fprintf(&contentBuilder, "[yellow]⚙️ 端口状态[white]")
	hasPort := false
	if config.Port != 0 {
		fmt.Fprintf(&contentBuilder, " | HTTP: %d", config.Port)
		hasPort = true
	}
	if config.SocksPort != 0 {
		fmt.Fprintf(&contentBuilder, " | SOCKS: %d", config.SocksPort)
		hasPort = true
	}
	if config.MixedPort != 0 {
		fmt.Fprintf(&contentBuilder, " | Mixed: %d", config.MixedPort)
		hasPort = true
	}
	if config.RedirPort != 0 {
		fmt.Fprintf(&contentBuilder, " | Redir: %d", config.RedirPort)
		hasPort = true
	}
	if config.TProxyPort != 0 {
		fmt.Fprintf(&contentBuilder, " | TProxy: %d", config.TProxyPort)
		hasPort = true
	}
	if !hasPort {
		contentBuilder.WriteString("  无活动端口")
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
			d.operationStatus.SetText("[white]就绪[white]")
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
	fmt.Fprintf(&content, "[blue]💾 内存使用[white] %s", memUsage)

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
		d.showOperationStatus("[red]错误: 无配置数据[white]")
		return
	}

	// Show operation in progress
	d.showOperationStatus("[yellow]正在切换 AllowLAN...[white]")

	// Create a new config with toggled Allow LAN setting
	newConfig := *config // Copy existing config
	newConfig.AllowLan = !config.AllowLan

	// Update the setting via API
	err := api.Client.UpdateConfig(&newConfig)

	if err != nil {
		d.showOperationStatus(fmt.Sprintf("[red]AllowLAN 切换失败: %v[white]", err))
		log.Printf("Failed to toggle Allow LAN: %v", err)
		return
	}

	status := "关闭"
	if newConfig.AllowLan {
		status = "开启"
	}
	d.showOperationStatus(fmt.Sprintf("[green]AllowLAN 已%s[white]", status))
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
		d.showOperationStatus("[red]错误: 无配置数据[white]")
		return
	}

	// Show operation in progress
	d.showOperationStatus("[yellow]正在切换 TUN...[white]")

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
		d.showOperationStatus(fmt.Sprintf("[red]TUN 切换失败: %v[white]", err))
		log.Printf("Failed to toggle TUN: %v", err)
		return
	}

	status := "关闭"
	if newTunEnabled {
		status = "开启"
	}
	d.showOperationStatus(fmt.Sprintf("[green]TUN 已%s[white]", status))
	log.Printf("TUN toggled to: %v", newTunEnabled)

	// Refresh data to show updated status
	go d.updateProxyStatusData()
}
