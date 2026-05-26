package pages

import (
	"log"
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	api.InitClient("http://test:9090", "test-secret")

	// Start a tview app with a simulation screen so QueueUpdateDraw calls
	// (which block until the event loop runs the function) do not hang.
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		panic(err)
	}
	app := tview.NewApplication().SetScreen(screen)
	ui.InitUpdater(app)
	go func() {
		if err := app.Run(); err != nil {
			log.Printf("tview app exited with error: %v", err)
		}
	}()
	time.Sleep(10 * time.Millisecond)
}

func TestNewConfigPage_NonNil(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)
	if page == nil {
		t.Fatal("NewConfigPage(config.NewManager()) returned nil")
	}
}

func TestNewConfig_NonNil(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfig(cfg)
	if page == nil {
		t.Fatal("NewConfig(config.NewManager()) returned nil")
	}
}

func TestConfigPage_FormInputFields(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)
	if page.form == nil {
		t.Fatal("NewConfigPage returned nil form")
	}

	tests := []struct {
		name  string
		label string
	}{
		{name: "API URL field", label: "API地址"},
		{name: "API Secret field", label: "API密钥"},
		{name: "HTTP Port field", label: "HTTP端口"},
		{name: "SOCKS Port field", label: "SOCKS端口"},
		{name: "Mixed Port field", label: "混合端口"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := page.form.GetFormItemByLabel(tt.label)
			if item == nil {
				t.Fatalf("form item with label %q not found", tt.label)
			}
			if _, ok := item.(*tview.InputField); !ok {
				t.Errorf("form item %q is not an *tview.InputField (got %T)", tt.label, item)
			}
		})
	}
}

func TestConfigPage_FormButtons(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)
	if page.form == nil {
		t.Fatal("NewConfigPage returned nil form")
	}

	tests := []struct {
		name  string
		index int
		label string
	}{
		{name: "Save button", index: 0, label: "保存"},
		{name: "Reset button", index: 1, label: "重置"},
		{name: "Test Connection button", index: 2, label: "测试连接"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			btn := page.form.GetButton(tt.index)
			if btn == nil {
				t.Fatalf("button at index %d not found", tt.index)
			}
			if got := btn.GetLabel(); got != tt.label {
				t.Errorf("button at index %d: expected label %q, got %q", tt.index, tt.label, got)
			}
		})
	}
}

func TestConfigPage_ActivateDoesNotPanic(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfig(cfg)

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Activate panicked: %v", r)
			}
			close(done)
		}()
		page.Activate()
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Activate timed out (background goroutine may be hung)")
	}
}

func TestConfigPage_DeactivateDoesNotPanic(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)
	page.Activate()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()
	page.Deactivate()
}

func TestConfigPage_GoroutineLeak(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfig(cfg)

	before := runtime.NumGoroutine()

	const cycles = 5
	for i := 0; i < cycles; i++ {
		page.Activate()
		page.Deactivate()
	}

	deadline := time.After(15 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			after := runtime.NumGoroutine()
			t.Errorf("goroutine count did not settle: before=%d, after=%d (diff=%d)",
				before, after, after-before)
			return
		case <-tick.C:
			if runtime.NumGoroutine() <= before+5 {
				return
			}
		}
	}
}

var _ ActivatablePage = (*ConfigPage)(nil)

func TestConfigPage_Deactivate(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	page.Deactivate()
}

func TestConfigPage_Layout(t *testing.T) {
	cfg := config.NewManager()
	page := NewConfigPage(cfg)

	if page.form == nil {
		t.Error("form should not be nil after construction")
	}
	if page.statusField == nil {
		t.Error("statusField should not be nil after construction")
	}
	if title := page.form.GetTitle(); title == "" {
		t.Error("form title should not be empty")
	}
}
