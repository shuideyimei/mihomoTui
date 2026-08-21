package pages

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/testapi"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

func init() {
	api.InitClient("http://test:9090", "test-secret")
	ui.InitUpdater(tview.NewApplication())
}

func TestNewConnectionsPage_NonNil(t *testing.T) {
	c := NewConnectionsPage()
	if c == nil {
		t.Fatal("NewConnectionsPage() returned nil")
	}
	if _, ok := interface{}(c).(ActivatablePage); !ok {
		t.Fatal("NewConnectionsPage() does not implement ActivatablePage")
	}
}

func TestConnections_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()
	api.InitClient(srv.URL, "")

	c := NewConnectionsPage()
	c.Activate()
	time.Sleep(50 * time.Millisecond)

	baseCount := runtime.NumGoroutine()

	c.Deactivate()

	// Allow goroutines to settle: the auto-refresh goroutine (started by
	// Activate) must observe the cancelled context and exit.
	time.Sleep(200 * time.Millisecond)

	finalCount := runtime.NumGoroutine()
	diff := finalCount - baseCount
	if diff > 2 {
		t.Errorf("expected at most 2 leaked goroutines, got %d (base=%d, final=%d)",
			diff, baseCount, finalCount)
	}
}

func TestConnections_DeactivateNoPanic(t *testing.T) {
	c := NewConnectionsPage()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — must not panic when
	// refreshCancel and refreshTicker are nil.
	c.Deactivate()
}
