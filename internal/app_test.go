package app

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/ui/pages"
)

type lifecycleRecorder struct {
	name   string
	mu     *sync.Mutex
	events *[]string
}

func (r *lifecycleRecorder) Activate() {
	r.record("activate:" + r.name)
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
