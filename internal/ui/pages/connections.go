package pages

import (
	"context"
	"fmt"
	"log"
	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Connections represents the connections page
type Connections struct {
	*ConnectionsPage
}

// Activate activates the connections page
func (c *Connections) Activate() {
	log.Printf("Activating connections page")
	c.ConnectionsPage.Activate()
}

// Deactivate deactivates the connections page
func (c *Connections) Deactivate() {
	log.Printf("Deactivating connections page")
	c.ConnectionsPage.Deactivate()
}

// ConnectionsPage represents the connections management page
type ConnectionsPage struct {
	*tview.Flex

	// Components
	connectionsTable *tview.Table
	statusText       *tview.TextView
	infoPanel        *tview.TextView

	// Data
	connections    []models.Connection
	selectedConnID string

	// Control
	mutex         sync.RWMutex
	refreshTicker *time.Ticker
	refreshCancel context.CancelFunc

	// State
	isActive     bool
	lastUpdate   time.Time
	autoRefresh  bool
	refreshCount int
}

// NewConnectionsPage creates a new connections management page
func NewConnectionsPage() *ConnectionsPage {
	page := &ConnectionsPage{
		Flex:        tview.NewFlex(),
		connections: make([]models.Connection, 0),
		autoRefresh: true,
	}

	page.setupLayout()
	page.setupEventHandlers()

	return page
}

// Activate initializes the page when it becomes active
func (c *ConnectionsPage) Activate() {
	c.mutex.Lock()
	c.isActive = true
	c.mutex.Unlock()

	// Load initial data
	c.loadConnectionsData()

	// Start auto-refresh
	c.startAutoRefresh()

	go ui.Updater.UpdateUi(func() {
		c.statusText.SetText("连接页面已激活")
		c.updateConnectionsTable()
	})
}

// Deactivate deactivates the connections page and stops auto-refresh
func (c *ConnectionsPage) Deactivate() {
	c.mutex.Lock()
	c.isActive = false
	// Clear data to free memory
	c.connections = make([]models.Connection, 0)
	c.selectedConnID = ""
	c.mutex.Unlock()

	// Stop auto-refresh
	c.stopAutoRefresh()
}

// setupLayout sets up the connections page layout
func (c *ConnectionsPage) setupLayout() {
	// Create components
	c.createConnectionsTable()
	c.createStatusText()
	c.createInfoPanel()

	// Create right panel (info panel + status)
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(c.infoPanel, 0, 2, false)
	rightPanel.AddItem(c.statusText, 6, 0, false)

	// Main layout
	c.SetDirection(tview.FlexColumn)
	c.AddItem(c.connectionsTable, 0, 3, true)
	c.AddItem(rightPanel, 40, 0, false)

	c.SetBorder(true)
	c.SetTitle(" 连接管理 ")
}

// createConnectionsTable creates the connections table
func (c *ConnectionsPage) createConnectionsTable() {
	c.connectionsTable = tview.NewTable().SetFixed(1, 0)
	c.connectionsTable.SetBorder(true)
	c.connectionsTable.SetTitle(" 活跃连接 ")
	c.connectionsTable.SetSelectable(true, false)

	// Set table headers
	headers := []string{"ID", "网络", "源地址", "目标地址", "代理链", "规则", "上传", "下载", "持续时间"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		c.connectionsTable.SetCell(0, i, cell)
	}
}

// createStatusText creates the status display
func (c *ConnectionsPage) createStatusText() {
	c.statusText = tview.NewTextView()
	c.statusText.SetBorder(true)
	c.statusText.SetTitle(" 状态 ")
	c.statusText.SetDynamicColors(true)
	c.statusText.SetText("初始化中...")
}

// createInfoPanel creates the connection info panel
func (c *ConnectionsPage) createInfoPanel() {
	c.infoPanel = tview.NewTextView()
	c.infoPanel.SetBorder(true)
	c.infoPanel.SetTitle(" 连接详情 ")
	c.infoPanel.SetDynamicColors(true)
	c.infoPanel.SetWordWrap(true)
	c.infoPanel.SetText("选择一个连接查看详细信息")
}

// setupEventHandlers sets up event handlers
func (c *ConnectionsPage) setupEventHandlers() {
	// Connections table selection handler
	c.connectionsTable.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 { // Skip header row
			c.mutex.RLock()
			if row-1 < len(c.connections) {
				connection := c.connections[row-1]
				c.selectedConnID = connection.ID
				c.updateInfoPanel(connection)
			}
			c.mutex.RUnlock()
		}
	})

	// Connections table input handler
	c.connectionsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDelete:
			c.closeSelectedConnection()
			return nil
		case tcell.KeyF5:
			c.refresh()
			return nil
		case tcell.KeyCtrlR:
			c.refresh()
			return nil
		}

		switch event.Rune() {
		case 'd', 'D':
			c.closeSelectedConnection()
			return nil
		case 'r', 'R':
			c.refresh()
			return nil
		case 't', 'T':
			c.toggleAutoRefresh()
			return nil
		}

		return event
	})
}

