package pages

import (
	"fmt"

	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

// buildHelpText builds the help text in the current language
func buildHelpText(pageInfo []ui.PageInfo) string {
	text := fmt.Sprintf(`[yellow::b]=== %s ===[-::-]

[green::b]%s:[-::-]
`, i18n.T("help.title"), i18n.T("help.section_nav"))
	for index, item := range pageInfo {
		ctrl := ""
		if index < 9 {
			ctrl = fmt.Sprintf(" / Ctrl+%d", index+1)
		} else if index == 9 {
			ctrl = " / Ctrl+0"
		}
		text += fmt.Sprintf("  [white]%-18s[-] %s\n", item.Function+ctrl+" / "+item.Alt, item.Label())
	}
	text += fmt.Sprintf(`
[green::b]%s:[-::-]
  [white]Esc[-]       %s
  [white]Tab[-]       %s
  [white]Ctrl+R[-]    %s
  [white]:[-]         %s
  [white]Ctrl+C/Q[-]  %s
  [white]? / F12[-]   %s

[green::b]%s:[-::-]
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s
  [white]%s:[-] %s

  [gray]%s[-]

[yellow]%s[-]`,
		i18n.T("help.section_global"),
		i18n.T("help.global_esc"),
		i18n.T("help.global_tab"),
		i18n.T("help.global_refresh"),
		i18n.T("help.global_commands"),
		i18n.T("help.global_quit"),
		i18n.T("help.global_help"),
		i18n.T("help.section_page"),
		i18n.T("nav.proxies"),
		i18n.T("help.page_proxies"),
		i18n.T("nav.connections"),
		i18n.T("help.page_connections"),
		i18n.T("nav.rules"),
		i18n.T("help.page_rules"),
		i18n.T("nav.logs"),
		i18n.T("help.page_logs"),
		i18n.T("nav.subscriptions"),
		i18n.T("help.page_subscriptions"),
		i18n.T("nav.proxygroups"),
		i18n.T("help.page_proxygroups"),
		i18n.T("nav.configmgr"),
		i18n.T("help.page_configmgr"),
		i18n.T("help.page_hint"),
		i18n.T("help.close_hint"),
	)
	return text
}

// NewHelpPage creates a help overlay panel showing all keyboard shortcuts
func NewHelpPage(pageInfo []ui.PageInfo) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignLeft).
		SetText(buildHelpText(pageInfo))

	textView.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", i18n.T("help.title"))).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(ui.ThemeBorderColor)

	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(textView, 0, 8, true).
		AddItem(nil, 0, 1, false)

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 0, 8, true).
		AddItem(nil, 0, 1, false)
}
