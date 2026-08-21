package components

import (
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FocusNavigator manages keyboard-based focus navigation among a set of primitives.
// Use it to implement consistent Tab/Shift+Tab cycling on pages.
type FocusNavigator struct {
	components []tview.Primitive
	index      int
}

// NewFocusNavigator creates a FocusNavigator for the given primitives.
func NewFocusNavigator(components ...tview.Primitive) *FocusNavigator {
	return &FocusNavigator{
		components: components,
		index:      0,
	}
}

// SetComponents replaces the navigable component list and resets the index.
func (fn *FocusNavigator) SetComponents(components []tview.Primitive) {
	fn.components = components
	fn.index = 0
}

// Next moves focus to the next component (wraps around).
func (fn *FocusNavigator) Next() {
	if len(fn.components) == 0 {
		return
	}
	fn.index = (fn.index + 1) % len(fn.components)
	ui.Updater.SetFocus(fn.components[fn.index])
}

// Prev moves focus to the previous component (wraps around).
func (fn *FocusNavigator) Prev() {
	if len(fn.components) == 0 {
		return
	}
	fn.index = (fn.index - 1 + len(fn.components)) % len(fn.components)
	ui.Updater.SetFocus(fn.components[fn.index])
}

// Index returns the current focus index.
func (fn *FocusNavigator) Index() int {
	return fn.index
}

// FocusNext returns an InputCapture handler that calls Next on Tab and Prev on Backtab.
func (fn *FocusNavigator) FocusNext() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyBacktab || (event.Key() == tcell.KeyTAB && event.Modifiers()&tcell.ModShift != 0) {
			fn.Prev()
			return nil
		}
		if event.Key() == tcell.KeyTAB {
			fn.Next()
			return nil
		}
		return event
	}
}