// loadConnectionsData loads data from /connections API
func (c *ConnectionsPage) loadConnectionsData() {
	// Check if page is still active
	c.mutex.RLock()
	isActive := c.isActive
	c.mutex.RUnlock()

	if !isActive {
		return // Page is not active
	}

	connections, err := api.Client.GetConnections()
	if err != nil {
		c.showError(fmt.Sprintf("获取连接数据失败: %v", err))
		return
	}

	c.mutex.Lock()
	// Double check if we're still active after API call
	if c.isActive {
		c.connections = connections
		c.lastUpdate = time.Now()
		c.refreshCount++
	}
	c.mutex.Unlock()
}

// updateConnectionsTable updates the connections table
func (c *ConnectionsPage) updateConnectionsTable() {
	c.mutex.RLock()
	connections := c.connections
	c.mutex.RUnlock()

	// Clear existing rows (except header)
	rowCount := c.connectionsTable.GetRowCount()
	for i := rowCount - 1; i > 0; i-- {
		c.connectionsTable.RemoveRow(i)
	}

	// Add connection rows
	for i, conn := range connections {
		row := i + 1

		// Truncate long IDs for display
		idDisplay := conn.ID
		if len(idDisplay) > 8 {
			idDisplay = idDisplay[:8] + "..."
		}

		// Format source address
		sourceAddr := fmt.Sprintf("%s:%s", conn.Metadata.SourceIP, conn.Metadata.SourcePort)

		// Format destination address
		destAddr := fmt.Sprintf("%s:%s", conn.Metadata.DestinationIP, conn.Metadata.DestinationPort)
		if conn.Metadata.Host != "" {
			destAddr = fmt.Sprintf("%s:%s", conn.Metadata.Host, conn.Metadata.DestinationPort)
		}

		// Format proxy chains
		chains := "DIRECT"
		if len(conn.Chains) > 0 {
			chains = strings.Join(conn.Chains, " → ")
		}

		// Format rule
		rule := conn.Rule
		if rule == "" {
			rule = "DIRECT"
		}

		// Format upload/download
		upload := c.formatBytes(conn.Upload)
		download := c.formatBytes(conn.Download)

		// Calculate duration
		duration := time.Since(conn.Start).Truncate(time.Second).String()

		// Set cell data
		// Limit destAddr to 20 characters for display
		if len(destAddr) > 20 {
			destAddr = destAddr[:17] + "..."
		}

		cells := []struct {
			text  string
			color tcell.Color
			align int
		}{
			{idDisplay, tcell.ColorWhite, tview.AlignLeft},
			{conn.Metadata.Network, tcell.ColorLightBlue, tview.AlignCenter},
			{sourceAddr, tcell.ColorBlue, tview.AlignLeft},
			{destAddr, tcell.ColorGreen, tview.AlignLeft},
			{chains, tcell.ColorYellow, tview.AlignLeft},
			{rule, tcell.ColorPurple, tview.AlignLeft},
			{upload, tcell.ColorRed, tview.AlignRight},
			{download, tcell.ColorGreen, tview.AlignRight},
			{duration, tcell.ColorGray, tview.AlignRight},
		}

		for j, cellData := range cells {
			cell := tview.NewTableCell(cellData.text).
				SetTextColor(cellData.color).
				SetAlign(cellData.align)
			cell.SetReference(conn.ID) // Store full ID in reference
			c.connectionsTable.SetCell(row, j, cell)
		}
	}

	// Update status
	c.updateStatus()

	// Select first connection if none selected and connections exist
	if len(connections) > 0 {
		row, _ := c.connectionsTable.GetSelection()
		if row == 0 {
			c.connectionsTable.Select(1, 0)
		}
	}
}

