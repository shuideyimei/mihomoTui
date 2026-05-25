package ruleproviders

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

// initRuleProvidersTestOnce ensures test setup runs only once.
var initRuleProvidersTestOnce sync.Once

func initRuleProvidersTest() {
	initRuleProvidersTestOnce.Do(func() {
		// Init API client so api.Client is non-nil (used by refresh via Activate).
		api.InitClient("http://127.0.0.1:1", "")

		// Init UI updater so showError / showStatus calls don't panic.
		// tview.NewApplication() is safe in tests — it only allocates internal queues.
		ui.InitUpdater(tview.NewApplication())
	})
}

// ---------------------------------------------------------------------------
// TestNewPage_NonNil
// ---------------------------------------------------------------------------

func TestNewPage_NonNil(t *testing.T) {
	initRuleProvidersTest()

	page := NewPage()
	if page == nil {
		t.Fatal("NewPage() returned nil")
	}
}

// ---------------------------------------------------------------------------
// TestPage_ActivateDeactivate_GoroutineLeak
// ---------------------------------------------------------------------------

func TestPage_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	initRuleProvidersTest()

	page := NewPage()
	baseline := runtime.NumGoroutine()

	// Activate spawns a goroutine via go p.refresh().  refresh() will try to
	// read the mihomo config (which does not exist in this test environment)
	// and fail, calling showError which queues a UI update in a goroutine.
	// Both goroutines should terminate quickly.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Activate panicked: %v", r)
			}
		}()
		page.Activate()
	}()

	// Give spawned goroutines a moment to start and finish.
	time.Sleep(50 * time.Millisecond)

	// Deactivate is a no-op but should not panic.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Deactivate panicked: %v", r)
			}
		}()
		page.Deactivate()
	}()

	// Wait for all goroutines to settle back to baseline.
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			got := runtime.NumGoroutine()
			t.Fatalf(
				"goroutine count did not return to baseline within deadline: initial=%d, current=%d",
				baseline, got,
			)
		case <-tick.C:
			if runtime.NumGoroutine() <= baseline+5 {
				return
			}
		}
	}
}
