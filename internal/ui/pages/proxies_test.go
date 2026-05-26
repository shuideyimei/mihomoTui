package pages

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Compile-time check: verify ProxiesPage implements ActivatablePage.
var _ ActivatablePage = (*ProxiesPage)(nil)

func TestNewProxies_NonNil(t *testing.T) {
	p := NewProxies()
	if p == nil {
		t.Fatal("NewProxies() returned nil")
	}
	// Verify Flex layout exists (embedded via NewProxiesPage now returns *ProxiesPage)
	if p.Flex == nil {
		t.Fatal("Flex layout is nil after NewProxiesPage()")
	}

	// Verify all layout sub-components are non-nil
	if p.groupsList == nil {
		t.Error("groupsList should not be nil")
	}
	if p.switchButtons == nil {
		t.Error("switchButtons should not be nil")
	}
	if p.nodesList == nil {
		t.Error("nodesList should not be nil")
	}
	if p.statusText == nil {
		t.Error("statusText should not be nil")
	}
	if p.searchInput == nil {
		t.Error("searchInput should not be nil")
	}

	// Verify title is set (from setupLayout)
	if title := p.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

func TestProxies_ActivateDeactivate_GoroutineLeak(t *testing.T) {
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

	p := NewProxies()
	before := runtime.NumGoroutine()

	// Activate starts:
	//   1. loadProvidersData goroutine (calls UpdateUi → unblocked by event loop)
	//   2. startAutoRefresh goroutine (ticker + context)
	//   3. syncModeFromStatusBar goroutine (returns quickly, statBar is nil)
	p.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate stops the auto-refresh goroutine and cancels the context.
	p.Deactivate()
	time.Sleep(100 * time.Millisecond) // allow goroutines to settle

	// Stop the application event loop so the Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

func TestProxies_DeactivateNoPanic(t *testing.T) {
	p := NewProxies()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	p.Deactivate()
}
