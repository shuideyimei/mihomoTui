package app

import "github.com/gdamore/tcell/v2"

// handleGlobalKeys handles global keyboard shortcuts
func (a *App) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	// Handle Escape key to return to sidebar (only when not on sidebar)
	if event.Key() == tcell.KeyEscape && !a.focusOnSidebar {
		a.setFocus(true)
		return nil
	}

	// Global keys that work regardless of focus
	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
		return nil
	case tcell.KeyF1:
		a.switchPage(0) // Dashboard
		return nil
	case tcell.KeyF2:
		a.switchPage(1) // Proxies
		return nil
	case tcell.KeyF3:
		a.switchPage(2) // Connections
		return nil
	case tcell.KeyF4:
		a.switchPage(3) // Config
		return nil
	case tcell.KeyF5:
		a.switchPage(4) // Logs
		return nil
	case tcell.KeyF6:
		a.switchPage(5) // Subscriptions
		return nil
	case tcell.KeyF7:
		a.switchPage(6) // Proxy Groups
		return nil
	}

	// Handle Ctrl + number keys and Ctrl+Q
	if event.Modifiers()&tcell.ModCtrl != 0 {
		switch event.Rune() {
		case '1':
			a.switchPage(0) // Ctrl+1: Dashboard
			return nil
		case '2':
			a.switchPage(1) // Ctrl+2: Proxies
			return nil
		case '3':
			a.switchPage(2) // Ctrl+3: Connections
			return nil
		case '4':
			a.switchPage(3) // Ctrl+4: Config
			return nil
		case '5':
			a.switchPage(4) // Ctrl+5: Logs
			return nil
		case '6':
			a.switchPage(5) // Ctrl+6: Subscriptions
			return nil
		case '7':
			a.switchPage(6) // Ctrl+7: Proxy Groups
			return nil
		case 'q', 'Q':
			a.Stop() // Ctrl+Q: Quit
			return nil
		}
	}

	// Handle Alt + letter keys
	if event.Modifiers()&tcell.ModAlt != 0 {
		switch event.Rune() {
		case 'd', 'D':
			a.switchPage(0)
			return nil
		case 'p', 'P':
			a.switchPage(1)
			return nil
		case 'r', 'R':
			a.switchPage(2)
			return nil
		case 'c', 'C':
			a.switchPage(3)
			return nil
		case 'l', 'L':
			a.switchPage(4)
			return nil
		case 's', 'S':
			a.switchPage(5) // Alt+S: Subscriptions
			return nil
		case 'g', 'G':
			a.switchPage(6) // Alt+G: Proxy Groups
			return nil
		}
	}

	return event
}
