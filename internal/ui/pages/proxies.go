package pages

import (
	"context"
	"fmt"
	"log"
	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Proxies represents the proxies page
type Proxies struct {
	*ProxiesPage
}

// Activate activates the proxies page
func (p *Proxies) Activate() {
	log.Printf("Activating proxies page")
	p.ProxiesPage.Activate()
}

// Deactivate deactivates the proxies page
func (p *Proxies) Deactivate() {
	log.Printf("Deactivating proxies page")
	p.ProxiesPage.Deactivate()
}

// ProxiesPage represents the proxies management page
type ProxiesPage struct {
	*tview.Flex

	// Components
	groupsList    *tview.List
	switchButtons *tview.Flex
	nodesList     *tview.Table
	statusText    *tview.TextView // Simple status display

	// Button references for mode switching
	ruleBtn   *tview.Button
	globalBtn *tview.Button
	directBtn *tview.Button

	// Data
	providersData map[string]*models.ProxyProvider
	proxiesData   map[string]*models.Proxy // Flat proxy data from /proxies
	proxyGroups   []*models.Proxy          // Extracted groups
	selectedGroup string
	selectedNode  string
	currentMode   string
	groupDelays   map[string]map[string]int // group name -> node name -> delay

	// Control
	cancel          context.CancelFunc
	autoRefreshCtx  context.Context
	autoRefreshStop context.CancelFunc
	mutex           sync.RWMutex

	// State
	isActive       bool
	isTestingDelay bool
	lastUpdate     time.Time

	// Navigation
	focusableComponents []tview.Primitive
	currentFocusIndex   int
	switchButtonIndex   int
	focusableButtons    []*tview.Button
}

// NewProxiesPage creates a new proxies management page
func NewProxiesPage() *ProxiesPage {
	page := &ProxiesPage{
		Flex:          tview.NewFlex(),
		providersData: make(map[string]*models.ProxyProvider),
		proxiesData:   make(map[string]*models.Proxy),
		currentMode:   "rule", // Default mode
	}

	page.setupLayout()
	page.setupEventHandlers()

	return page
}

// Activate initializes the page when it becomes active
func (p *ProxiesPage) Activate() {
	p.mutex.Lock()
	p.isActive = true
	p.mutex.Unlock()

	// Load data and start auto-refresh in background
	go func() {
		p.loadProvidersData()
		ui.Updater.UpdateUi(func() {
			p.updateGroupsList()
			p.statusText.SetText("加载完成")
		})
	}()

	// Start auto-refresh
	p.startAutoRefresh()

	// Sync mode state from StatusBar
	go p.syncModeFromStatusBar()
}

// Deactivate deactivates the proxies page
func (p *ProxiesPage) Deactivate() {
	p.mutex.Lock()
	p.isActive = false
	p.mutex.Unlock()

	// Stop auto-refresh
	p.stopAutoRefresh()

	// Cancel any ongoing operations
	if p.cancel != nil {
		p.cancel()
	}
}

// setupLayout sets up the proxies page layout
func (p *ProxiesPage) setupLayout() {
	// Create components
	p.createGroupsList()
	p.createSwitchButtons()
	p.createNodesList()
	p.createStatusText()

	// Initialize navigation system
	p.initializeNavigation()

	// Create left panel (groups list + switch buttons + status)
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.groupsList, 0, 3, true)
	leftPanel.AddItem(p.switchButtons, 3, 0, false)
	leftPanel.AddItem(p.statusText, 3, 0, false)

	// Main layout
	p.SetDirection(tview.FlexColumn)
	p.AddItem(leftPanel, 0, 1, true)
	p.AddItem(p.nodesList, 0, 2, false)

	p.SetBorder(true)
	p.SetTitle(" 代理管理 ")
}

// createStatusText creates the status display
func (p *ProxiesPage) createStatusText() {
	p.statusText = tview.NewTextView()
	p.statusText.SetBorder(true)
	p.statusText.SetTitle(" 状态 ")
	p.statusText.SetDynamicColors(true)
	p.statusText.SetText("加载中...")
}

// createGroupsList creates the proxy groups list
func (p *ProxiesPage) createGroupsList() {
	p.groupsList = tview.NewList()
	p.groupsList.SetBorder(true)
	p.groupsList.SetTitle(" 代理组 ")
}

