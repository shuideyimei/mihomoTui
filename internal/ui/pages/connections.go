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

// ConnectionsPage represents the connections management page
type ConnectionsPage struct {
	*tview.Flex

	// Components
	pages            *tview.Pages
	mainFlex         *tview.Flex
	connectionsTable *tview.Table
	statusText       *tview.TextView
	infoPanel        *tview.TextView
	searchInput      *tview.InputField
	filterText       string

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
		pages:       tview.NewPages(),
		connections: make([]models.Connection, 0),
		autoRefresh: true,
	}

	page.setupLayout()
	page.setupEventHandlers()

	return page
}

// Activate initializes the page when it becomes active
func (c *ConnectionsPage) Activate() {
	c.Deactivate()

	c.mutex.Lock()
	c.isActive = true
	c.mutex.Unlock()

	// Load initial data
	c.loadConnectionsData()

	// Start auto-refresh
	c.startAutoRefresh()

	ui.Updater.PostUi(func() {
		c.statusText.SetText(i18n.T("conn.title"))
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

// UpdateTexts refreshes all localized labels and titles on the connections page
func (c *ConnectionsPage) UpdateTexts() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.mainFlex.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.title")))
	if c.searchInput != nil {
		c.searchInput.SetLabel(" " + i18n.T("rule.search_label"))
	}
	if c.connectionsTable != nil {
		c.connectionsTable.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.active_conn")))
		headers := []string{
			i18n.T("conn.col_id"), i18n.T("conn.col_network"), i18n.T("conn.col_source"),
			i18n.T("conn.col_dest"), i18n.T("conn.col_chain"), i18n.T("conn.col_rule"),
			i18n.T("conn.col_upload"), i18n.T("conn.col_download"), i18n.T("conn.col_duration"),
		}
		for i, header := range headers {
			cell := tview.NewTableCell(header).
				SetTextColor(tcell.ColorGray).
				SetAlign(tview.AlignCenter).
				SetSelectable(false)
			c.connectionsTable.SetCell(0, i, cell)
		}
	}
	if c.infoPanel != nil {
		c.infoPanel.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.detail_title")))
	}
	c.updateStatus()
}

// setupLayout sets up the connections page layout
func (c *ConnectionsPage) setupLayout() {
	// Create components
	c.createSearchInput()
	c.createConnectionsTable()
	c.createStatusText()
	c.createInfoPanel()

	// Create left panel (search + table)
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(c.searchInput, 1, 0, false)
	leftPanel.AddItem(c.connectionsTable, 0, 1, true)

	// Create right panel (info panel + status)
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(c.infoPanel, 0, 2, false)
	rightPanel.AddItem(c.statusText, 1, 0, false)

	// Main layout
	c.mainFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	c.mainFlex.AddItem(components.NewResponsiveSplit(leftPanel, rightPanel, 110, 3, 2), 0, 1, true)

	c.mainFlex.SetBorder(false)
	c.mainFlex.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.title")))

	c.pages.AddPage("main", c.mainFlex, true, true)
	c.AddItem(c.pages, 0, 1, true)
}

func (c *ConnectionsPage) createSearchInput() {
	c.searchInput = tview.NewInputField().
		SetLabel(" " + i18n.T("rule.search_label")).
		SetFieldWidth(0)
	c.searchInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	c.searchInput.SetChangedFunc(func(text string) {
		c.mutex.Lock()
		c.filterText = strings.TrimSpace(text)
		c.mutex.Unlock()
		c.updateConnectionsTable()
	})
	c.searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			ui.Updater.SetFocus(c.connectionsTable)
			return nil
		}
		return event
	})
}

// createConnectionsTable creates the connections table
func (c *ConnectionsPage) createConnectionsTable() {
	c.connectionsTable = tview.NewTable().SetFixed(1, 0)
	c.connectionsTable.SetBorder(true)
	c.connectionsTable.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.active_conn")))
	components.StyleFocusBorder(c.connectionsTable)
	c.connectionsTable.SetSelectable(true, false)

	// Set table headers
	headers := []string{
		i18n.T("conn.col_id"), i18n.T("conn.col_network"), i18n.T("conn.col_source"),
		i18n.T("conn.col_dest"), i18n.T("conn.col_chain"), i18n.T("conn.col_rule"),
		i18n.T("conn.col_upload"), i18n.T("conn.col_download"), i18n.T("conn.col_duration"),
	}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		c.connectionsTable.SetCell(0, i, cell)
	}
}

