package app

import "github.com/gdamore/tcell/v2"

// handleGlobalKeys handles global keyboard shortcuts
func (a *App) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Rune() == '?' {
		if a.showingHelp {
			a.rootPages.HidePage("help")
			a.showingHelp = false
		} else {
			a.rootPages.ShowPage("help")
			a.showingHelp = true
		}
		return nil
	}

	if event.Key() == tcell.KeyEscape && a.showingHelp {
		a.rootPages.HidePage("help")
		a.showingHelp = false
		return nil
	}

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
	case tcell.KeyF8:
		a.switchPage(7) // Rules
		return nil
	case tcell.KeyF9:
		a.switchPage(8) // Rule Providers
		return nil
	case tcell.KeyF10:
		a.switchPage(9) // Settings
		return nil
	case tcell.KeyF11:
		a.switchPage(10)
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
		case '8':
			a.switchPage(7) // Ctrl+8: Rules
			return nil
		case '9':
			a.switchPage(8) // Ctrl+9: Rule Providers
			return nil
		case '0':
			a.switchPage(9) // Ctrl+0: Settings
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
		case 'u', 'U':
			a.switchPage(7) // Alt+U: Rules
			return nil
		case 't', 'T':
			a.switchPage(8) // Alt+T: Rule Providers
			return nil
		case 'k', 'K':
			a.switchPage(9) // Alt+K: Settings
			return nil
		case 'm', 'M':
			a.switchPage(10) // Alt+M: Config Manager
			return nil
		}
	}

	return event
}
