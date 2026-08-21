package ui

import (
	"fmt"
	"sync"

	"mihomoTui/internal/i18n"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	once    sync.Once
	Updater *UiUpdater
)

type UiUpdater struct {
	mu       sync.RWMutex
	app      *tview.Application
	overlays *tview.Pages
}

func InitUpdater(app *tview.Application) {
	once.Do(func() {
		Updater = &UiUpdater{}
	})
	if Updater != nil {
		Updater.mu.Lock()
		Updater.app = app
		Updater.mu.Unlock()
	}
}

func (u *UiUpdater) GetApp() *tview.Application {
	if u == nil {
		return nil
	}
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.app
}

func (u *UiUpdater) SetOverlayPages(pages *tview.Pages) {
	if u == nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.overlays = pages
}

// ShowConfirm displays a shared confirmation dialog above the current page.
func (u *UiUpdater) ShowConfirm(title, message string, onConfirm func()) {
	u.showConfirm(title, message, i18n.T("common.confirm"), onConfirm)
}

func (u *UiUpdater) showConfirm(title, message, confirmLabel string, onConfirm func()) {
	u.PostUi(func() {
		if u.overlays == nil {
			return
		}
		returnFocus := u.app.GetFocus()
		closeModal := func() {
			u.overlays.RemovePage("confirm")
			if returnFocus != nil {
				u.app.SetFocus(returnFocus)
			}
		}
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{i18n.T("common.cancel"), confirmLabel}).
			SetDoneFunc(func(buttonIndex int, _ string) {
				closeModal()
				if buttonIndex == 1 && onConfirm != nil {
					onConfirm()
				}
			})
		modal.SetTitle(" " + title + " ")
		modal.SetTitleAlign(tview.AlignCenter)
		modal.SetBorderColor(ThemeFocusColor)
		modal.SetButtonBackgroundColor(ThemeButtonBg)
		modal.SetButtonTextColor(tcell.ColorWhite)
		modal.SetButtonActivatedStyle(tcell.StyleDefault.Background(ThemeFocusColor).Foreground(tcell.ColorBlack))
		modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape {
				closeModal()
				return nil
			}
			return event
		})
		u.overlays.AddPage("confirm", modal, true, true)
		u.app.SetFocus(modal)
	})
}

func (u *UiUpdater) ConfirmDelete(label string, onConfirm func()) {
	u.showConfirm(
		i18n.T("common.delete_confirm_title"),
		fmt.Sprintf(i18n.T("common.delete_confirm"), label),
		i18n.T("common.delete"),
		onConfirm,
	)
}

func (u *UiUpdater) UpdateUi(fn func()) {
	u.PostUi(fn)
}

// PostUi queues a UI update without blocking the caller.
// Safe to call from any goroutine. The function will run on
// the UI goroutine and the screen will be redrawn afterwards.
func (u *UiUpdater) PostUi(fn func()) {
	if u == nil {
		return
	}
	u.mu.RLock()
	app := u.app
	u.mu.RUnlock()
	if app == nil {
		return
	}
	go func() {
		defer func() {
			_ = recover()
		}()
		app.QueueUpdateDraw(fn)
	}()
}

func (u *UiUpdater) SetFocus(focusable tview.Primitive) {
	if u == nil || focusable == nil {
		return
	}
	u.PostUi(func() {
		u.mu.RLock()
		app := u.app
		u.mu.RUnlock()
		if app != nil {
			app.SetFocus(focusable)
		}
	})
}
