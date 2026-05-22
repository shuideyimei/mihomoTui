package event

import (
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/types"
)

func TestBus_PublishSubscribe(t *testing.T) {
	bus := NewBus()
	received := make(chan string, 1)

	bus.Subscribe(types.EventSubAdded, func(e types.Event) {
		received <- e.Payload.(string)
	})

	bus.Publish(types.Event{Type: types.EventSubAdded, Payload: "hello"})

	select {
	case msg := <-received:
		if msg != "hello" {
			t.Errorf("expected 'hello', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	bus := NewBus()
	received := make(chan string, 1)

	unsub := bus.Subscribe(types.EventSubAdded, func(e types.Event) {
		received <- "got-it"
	})
	unsub()

	bus.Publish(types.Event{Type: types.EventSubAdded})

	select {
	case <-received:
		t.Error("received event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	var received []string

	bus.Subscribe(types.EventSubUpdated, func(e types.Event) {
		mu.Lock()
		received = append(received, "a")
		mu.Unlock()
	})
	bus.Subscribe(types.EventSubUpdated, func(e types.Event) {
		mu.Lock()
		received = append(received, "b")
		mu.Unlock()
	})

	bus.Publish(types.Event{Type: types.EventSubUpdated})

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if len(received) != 2 {
		t.Errorf("expected 2 handlers called, got %d", len(received))
	}
	mu.Unlock()
}

func TestBus_EventHandlerPanicRecovery(t *testing.T) {
	bus := NewBus()
	received := make(chan string, 1)

	bus.Subscribe(types.EventSubAdded, func(e types.Event) {
		panic("intentional panic in handler")
	})
	bus.Subscribe(types.EventSubAdded, func(e types.Event) {
		received <- "second-handler-called"
	})

	// Should not panic — second handler should still run
	bus.Publish(types.Event{Type: types.EventSubAdded})

	select {
	case msg := <-received:
		if msg != "second-handler-called" {
			t.Errorf("unexpected message: %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("second handler was not called after panic")
	}
}

func TestBus_SubscriberCount(t *testing.T) {
	bus := NewBus()

	if bus.SubscriberCount(types.EventSubAdded) != 0 {
		t.Error("expected 0 subscribers initially")
	}

	unsub1 := bus.Subscribe(types.EventSubAdded, func(types.Event) {})
	bus.Subscribe(types.EventSubAdded, func(types.Event) {})

	if bus.SubscriberCount(types.EventSubAdded) != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount(types.EventSubAdded))
	}

	unsub1()
	if bus.SubscriberCount(types.EventSubAdded) != 1 {
		t.Errorf("expected 1 subscriber after unsub, got %d", bus.SubscriberCount(types.EventSubAdded))
	}
}

func TestBus_DifferentEventTypes(t *testing.T) {
	bus := NewBus()
	subEvents := make(chan string, 2)
	configEvents := make(chan string, 2)

	bus.Subscribe(types.EventSubAdded, func(e types.Event) {
		subEvents <- "sub"
	})
	bus.Subscribe(types.EventConfigReloaded, func(e types.Event) {
		configEvents <- "config"
	})

	bus.Publish(types.Event{Type: types.EventSubAdded})
	bus.Publish(types.Event{Type: types.EventConfigReloaded})
	bus.Publish(types.Event{Type: types.EventConfigReloaded})

	// Only sub channel should have 1
	select {
	case <-subEvents:
		// ok
	case <-time.After(time.Second):
		t.Fatal("sub event not received")
	}

	// Config channel should have 2
	select {
	case <-configEvents:
	case <-time.After(time.Second):
		t.Fatal("config event 1 not received")
	}
	select {
	case <-configEvents:
	case <-time.After(time.Second):
		t.Fatal("config event 2 not received")
	}
}