// createSwitchButtons creates the mode switch buttons
func (p *ProxiesPage) createSwitchButtons() {
	p.switchButtons = tview.NewFlex()
	p.switchButtons.SetBorder(true)
	p.switchButtons.SetTitle(" 模式切换 ")

	// Create buttons for rule, global, direct modes
	p.ruleBtn = tview.NewButton("Rule")
	p.globalBtn = tview.NewButton("Global")
	p.directBtn = tview.NewButton("Direct")

	// Initialize button navigation
	p.focusableButtons = []*tview.Button{p.ruleBtn, p.globalBtn, p.directBtn}
	p.switchButtonIndex = 0

	// Set initial button styles
	p.updateButtonStyles()

	// Add button click handlers for mode switching
	p.ruleBtn.SetSelectedFunc(func() {
		p.switchMode("rule")
	})
	p.globalBtn.SetSelectedFunc(func() {
		p.switchMode("global")
	})
	p.directBtn.SetSelectedFunc(func() {
		p.switchMode("direct")
	})

	p.switchButtons.AddItem(p.ruleBtn, 0, 1, false)
	p.switchButtons.AddItem(p.globalBtn, 0, 1, false)
	p.switchButtons.AddItem(p.directBtn, 0, 1, false)

	// Set up input capture for button navigation
	p.switchButtons.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			p.switchToPrevButton()
			return nil
		case tcell.KeyRight:
			p.switchToNextButton()
			return nil
		}
		return event
	})
}

// updateButtonStyles updates button styles based on current mode
func (p *ProxiesPage) updateButtonStyles() {
	// Reset all buttons to gray
	style := tcell.StyleDefault.Background(tcell.ColorGray).Foreground(tcell.ColorWhite)
	activatedStyle := tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorWhite)
	p.ruleBtn.SetStyle(style)
	p.globalBtn.SetStyle(style)
	p.directBtn.SetStyle(style)

	// Highlight current mode
	switch p.currentMode {
	case "rule":
		p.ruleBtn.SetStyle(activatedStyle)
	case "global":
		p.globalBtn.SetStyle(activatedStyle)
	case "direct":
		p.directBtn.SetStyle(activatedStyle)
	}
}

// syncModeFromStatusBar synchronizes mode state from StatusBar
func (p *ProxiesPage) syncModeFromStatusBar() {
	currentMode := ui.Updater.GetCurrentMode()
	if currentMode != "" && currentMode != "Unknown" {
		// Convert mode to lowercase to match our internal format
		switch currentMode {
		case "Rule":
			p.currentMode = "rule"
		case "Global":
			p.currentMode = "global"
		case "Direct":
			p.currentMode = "direct"
		default:
			p.currentMode = strings.ToLower(currentMode)
		}
		// Update button highlights
		ui.Updater.UpdateUi(func() {
			p.updateButtonStyles()
		})
	}
}

// createNodesList creates the proxy nodes table
func (p *ProxiesPage) createNodesList() {
	p.nodesList = tview.NewTable().SetFixed(1, 0)
	p.nodesList.SetBorder(true)
	p.nodesList.SetTitle(" 代理节点 ")
	p.nodesList.SetSelectable(true, false)

	// Set table headers
	headers := []string{"节点名称", "类型", "延迟", "状态"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		p.nodesList.SetCell(0, i, cell)
	}
}

// setupEventHandlers sets up event handlers
func (p *ProxiesPage) setupEventHandlers() {
	// Groups list selection handler
	p.groupsList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		p.selectedGroup = extractGroupName(mainText)
		p.updateNodesListContent()
	})

	// Groups list input handler
	p.groupsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			p.focusNodesList()
			return nil
		case tcell.KeyCtrlR:
			p.Refresh()
			return nil
		}
		return event
	})

	// Nodes table selection handler
	p.nodesList.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 { // Skip header row
			cell := p.nodesList.GetCell(row, 0)
			if cell != nil {
				// Get the clean node name from reference or text
				if ref := cell.GetReference(); ref != nil {
					if nodeName, ok := ref.(string); ok {
						p.selectedNode = nodeName
					} else {
						p.selectedNode = strings.TrimPrefix(cell.Text, "✓ ")
					}
				} else {
					p.selectedNode = strings.TrimPrefix(cell.Text, "✓ ")
				}
			}
		}
	})

	// Nodes table input handler
	p.nodesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			p.focusGroupsList()
			return nil
		case tcell.KeyEnter:
			p.selectCurrentNode()
			return nil
		case tcell.KeyCtrlR:
			p.Refresh()
			return nil
		}

		switch event.Rune() {
		case ' ':
			p.testGroupDelay()
			return nil
		case 'r', 'R':
			p.testSelectedNodeDelay()
			return nil
		}

		return event
	})
}

