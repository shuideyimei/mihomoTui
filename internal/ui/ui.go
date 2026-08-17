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

type statusBar interface {
	GetCurrentMode() string
	UpdateConfig(config interface{})
}

type UiUpdater struct {
	app      *tview.Application
	statBar  statusBar
	overlays *tview.Pages

	queueMu   sync.Mutex
	queueCond *sync.Cond
	queue     []func()
}

func InitUpdater(app *tview.Application) {
	once.Do(func() {
		Updater = &UiUpdater{
			app: app,
		}
		Updater.queueCond = sync.NewCond(&Updater.queueMu)
		go Updater.run()
	})
}

func (u *UiUpdater) run() {
	for {
		u.queueMu.Lock()
		for len(u.queue) == 0 {
			u.queueCond.Wait()
		}
		fn := u.queue[0]
		u.queue[0] = nil
		u.queue = u.queue[1:]
		u.queueMu.Unlock()
		u.app.QueueUpdateDraw(fn)
	}
}

func (u *UiUpdater) GetApp() *tview.Application {
	return u.app
}

func (u *UiUpdater) SetStatusBar(statusBar statusBar) {
	u.statBar = statusBar
}

func (u *UiUpdater) SetOverlayPages(pages *tview.Pages) {
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

func (u *UiUpdater) GetCurrentMode() string {
	if u.statBar != nil {
		return u.statBar.GetCurrentMode()
	}
	return ""
}

func (u *UiUpdater) UpdateStatusBarConfig(config interface{}) {
	if u.statBar != nil {
		u.PostUi(func() {
			u.statBar.UpdateConfig(config)
		})
	}
}

func (u *UiUpdater) UpdateUi(fn func()) {
	u.app.QueueUpdateDraw(fn)
}

// PostUi queues a UI update without blocking the caller.
// Safe to call from any goroutine. The function will run on
// the UI goroutine and the screen will be redrawn afterwards.
func (u *UiUpdater) PostUi(fn func()) {
	u.queueMu.Lock()
	u.queue = append(u.queue, fn)
	u.queueMu.Unlock()
	u.queueCond.Signal()
}

func (u *UiUpdater) SetFocus(focusable tview.Primitive) {
	u.PostUi(func() {
		u.app.SetFocus(focusable)
	})
}
