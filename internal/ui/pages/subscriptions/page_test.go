package subscriptions

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ---------------------------------------------------------------------------
// TestNewPage_NonNil
// ---------------------------------------------------------------------------

func TestNewPage_NonNil(t *testing.T) {
	subMgr := subscription.NewManager()
	page := NewPage(subMgr)
	if page == nil {
		t.Fatal("NewPage() returned nil")
	}

	// Verify embedded Flex layout exists
	if page.Flex == nil {
		t.Fatal("Flex layout is nil after NewPage()")
	}

	// Verify sub-components are non-nil
	if page.subList == nil {
		t.Error("subList should not be nil")
	}
	if page.detailView == nil {
		t.Error("detailView should not be nil")
	}
	if page.actionBar == nil {
		t.Error("actionBar should not be nil")
	}
	if page.status == nil {
		t.Error("status should not be nil")
	}
	if page.nameInput == nil {
		t.Error("nameInput should not be nil")
	}
	if page.urlInput == nil {
		t.Error("urlInput should not be nil")
	}
	if page.intervalInput == nil {
		t.Error("intervalInput should not be nil")
	}
	if page.importForm == nil {
		t.Error("importForm should not be nil")
	}

	// Verify title is set (from setupUI)
	if title := page.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

// ---------------------------------------------------------------------------
// TestPage_ActivateDeactivate_GoroutineLeak
// ---------------------------------------------------------------------------

func TestPage_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	// We need a running tview application so that QueueUpdateDraw calls
	// (which block until the event loop processes them) do not hang.
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)

	// Start the application event loop in the background so that
	// ui.Updater.UpdateUi -> app.QueueUpdateDraw is processed.
	go app.Run()

	subMgr := subscription.NewManager()
	page := NewPage(subMgr)
	before := runtime.NumGoroutine()

	// Activate starts: go p.refresh() which calls config.GetMihomoProxyProviders()
	// (will fail quickly — no config) then queues a UI update via ui.Updater.UpdateUi.
	page.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate is a no-op for this page, but call it to exercise the method.
	page.Deactivate()
	time.Sleep(100 * time.Millisecond) // allow goroutines to settle

	// Stop the application event loop so Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

// ---------------------------------------------------------------------------
// TestPage_DeactivateNoPanic
// ---------------------------------------------------------------------------

func TestPage_DeactivateNoPanic(t *testing.T) {
	subMgr := subscription.NewManager()
	page := NewPage(subMgr)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	page.Deactivate()
}
