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
