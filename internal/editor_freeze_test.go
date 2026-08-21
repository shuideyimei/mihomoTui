package app

import (
	"testing"
	"time"

	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/pages"

	"github.com/gdamore/tcell/v2"
)

func TestEditorShiftTabLiveEventLoop(t *testing.T) {
	appInstance, _ := createTestApp(t)
	go appInstance.Run()
	defer appInstance.Stop()

	// Switch to Editor page (index 7)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(7)
	})
	time.Sleep(50 * time.Millisecond)

	_, ok := appInstance.pageLifecycle["editor"].(*pages.EditorPage)
	if !ok {
		t.Fatal("editor page not found")
	}

	done := make(chan bool, 1)
	go func() {
		// Send Shift+Tab event to the application
		// In a real terminal, Shift+Tab can arrive as KeyBacktab or KeyTab+ModShift
		backtab := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		appInstance.app.QueueEvent(backtab)
		time.Sleep(50 * time.Millisecond)

		shiftTab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModShift)
		appInstance.app.QueueEvent(shiftTab)
		time.Sleep(50 * time.Millisecond)

		tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		appInstance.app.QueueEvent(tab)
		time.Sleep(50 * time.Millisecond)

		done <- true
	}()

	select {
	case <-done:
		t.Log("Shift+Tab and Tab event loop completed successfully")
	case <-time.After(2 * time.Second):
		t.Fatal("Shift+Tab deadlocked the application event loop!")
	}
}
