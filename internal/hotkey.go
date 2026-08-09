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
		// Except we want Esc to exit input fields and go to navbar, unless it's a Modal
		if event.Key() == tcell.KeyEscape {
			if _, isModal := a.app.GetFocus().(*tview.Modal); isModal {
				return event
			}
			// Let Esc fall through to global handler
			break
		}
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
		a.switchPage(3) // Logs
		return nil
	case tcell.KeyF5:
		if a.currentPage == 2 {
			return event
		}
		a.switchPage(5) // Profiles
		return nil
	case tcell.KeyF6:
		a.switchPage(6) // Subscriptions
		return nil
	case tcell.KeyF7:
		a.switchPage(7) // Editor
		return nil
	case tcell.KeyF8:
		a.switchPage(8) // Settings
		return nil
	}

	// Handle Ctrl + number keys
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
			a.switchPage(3) // Ctrl+4: Logs
			return nil
		case '5':
			a.switchPage(5) // Ctrl+5: Profiles
			return nil
		case '6':
			a.switchPage(6) // Ctrl+6: Subscriptions
			return nil
		case '7':
			a.switchPage(7) // Ctrl+7: Editor
			return nil
		case '8':
			a.switchPage(8) // Ctrl+8: Settings
			return nil
		}
	}

	// Handle Alt + number keys
	if event.Modifiers()&tcell.ModAlt != 0 {
		switch event.Rune() {
		case '1':
			a.switchPage(0)
			return nil
		case '2':
			a.switchPage(1)
			return nil
		case '3':
			a.switchPage(2)
			return nil
		case '4':
			a.switchPage(3)
			return nil
		case '5':
			a.switchPage(5)
			return nil
		case '6':
			a.switchPage(6)
			return nil
		case '7':
			a.switchPage(7)
			return nil
		case '8':
			a.switchPage(8)
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