// updateInfoPanel updates the connection info panel
func (c *ConnectionsPage) updateInfoPanel(conn models.Connection) {
	info := fmt.Sprintf(
		`[yellow]ID:[white] %s
[yellow]网络:[white] %s (%s)
[yellow]源:[white] %s:%s
[yellow]目标:[white] %s:%s (%s)
[yellow]代理链:[white] %s
[yellow]规则:[white] %s (%s)
[yellow]上传/下载:[white] %s / %s (总: %s)
[yellow]开始:[white] %s
[yellow]持续:[white] %s
[yellow]DNS:[white] %s
[yellow]进程:[white] %s
[yellow]特殊代理:[white] %s`,
		conn.ID,
		conn.Metadata.Network, conn.Metadata.Type,
		conn.Metadata.SourceIP, conn.Metadata.SourcePort,
		conn.Metadata.DestinationIP, conn.Metadata.DestinationPort, c.safeString(conn.Metadata.Host, "未知"),
		c.formatChains(conn.Chains),
		c.safeString(conn.Rule, "DIRECT"), c.safeString(conn.RulePayload, "无"),
		c.formatBytes(conn.Upload), c.formatBytes(conn.Download), c.formatBytes(conn.Upload+conn.Download),
		conn.Start.Format("2006-01-02 15:04:05"),
		time.Since(conn.Start).Truncate(time.Second).String(),
		c.safeString(conn.Metadata.DNSMode, "未知"),
		c.safeString(conn.Metadata.ProcessPath, "未知"),
		c.safeString(conn.Metadata.SpecialProxy, "无"),
	)

	go ui.Updater.UpdateUi(func() {
		c.infoPanel.SetText(info)
	})
}

// formatBytes formats byte count to human readable format
func (c *ConnectionsPage) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatChains formats proxy chains for display
func (c *ConnectionsPage) formatChains(chains []string) string {
	if len(chains) == 0 {
		return "DIRECT"
	}
	return strings.Join(chains, " → ")
}

