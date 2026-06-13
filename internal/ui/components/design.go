package components

import (
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ButtonKind int

const (
	ButtonNormal ButtonKind = iota
	ButtonPrimary
	ButtonDanger
)

// StyleButton applies the shared button states used across the application.
func StyleButton(button *tview.Button, kind ButtonKind) *tview.Button {
	background := ui.ThemeButtonBg
	foreground := tcell.ColorWhite
	if kind == ButtonDanger {
		foreground = ui.ThemeDangerColor
	}
	button.SetStyle(tcell.StyleDefault.Background(background).Foreground(foreground))
	button.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeFocusColor).Foreground(tcell.ColorBlack).Bold(kind == ButtonPrimary))
	return button
}

func NewButton(label string, kind ButtonKind, selected func()) *tview.Button {
	button := StyleButton(tview.NewButton(label), kind)
	button.SetSelectedFunc(selected)
	return button
}

// StyleSelectableList makes focused, blurred, and selected states distinct.
func StyleSelectableList(list *tview.List) *tview.List {
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorGray)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(ui.ThemeSelectionColor)
	list.SetFocusFunc(func() {
		list.SetBorderColor(ui.ThemeFocusColor)
		list.SetSelectedBackgroundColor(ui.ThemeSelectionColor)
	})
	list.SetBlurFunc(func() {
		list.SetBorderColor(ui.ThemeBorderColor)
		list.SetSelectedBackgroundColor(ui.ThemeSelectionBlurColor)
	})
	return list
}

func StyleStatusLine(view *tview.TextView) *tview.TextView {
	view.SetBorder(false)
	view.SetDynamicColors(true)
	view.SetTextAlign(tview.AlignLeft)
	return view
}

func StyleActionBar(bar *tview.Flex) *tview.Flex {
	bar.SetBorder(false)
	return bar
}

type focusableBox interface {
	SetFocusFunc(func()) *tview.Box
	SetBlurFunc(func()) *tview.Box
	SetBorderColor(tcell.Color) *tview.Box
}

func StyleFocusBorder(box focusableBox) {
	box.SetFocusFunc(func() { box.SetBorderColor(ui.ThemeFocusColor) })
	box.SetBlurFunc(func() { box.SetBorderColor(ui.ThemeBorderColor) })
}
