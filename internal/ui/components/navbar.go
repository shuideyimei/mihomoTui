package components

import (
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NavItem represents a navigation bar menu item
type NavItem struct {
	Name     string // display name
	Shortcut string // keyboard shortcut
}

// NavBar is a single-row horizontal navigation bar at the top of the page.
type NavBar struct {
	*tview.Flex
	items       []NavItem
	buttons     []*tview.Button
	onSelect    func(int, string)

	currentIndex int
	navigator    *FocusNavigator
}

// NewNavBar creates a new navigation bar component.
func NewNavBar() *NavBar {
	n := &NavBar{
		Flex: tview.NewFlex(),
		items: []NavItem{
			{Name: "仪表板", Shortcut: "D"},
			{Name: "代理", Shortcut: "P"},
			{Name: "连接", Shortcut: "R"},
			{Name: "配置", Shortcut: "C"},
			{Name: "日志", Shortcut: "L"},
			{Name: "订阅", Shortcut: "S"},
			{Name: "代理组", Shortcut: "G"},
			{Name: "规则", Shortcut: "U"},
			{Name: "规则提供者", Shortcut: "T"},
			{Name: "设置", Shortcut: "K"},
			{Name: "配置管理", Shortcut: "M"},
		},
		currentIndex: 0,
	}

	n.setupBar()
	return n
}

// setupBar creates the single-row button bar.
func (n *NavBar) setupBar() {
	n.SetDirection(tview.FlexColumn)

	n.buttons = make([]*tview.Button, 0, len(n.items))

	for idx, item := range n.items {
		index := idx // capture
		label := " " + item.Name
		btn := tview.NewButton(label)
		btn.SetSelectedFunc(func() {
			n.onItemSelected(index)
		})
		btn.SetStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkGray).
			Foreground(tcell.ColorGray),
		)
		btn.SetActivatedStyle(tcell.StyleDefault.
			Background(tcell.ColorGray).
			Foreground(tcell.ColorWhite),
		)

		n.buttons = append(n.buttons, btn)
		n.AddItem(btn, 0, 1, idx == 0)
	}

	// Build focus navigator from all buttons
	navigatorPrimitives := make([]tview.Primitive, len(n.buttons))
	for i, b := range n.buttons {
		navigatorPrimitives[i] = b
	}
	n.navigator = NewFocusNavigator(navigatorPrimitives...)

	// Keyboard navigation via Tab/arrows
	n.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB, tcell.KeyRight:
			n.navigator.Next()
			return nil
		case tcell.KeyBacktab, tcell.KeyLeft:
			n.navigator.Prev()
			return nil
		}
		return event
	})

	// Focus: highlight current item
	n.SetFocusFunc(func() {
		n.highlightCurrent()
	})

	n.SetBlurFunc(func() {
		n.blurAll()
	})
}

// onItemSelected handles a navigation item being selected.
func (n *NavBar) onItemSelected(index int) {
	if index < 0 || index >= len(n.items) {
		return
	}
	n.currentIndex = index
	n.highlightCurrent()
	if n.onSelect != nil {
		n.onSelect(index, n.items[index].Name)
	}
}

// highlightCurrent applies the focused/active style to the current button
// and the unfocused style to all others.
func (n *NavBar) highlightCurrent() {
	for i, btn := range n.buttons {
		if i == n.currentIndex {
			btn.SetStyle(tcell.StyleDefault.
				Background(ui.ThemeHighlightBg).
				Foreground(tcell.ColorBlack),
			)
			btn.SetActivatedStyle(tcell.StyleDefault.
				Background(ui.ThemeHighlightBg).
				Foreground(tcell.ColorBlack),
			)
		} else {
			btn.SetStyle(tcell.StyleDefault.
				Background(tcell.ColorDarkGray).
				Foreground(tcell.ColorGray),
			)
			btn.SetActivatedStyle(tcell.StyleDefault.
				Background(tcell.ColorGray).
				Foreground(tcell.ColorWhite),
			)
		}
	}
}

// blurAll sets all buttons to the blur (unfocused) style.
func (n *NavBar) blurAll() {
	for _, btn := range n.buttons {
		btn.SetStyle(tcell.StyleDefault.
			Background(tcell.ColorDarkGray).
			Foreground(tcell.ColorGray),
		)
		btn.SetActivatedStyle(tcell.StyleDefault.
			Background(tcell.ColorGray).
			Foreground(tcell.ColorWhite),
		)
	}
}

// SetOnSelect sets the callback for when a navigation item is selected.
func (n *NavBar) SetOnSelect(callback func(int, string)) {
	n.onSelect = callback
}

// GetCurrentItem returns the currently selected item index and name.
func (n *NavBar) GetCurrentItem() (int, string) {
	if n.currentIndex >= 0 && n.currentIndex < len(n.items) {
		return n.currentIndex, n.items[n.currentIndex].Name
	}
	return -1, ""
}

// SelectItem selects a specific item by index.
func (n *NavBar) SelectItem(index int) {
	if index >= 0 && index < len(n.items) {
		n.currentIndex = index
		n.highlightCurrent()
	}
}