// safeString returns defaultValue if s is empty
func (c *ConnectionsPage) safeString(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

// updateStatus updates the status text
func (c *ConnectionsPage) updateStatus() {
	c.mutex.RLock()
	connCount := len(c.connections)
	lastUpdate := c.lastUpdate
	refreshCount := c.refreshCount
	autoRefresh := c.autoRefresh
	c.mutex.RUnlock()

	autoRefreshIcon := "⏸"
	if autoRefresh {
		autoRefreshIcon = "▶"
	}

	status := fmt.Sprintf(`[green]● %d[white] 连接 | [yellow]%s[white] 更新:%d | [cyan]%s[white] 自动刷新

[gray]快捷键:[white]
[yellow]F5/R[white] 刷新 [yellow]T[white] 自动 [yellow]D/Del[white] 关闭连接
[yellow]↑↓[white] 选择连接`,
		connCount,
		lastUpdate.Format("15:04"),
		refreshCount,
		autoRefreshIcon,
	)

	go ui.Updater.UpdateUi(func() {
		c.statusText.SetText(status)
	})
}

// closeSelectedConnection closes the selected connection
func (c *ConnectionsPage) closeSelectedConnection() {
	if c.selectedConnID == "" {
		c.showError("请先选择一个连接")
		return
	}

	c.showInfo(fmt.Sprintf("正在关闭连接: %s", c.selectedConnID[:8]))

	go func() {
		// Check if we're still active before making API call
		c.mutex.RLock()
		isActive := c.isActive
		c.mutex.RUnlock()

		if !isActive {
			return
		}

		err := api.Client.CloseConnection(c.selectedConnID)
		if err != nil {
			c.showError(fmt.Sprintf("关闭连接失败: %v", err))
			return
		}

		c.showSuccess("连接已关闭")

		// Refresh data to remove closed connection
		time.Sleep(500 * time.Millisecond)

		// Check again if we're still active
		c.mutex.RLock()
		isActive = c.isActive
		c.mutex.RUnlock()

		if isActive {
			c.loadConnectionsData()
			go ui.Updater.UpdateUi(func() {
				c.updateConnectionsTable()
			})
		}
	}()
}

// refresh manually refreshes the connections data
func (c *ConnectionsPage) refresh() {
	// Check if we're still active
	c.mutex.RLock()
	isActive := c.isActive
	c.mutex.RUnlock()

	if !isActive {
		return
	}

	c.showInfo("正在刷新连接数据...")
	c.loadConnectionsData()

	go ui.Updater.UpdateUi(func() {
		c.updateConnectionsTable()
	})
}

// toggleAutoRefresh toggles auto-refresh on/off
func (c *ConnectionsPage) toggleAutoRefresh() {
	c.mutex.Lock()
	c.autoRefresh = !c.autoRefresh
	autoRefresh := c.autoRefresh
	c.mutex.Unlock()

	if autoRefresh {
		c.startAutoRefresh()
		c.showSuccess("自动刷新已开启")
	} else {
		c.stopAutoRefresh()
		c.showSuccess("自动刷新已关闭")
	}

	c.updateStatus()
}

// startAutoRefresh starts the auto-refresh goroutine
func (c *ConnectionsPage) startAutoRefresh() {
	c.stopAutoRefresh() // Stop existing refresh if any

	c.mutex.Lock()
	isActive := c.isActive
	c.mutex.Unlock()

	if !isActive {
		return // Page is not active
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.refreshCancel = cancel
	c.refreshTicker = time.NewTicker(2 * time.Second) // Refresh every 2 seconds

	go func() {
		defer c.refreshTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.refreshTicker.C:
				c.mutex.RLock()
				isActive := c.isActive
				autoRefresh := c.autoRefresh
				c.mutex.RUnlock()

				if !isActive || !autoRefresh {
					return
				}

				c.loadConnectionsData()
				go ui.Updater.UpdateUi(func() {
					c.updateConnectionsTable()
				})
			}
		}
	}()
}

// stopAutoRefresh stops the auto-refresh goroutine
func (c *ConnectionsPage) stopAutoRefresh() {
	if c.refreshCancel != nil {
		c.refreshCancel()
		c.refreshCancel = nil
	}
	if c.refreshTicker != nil {
		c.refreshTicker.Stop()
		c.refreshTicker = nil
	}
}

// showError shows an error message
func (c *ConnectionsPage) showError(message string) {
	log.Printf("Error: %s", message)
	// We'll update status instead of having a separate error display
}

// showSuccess shows a success message
func (c *ConnectionsPage) showSuccess(message string) {
	log.Printf("Success: %s", message)
	// We'll update status instead of having a separate success display
}

// showInfo shows an info message
func (c *ConnectionsPage) showInfo(message string) {
	log.Printf("Info: %s", message)
	// We'll update status instead of having a separate info display
}

// GetInputCapture returns the input capture function for keyboard shortcuts
func (c *ConnectionsPage) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF5:
			c.refresh()
			return nil
		case tcell.KeyCtrlR:
			c.refresh()
			return nil
		case tcell.KeyDelete:
			c.closeSelectedConnection()
			return nil
		}

		switch event.Rune() {
		case 'r', 'R':
			c.refresh()
			return nil
		case 'd', 'D':
			c.closeSelectedConnection()
			return nil
		case 't', 'T':
			c.toggleAutoRefresh()
			return nil
		}

		return event
	}
}
