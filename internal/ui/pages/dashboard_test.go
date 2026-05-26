package pages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
)

// newDashboardMockServer creates an httptest.Server that serves the minimal
// endpoints needed by the Dashboard page:
//   - GET /configs     — used by periodicUpdates → updateProxyStatusData
//   - GET /connections — used by periodicUpdates → updateConnectionsData
//   - GET /memory      — blocks until request context is cancelled, simulating
//                        the long-lived SSE connection of the real /memory endpoint
//
// The /memory handler is designed so that startStreamMemoryUsage exits promptly
// when Deactivate cancels the dashboard's context, avoiding a 5-second retry sleep.
func newDashboardMockServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/configs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.Config{
			Port:      7890,
			SocksPort: 7891,
			MixedPort: 7890,
			AllowLan:  false,
			Mode:      "rule",
			LogLevel:  "info",
			Tun:       map[string]interface{}{"enable": true},
		})
	})

	mux.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Connections     []models.Connection `json:"connections"`
			ConnectionCount int                 `json:"connectionCount"`
		}{
			Connections:     []models.Connection{},
			ConnectionCount: 0,
		})
	})

	// /memory holds the connection open (simulating SSE) until the client
	// cancels the request context.  This lets startStreamMemoryUsage observe
	// the cancellation and exit without the 5s retry sleep.
	mux.HandleFunc("/memory", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	})

	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// TestNewDashboard_NonNil
// ---------------------------------------------------------------------------

func TestNewDashboard_NonNil(t *testing.T) {
	dash := NewDashboard()
	if dash == nil {
		t.Fatal("NewDashboard() returned nil")
	}
	// NewDashboard now returns *DashboardPage directly
	if _, ok := interface{}(dash).(ActivatablePage); !ok {
		t.Fatal("NewDashboard() does not implement ActivatablePage")
	}
}

// ---------------------------------------------------------------------------
// TestDashboard_ActivateDeactivate_GoroutineLeak
// ---------------------------------------------------------------------------

func TestDashboard_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	srv := newDashboardMockServer()
	defer srv.Close()
	api.UpdateClient(srv.URL, "")

	baseCount := runtime.NumGoroutine()

	dash := NewDashboard()
	dash.Activate()
	dash.Deactivate()

	// Give background goroutines time to observe the cancelled context and
	// exit.  periodicUpdates reads ctx.Done() and returns; the blocking
	// /memory mock unblocks when the request context is cancelled so
	// startStreamMemoryUsage also returns quickly.
	time.Sleep(500 * time.Millisecond)

	finalCount := runtime.NumGoroutine()
	diff := finalCount - baseCount
	if diff > 2 {
		t.Errorf("expected at most 2 leaked goroutines, got %d (base=%d, final=%d)",
			diff, baseCount, finalCount)
	}
}

// ---------------------------------------------------------------------------
// TestDashboard_BasicLayout
// ---------------------------------------------------------------------------

func TestDashboard_BasicLayout(t *testing.T) {
	dash := NewDashboard()
	if dash == nil {
		t.Fatal("NewDashboard() returned nil")
	}

	// All of these are created in setupLayout / create* helpers and must be
	// non-nil after construction.

	if dash.connectionsBox == nil {
		t.Error("connectionsBox is nil — expected *tview.TextView")
	}
	if dash.systemInfoBox == nil {
		t.Error("systemInfoBox is nil — expected *tview.TextView")
	}
	if dash.controlButtons == nil {
		t.Error("controlButtons is nil — expected *tview.Flex")
	}
	if dash.allowLanBtn == nil {
		t.Error("allowLanBtn is nil — expected *tview.Button")
	}
	if dash.tunBtn == nil {
		t.Error("tunBtn is nil — expected *tview.Button")
	}
	if dash.statusText == nil {
		t.Error("statusText is nil — expected *tview.TextView")
	}
	if dash.operationStatus == nil {
		t.Error("operationStatus is nil — expected *tview.TextView")
	}
	if dash.buttonNav == nil {
		t.Error("buttonNav is nil — expected *components.FocusNavigator")
	}
	if dash.Flex == nil {
		t.Error("Flex is nil — expected *tview.Flex")
	}
}

// ---------------------------------------------------------------------------
// TestDashboard_RefreshDoesNotPanic
// ---------------------------------------------------------------------------

func TestDashboard_RefreshDoesNotPanic(t *testing.T) {
	srv := newDashboardMockServer()
	defer srv.Close()
	api.UpdateClient(srv.URL, "")

	dash := NewDashboard()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Refresh panicked: %v", r)
		}
	}()

	// Refresh is synchronous (it launches a goroutine and returns), but the
	// goroutine it spawns calls updateConnectionsData, updateProxyStatusData
	// and updateSystemInfo, all of which make API calls and queue UI updates.
	dash.Refresh()

	// Allow the spawned goroutine to complete its work.
	time.Sleep(500 * time.Millisecond)
}
