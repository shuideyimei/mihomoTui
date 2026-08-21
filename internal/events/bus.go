package events

import (
	"sync"
)

// Handler represents an event listener function.
type Handler func(event interface{})

type subscription struct {
	id      uint64
	handler Handler
}

// Bus is a thread-safe event bus for publishing and subscribing to application events.
type Bus struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[string][]subscription
}

// NewBus creates a new EventBus instance.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string][]subscription),
	}
}

// Subscribe registers a handler for a specific event topic and returns an unsubscribe function.
func (b *Bus) Subscribe(topic string, handler Handler) func() {
	if b == nil || handler == nil {
		return func() {}
	}

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.subscribers[topic] = append(b.subscribers[topic], subscription{
		id:      id,
		handler: handler,
	})
	b.mu.Unlock()

	var unsubOnce sync.Once
	return func() {
		unsubOnce.Do(func() {
			b.unsubscribe(topic, id)
		})
	}
}

func (b *Bus) unsubscribe(topic string, id uint64) {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[topic]
	for i, sub := range subs {
		if sub.id == id {
			copy(subs[i:], subs[i+1:])
			subs[len(subs)-1] = subscription{}
			b.subscribers[topic] = subs[:len(subs)-1]
			if len(b.subscribers[topic]) == 0 {
				delete(b.subscribers, topic)
			}
			break
		}
	}
}

// Publish broadcasts an event to all subscribers of the specified topic.
// Handlers are invoked synchronously in the order they were registered.
// Each handler execution is protected with panic recovery to prevent total bus failure.
func (b *Bus) Publish(topic string, event interface{}) {
	if b == nil {
		return
	}

	b.mu.RLock()
	subs := b.subscribers[topic]
	handlers := make([]Handler, 0, len(subs))
	for _, sub := range subs {
		if sub.handler != nil {
			handlers = append(handlers, sub.handler)
		}
	}
	b.mu.RUnlock()

	for _, handler := range handlers {
		func() {
			defer func() {
				_ = recover()
			}()
			handler(event)
		}()
	}
}
