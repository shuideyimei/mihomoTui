package pages

import (
	"fmt"
	"strings"

	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"
	proxygroupspage "mihomoTui/internal/ui/pages/proxygroups"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// EditorPage represents the unified configuration editor page.
type EditorPage struct {
	*tview.Flex

	pages  *tview.Pages
	tabBar *tview.TextView

	// Sub-pages
	proxyGroups *proxygroupspage.Page
	rules       *RulesPage
	providers   *rproviders.Page

	tabs      []string
	activeTab int
}

// NewEditorPage creates a new unified editor page.
func NewEditorPage(proxyGroups *proxygroupspage.Page, rules *RulesPage, providers *rproviders.Page) *EditorPage {
	e := &EditorPage{
		Flex:  tview.NewFlex().SetDirection(tview.FlexRow),
		pages: tview.NewPages(),
		tabBar: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWrap(false).
			SetTextAlign(tview.AlignCenter),
		proxyGroups: proxyGroups,
		rules:       rules,
		providers:   providers,
		tabs:        []string{"proxygroups", "rules", "ruleproviders"},
	}

	e.setupUI()
	return e
}

func (e *EditorPage) setupUI() {
	e.tabBar.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			target := added[0]
			for i, tab := range e.tabs {
				if tab == target {
					if i != e.activeTab {
						e.switchTab(i)
					} else {
						e.tabBar.Highlight()
					}
					break
				}
			}
		}
	})

	e.pages.AddPage("proxygroups", e.proxyGroups, true, true)
	e.pages.AddPage("rules", e.rules, true, false)
	e.pages.AddPage("ruleproviders", e.providers, true, false)

	// Combine into layout
	e.AddItem(e.tabBar, 1, 0, false)
	e.AddItem(e.pages, 0, 1, true)

	e.updateTabBar()

	// Handle Tab/Shift+Tab to switch subtabs (unless inside an input/form)
	e.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if app := ui.Updater.GetApp(); app != nil {
			switch app.GetFocus().(type) {
			case *tview.InputField, *tview.TextArea, *tview.Form, *tview.DropDown:
				return event
			}
		}

		isBacktab := event.Key() == tcell.KeyBacktab || (event.Key() == tcell.KeyTab && event.Modifiers()&tcell.ModShift != 0)
		if isBacktab {
			e.switchTab((e.activeTab - 1 + len(e.tabs)) % len(e.tabs))
			return nil
		}
		if event.Key() == tcell.KeyTab {
			e.switchTab((e.activeTab + 1) % len(e.tabs))
			return nil
		}
		return event
	})
}

func (e *EditorPage) switchTab(index int) {
	if index < 0 || index >= len(e.tabs) {
		return
	}
	if e.activeTab == index {
		return
	}
	if name, act := e.pages.GetFrontPage(); name != "" {
		if a, ok := act.(ActivatablePage); ok {
			a.Deactivate()
		}
	}
	e.activeTab = index
	e.pages.SwitchToPage(e.tabs[index])
	e.updateTabBar()
	if name, act := e.pages.GetFrontPage(); name != "" {
		if a, ok := act.(ActivatablePage); ok {
			a.Activate()
		}
		if p, ok := act.(tview.Primitive); ok {
			ui.Updater.SetFocus(p)
		}
	}
}

func (e *EditorPage) updateTabBar() {
	var sb strings.Builder
	for i, tab := range e.tabs {
		title := ""
		switch tab {
		case "proxygroups":
			title = i18n.T("nav.proxygroups")
		case "rules":
			title = i18n.T("nav.rules")
		case "ruleproviders":
			title = i18n.T("nav.ruleproviders")
		}

		if i == e.activeTab {
			sb.WriteString(fmt.Sprintf(`["%s"][black:green] %s [-:-:-][""]  `, tab, title))
		} else {
			sb.WriteString(fmt.Sprintf(`["%s"][white] %s [-:-:-][""]  `, tab, title))
		}
	}
	// Add hint
	sb.WriteString("  [gray](Tab / Shift+Tab to switch)[-]")

	e.tabBar.SetText(sb.String())
	e.tabBar.Highlight()
}

func (e *EditorPage) Activate() {
	if name, act := e.pages.GetFrontPage(); name != "" {
		if a, ok := act.(ActivatablePage); ok {
			a.Activate()
		}
		if p, ok := act.(tview.Primitive); ok {
			ui.Updater.SetFocus(p)
		}
	}
}

func (e *EditorPage) Deactivate() {
	if name, act := e.pages.GetFrontPage(); name != "" {
		if a, ok := act.(ActivatablePage); ok {
			a.Deactivate()
		}
	}
}

// UpdateTexts refreshes tab bar texts and sub-page texts on language change
func (e *EditorPage) UpdateTexts() {
	ui.Updater.PostUi(func() {
		e.updateTabBar()
		if e.proxyGroups != nil {
			if u, ok := interface{}(e.proxyGroups).(interface{ UpdateTexts() }); ok {
				u.UpdateTexts()
			}
		}
		if e.rules != nil {
			e.rules.UpdateTexts()
		}
		if e.providers != nil {
			if u, ok := interface{}(e.providers).(interface{ UpdateTexts() }); ok {
				u.UpdateTexts()
			}
		}
	})
}
