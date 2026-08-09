package components

import (
	"fmt"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NavBar is a vertical page navigator. The active page marker remains visible
// even while another component owns keyboard focus.
type NavBar struct {
	*tview.List
	items        []ui.PageInfo
	onSelect     func(int, string)
	currentIndex int
}

func NewNavBar(items []ui.PageInfo) *NavBar {
	n := &NavBar{
		List:         tview.NewList(),
		items:        append([]ui.PageInfo(nil), items...),
		currentIndex: 0,
	}
	n.SetBorder(true)
	n.SetTitle(" " + i18n.T("nav.title") + " ")
	n.ShowSecondaryText(true)
	n.SetHighlightFullLine(true)
	StyleSelectableList(n.List)
	n.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		n.onItemSelected(index)
	})
	n.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(n.items) && n.items[index].Separator {
			// Find the next valid item depending on direction
			// For simplicity, just find the next non-separator downward
			next := index + 1
			for next < len(n.items) && n.items[next].Separator {
				next++
			}
			if next < len(n.items) {
				n.SetCurrentItem(next)
			} else {
				// Try backward
				prev := index - 1
				for prev >= 0 && n.items[prev].Separator {
					prev--
				}
				if prev >= 0 {
					n.SetCurrentItem(prev)
				}
			}
		}
	})
	n.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight, tcell.KeyEnter:
			if !n.items[n.GetCurrentItem()].Separator {
				n.onItemSelected(n.GetCurrentItem())
			}
			return nil
		}
		return event
	})
	n.rebuild()
	return n
}

func (n *NavBar) rebuild() {
	cursor := n.GetCurrentItem()
	n.Clear()
	for index, item := range n.items {
		if item.Separator {
			n.AddItem("[gray]────────────────[-]  ", "", 0, nil)
			continue
		}
		marker := "  "
		if index == n.currentIndex {
			marker = "[green]●[-] "
		}
		secondary := item.Function
		if item.Alt != "" {
			secondary = fmt.Sprintf("%s  %s", item.Function, item.Alt)
		}
		n.AddItem(marker+item.Label(), secondary, 0, nil)
	}
	if cursor >= 0 && cursor < len(n.items) {
		if n.items[cursor].Separator {
			// Find next valid
			for i := cursor; i < len(n.items); i++ {
				if !n.items[i].Separator {
					n.SetCurrentItem(i)
					break
				}
			}
		} else {
			n.SetCurrentItem(cursor)
		}
	}
}

func (n *NavBar) onItemSelected(index int) {
	if index < 0 || index >= len(n.items) {
		return
	}
	n.currentIndex = index
	n.rebuild()
	if n.onSelect != nil {
		n.onSelect(index, n.items[index].Label())
	}
}

func (n *NavBar) SetOnSelect(callback func(int, string)) {
	n.onSelect = callback
}

func (n *NavBar) GetCurrentPage() (int, string) {
	if n.currentIndex >= 0 && n.currentIndex < len(n.items) {
		return n.currentIndex, n.items[n.currentIndex].Label()
	}
	return -1, ""
}

func (n *NavBar) SelectItem(index int) {
	if index >= 0 && index < len(n.items) {
		n.currentIndex = index
		n.rebuild()
	}
}
