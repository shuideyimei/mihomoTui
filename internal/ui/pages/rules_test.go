package pages

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Compile-time check: verify Rules implements ActivatablePage.
var _ ActivatablePage = (*Rules)(nil)

func TestNewRules_NonNil(t *testing.T) {
	r := NewRules()
	if r == nil {
		t.Fatal("NewRules() returned nil")
	}
	if r.RulesPage == nil {
		t.Fatal("RulesPage is nil after NewRules()")
	}

	// Verify embedded Flex layout exists
	if r.RulesPage.Flex == nil {
		t.Fatal("Flex layout is nil after NewRulesPage()")
	}

	// Verify sub-components are non-nil
	if r.RulesPage.table == nil {
		t.Error("table should not be nil")
	}
	if r.RulesPage.searchInput == nil {
		t.Error("searchInput should not be nil")
	}
	if r.RulesPage.statusText == nil {
		t.Error("statusText should not be nil")
	}
	if r.RulesPage.actionBar == nil {
		t.Error("actionBar should not be nil")
	}

	// Verify title is set (from setupUI)
	if title := r.RulesPage.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

func TestRules_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	// Initialize API client so api.Client.GetRules doesn't panic.
	api.InitClient("http://127.0.0.1:1", "")

	// Need a running tview application so QueueUpdateDraw calls do not hang.
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)

	// Start the application event loop in the background.
	go app.Run()
	time.Sleep(10 * time.Millisecond)

	r := NewRules()
	before := runtime.NumGoroutine()

	// Activate starts refresh() which launches goroutines (UpdateUi + API call).
	r.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate clears internal state.
	r.Deactivate()
	time.Sleep(100 * time.Millisecond)

	// Stop the application event loop so Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

func TestRules_DeactivateNoPanic(t *testing.T) {
	r := NewRules()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	r.Deactivate()
}