// loadProvidersData loads data from /providers/proxies and /proxies APIs
func (p *ProxiesPage) loadProvidersData() {
	providers, pErr := api.Client.GetProviders()
	proxies, xErr := api.Client.GetProxies()

	p.mutex.Lock()
	if providers != nil {
		p.providersData = providers.Providers
		log.Printf("Loaded %d providers", len(providers.Providers))
	} else if pErr != nil {
		log.Printf("Failed to get providers: %v", pErr)
	}
	if proxies != nil {
		p.proxiesData = proxies
		groupCount := 0
		for _, proxy := range proxies {
			if proxy != nil && (len(proxy.All) > 0 || proxy.Now != "") {
				groupCount++
			}
		}
		log.Printf("Loaded %d proxies (%d groups)", len(proxies), groupCount)
	} else if xErr != nil {
		log.Printf("Failed to get proxies: %v", xErr)
	}
	p.lastUpdate = time.Now()
	p.mutex.Unlock()
}

// extractGroups returns groups from /proxies API only.
func (p *ProxiesPage) extractGroups() []*models.Proxy {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var groups []*models.Proxy
	seen := make(map[string]bool)

	for name, proxy := range p.proxiesData {
		if proxy == nil || name == "" || seen[name] {
			continue
		}
		if len(proxy.All) > 0 || proxy.Now != "" {
			seen[name] = true
			if proxy.Name == "" {
				proxy.Name = name
			}
			groups = append(groups, proxy)
			log.Printf("group %q: from API (type=%s all=%d now=%q)", name, proxy.Type, len(proxy.All), proxy.Now)
		}
	}

	log.Printf("extractGroups result: %d groups", len(groups))
	return groups
}

// sortGroups sorts groups: "Proxy" first, "Auto" second, "GLOBAL" third, rest alphabetically
func (p *ProxiesPage) sortGroups(groups []*models.Proxy) []*models.Proxy {
	sort.SliceStable(groups, func(i, j int) bool {
		ni, nj := groups[i].Name, groups[j].Name
		if ni == "Proxy" {
			return true
		}
		if nj == "Proxy" {
			return false
		}
		if ni == "Auto" {
			return true
		}
		if nj == "Auto" {
			return false
		}
		if ni == "GLOBAL" {
			return true
		}
		if nj == "GLOBAL" {
			return false
		}
		return ni < nj
	})
	return groups
}

// updateGroupsList updates the groups list
func (p *ProxiesPage) updateGroupsList() {
	groups := p.extractGroups()
	groups = p.sortGroups(groups)

	p.mutex.Lock()
	p.proxyGroups = groups
	p.mutex.Unlock()

	p.groupsList.Clear()

	for _, group := range groups {
		secondaryText := ""
		if group.Now != "" {
			secondaryText = fmt.Sprintf("当前: %s", group.Now)
		}

		mainText := fmt.Sprintf("[%s] %s", group.Type, group.Name)
		p.groupsList.AddItem(mainText, secondaryText, 0, nil)
	}

	if len(groups) > 0 {
		p.groupsList.SetCurrentItem(0)
		// Get the selected group name from the groupsList item
		mainText, _ := p.groupsList.GetItemText(0)
		p.selectedGroup = extractGroupName(mainText)
		p.updateNodesListContent()
	}
}

// extractGroupName strips the type prefix from displayed group name
func extractGroupName(mainText string) string {
	idx := strings.Index(mainText, "] ")
	if idx >= 0 {
		return mainText[idx+2:]
	}
	return mainText
}

