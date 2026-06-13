package app

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/ui/pages"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type lifecycleRecorder struct {
	name   string
	mu     *sync.Mutex
	events *[]string
}

type navigationGuardRecorder struct {
	lifecycleRecorder
	requested bool
}

func (g *navigationGuardRecorder) RequestDeactivate(func()) bool {
	g.requested = true
	return false
}

func (r *lifecycleRecorder) Activate() {
	r.record("activate:" + r.name)
}

func TestGlobalShortcutsDoNotInterceptTextInput(t *testing.T) {
	app := NewApp("test", "test")
	input := tview.NewInputField()
	app.app.SetFocus(input)
	event := tcell.NewEventKey(tcell.KeyRune, '?', tcell.ModNone)

	if got := app.handleGlobalKeys(event); got != event {
		t.Fatal("text input event should be returned to the focused input")
	}
	if app.showingHelp {
		t.Fatal("typing ? in an input should not open help")
	}
}

func TestConnectionsPageOwnsF5Refresh(t *testing.T) {
	app := NewApp("test", "test")
	app.currentPage = 2
	event := tcell.NewEventKey(tcell.KeyF5, 0, tcell.ModNone)

	if got := app.handleGlobalKeys(event); got != event {
		t.Fatal("F5 should be returned to the connections page")
	}
	if app.currentPage != 2 {
		t.Fatal("F5 on connections should not navigate to logs")
	}
}

func TestNavigationGuardCanDeferPageSwitch(t *testing.T) {
	var mu sync.Mutex
	var events []string
	guard := &navigationGuardRecorder{
		lifecycleRecorder: lifecycleRecorder{name: "guard", mu: &mu, events: &events},
	}
	app := NewApp("test", "test")
	app.pageLifecycle["dashboard"] = guard

	app.switchPage(1)

	if !guard.requested {
		t.Fatal("expected current page navigation guard to be consulted")
	}
	if app.currentPage != 0 {
		t.Fatal("guarded navigation should leave the current page unchanged")
	}
}

func TestCommandPaletteOpensAndCloses(t *testing.T) {
	app := NewApp("test", "test")
	app.rootPages = tview.NewPages()
	previous := tview.NewBox()
	app.app.SetFocus(previous)

	app.showCommandPalette()
	if !app.showingPalette {
		t.Fatal("command palette should be visible")
	}
	app.hideCommandPalette()
	if app.showingPalette {
		t.Fatal("command palette should be hidden")
	}
	if app.app.GetFocus() != previous {
		t.Fatal("closing command palette should restore previous focus")
	}
}

func TestHelpAllowsScrollKeys(t *testing.T) {
	app := NewApp("test", "test")
	app.showingHelp = true
	event := tcell.NewEventKey(tcell.KeyPgDn, 0, tcell.ModNone)

	if got := app.handleGlobalKeys(event); got != event {
		t.Fatal("help should pass scroll keys to the help view")
	}
}

func (r *lifecycleRecorder) Deactivate() {
	r.record("deactivate:" + r.name)
}

func (r *lifecycleRecorder) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.events = append(*r.events, event)
}

func TestLifecycleTransitionsAreOrdered(t *testing.T) {
	var mu sync.Mutex
	var events []string
	first := &lifecycleRecorder{name: "first", mu: &mu, events: &events}
	second := &lifecycleRecorder{name: "second", mu: &mu, events: &events}

	app := NewApp("test", "test")
	app.pageLifecycle = map[string]pages.ActivatablePage{
		"first":  first,
		"second": second,
	}
	go app.runLifecycle()

	app.activatePage("first")
	app.transitionPages("first", "second")
	app.enqueueLifecycle(lifecycleTransition{deactivate: second, stop: true})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := len(events) == 4
		mu.Unlock()
		if done {
			break
		}
		time.Sleep(time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"activate:first",
		"deactivate:first",
		"activate:second",
		"deactivate:second",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("expected lifecycle order %v, got %v", want, events)
	}
}
