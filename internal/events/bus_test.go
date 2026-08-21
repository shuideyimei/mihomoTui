package events

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestEventBus_Basic(t *testing.T) {
	bus := NewBus()
	var called int32

	unsub := bus.Subscribe(TopicLanguageChanged, func(ev interface{}) {
		event, ok := ev.(LanguageChangedEvent)
		if !ok || event.Language != "en" {
			t.Errorf("unexpected event: %v", ev)
		}
		atomic.AddInt32(&called, 1)
	})
	if unsub == nil {
		t.Fatal("expected non-nil unsubscribe function")
	}

	bus.Publish(TopicLanguageChanged, LanguageChangedEvent{Language: "en"})

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected handler to be called once, got %d", called)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewBus()
	var count int32

	unsub := bus.Subscribe("test.topic", func(ev interface{}) {
		atomic.AddInt32(&count, 1)
	})

	bus.Publish("test.topic", "hello")
	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	// Unsubscribe
	unsub()

	// Calling unsub again should be a safe no-op
	unsub()

	bus.Publish("test.topic", "world")
	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected count to remain 1 after unsub, got %d", count)
	}
}

func TestEventBus_MultipleSubscribersAndOrder(t *testing.T) {
	bus := NewBus()
	var order []int
	var mu sync.Mutex

	unsub1 := bus.Subscribe("ordered", func(ev interface{}) {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
	})
	unsub2 := bus.Subscribe("ordered", func(ev interface{}) {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
	})
	unsub3 := bus.Subscribe("ordered", func(ev interface{}) {
		mu.Lock()
		order = append(order, 3)
		mu.Unlock()
	})

	bus.Publish("ordered", "ping")
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("expected execution order [1 2 3], got %v", order)
	}

	// Remove middle subscriber
	unsub2()
	mu.Lock()
	order = nil
	mu.Unlock()

	bus.Publish("ordered", "ping2")
	if len(order) != 2 || order[0] != 1 || order[1] != 3 {
		t.Fatalf("expected execution order [1 3], got %v", order)
	}

	unsub1()
	unsub3()
}

func TestEventBus_PanicRecovery(t *testing.T) {
	bus := NewBus()
	var secondCalled bool

	bus.Subscribe("panic.topic", func(ev interface{}) {
		panic("boom!")
	})
	bus.Subscribe("panic.topic", func(ev interface{}) {
		secondCalled = true
	})

	// Publish should recover from panic and continue to second handler
	bus.Publish("panic.topic", "test")

	if !secondCalled {
		t.Fatal("expected second subscriber to run despite first subscriber panic")
	}
}

func TestEventBus_Concurrent(t *testing.T) {
	bus := NewBus()
	var wg sync.WaitGroup
	var counter int64

	const numSubscribers = 20
	const numPublishers = 20
	const eventsPerPublisher = 50

	unsubs := make([]func(), numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		unsubs[i] = bus.Subscribe("concurrent.topic", func(ev interface{}) {
			atomic.AddInt64(&counter, 1)
		})
	}

	for p := 0; p < numPublishers; p++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for e := 0; e < eventsPerPublisher; e++ {
				bus.Publish("concurrent.topic", id)
			}
		}(p)
	}

	wg.Wait()

	expected := int64(numSubscribers * numPublishers * eventsPerPublisher)
	if atomic.LoadInt64(&counter) != expected {
		t.Fatalf("expected %d events processed, got %d", expected, counter)
	}

	for _, unsub := range unsubs {
		unsub()
	}

	bus.Publish("concurrent.topic", "after")
	if atomic.LoadInt64(&counter) != expected {
		t.Fatalf("expected counter to not increase after unsubscribe, got %d", counter)
	}
}

func TestEventBus_NilSafety(t *testing.T) {
	var bus *Bus

	// nil bus subscribe
	unsub := bus.Subscribe("topic", func(ev interface{}) {})
	if unsub == nil {
		t.Fatal("expected non-nil dummy unsub function")
	}
	unsub()

	// nil bus publish
	bus.Publish("topic", "data")

	// non-nil bus, nil handler
	b := NewBus()
	u2 := b.Subscribe("topic", nil)
	if u2 == nil {
		t.Fatal("expected non-nil dummy unsub function")
	}
	u2()
	b.Publish("topic", "data")
}
