package components

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mihomoTui/internal/events"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestStatusBar_SetBusAndPublishMode(t *testing.T) {
	bus := events.NewBus()
	sb := NewStatusBar(bus)

	var receivedMode string
	var called int32

	bus.Subscribe(events.TopicProxyModeChanged, func(ev interface{}) {
		if event, ok := ev.(events.ProxyModeChangedEvent); ok {
			receivedMode = event.Mode
			atomic.AddInt32(&called, 1)
		}
	})

	// Updating config with non-empty mode should publish to bus
	sb.UpdateConfig(&models.Config{Mode: "rule"})

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected 1 event published, got %d", called)
	}
	if receivedMode != "rule" {
		t.Fatalf("expected mode rule, got %q", receivedMode)
	}
}

func TestStatusBar_ReceiveModeChangedEvent(t *testing.T) {
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)
	go app.Run()
	defer app.Stop()

	bus := events.NewBus()
	sb := NewStatusBar()
	sb.SetBus(bus)

	bus.Publish(events.TopicProxyModeChanged, events.ProxyModeChangedEvent{Mode: "global"})
	time.Sleep(50 * time.Millisecond)

	mode := sb.GetCurrentMode()
	if mode != "global" {
		t.Fatalf("expected mode global, got %q", mode)
	}

	done := make(chan struct{})
	app.QueueUpdateDraw(func() {
		text := sb.GetText(false)
		if !strings.Contains(text, "global") {
			t.Errorf("expected status bar text to contain global, got %q", text)
		}
		close(done)
	})
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for UI update")
	}
}
