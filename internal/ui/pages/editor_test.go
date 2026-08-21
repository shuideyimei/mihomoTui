package pages

import (
	"strings"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"
	proxygroupspage "mihomoTui/internal/ui/pages/proxygroups"
	rproviders "mihomoTui/internal/ui/pages/ruleproviders"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func createTestEditorPage() (*EditorPage, *tview.Application) {
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		panic(err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)

	pg := proxygroupspage.NewPage()
	rules := NewRulesPage()
	rp := rproviders.NewPage()
	editor := NewEditorPage(pg, rules, rp)

	return editor, app
}

func TestEditorPage_NonNil(t *testing.T) {
	editor, _ := createTestEditorPage()
	if editor == nil {
		t.Fatal("NewEditorPage returned nil")
	}
	if editor.Flex == nil {
		t.Fatal("editor.Flex is nil")
	}
	if editor.tabBar == nil {
		t.Fatal("editor.tabBar is nil")
	}
	if editor.pages == nil {
		t.Fatal("editor.pages is nil")
	}
}

func TestEditorPage_SubtabSwitchingKeyboard(t *testing.T) {
	editor, app := createTestEditorPage()
	go app.Run()
	defer app.Stop()

	// Initial active tab should be 0 (proxygroups)
	if editor.activeTab != 0 {
		t.Fatalf("expected activeTab 0, got %d", editor.activeTab)
	}

	fn := editor.GetInputCapture()
	if fn == nil {
		t.Fatal("expected non-nil InputCapture")
	}

	ui.Updater.PostUi(func() {
		// Press Tab -> switch to 1 (rules)
		tabEvent := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		res := fn(tabEvent)
		if res != nil {
			t.Error("expected Tab event to be handled and return nil")
		}
		if editor.activeTab != 1 {
			t.Errorf("expected activeTab 1, got %d", editor.activeTab)
		}

		// Press Tab again -> switch to 2 (ruleproviders)
		fn(tabEvent)
		if editor.activeTab != 2 {
			t.Errorf("expected activeTab 2, got %d", editor.activeTab)
		}

		// Press Tab again -> wrap around to 0 (proxygroups)
		fn(tabEvent)
		if editor.activeTab != 0 {
			t.Errorf("expected activeTab 0 after wrap, got %d", editor.activeTab)
		}

		// Press Shift+Tab (Backtab) -> wrap backwards to 2 (ruleproviders)
		backtabEvent := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		fn(backtabEvent)
		if editor.activeTab != 2 {
			t.Errorf("expected activeTab 2 after Backtab, got %d", editor.activeTab)
		}
	})
	time.Sleep(30 * time.Millisecond)
}

func TestEditorPage_ShiftTabBothEventTypes(t *testing.T) {
	editor, app := createTestEditorPage()
	go app.Run()
	defer app.Stop()

	// Initial active tab: 0 (proxygroups)
	fn := editor.GetInputCapture()
	if fn == nil {
		t.Fatal("expected non-nil InputCapture")
	}

	ui.Updater.PostUi(func() {
		// 1. Test KeyTab + ModShift backwards navigation (0 -> 2 -> 1 -> 0)
		shiftTabEvent := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModShift)
		fn(shiftTabEvent)
		if editor.activeTab != 2 {
			t.Errorf("expected activeTab 2 after Shift+Tab, got %d", editor.activeTab)
		}
		fn(shiftTabEvent)
		if editor.activeTab != 1 {
			t.Errorf("expected activeTab 1 after Shift+Tab, got %d", editor.activeTab)
		}
		fn(shiftTabEvent)
		if editor.activeTab != 0 {
			t.Errorf("expected activeTab 0 after Shift+Tab, got %d", editor.activeTab)
		}

		// 2. Test KeyBacktab backwards navigation (0 -> 2 -> 1 -> 0)
		backtabEvent := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		fn(backtabEvent)
		if editor.activeTab != 2 {
			t.Errorf("expected activeTab 2 after KeyBacktab, got %d", editor.activeTab)
		}
		fn(backtabEvent)
		if editor.activeTab != 1 {
			t.Errorf("expected activeTab 1 after KeyBacktab, got %d", editor.activeTab)
		}
		fn(backtabEvent)
		if editor.activeTab != 0 {
			t.Errorf("expected activeTab 0 after KeyBacktab, got %d", editor.activeTab)
		}
	})
	time.Sleep(30 * time.Millisecond)
}

