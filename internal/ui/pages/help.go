package pages

import (
	"fmt"

	"mihomoTui/internal/i18n"

	"github.com/rivo/tview"
	"mihomoTui/internal/ui"
)

// buildHelpText builds the help text in the current language
func buildHelpText() string {
	return fmt.Sprintf(`[yellow::b]=== %s ===[-::-]

[green::b]%s:[-::-]
  [white]F1 / Ctrl+1 / Alt+D[-]  %s
  [white]F2 / Ctrl+2 / Alt+P[-]  %s
  [white]F3 / Ctrl+3 / Alt+R[-]  %s
  [white]F4 / Ctrl+4 / Alt+C[-]  %s
  [white]F5 / Ctrl+5 / Alt+L[-]  %s
  [white]F6 / Ctrl+6 / Alt+S[-]  %s
  [white]F7 / Ctrl+7 / Alt+G[-]  %s
  [white]F8 / Ctrl+8 / Alt+U[-]  %s
  [white]F9 / Ctrl+9 / Alt+T[-]  %s
  [white]F10 / Ctrl+0 / Alt+K[-] %s
  [white]F11 / Alt+M[-]           %s

[green::b]%s:[-::-]
  [white]Esc[-]       %s
  [white]Tab[-]       %s
  [white]Ctrl+R[-]    %s
  [white]Ctrl+C/Q[-]  %s
  [white]? / F12[-]   %s

[green::b]%s:[-::-]
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s

  [gray]%s[-]

[yellow]%s[-]`,
		i18n.T("help.title"),
		i18n.T("help.section_nav"),
		i18n.T("help.nav_dashboard"),
		i18n.T("help.nav_proxies"),
		i18n.T("help.nav_connections"),
		i18n.T("help.nav_config"),
		i18n.T("help.nav_logs"),
		i18n.T("help.nav_subscriptions"),
		i18n.T("help.nav_proxygroups"),
		i18n.T("help.nav_rules"),
		i18n.T("help.nav_ruleproviders"),
		i18n.T("help.nav_settings"),
		i18n.T("help.nav_configmgr"),
		i18n.T("help.section_global"),
		i18n.T("help.global_esc"),
		i18n.T("help.global_tab"),
		i18n.T("help.global_refresh"),
		i18n.T("help.global_quit"),
		i18n.T("help.global_help"),
		i18n.T("help.section_page"),
		i18n.T("nav.proxies"),
		i18n.T("help.page_proxies"),
		i18n.T("nav.connections"),
		i18n.T("help.page_connections"),
		i18n.T("nav.rules"),
		i18n.T("help.page_rules"),
		i18n.T("help.page_hint"),
		i18n.T("help.close_hint"),
	)
}

// NewHelpPage creates a help overlay panel showing all keyboard shortcuts
func NewHelpPage() tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(buildHelpText())

	textView.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", i18n.T("help.title"))).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(ui.ThemeBorderColor)

	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(textView, 28, 0, true).
		AddItem(nil, 0, 1, false)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 70, 0, true).
		AddItem(nil, 0, 1, false)
}
