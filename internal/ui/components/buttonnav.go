package components

import (
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

// ButtonNavigator manages keyboard-based button navigation
type ButtonNavigator struct {
	buttons []*tview.Button
	index   int
}

// NewButtonNavigator creates a new ButtonNavigator
func NewButtonNavigator() *ButtonNavigator {
	return &ButtonNavigator{}
}

// SetButtons sets the navigable buttons and resets the index
func (bn *ButtonNavigator) SetButtons(buttons []*tview.Button) {
	bn.buttons = buttons
	bn.index = 0
}

// Next moves focus to the next button (wraps around)
func (bn *ButtonNavigator) Next() {
	if len(bn.buttons) == 0 {
		return
	}
	bn.index = (bn.index + 1) % len(bn.buttons)
	go ui.Updater.SetFocus(bn.buttons[bn.index])
}

// Prev moves focus to the previous button (wraps around)
func (bn *ButtonNavigator) Prev() {
	if len(bn.buttons) == 0 {
		return
	}
	bn.index = (bn.index - 1 + len(bn.buttons)) % len(bn.buttons)
	go ui.Updater.SetFocus(bn.buttons[bn.index])
}

// Index returns the current button index
func (bn *ButtonNavigator) Index() int {
	return bn.index
}
