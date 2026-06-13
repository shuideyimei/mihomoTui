package app

import (
	"strings"

	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) showCommandPalette() {
	if a.showingPalette {
		return
	}
	a.showingPalette = true
	a.paletteReturnFocus = a.app.GetFocus()

	input := tview.NewInputField().
		SetLabel(i18n.T("command.label")).
		SetPlaceholder(i18n.T("command.placeholder"))
	input.SetFieldBackgroundColor(ui.ThemeInputBg)

	list := components.StyleSelectableList(tview.NewList())
	list.ShowSecondaryText(true)
	var visible []int
	rebuild := func(query string) {
		query = strings.ToLower(strings.TrimSpace(query))
		visible = visible[:0]
		list.Clear()
		for index, item := range a.pageInfo {
			label := item.Label()
			if query != "" && !strings.Contains(strings.ToLower(label+" "+item.ID), query) {
				continue
			}
			visible = append(visible, index)
			list.AddItem(label, item.Function+"  "+item.Alt, 0, nil)
		}
	}
	choose := func() {
		index := list.GetCurrentItem()
		if index >= 0 && index < len(visible) {
			page := visible[index]
			a.hideCommandPalette()
			a.switchPage(page)
		}
	}
	rebuild("")
	input.SetChangedFunc(rebuild)
	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			choose()
		case tcell.KeyDown, tcell.KeyTab:
			a.app.SetFocus(list)
		}
	})
	list.SetSelectedFunc(func(_ int, _ string, _ string, _ rune) { choose() })
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyBacktab {
			a.app.SetFocus(input)
			return nil
		}
		return event
	})

	panel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true).
		AddItem(list, 0, 1, false)
	panel.SetBorder(true)
	panel.SetTitle(" " + i18n.T("command.title") + " ")
	panel.SetBorderColor(ui.ThemeFocusColor)

	center := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(panel, 0, 3, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)
	a.rootPages.AddPage("command-palette", center, true, true)
	a.app.SetFocus(input)
}

func (a *App) hideCommandPalette() {
	if !a.showingPalette {
		return
	}
	a.rootPages.RemovePage("command-palette")
	a.showingPalette = false
	if a.paletteReturnFocus != nil {
		a.app.SetFocus(a.paletteReturnFocus)
		a.paletteReturnFocus = nil
	}
}
