package ui

import (
	"sync"

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
	app     *tview.Application
	statBar statusBar
}

func InitUpdater(app *tview.Application) {
	once.Do(func() {
		Updater = &UiUpdater{
			app: app,
		}
	})
}

func (u *UiUpdater) SetStatusBar(statusBar statusBar) {
	u.statBar = statusBar
}

func (u *UiUpdater) GetCurrentMode() string {
	if u.statBar != nil {
		return u.statBar.GetCurrentMode()
	}
	return ""
}

func (u *UiUpdater) UpdateStatusBarConfig(config interface{}) {
	if u.statBar != nil {
		u.statBar.UpdateConfig(config)
	}
}

func (u *UiUpdater) UpdateUi(fn func()) {
	u.app.QueueUpdateDraw(fn)
}

// PostUi queues a UI update without blocking the caller.
// Safe to call from any goroutine. The function will run on
// the UI goroutine and the screen will be redrawn afterwards.
func (u *UiUpdater) PostUi(fn func()) {
	go u.app.QueueUpdateDraw(fn)
}

func (u *UiUpdater) SetFocus(focusable tview.Primitive) {
	go u.app.QueueUpdateDraw(func() {
		u.app.SetFocus(focusable)
	})
}