// createStatusText creates the status display
func (c *ConnectionsPage) createStatusText() {
	c.statusText = tview.NewTextView()
	components.StyleStatusLine(c.statusText)
	c.statusText.SetText(i18n.T("conn.initializing"))
}

// createInfoPanel creates the connection info panel
func (c *ConnectionsPage) createInfoPanel() {
	c.infoPanel = tview.NewTextView()
	c.infoPanel.SetBorder(true)
	c.infoPanel.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.detail_title")))
	c.infoPanel.SetDynamicColors(true)
	c.infoPanel.SetWordWrap(true)
	c.infoPanel.SetText(i18n.T("conn.select_hint"))
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
		case tcell.KeyEnter:
			c.showConnectionDetails()
			return nil
		case tcell.KeyDelete:
			c.closeSelectedConnection()
			return nil
		case tcell.KeyF5, tcell.KeyCtrlR:
			c.refresh()
			return nil
		case tcell.KeyCtrlX:
			c.confirmCloseAllConnections()
			return nil
		}

		switch event.Rune() {
		case '/':
			ui.Updater.SetFocus(c.searchInput)
			return nil
		case 'x', 'X':
			c.confirmCloseAllConnections()
			return nil
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

func (c *ConnectionsPage) confirmCloseAllConnections() {
	ui.Updater.ShowConfirm(i18n.T("conn.title"), i18n.T("conn.close_all_confirm"), func() {
		go c.closeAllConnections()
	})
}

func (c *ConnectionsPage) closeAllConnections() {
	c.mutex.RLock()
	conns := make([]models.Connection, len(c.connections))
	copy(conns, c.connections)
	c.mutex.RUnlock()

	closed := 0
	for _, conn := range conns {
		if err := api.Client.CloseConnection(conn.ID); err == nil {
			closed++
		}
	}

	ui.Updater.PostUi(func() {
		c.statusText.SetText(fmt.Sprintf("[green]"+i18n.T("conn.close_all_done")+"[white]", closed))
		c.refresh()
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
		ui.Updater.PostUi(func() {
			c.statusText.SetText(fmt.Sprintf(i18n.T("dash.conn_fetch_fail"), err))
		})
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
	var connections []models.Connection
	filter := strings.ToLower(c.filterText)
	for _, conn := range c.connections {
		if filter != "" {
			target := strings.ToLower(conn.Metadata.Host + " " + conn.Metadata.DestinationIP + " " + conn.Rule + " " + strings.Join(conn.Chains, " ") + " " + conn.Metadata.ProcessPath)
			if !strings.Contains(target, filter) {
				continue
			}
		}
		connections = append(connections, conn)
	}
	c.mutex.RUnlock()

	rowCount := c.connectionsTable.GetRowCount()

	// In-place update when the row count and per-row IDs match the new data.
	// This avoids the O(n²) full rebuild on every 2-second refresh ticker.
	inPlace := rowCount-1 == len(connections)
	if inPlace {
		for i := 0; i < len(connections); i++ {
			cell := c.connectionsTable.GetCell(i+1, 0)
			if cell == nil {
				inPlace = false
				break
			}
			ref, _ := cell.GetReference().(string)
			if ref != connections[i].ID {
				inPlace = false
				break
			}
		}
	}

	if !inPlace {
		// Clear existing rows (except header)
		for i := c.connectionsTable.GetRowCount() - 1; i > 0; i-- {
			c.connectionsTable.RemoveRow(i)
		}
	}

	// Fill rows (SetCell updates existing rows or appends new ones)
	row := 1
	for _, conn := range connections {
		c.setConnectionRow(row, conn)
		row++
	}

	// Trim surplus rows left over from a previous larger connection set
	for i := c.connectionsTable.GetRowCount() - 1; i >= row; i-- {
		c.connectionsTable.RemoveRow(i)
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

// setConnectionRow writes one connection into the given table row.
func (c *ConnectionsPage) setConnectionRow(row int, conn models.Connection) {
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
		chains = ui.StripFlagEmoji(chains)
	}

	// Format rule
	rule := conn.Rule
	if rule == "" {
		rule = "DIRECT"
	}

	// Format upload/download
	upload := utils.FormatBytes(conn.Upload)
	download := utils.FormatBytes(conn.Download)

	// Calculate duration
	duration := time.Since(conn.Start).Truncate(time.Second).String()

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
		{conn.Metadata.Network, tcell.ColorWhite, tview.AlignCenter},
		{sourceAddr, tcell.ColorGray, tview.AlignLeft},
		{destAddr, tcell.ColorWhite, tview.AlignLeft},
		{chains, tcell.ColorGray, tview.AlignLeft},
		{rule, tcell.ColorGray, tview.AlignLeft},
		{upload, tcell.ColorWhite, tview.AlignRight},
		{download, tcell.ColorWhite, tview.AlignRight},
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

// updateInfoPanel updates the connection info panel
func (c *ConnectionsPage) updateInfoPanel(conn models.Connection) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_id"), conn.ID))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_network"), conn.Metadata.Network, conn.Metadata.Type))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_source"), conn.Metadata.SourceIP, conn.Metadata.SourcePort))
	b.WriteString("\n")
	destAddr := fmt.Sprintf("%s:%s", conn.Metadata.DestinationIP, conn.Metadata.DestinationPort)
	if conn.Metadata.Host != "" {
		destAddr = fmt.Sprintf("%s:%s", conn.Metadata.Host, conn.Metadata.DestinationPort)
	}
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_dest"), destAddr))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_chain"), c.formatChains(conn.Chains)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_rule"), c.safeString(conn.Rule, "DIRECT"), c.safeString(conn.RulePayload, i18n.T("conn.none"))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_updown"), utils.FormatBytes(conn.Upload), utils.FormatBytes(conn.Download), utils.FormatBytes(conn.Upload+conn.Download)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_start"), conn.Start.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_duration"), time.Since(conn.Start).Truncate(time.Second).String()))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_dns"), c.safeString(conn.Metadata.DNSMode, i18n.T("conn.unknown"))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_process"), c.safeString(conn.Metadata.ProcessPath, i18n.T("conn.unknown"))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(i18n.T("conn.info_special"), ui.StripFlagEmoji(c.safeString(conn.Metadata.SpecialProxy, i18n.T("conn.none")))))

	info := b.String()
	ui.Updater.PostUi(func() {
		c.infoPanel.SetText(info)
	})
}