// updateNodesListContent updates the nodes list content from the selected group's All list
func (p *ProxiesPage) updateNodesListContent() {
	if p.selectedGroup == "" {
		return
	}

	log.Printf("updateNodesListContent: selectedGroup=%q", p.selectedGroup)

	// Find the selected group in proxyGroups
	var selectedGroupProxy *models.Proxy
	p.mutex.RLock()
	for _, g := range p.proxyGroups {
		if g != nil && g.Name == p.selectedGroup {
			selectedGroupProxy = g
			break
		}
	}
	p.mutex.RUnlock()

	if selectedGroupProxy == nil {
		log.Printf("updateNodesListContent: group %q not found in proxyGroups", p.selectedGroup)
		return
	}

	log.Printf("updateNodesListContent: group %q type=%s all=%v now=%q",
		p.selectedGroup, selectedGroupProxy.Type, selectedGroupProxy.All, selectedGroupProxy.Now)

	p.nodesList.Clear()

	// Re-add headers
	headers := []string{"节点名称", "类型", "延迟", "状态"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		p.nodesList.SetCell(0, i, cell)
	}

	allNodes := selectedGroupProxy.All

	// Build node lookup from all providers
	nodeLookup := p.buildNodeLookup()
	log.Printf("updateNodesListContent: nodeLookup has %d entries", len(nodeLookup))

	// If still no nodes, show empty state
	if len(allNodes) == 0 {
		log.Printf("updateNodesListContent: no nodes found for group %q, showing empty state", p.selectedGroup)
		emptyCell := tview.NewTableCell("  (无可用节点)  ").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignCenter)
		p.nodesList.SetCell(1, 0, emptyCell)
		p.nodesList.SetCell(1, 1, tview.NewTableCell("-").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignCenter))
		p.nodesList.SetCell(1, 2, tview.NewTableCell("-").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignCenter))
		p.nodesList.SetCell(1, 3, tview.NewTableCell("-").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignCenter))
		return
	}

	// Display each entry in the group's All list
	for i, name := range allNodes {
		row := i + 1

		// Name cell
		nameText := name
		nameColor := tcell.ColorWhite
		if name == selectedGroupProxy.Now {
			nameText = "✓ " + name
			nameColor = tcell.ColorGreen
		}

		nameCell := tview.NewTableCell(nameText).SetTextColor(nameColor)
		nameCell.SetReference(name)
		p.nodesList.SetCell(row, 0, nameCell)

		// Type cell
		typeText := "-"
		typeColor := tcell.ColorGray
		switch name {
		case "DIRECT":
			typeText = "DIRECT"
			typeColor = tcell.ColorGreen
		case "REJECT", "REJECT-DROP", "REJECT-TLS":
			typeText = name
			typeColor = tcell.ColorRed
		default:
			if node, found := nodeLookup[name]; found {
				typeText = strings.ToUpper(node.Type)
				typeColor = tcell.ColorBlue
			}
		}

		typeCell := tview.NewTableCell(typeText).
			SetTextColor(typeColor).
			SetAlign(tview.AlignCenter)
		p.nodesList.SetCell(row, 1, typeCell)

		// Delay cell
		delayText := "未测试"
		delayColor := tcell.ColorGray

		// Check group delay data first
		if p.groupDelays != nil {
			if gd, ok := p.groupDelays[p.selectedGroup]; ok {
				if delay, ok := gd[name]; ok && delay > 0 {
					delayText = fmt.Sprintf("%dms", delay)
					if delay < 100 {
						delayColor = tcell.ColorGreen
					} else if delay < 300 {
						delayColor = tcell.ColorYellow
					} else {
						delayColor = tcell.ColorRed
					}
				} else if delay == 0 {
					delayText = "超时"
					delayColor = tcell.ColorRed
				}
			}
		}

		// Fallback to node's own history
		if delayText == "未测试" {
			if node, found := nodeLookup[name]; found && len(node.History) > 0 {
				lastDelay := node.History[len(node.History)-1].Delay
				if lastDelay > 0 {
					delayText = fmt.Sprintf("%dms", lastDelay)
					if lastDelay < 100 {
						delayColor = tcell.ColorGreen
					} else if lastDelay < 300 {
						delayColor = tcell.ColorYellow
					} else {
						delayColor = tcell.ColorRed
					}
				} else {
					delayText = "超时"
					delayColor = tcell.ColorRed
				}
			}
		}

		delayCell := tview.NewTableCell(delayText).
			SetTextColor(delayColor).
			SetAlign(tview.AlignCenter)
		p.nodesList.SetCell(row, 2, delayCell)

		// Status cell
		statusText := "-"
		statusColor := tcell.ColorGray
		switch name {
		case "DIRECT", "REJECT", "REJECT-DROP", "REJECT-TLS":
			statusText = "-"
			statusColor = tcell.ColorGray
		default:
			if node, found := nodeLookup[name]; found {
				if node.UDP {
					statusText = "OK"
					statusColor = tcell.ColorGreen
				} else {
					statusText = "TCP"
					statusColor = tcell.ColorYellow
				}
			}
		}

		statusCell := tview.NewTableCell(statusText).
			SetTextColor(statusColor).
			SetAlign(tview.AlignCenter)
		p.nodesList.SetCell(row, 3, statusCell)
	}

	// Select first node
	if len(allNodes) > 0 {
		p.nodesList.Select(1, 0)
		p.selectedNode = allNodes[0]
	}
}

