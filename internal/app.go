package app

import (
	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"
	"mihomoTui/internal/ui/pages"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"
	subscriptionspage "mihomoTui/internal/ui/pages/subscriptions"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type lifecycleTransition struct {
	deactivate pages.ActivatablePage
	activate   pages.ActivatablePage
	stop       bool
}

// App represents the main application
type App struct {
	app *tview.Application

	configManager *config.Manager
	subMgr        *subscription.Manager

	header    *components.Header
	navBar    *components.NavBar
	statusBar *components.StatusBar
	content   tview.Primitive

	rootLayout    *tview.Flex
	bodyLayout    *tview.Flex
	rootPages     *tview.Pages
	mainLayout    *tview.Flex
	pages         *tview.Pages
	helpPage      tview.Primitive
	pageNames     []string
	pageInfo      []ui.PageInfo
	currentPage   int
	pageLifecycle map[string]pages.ActivatablePage

	lifecycleQueue chan lifecycleTransition
	lifecycleMu    sync.Mutex
	lifecycleStop  bool
	stopOnce       sync.Once

	focusOnNav         bool
	showingHelp        bool
	showingPalette     bool
	helpReturnFocus    tview.Primitive
	paletteReturnFocus tview.Primitive
	appName            string
	appVersion         string
}

// NewApp creates a new application instance
func NewApp(appName, appVersion string) *App {
	pageInfo := ui.DefaultPageInfo()
	pageNames := make([]string, len(pageInfo))
	for index, item := range pageInfo {
		pageNames[index] = item.ID
	}
	return &App{
		app:            tview.NewApplication(),
		pageNames:      pageNames,
		pageInfo:       pageInfo,
		pageLifecycle:  make(map[string]pages.ActivatablePage),
		lifecycleQueue: make(chan lifecycleTransition, 64),
		focusOnNav:     true,
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

	// Initialize global theme (border colors, etc.)
	ui.InitTheme()

	// Initialize language from config
	i18n.SetLanguage(i18n.Language(a.configManager.GetLanguage()))

	// Initialize API client
	cfg := a.configManager.GetAPI()
	api.InitClient(cfg.BaseURL, cfg.Secret)

	a.subMgr = subscription.NewManager()

	// Initialize UI components
	a.setupUI()
	go a.runLifecycle()

	// Start initialization tasks
	go a.basicInitData()

	// Activate the initial page (dashboard)
	if len(a.pageNames) > 0 {
		a.activatePage(a.pageNames[0])
	}

	return nil
}

// setFocus sets focus to either navbar or content
func (a *App) setFocus(toNav bool) {
	a.focusOnNav = toNav
	if toNav {
		a.app.SetFocus(a.navBar)
	} else {
		a.app.SetFocus(a.content)
	}
}

// setupUI initializes the user interface
func (a *App) setupUI() {
	// Create components
	a.header = components.NewHeader(a.appName, a.appVersion)
	a.navBar = components.NewNavBar(a.pageInfo)
	a.statusBar = components.NewStatusBar()

	// Create pages
	a.pages = tview.NewPages()
	a.setupPages()

	// Set initial content
	a.content = a.pages

	// Setup navigation bar selection handler
	a.navBar.SetOnSelect(func(index int, label string) {
		a.switchPage(index)
	})

	// Create layouts
	a.setupLayouts()

	// Give initial focus to navbar
	a.app.SetFocus(a.navBar)

	a.rootPages = tview.NewPages()
	a.rootPages.AddPage("main", a.rootLayout, true, true)
	a.helpPage = pages.NewHelpPage(a.pageInfo)
	a.rootPages.AddPage("help", a.helpPage, true, false)

	// Configure application
	a.app.SetRoot(a.rootPages, true)
	a.app.EnableMouse(true) // Enable mouse support by default
	a.app.EnablePaste(true) // Enable bracketed paste mode for reliable paste handling

	// Set global key handlers
	a.app.SetInputCapture(a.handleGlobalKeys)
	a.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, _ := screen.Size()
		navWidth := 24
		if width < 100 {
			navWidth = 18
		}
		if width < 72 {
			navWidth = 14
		}
		a.bodyLayout.ResizeItem(a.navBar, navWidth, 0)
		return false
	})

	// Set StatusBar reference in UI Updater
	ui.Updater.SetStatusBar(a.statusBar)
	ui.Updater.SetOverlayPages(a.rootPages)
}