func (c *ConnectionsPage) showConnectionDetails() {
	c.mutex.RLock()
	connID := c.selectedConnID
	connections := c.connections
	c.mutex.RUnlock()

	if connID == "" || len(connections) == 0 {
		return
	}

	var conn *models.Connection
	for i := range connections {
		if connections[i].ID == connID {
			conn = &connections[i]
			break
		}
	}
	if conn == nil {
		return
	}

	detailFlex := c.createDetailView(conn)
	c.pages.AddPage("detail", detailFlex, true, true)
	c.pages.SwitchToPage("detail")
	ui.Updater.SetFocus(detailFlex)
}

func (c *ConnectionsPage) createDetailView(conn *models.Connection) *tview.Flex {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_id"), conn.ID))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_source"), conn.Metadata.SourceIP, conn.Metadata.SourcePort))
	destAddr := fmt.Sprintf("%s:%s", conn.Metadata.DestinationIP, conn.Metadata.DestinationPort)
	if conn.Metadata.Host != "" {
		destAddr = fmt.Sprintf("%s:%s", conn.Metadata.Host, conn.Metadata.DestinationPort)
	}
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_dest"), destAddr))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_host"), c.safeString(conn.Metadata.Host, i18n.T("conn.none"))))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_upload"), utils.FormatBytes(conn.Upload)))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_download"), utils.FormatBytes(conn.Download)))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_total"), utils.FormatBytes(conn.Upload+conn.Download)))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_start"), conn.Start.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_duration"), time.Since(conn.Start).Truncate(time.Second).String()))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_chain"), c.formatChains(conn.Chains)))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_rule"), conn.Rule))
	if conn.RulePayload != "" {
		b.WriteString(fmt.Sprintf(" (%s)", conn.RulePayload))
	}
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_process"), c.safeString(conn.Metadata.ProcessPath, i18n.T("conn.unknown"))))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_network_type"), conn.Metadata.Network))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_conn_type"), conn.Metadata.Type))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_dns_mode"), c.safeString(conn.Metadata.DNSMode, i18n.T("conn.unknown"))))
	b.WriteString(fmt.Sprintf(i18n.T("conn.detail_special_proxy"), ui.StripFlagEmoji(c.safeString(conn.Metadata.SpecialProxy, i18n.T("conn.none")))))
	b.WriteString(i18n.T("conn.detail_back"))

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetText(b.String())

	closeBtn := components.NewButton(i18n.T("conn.close_btn"), components.ButtonNormal, func() {
		c.pages.RemovePage("detail")
		c.pages.SwitchToPage("main")
		ui.Updater.SetFocus(c.connectionsTable)
	})

	popupFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	popupFlex.SetBorder(true)
	popupFlex.SetTitle(fmt.Sprintf(" %s ", i18n.T("conn.detail_title")))
	popupFlex.AddItem(textView, 0, 1, false)
	popupFlex.AddItem(closeBtn, 1, 0, true)

	popupFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			c.pages.RemovePage("detail")
			c.pages.SwitchToPage("main")
			ui.Updater.SetFocus(c.connectionsTable)
			return nil
		}
		return event
	})

	hFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(popupFlex, 0, 3, false).
		AddItem(nil, 0, 1, false)

	vFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(hFlex, 0, 3, false).
		AddItem(nil, 0, 1, false)

	return vFlex
}

