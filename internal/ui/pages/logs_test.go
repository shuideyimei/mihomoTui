package pages

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

// initOnce ensures ui.Updater is initialized only once for all tests in this package.
// Using a separate sync.Once because ui.InitUpdater uses its own sync.Once internally,
// but we call it here to guarantee initialization before any Logs test runs.
var initLogsTestOnce sync.Once

func initLogsTest() {
	initLogsTestOnce.Do(func() {
		// Init API client so api.StreamClient is non-nil (required by startLogStream).
		api.InitClient("http://127.0.0.1:1", "")

		// Init UI updater so PostUi calls don't panic.
		// tview.NewApplication() is safe in tests — it only allocates internal queues.
		ui.InitUpdater(tview.NewApplication())
	})
}

func TestNewLogs_NonNil(t *testing.T) {
	initLogsTest()

	l := NewLogs()
	if l == nil {
		t.Fatal("NewLogs() returned nil")
	}
	if l.LogsPage == nil {
		t.Fatal("NewLogs().LogsPage is nil")
	}
}

func TestLogs_ActivateDeactivate_StreamLifecycle(t *testing.T) {
	initLogsTest()

	page := NewLogs()
	baseline := runtime.NumGoroutine()

	// Activate starts a background log stream goroutine.
	// The stream cannot connect (127.0.0.1:1 refuses / has nothing listening),
	// so it should fail gracefully without panic.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Activate panicked: %v", r)
			}
		}()
		page.Activate()
	}()

	// Give the goroutine time to start and attempt the connection.
	time.Sleep(50 * time.Millisecond)

	// Deactivate cancels the context — the goroutine should exit promptly
	// (if the HTTP request is in-flight, context cancellation aborts it;
	// if the request already failed and the goroutine is in the error sleep,
	// it will exit up to ~5s later).
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Deactivate panicked: %v", r)
			}
		}()
		page.Deactivate()
	}()

	// Wait for goroutines to settle back to baseline.
	// The stream goroutine exits immediately when the context is cancelled
	// while the HTTP request is in-flight, but if the request failed before
	// Deactivate was called, the goroutine enters a 5s error sleep.
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

func TestLogs_ClearDoesNotPanic(t *testing.T) {
	initLogsTest()

	page := NewLogs()

	// Activate first to get some internal state populated (addLog is called).
	page.Activate()
	time.Sleep(30 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Clear panicked from goroutine: %v", r)
			}
		}()
		page.Clear()
	}()

	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("Clear timed out (possible deadlock)")
	}

	page.Deactivate()
}

func TestLogs_TogglePause(t *testing.T) {
	initLogsTest()

	page := NewLogs()

	// Start the log stream.
	page.Activate()
	time.Sleep(50 * time.Millisecond)

	// Toggle to pause — should cancel the current stream without starting a new one.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("togglePause (pause) panicked: %v", r)
			}
		}()
		page.togglePause()
	}()

	page.LogsPage.mu.Lock()
	paused := page.LogsPage.isPaused
	page.LogsPage.mu.Unlock()
	if !paused {
		t.Error("expected isPaused=true after togglePause")
	}

	// Toggle to resume — should start a new stream.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("togglePause (resume) panicked: %v", r)
			}
		}()
		page.togglePause()
	}()

	page.LogsPage.mu.Lock()
	paused = page.LogsPage.isPaused
	page.LogsPage.mu.Unlock()
	if paused {
		t.Error("expected isPaused=false after second togglePause")
	}

	// Clean up.
	page.Deactivate()
}

// Ensure Logs satisfies the ActivatablePage interface at compile time.
var _ ActivatablePage = (*Logs)(nil)