// buildNodeLookup builds a lookup map of proxy name to proxy data across all providers and flat proxies
func (p *ProxiesPage) buildNodeLookup() map[string]*models.Proxy {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	lookup := make(map[string]*models.Proxy)

	// Collect nodes from providers
	for _, provider := range p.providersData {
		if provider == nil {
			continue
		}
		for _, proxy := range provider.Proxies {
			if proxy != nil && len(proxy.All) == 0 {
				// Only include actual nodes (not groups, which have All)
				lookup[proxy.Name] = proxy
			}
		}
	}

	// Also collect nodes from flat /proxies data if not already found
	for name, proxy := range p.proxiesData {
		if proxy != nil && len(proxy.All) == 0 {
			if _, exists := lookup[name]; !exists {
				lookup[name] = proxy
			}
		}
	}

	return lookup
}

// switchMode switches the proxy mode
func (p *ProxiesPage) switchMode(mode string) {
	if p.currentMode == mode {
		return // Already in this mode
	}

	p.showInfo(fmt.Sprintf("正在切换到 %s 模式...", mode))

	go func() {
		// Get current config
		config, err := api.Client.GetConfig()
		if err != nil {
			p.showError(fmt.Sprintf("获取配置失败: %v", err))
			return
		}

		// Update mode
		config.Mode = mode
		err = api.Client.UpdateConfig(config)
		if err != nil {
			p.showError(fmt.Sprintf("切换模式失败: %v", err))
			return
		}

		// Update local state
		p.currentMode = mode

		// Update StatusBar with new config
		ui.Updater.UpdateStatusBarConfig(config)

		// Update button colors
		go ui.Updater.UpdateUi(func() {
			p.updateButtonStyles()
		})

		p.showSuccess(fmt.Sprintf("已切换到 %s 模式", mode))
		log.Printf("Switched to mode: %s", mode)
	}()
}

// selectCurrentNode selects the currently highlighted node
func (p *ProxiesPage) selectCurrentNode() {
	if p.selectedGroup == "" || p.selectedNode == "" {
		return
	}

	go func() {
		err := api.Client.SelectProxy(p.selectedGroup, p.selectedNode)
		if err != nil {
			p.showError(fmt.Sprintf("切换代理失败: %v", err))
			return
		}

		p.showSuccess(fmt.Sprintf("已切换到代理: %s", p.selectedNode))

		// Update current selection in proxyGroups
		p.mutex.Lock()
		for _, g := range p.proxyGroups {
			if g != nil && g.Name == p.selectedGroup {
				g.Now = p.selectedNode
				break
			}
		}
		p.mutex.Unlock()

		// Update UI to reflect the change
		go ui.Updater.UpdateUi(func() {
			p.updateGroupsList()
			p.updateNodesListContent()
		})
	}()
}

