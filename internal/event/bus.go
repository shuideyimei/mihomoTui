package event

import (
	"log"
	"strconv"
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
// A snapshot of handlers is taken under RLock to avoid deadlock if a handler
// calls Subscribe() (which acquires a write lock) during event dispatch.
func (b *Bus) Publish(event types.Event) {
	b.mu.RLock()
	handlers, ok := b.subscribers[event.Type]
	var handlerList []types.EventHandler
	if ok {
		handlerList = make([]types.EventHandler, 0, len(handlers))
		for _, h := range handlers {
			handlerList = append(handlerList, h)
		}
	}
	b.mu.RUnlock()

	for _, handler := range handlerList {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[eventbus] panic in handler for %s: %v", event.Type, r)
				}
			}()
			handler(event)
		}()
	}
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
	return string(et) + ":" + strconv.Itoa(n)
}