// setupPages initializes all pages
func (a *App) setupPages() {
	// Dashboard page
	dashboardPage := pages.NewDashboard()
	dashboardPage.SetInputCapture(dashboardPage.GetInputCapture())
	a.pages.AddPage("dashboard", dashboardPage, true, true)
	a.pageLifecycle["dashboard"] = dashboardPage

	// Proxies page
	proxiesPage := pages.NewProxies()
	// ProxiesPage sets its own InputCapture internally via initializeNavigation()
	// which already includes Tab navigation and shortcuts
	a.pages.AddPage("proxies", proxiesPage, true, false)
	a.pageLifecycle["proxies"] = proxiesPage

	// Connections page
	connectionsPage := pages.NewConnections()
	connectionsPage.SetInputCapture(connectionsPage.GetInputCapture())
	a.pages.AddPage("connections", connectionsPage, true, false)
	a.pageLifecycle["connections"] = connectionsPage

	// Config page
	configPage := pages.NewConfig(a.configManager)
	a.pages.AddPage("config", configPage, true, false)
	a.pageLifecycle["config"] = configPage

	// Logs page
	logsPage := pages.NewLogs()
	a.pages.AddPage("logs", logsPage, true, false)
	a.pageLifecycle["logs"] = logsPage

	// Subscriptions page
	subPage := subscriptionspage.NewPage(a.subMgr)
	a.pages.AddPage("subscriptions", subPage, true, false)
	a.pageLifecycle["subscriptions"] = subPage

	proxyGroupsPage := pages.NewProxyGroups()
	a.pages.AddPage("proxygroups", proxyGroupsPage, true, false)
	a.pageLifecycle["proxygroups"] = proxyGroupsPage

	// Rules page
	rulesPage := pages.NewRules()
	a.pages.AddPage("rules", rulesPage, true, false)
	a.pageLifecycle["rules"] = rulesPage

	// Rule providers page
	rpPage := rproviders.NewPage()
	a.pages.AddPage("ruleproviders", rpPage, true, false)
	a.pageLifecycle["ruleproviders"] = rpPage

	// Settings page
	settingsPage := pages.NewSettings(a.configManager, a.appName, a.appVersion)
	settingsPage.SetInputCapture(settingsPage.GetInputCapture())
	a.pages.AddPage("settings", settingsPage, true, false)

	configMgrPage := pages.NewConfigManager()
	a.pages.AddPage("configmgr", configMgrPage, true, false)
	a.pageLifecycle["configmgr"] = configMgrPage
}

// setupLayouts creates the application layout
func (a *App) setupLayouts() {
	// Main layout (content full width)
	a.mainLayout = tview.NewFlex().
		AddItem(a.content, 0, 1, false)

	a.bodyLayout = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.navBar, 24, 0, true).
		AddItem(a.mainLayout, 0, 1, false)

	// Root layout (header + side navigation/content + status)
	a.rootLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 2, 0, false).
		AddItem(a.bodyLayout, 0, 1, true).
		AddItem(a.statusBar, 2, 0, false)
}

// basicInitData initializes basic application data.
// Called via go routine from Initialize() so all calls here are already async.
func (a *App) basicInitData() {
	a.header.SetHeaderInfo()
	a.statusBar.Active()
}

// switchPage switches to a specific page
func (a *App) switchPage(index int) {
	if index >= 0 && index < len(a.pageNames) {
		currentPageName := a.pageNames[a.currentPage]
		if index != a.currentPage {
			if guard, ok := a.pageLifecycle[currentPageName].(pages.NavigationGuard); ok {
				if !guard.RequestDeactivate(func() { a.switchPage(index) }) {
					return
				}
			}
			// Switch to new page
			a.currentPage = index
			pageName := a.pageNames[index]
			a.pages.SwitchToPage(pageName)
			a.navBar.SelectItem(index)

			a.transitionPages(currentPageName, pageName)
		}

		// Switch focus to content when switching pages
		a.setFocus(false)
	}
}

// activatePage activates a page if it implements ActivatablePage
func (a *App) activatePage(pageName string) {
	a.enqueueLifecycle(lifecycleTransition{activate: a.pageLifecycle[pageName]})
}

func (a *App) transitionPages(from, to string) {
	a.enqueueLifecycle(lifecycleTransition{
		deactivate: a.pageLifecycle[from],
		activate:   a.pageLifecycle[to],
	})
}

func (a *App) enqueueLifecycle(transition lifecycleTransition) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	if a.lifecycleStop {
		return
	}
	a.lifecycleQueue <- transition
}

func (a *App) runLifecycle() {
	for transition := range a.lifecycleQueue {
		if transition.deactivate != nil {
			transition.deactivate.Deactivate()
		}
		if transition.activate != nil {
			transition.activate.Activate()
		}
		if transition.stop {
			return
		}
	}
}

// Run starts the application
func (a *App) Run() error {
	// Start background tasks
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.stopOnce.Do(func() {
		a.statusBar.Deactivate()
		if a.currentPage >= 0 && a.currentPage < len(a.pageNames) {
			currentPageName := a.pageNames[a.currentPage]
			a.enqueueLifecycle(lifecycleTransition{
				deactivate: a.pageLifecycle[currentPageName],
				stop:       true,
			})
		}

		a.lifecycleMu.Lock()
		a.lifecycleStop = true
		a.lifecycleMu.Unlock()

		a.app.Suspend(func() {
			a.app.Stop()
		})
	})
}