// formatChains formats proxy chains for display
func (c *ConnectionsPage) formatChains(chains []string) string {
	if len(chains) == 0 {
		return "DIRECT"
	}
	return ui.StripFlagEmoji(strings.Join(chains, " → "))
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
	c.mutex.RUnlock()

	status := fmt.Sprintf(i18n.T("conn.status_summary"),
		connCount,
		lastUpdate.Format("15:04"),
		refreshCount,
	)

	c.statusText.SetText(status)
}

// closeSelectedConnection closes the selected connection
func (c *ConnectionsPage) closeSelectedConnection() {
	if c.selectedConnID == "" {
		c.showError(i18n.T("conn.select_first"))
		return
	}

	c.showInfo(fmt.Sprintf(i18n.T("conn.closing"), c.selectedConnID[:8]))

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
			c.showError(fmt.Sprintf(i18n.T("conn.close_failed"), err))
			return
		}

		c.showSuccess(i18n.T("conn.closed"))

		// Refresh data to remove closed connection
		time.Sleep(500 * time.Millisecond)

		// Check again if we're still active
		c.mutex.RLock()
		isActive = c.isActive
		c.mutex.RUnlock()

		if isActive {
			c.loadConnectionsData()
			ui.Updater.PostUi(func() {
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

	c.showInfo(i18n.T("conn.refreshing"))
	c.loadConnectionsData()

	ui.Updater.PostUi(func() {
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
		c.showSuccess(i18n.T("conn.auto_refresh_on"))
	} else {
		c.stopAutoRefresh()
		c.showSuccess(i18n.T("conn.auto_refresh_off"))
	}

	c.updateStatus()
}

// startAutoRefresh starts the auto-refresh goroutine
func (c *ConnectionsPage) startAutoRefresh() {
	c.stopAutoRefresh() // Stop existing refresh if any

	c.mutex.Lock()
	if !c.isActive || !c.autoRefresh {
		c.mutex.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.refreshCancel = cancel
	ticker := time.NewTicker(2 * time.Second) // Refresh every 2 seconds
	c.refreshTicker = ticker
	c.mutex.Unlock()

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mutex.RLock()
				isActive := c.isActive
				autoRefresh := c.autoRefresh
				c.mutex.RUnlock()

				if !isActive || !autoRefresh || ctx.Err() != nil {
					return
				}

				c.loadConnectionsData()
				if ctx.Err() != nil {
					return
				}
				ui.Updater.PostUi(func() {
					c.updateConnectionsTable()
				})
			}
		}
	}()
}

// stopAutoRefresh stops the auto-refresh goroutine
func (c *ConnectionsPage) stopAutoRefresh() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.refreshCancel != nil {
		c.refreshCancel()
		c.refreshCancel = nil
	}
	if c.refreshTicker != nil {
		c.refreshTicker.Stop()
		c.refreshTicker = nil
	}
}

// Stop stops the connections page
func (c *ConnectionsPage) Stop() {
	c.Deactivate()
}

// showError shows an error message in the status area
func (c *ConnectionsPage) showError(message string) {
	log.Printf("Error: %s", message)
	ui.Updater.PostUi(func() {
		c.statusText.SetText(fmt.Sprintf("[red]%s", message))
	})
}

// showSuccess shows a success message in the status area
func (c *ConnectionsPage) showSuccess(message string) {
	log.Printf("Success: %s", message)
	ui.Updater.PostUi(func() {
		c.statusText.SetText(fmt.Sprintf("[green]%s", message))
	})
}

// showInfo shows an info message in the status area
func (c *ConnectionsPage) showInfo(message string) {
	log.Printf("Info: %s", message)
	ui.Updater.PostUi(func() {
		c.statusText.SetText(fmt.Sprintf("[yellow]%s", message))
	})
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
