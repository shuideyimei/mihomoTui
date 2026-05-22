package event

import (
	"log"
	"sync"

	"mihomoTui/internal/types"
)

// Bus is a concurrency-safe pub/sub event bus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[types.EventType]map[string]types.EventHandler
	nextID      int
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[types.EventType]map[string]types.EventHandler),
	}
}

// Publish sends an event to all subscribers of its type.
// Handlers are called synchronously; for non-blocking dispatch, wrap in a goroutine.
func (b *Bus) Publish(event types.Event) {
	b.mu.RLock()
	handlers, ok := b.subscribers[event.Type]
	b.mu.RUnlock()

	if !ok {
		return
	}

	b.mu.RLock()
	for id, handler := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[eventbus] panic in handler %s for %s: %v", id, event.Type, r)
				}
			}()
			handler(event)
		}()
	}
	b.mu.RUnlock()
}

// Subscribe registers a handler for an event type. Returns an unsubscribe function.
func (b *Bus) Subscribe(eventType types.EventType, handler types.EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.makeID(eventType, b.nextID)

	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = make(map[string]types.EventHandler)
	}
	b.subscribers[eventType][id] = handler

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if handlers, ok := b.subscribers[eventType]; ok {
			delete(handlers, id)
			if len(handlers) == 0 {
				delete(b.subscribers, eventType)
			}
		}
	}
	return unsub
}

// SubscribeAsync is like Subscribe but dispatches the handler in a new goroutine.
func (b *Bus) SubscribeAsync(eventType types.EventType, handler types.EventHandler) func() {
	wrapped := func(e types.Event) {
		go handler(e)
	}
	return b.Subscribe(eventType, wrapped)
}

// Unsubscribe removes a specific subscription by ID.
func (b *Bus) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, handlers := range b.subscribers {
		delete(handlers, id)
	}
}

// SubscriberCount returns the number of handlers for a given event type.
func (b *Bus) SubscriberCount(eventType types.EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[eventType])
}

func (b *Bus) makeID(et types.EventType, n int) string {
	return string(et) + ":" + itoa(n)
}

// Simple int-to-string for zero allocs (avoids fmt import).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
