package app

import (
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/events"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/repository"
	"mihomoTui/internal/service"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"
	"mihomoTui/internal/ui/pages"
	proxygroupspage "mihomoTui/internal/ui/pages/proxygroups"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"
	subscriptionspage "mihomoTui/internal/ui/pages/subscriptions"

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

	eventBus        *events.Bus
	yamlDriver      *repository.LocalYAMLDriver
	ruleService     *service.RuleService
	groupService    *service.ProxyGroupService
	providerService *service.ProviderService
	profileService  *service.ProfileService

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
		if !item.Separator {
			pageNames[index] = item.ID
		}
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
	a.eventBus = events.NewBus()
	a.yamlDriver = repository.NewLocalYAMLDriver()
	a.ruleService = service.NewRuleService(a.yamlDriver)
	a.groupService = service.NewProxyGroupService(a.yamlDriver)
	a.providerService = service.NewProviderService(a.yamlDriver, a.subMgr)
	a.profileService = service.NewProfileService(a.yamlDriver, a.eventBus)

	// Subscribe to language changes for dynamic UI re-render
	a.eventBus.Subscribe(events.TopicLanguageChanged, func(event interface{}) {
		a.handleLanguageChanged()
	})

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
	a.statusBar = components.NewStatusBar(a.eventBus)

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

	ui.Updater.SetOverlayPages(a.rootPages)
}

// setupPages initializes all pages
func (a *App) setupPages() {
	// Dashboard page
	dashboardPage := pages.NewDashboardPage()
	dashboardPage.SetInputCapture(dashboardPage.GetInputCapture())
	a.pages.AddPage("dashboard", dashboardPage, true, true)
	a.pageLifecycle["dashboard"] = dashboardPage

	// Proxies page
	proxiesPage := pages.NewProxiesPage(a.eventBus)
	// ProxiesPage sets its own InputCapture internally via initializeNavigation()
	// which already includes Tab navigation and shortcuts
	a.pages.AddPage("proxies", proxiesPage, true, false)
	a.pageLifecycle["proxies"] = proxiesPage

	// Connections page
	connectionsPage := pages.NewConnectionsPage()
	connectionsPage.SetInputCapture(connectionsPage.GetInputCapture())
	a.pages.AddPage("connections", connectionsPage, true, false)
	a.pageLifecycle["connections"] = connectionsPage

	// Logs page
	logsPage := pages.NewLogsPage()
	a.pages.AddPage("logs", logsPage, true, false)
	a.pageLifecycle["logs"] = logsPage

	// Profiles page (formerly configmgr)
	configMgrPage := pages.NewConfigManager(a.profileService, a.eventBus)
	a.pages.AddPage("configmgr", configMgrPage, true, false)
	a.pageLifecycle["configmgr"] = configMgrPage

	// Subscriptions page
	subPage := subscriptionspage.NewPage(a.providerService)
	a.pages.AddPage("subscriptions", subPage, true, false)
	a.pageLifecycle["subscriptions"] = subPage

	// Unified Editor Page (Proxy Groups, Rules, Providers)
	proxyGroupsPage := proxygroupspage.NewPage(a.groupService)
	rulesPage := pages.NewRulesPage(a.ruleService)
	rpPage := rproviders.NewPage(a.providerService)
	editorPage := pages.NewEditorPage(proxyGroupsPage, rulesPage, rpPage)
	a.pages.AddPage("editor", editorPage, true, false)
	a.pageLifecycle["editor"] = editorPage

	// Unified Settings page
	settingsPage := pages.NewUnifiedSettings(a.configManager, a.appName, a.appVersion, a.eventBus)
	a.pages.AddPage("settings", settingsPage, true, false)
	a.pageLifecycle["settings"] = settingsPage
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
		AddItem(a.header, 1, 0, false).
		AddItem(a.bodyLayout, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)
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
		pageName := a.pageNames[index]
		if pageName == "" {
			return // Separator or empty
		}
		currentPageName := a.pageNames[a.currentPage]
		if index != a.currentPage {
			if guard, ok := a.pageLifecycle[currentPageName].(pages.NavigationGuard); ok {
				if !guard.RequestDeactivate(func() { a.switchPage(index) }) {
					return
				}
			}
			// Switch to new page
			a.currentPage = index
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

		a.app.Stop()
	})
}

// handleLanguageChanged updates all UI components and pages when language changes
func (a *App) handleLanguageChanged() {
	ui.Updater.PostUi(func() {
		if a.header != nil {
			a.header.UpdateTexts()
		}
		if a.navBar != nil {
			a.pageInfo = ui.DefaultPageInfo()
			a.navBar.UpdateItems(a.pageInfo)
		}
		if a.statusBar != nil {
			a.statusBar.UpdateTexts()
		}
		if a.rootPages != nil {
			a.helpPage = pages.NewHelpPage(a.pageInfo)
			a.rootPages.AddPage("help", a.helpPage, true, a.showingHelp)
		}
		for _, page := range a.pageLifecycle {
			if updater, ok := page.(interface{ UpdateTexts() }); ok {
				updater.UpdateTexts()
			}
		}
	})
}