// testGroupDelay tests the delay of all nodes in the selected group
func (p *ProxiesPage) testGroupDelay() {
	if p.selectedGroup == "" || p.isTestingDelay {
		return
	}

	p.isTestingDelay = true
	p.showInfo("正在测试组内所有节点延迟...")

	go func() {
		defer func() {
			p.isTestingDelay = false
		}()

		delays, err := api.Client.TestGroupDelay(p.selectedGroup, "http://www.gstatic.com/generate_204", 3000)
		if err != nil {
			p.showError(fmt.Sprintf("组延迟测试失败: %v", err))
			return
		}

		p.showSuccess("组延迟测试完成")

		// Store delays and update UI
		p.mutex.Lock()
		if p.groupDelays == nil {
			p.groupDelays = make(map[string]map[string]int)
		}
		p.groupDelays[p.selectedGroup] = delays
		p.mutex.Unlock()

		go ui.Updater.UpdateUi(func() {
			p.updateNodesListContent()
		})
	}()
}

// testSelectedNodeDelay tests delay for the selected node using R shortcut
func (p *ProxiesPage) testSelectedNodeDelay() {
	if p.selectedNode == "" || p.isTestingDelay {
		return
	}

	p.isTestingDelay = true
	p.showInfo(fmt.Sprintf("正在测试 %s 延迟...", p.selectedNode))

	go func() {
		defer func() {
			p.isTestingDelay = false
		}()

		delay, err := api.Client.TestProxyDelay(p.selectedNode, "http://www.gstatic.com/generate_204", 5000)
		if err != nil {
			p.showError(fmt.Sprintf("延迟测试失败: %v", err))
			return
		}

		if delay > 0 {
			p.showSuccess(fmt.Sprintf("%s 延迟: %dms", p.selectedNode, delay))

			// Update delay in data structure directly
			p.mutex.Lock()
			// Update in groupDelays for the selected group
			if p.groupDelays == nil {
				p.groupDelays = make(map[string]map[string]int)
			}
			if p.groupDelays[p.selectedGroup] == nil {
				p.groupDelays[p.selectedGroup] = make(map[string]int)
			}
			p.groupDelays[p.selectedGroup][p.selectedNode] = delay

			// Also update in node history for persistence
			for _, provider := range p.providersData {
				for _, proxy := range provider.Proxies {
					if proxy != nil && proxy.Name == p.selectedNode {
						newHistory := models.ProxyHistory{
							Time:  time.Now(),
							Delay: delay,
						}
						proxy.History = append(proxy.History, newHistory)
						if len(proxy.History) > 10 {
							proxy.History = proxy.History[len(proxy.History)-10:]
						}
						break
					}
				}
			}
			p.mutex.Unlock()

			go ui.Updater.UpdateUi(func() {
				p.updateNodesListContent()
			})
		} else {
			p.showError(fmt.Sprintf("%s 延迟测试超时", p.selectedNode))
		}
	}()
}

// Refresh refreshes the proxies data
func (p *ProxiesPage) Refresh() {
	if !p.isActive {
		return
	}

	p.showInfo("正在刷新代理数据...")
	p.loadProvidersData()

	go ui.Updater.UpdateUi(func() {
		p.updateGroupsList()
	})
}

// startAutoRefresh starts periodic data refresh every 10 seconds
func (p *ProxiesPage) startAutoRefresh() {
	if p.autoRefreshStop != nil {
		p.autoRefreshStop() // Stop existing refresh first
	}

	p.autoRefreshCtx, p.autoRefreshStop = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-p.autoRefreshCtx.Done():
				return
			case <-ticker.C:
				p.mutex.RLock()
				active := p.isActive
				p.mutex.RUnlock()
				if !active {
					return
				}

				// Reload provider data silently
				p.loadProvidersData()

				// Update UI with new data
				ui.Updater.UpdateUi(func() {
					p.updateGroupsList()
					if p.selectedGroup != "" {
						p.updateNodesListContent()
					}
				})
			}
		}
	}()
}

// stopAutoRefresh stops the periodic refresh
func (p *ProxiesPage) stopAutoRefresh() {
	if p.autoRefreshStop != nil {
		p.autoRefreshStop()
		p.autoRefreshStop = nil
	}
}

// Stop stops all background processes
func (p *ProxiesPage) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

// showError shows an error message
func (p *ProxiesPage) showError(message string) {
	log.Printf("Error: %s", message)
	if p.statusText != nil {
		go ui.Updater.UpdateUi(func() {
			p.statusText.SetText(fmt.Sprintf("[red]错误:[white] %s", message))
		})
	}
}