func TestEditorPage_SubtabSwitchingMouseClick(t *testing.T) {
	editor, app := createTestEditorPage()
	go app.Run()
	defer app.Stop()

	ui.Updater.PostUi(func() {
		// Simulate clicking region "rules"
		editor.switchTab(1)
		if editor.activeTab != 1 {
			t.Errorf("expected activeTab 1 after switchTab(1), got %d", editor.activeTab)
		}

		// Simulate clicking region "ruleproviders"
		editor.switchTab(2)
		if editor.activeTab != 2 {
			t.Errorf("expected activeTab 2 after switchTab(2), got %d", editor.activeTab)
		}

		// Simulate clicking region "proxygroups"
		editor.switchTab(0)
		if editor.activeTab != 0 {
			t.Errorf("expected activeTab 0 after switchTab(0), got %d", editor.activeTab)
		}
	})
	time.Sleep(30 * time.Millisecond)
}

func TestEditorPage_TabNotInterceptedInsideInputs(t *testing.T) {
	editor, app := createTestEditorPage()
	go app.Run()
	defer app.Stop()

	input := tview.NewInputField()
	ui.Updater.PostUi(func() {
		app.SetFocus(input)

		// When input has focus, Tab should NOT switch tab in EditorPage
		tabEvent := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		if fn := editor.GetInputCapture(); fn != nil {
			res := fn(tabEvent)
			if res != tabEvent {
				t.Error("expected Tab event to be returned unmodified when input has focus")
			}
		}
		if editor.activeTab != 0 {
			t.Errorf("activeTab should remain 0 when input has focus, got %d", editor.activeTab)
		}
	})
	time.Sleep(30 * time.Millisecond)
}

func TestEditorPage_ActivateDeactivateNoHang(t *testing.T) {
	api.InitClient("http://127.0.0.1:1", "")
	editor, app := createTestEditorPage()
	go app.Run()
	defer app.Stop()

	// Rapidly activate, switch subtabs, and deactivate on UI goroutine
	for i := 0; i < 5; i++ {
		ui.Updater.PostUi(func() {
			editor.Activate()
			editor.switchTab(1)
			editor.switchTab(2)
			editor.switchTab(0)
			editor.Deactivate()
		})
		time.Sleep(15 * time.Millisecond)
	}
}

func TestEditorPage_TabBarFormatting(t *testing.T) {
	editor, _ := createTestEditorPage()

	// Tab 0 active: proxygroups should have [black:green] and close with [-:-:-]
	editor.activeTab = 0
	editor.updateTabBar()
	text0 := editor.tabBar.GetText(false)
	if !strings.Contains(text0, `["proxygroups"][black:green]`) {
		t.Errorf("active tab 0 should contain [black:green], got %q", text0)
	}
	if !strings.Contains(text0, `[-:-:-][""]`) {
		t.Errorf("active tab 0 should reset with [-:-:-], got %q", text0)
	}
	if !strings.Contains(text0, `["rules"][white]`) {
		t.Errorf("inactive tab 1 should have [white], got %q", text0)
	}

	// Tab 1 active: rules should have [black:green] and proxygroups should have [white]
	editor.activeTab = 1
	editor.updateTabBar()
	text1 := editor.tabBar.GetText(false)
	if !strings.Contains(text1, `["rules"][black:green]`) {
		t.Errorf("active tab 1 should contain [black:green], got %q", text1)
	}
	if !strings.Contains(text1, `["proxygroups"][white]`) {
		t.Errorf("inactive tab 0 should have [white], got %q", text1)
	}
}
