package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// handleGlobalKeys handles global keyboard shortcuts
func (a *App) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlC ||
		(event.Modifiers()&tcell.ModCtrl != 0 && (event.Rune() == 'q' || event.Rune() == 'Q')) {
		a.Stop()
		return nil
	}

	if a.showingHelp {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF12 || event.Rune() == '?' {
			a.hideHelp()
			return nil
		}
		return event
	}

	if a.showingPalette {
		if event.Key() == tcell.KeyEscape {
			a.hideCommandPalette()
			return nil
		}
		return event
	}

	if event.Key() == tcell.KeyEscape {
		if _, isForm := a.app.GetFocus().(*tview.Form); isForm {
			a.setFocus(true)
			return nil
		}
	}

	// Inputs, forms, drop-downs, and dialogs own their keys before global
	// navigation. This prevents ?, Esc, and function keys from interrupting edits.
	switch a.app.GetFocus().(type) {
	case *tview.InputField, *tview.TextArea, *tview.Form, *tview.DropDown, *tview.Modal:
		return event
	}

	// Check if the event is a rune event (printable character)
	// ? can come as Rune '?' directly, or as '/' with Shift modifier on some terminals
	isQuestionMark := event.Rune() == '?'
	if event.Key() == tcell.KeyRune && event.Rune() == '/' && event.Modifiers()&tcell.ModShift != 0 {
		isQuestionMark = true
	}

	if isQuestionMark {
		a.toggleHelp()
		return nil
	}

	if event.Rune() == ':' {
		a.showCommandPalette()
		return nil
	}

	// F12 as alternative help shortcut
	if event.Key() == tcell.KeyF12 {
		a.toggleHelp()
		return nil
	}

	// Handle Escape key to return to navigation bar (only when not on navbar)
	if event.Key() == tcell.KeyEscape && !a.focusOnNav {
		a.setFocus(true)
		return nil
	}

	// Global keys that work regardless of focus
	switch event.Key() {
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
		if a.currentPage == 2 {
			return event
		}
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

// toggleHelp toggles the help overlay visibility
func (a *App) toggleHelp() {
	if a.showingHelp {
		a.hideHelp()
	} else {
		a.showHelp()
	}
}

// showHelp displays the help overlay
func (a *App) showHelp() {
	a.helpReturnFocus = a.app.GetFocus()
	a.rootPages.ShowPage("help")
	a.showingHelp = true
	a.app.SetFocus(a.helpPage)
}

// hideHelp hides the help overlay and restores focus
func (a *App) hideHelp() {
	a.rootPages.HidePage("help")
	a.showingHelp = false
	if a.helpReturnFocus != nil {
		a.app.SetFocus(a.helpReturnFocus)
		a.helpReturnFocus = nil
		return
	}
	a.setFocus(a.focusOnNav)
}