// showSuccess shows a success message
func (p *ProxiesPage) showSuccess(message string) {
	log.Printf("Success: %s", message)
	if p.statusText != nil {
		go ui.Updater.UpdateUi(func() {
			p.statusText.SetText(fmt.Sprintf("[green]成功:[white] %s", message))
		})
	}
}

// showInfo shows an info message
func (p *ProxiesPage) showInfo(message string) {
	log.Printf("Info: %s", message)
	if p.statusText != nil {
		go ui.Updater.UpdateUi(func() {
			p.statusText.SetText(fmt.Sprintf("[yellow]信息:[white] %s", message))
		})
	}
}

// GetInputCapture returns the input capture function for keyboard shortcuts
func (p *ProxiesPage) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			p.Refresh()
			return nil
		}

		switch event.Rune() {
		case 'r', 'R':
			p.testSelectedNodeDelay()
			return nil
		}

		return event
	}
}

// Navigation methods

// initializeNavigation initializes the navigation system
func (p *ProxiesPage) initializeNavigation() {
	// Set up focusable components array
	p.focusableComponents = []tview.Primitive{p.groupsList, p.switchButtons, p.nodesList}
	p.currentFocusIndex = 0
	p.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			p.switchToNextComponent()
			return nil
		case tcell.KeyBacktab:
			p.switchToPrevComponent()
			return nil
		}

		switch event.Rune() {
		case 'h', 'H':
			p.showInfo("使用 TAB 切换组件，使用方向键移动选择")
			return nil
		}
		return event
	})
}

// switchToNextComponent switches focus to the next component
func (p *ProxiesPage) switchToNextComponent() {
	if len(p.focusableComponents) == 0 {
		return
	}

	p.currentFocusIndex = (p.currentFocusIndex + 1) % len(p.focusableComponents)
	ui.Updater.SetFocus(p.focusableComponents[p.currentFocusIndex])

	// If switching to switchButtons, focus the current button
	if p.currentFocusIndex == 1 && len(p.focusableButtons) > 0 {
		ui.Updater.SetFocus(p.focusableButtons[p.switchButtonIndex])
	}
}

// switchToPrevComponent switches focus to the previous component
func (p *ProxiesPage) switchToPrevComponent() {
	if len(p.focusableComponents) == 0 {
		return
	}

	p.currentFocusIndex = (p.currentFocusIndex - 1 + len(p.focusableComponents)) % len(p.focusableComponents)
	ui.Updater.SetFocus(p.focusableComponents[p.currentFocusIndex])

	// If switching to switchButtons, focus the current button
	if p.currentFocusIndex == 1 && len(p.focusableButtons) > 0 {
		ui.Updater.SetFocus(p.focusableButtons[p.switchButtonIndex])
	}
}

// switchToNextButton switches to the next button in switchButtons
func (p *ProxiesPage) switchToNextButton() {
	if len(p.focusableButtons) == 0 {
		return
	}

	p.switchButtonIndex = (p.switchButtonIndex + 1) % len(p.focusableButtons)
	ui.Updater.SetFocus(p.focusableButtons[p.switchButtonIndex])
}

// switchToPrevButton switches to the previous button in switchButtons
func (p *ProxiesPage) switchToPrevButton() {
	if len(p.focusableButtons) == 0 {
		return
	}

	p.switchButtonIndex = (p.switchButtonIndex - 1 + len(p.focusableButtons)) % len(p.focusableButtons)
	ui.Updater.SetFocus(p.focusableButtons[p.switchButtonIndex])
}

// focusGroupsList focuses the groups list
func (p *ProxiesPage) focusGroupsList() {
	p.currentFocusIndex = 0
	ui.Updater.SetFocus(p.groupsList)
}

// focusNodesList focuses the nodes list
func (p *ProxiesPage) focusNodesList() {
	p.currentFocusIndex = 2
	ui.Updater.SetFocus(p.nodesList)
}

// focusSwitchButtons focuses the switch buttons
func (p *ProxiesPage) focusSwitchButtons() {
	p.currentFocusIndex = 1
	if len(p.focusableButtons) > 0 {
		ui.Updater.SetFocus(p.focusableButtons[p.switchButtonIndex])
	}
}
