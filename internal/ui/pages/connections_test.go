package pages

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

func init() {
	api.InitClient("http://test:9090", "test-secret")
	ui.InitUpdater(tview.NewApplication())
}

func TestNewConnections_NonNil(t *testing.T) {
	c := NewConnections()
	if c == nil {
		t.Fatal("NewConnections() returned nil")
	}
	if c.ConnectionsPage == nil {
		t.Fatal("NewConnections().ConnectionsPage is nil")
	}
}

func TestConnections_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	// Record baseline goroutine count before creating any pages
	baseCount := runtime.NumGoroutine()

	c := NewConnections()
	c.Activate()
	c.Deactivate()

	// Allow goroutines to settle: the auto-refresh goroutine (started by
	// Activate) must observe the cancelled context and exit, and any
	// inflight API call or queued UI update must complete.
	time.Sleep(200 * time.Millisecond)

	finalCount := runtime.NumGoroutine()
	diff := finalCount - baseCount
	if diff > 2 {
		t.Errorf("expected at most 2 leaked goroutines, got %d (base=%d, final=%d)",
			diff, baseCount, finalCount)
	}
}

func TestConnections_DeactivateNoPanic(t *testing.T) {
	c := NewConnections()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — must not panic when
	// refreshCancel and refreshTicker are nil.
	c.Deactivate()
}
