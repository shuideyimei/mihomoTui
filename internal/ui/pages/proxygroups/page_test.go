package proxygroups

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ---------------------------------------------------------------------------
// TestNewPage_NonNil
// ---------------------------------------------------------------------------

func TestNewPage_NonNil(t *testing.T) {
	p := NewPage()
	if p == nil {
		t.Fatal("NewPage() returned nil")
	}

	// Verify Flex layout exists (embedded via Page)
	if p.Flex == nil {
		t.Fatal("Flex layout is nil after NewPage()")
	}

	// Verify all layout sub-components are non-nil
	if p.groupList == nil {
		t.Error("groupList should not be nil")
	}
	if p.detailView == nil {
		t.Error("detailView should not be nil")
	}
	if p.actionBar == nil {
		t.Error("actionBar should not be nil")
	}
	if p.status == nil {
		t.Error("status should not be nil")
	}
	if p.nameInput == nil {
		t.Error("nameInput should not be nil")
	}
	if p.typeSelect == nil {
		t.Error("typeSelect should not be nil")
	}
	if p.proxiesInput == nil {
		t.Error("proxiesInput should not be nil")
	}
	if p.useInput == nil {
		t.Error("useInput should not be nil")
	}
	if p.urlInput == nil {
		t.Error("urlInput should not be nil")
	}
	if p.filterInput == nil {
		t.Error("filterInput should not be nil")
	}
	if p.groupForm == nil {
		t.Error("groupForm should not be nil")
	}

	// Verify title is set (from setupUI)
	if title := p.GetTitle(); title == "" {
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
	// ui.Updater.UpdateUi → app.QueueUpdateDraw is processed.
	go app.Run()

	p := NewPage()
	before := runtime.NumGoroutine()

	// Activate starts: go p.refresh() which calls config.GetAllProxyGroupsFromConfig()
	// (will fail quickly — no config) then queues a UI update via ui.Updater.UpdateUi.
	p.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate is a no-op for this page, but call it to exercise the method.
	p.Deactivate()
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
	p := NewPage()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	p.Deactivate()
}
