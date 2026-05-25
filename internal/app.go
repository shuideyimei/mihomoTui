package app

import (
	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"
	"mihomoTui/internal/ui/pages"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"
	subscriptionspage "mihomoTui/internal/ui/pages/subscriptions"

	"github.com/rivo/tview"
)

// App represents the main application
type App struct {
	app *tview.Application

	configManager *config.Manager
	subMgr        *subscription.Manager

	header    *components.Header
	sidebar   *components.Sidebar
	statusBar *components.StatusBar
	content   tview.Primitive

	rootLayout   *tview.Flex
	rootPages    *tview.Pages
	mainLayout   *tview.Flex
	pages        *tview.Pages
	pageNames    []string
	currentPage  int

	focusOnSidebar bool
	showingHelp    bool
	appName        string
	appVersion     string
}

// NewApp creates a new application instance
func NewApp(appName, appVersion string) *App {
	return &App{
		app:            tview.NewApplication(),
		pageNames:      []string{"dashboard", "proxies", "connections", "config", "logs", "subscriptions", "proxygroups", "rules", "ruleproviders", "settings", "configmgr"},
		focusOnSidebar: true,
		appName:        appName,
		appVersion:     appVersion,
	}
}

// Initialize initializes the application
func (a *App) Initialize() error {
	// Initialize configuration
	a.configManager = config.NewManager()
	if err := a.configManager.Load(); err != nil {
		return err
	}
	// Set application
	ui.InitUpdater(a.app)

	// Initialize API client
	cfg := a.configManager.GetAPI()
	api.InitClient(cfg.BaseURL, cfg.Secret)

	a.subMgr = subscription.NewManager()

	// Initialize UI components
	a.setupUI()

	// Start initialization tasks
	go a.basicInitData()

	// Activate the initial page (dashboard)
	if len(a.pageNames) > 0 {
		a.activatePage(a.pageNames[0])
	}

	return nil
}

// setupUI initializes the user interface
func (a *App) setupUI() {
	// Create components
	a.header = components.NewHeader(a.appName, a.appVersion)
	a.sidebar = components.NewSidebar()
	a.statusBar = components.NewStatusBar()

	// Create pages
	a.pages = tview.NewPages()
	a.setupPages()

	// Set initial content
	a.content = a.pages

	// Setup sidebar selection handler
	a.sidebar.SetOnSelect(func(index int, label string) {
		a.switchPage(index)
	})

	// Create layouts
	a.setupLayouts()

	a.rootPages = tview.NewPages()
	a.rootPages.AddPage("main", a.rootLayout, true, true)
	a.rootPages.AddPage("help", pages.NewHelpPage(), true, false)

	// Configure application
	a.app.SetRoot(a.rootPages, true)
	a.app.EnableMouse(true)   // Enable mouse support by default
	a.app.EnablePaste(true)   // Enable bracketed paste mode for reliable paste handling

	// Set global key handlers
	a.app.SetInputCapture(a.handleGlobalKeys)

	// Set initial focus to sidebar
	a.setFocus(true)

	// Set StatusBar reference in UI Updater
	ui.Updater.SetStatusBar(a.statusBar)
}

// setupPages initializes all pages
func (a *App) setupPages() {
	// Dashboard page
	dashboardPage := pages.NewDashboard()
	dashboardPage.SetInputCapture(dashboardPage.GetInputCapture())
	a.pages.AddPage("dashboard", dashboardPage, true, true)

	// Proxies page
	proxiesPage := pages.NewProxies()
	a.pages.AddPage("proxies", proxiesPage, true, false)

	// Connections page
	connectionsPage := pages.NewConnections()
	a.pages.AddPage("connections", connectionsPage, true, false)

	// Config page
	configPage := pages.NewConfig(a.configManager)
	a.pages.AddPage("config", configPage, true, false)

	// Logs page
	logsPage := pages.NewLogs()
	a.pages.AddPage("logs", logsPage, true, false)

	// Subscriptions page
	subPage := subscriptionspage.NewPage(a.subMgr)
	a.pages.AddPage("subscriptions", subPage, true, false)

	proxyGroupsPage := pages.NewProxyGroups()
	a.pages.AddPage("proxygroups", proxyGroupsPage, true, false)

	// Rules page
	rulesPage := pages.NewRules()
	a.pages.AddPage("rules", rulesPage, true, false)

	// Rule providers page
	rpPage := rproviders.NewPage()
	a.pages.AddPage("ruleproviders", rpPage, true, false)

	// Settings page
	settingsPage := pages.NewSettings(a.configManager, a.appName, a.appVersion)
	a.pages.AddPage("settings", settingsPage, true, false)

	configMgrPage := pages.NewConfigManager()
	a.pages.AddPage("configmgr", configMgrPage, true, false)
}

// setupLayouts creates the application layout
func (a *App) setupLayouts() {
	// Main layout (sidebar + content)
	a.mainLayout = tview.NewFlex().
		AddItem(a.sidebar, 12, 0, true).
		AddItem(a.content, 0, 1, false)

	// Root layout (header + main + status)
	a.rootLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 3, 0, false).
		AddItem(a.mainLayout, 0, 1, true).
		AddItem(a.statusBar, 3, 0, false)
}

// basicInitData initializes basic application data
func (a *App) basicInitData() {
	// Initial connection check
	go a.header.SetHeaderInfo()

	// Update status bar
	go a.statusBar.Active()
}

// switchPage switches to a specific page
func (a *App) switchPage(index int) {
	if index >= 0 && index < len(a.pageNames) {
		currentPageName := a.pageNames[a.currentPage]
		if index != a.currentPage {
			if currentPageName != "" {
				a.deactivatePage(currentPageName)
			}
			// Switch to new page
			a.currentPage = index
			pageName := a.pageNames[index]
			a.pages.SwitchToPage(pageName)
			a.sidebar.SelectItem(index)

			// Activate new page if it's activatable
			a.activatePage(pageName)
		}

		// Switch focus to content when switching pages
		a.setFocus(false)
	}
}

// activatePage activates a page if it implements ActivatablePage
func (a *App) activatePage(pageName string) {
	name, primitive := a.pages.GetFrontPage()
	if name == pageName && primitive != nil {
		if activatable, ok := primitive.(pages.ActivatablePage); ok {
			go activatable.Activate()
		}
	}
}

// deactivatePage deactivates a page if it implements ActivatablePage
func (a *App) deactivatePage(pageName string) {
	name, primitive := a.pages.GetFrontPage()
	if name == pageName && primitive != nil {
		if activatable, ok := primitive.(pages.ActivatablePage); ok {
			go activatable.Deactivate()
		}
	}
}

// setFocus sets focus to either sidebar or content
func (a *App) setFocus(toSidebar bool) {
	a.focusOnSidebar = toSidebar
	if toSidebar {
		a.app.SetFocus(a.sidebar)
	} else {
		a.app.SetFocus(a.content)
	}
}

// Run starts the application
func (a *App) Run() error {
	// Start background tasks
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.app.Suspend(func() {
		a.app.Stop()
	})

	// Deactivate current page if it's activatable
	if a.currentPage >= 0 && a.currentPage < len(a.pageNames) {
		currentPageName := a.pageNames[a.currentPage]
		if currentPageName != "" {
			a.deactivatePage(currentPageName)
		}
	}

}
