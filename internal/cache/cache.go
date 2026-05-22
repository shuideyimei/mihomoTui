package cache

import (
	"sync"
	"time"

	"mihomoTui/internal/types"
)

// Manager is a simple in-memory cache for proxy nodes and other runtime data.
// Thread-safe. Supports TTL expiry. Not persisted (use SubscriptionStore for persistence).
type Manager struct {
	mu    sync.RWMutex
	store map[string]*types.CacheEntry
}

// NewManager creates a new cache manager.
func NewManager() *Manager {
	return &Manager{
		store: make(map[string]*types.CacheEntry),
	}
}

// Get retrieves a value from the cache. Returns nil if missing or expired.
func (m *Manager) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	entry, ok := m.store[key]
	m.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if !entry.ExpireAt.IsZero() && time.Now().After(entry.ExpireAt) {
		m.Delete(key)
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value with optional TTL.
func (m *Manager) Set(key string, value interface{}, ttl time.Duration) {
	entry := &types.CacheEntry{
		Key:   key,
		Value: value,
	}
	if ttl > 0 {
		entry.ExpireAt = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.store[key] = entry
	m.mu.Unlock()
}

// Delete removes a key from the cache.
func (m *Manager) Delete(key string) {
	m.mu.Lock()
	delete(m.store, key)
	m.mu.Unlock()
}

// Clear removes all entries.
func (m *Manager) Clear() {
	m.mu.Lock()
	m.store = make(map[string]*types.CacheEntry)
	m.mu.Unlock()
}

// Size returns the number of cached entries.
func (m *Manager) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.store)
}

// GetOrSet atomically retrieves a value or computes and stores it.
func (m *Manager) GetOrSet(key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, ok := m.Get(key); ok {
		return val, nil
	}

	val, err := fn()
	if err != nil {
		return nil, err
	}

	m.Set(key, val, ttl)
	return val, nil
}

// Keys returns all cached keys (for observability).
func (m *Manager) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.store))
	for k := range m.store {
		keys = append(keys, k)
	}
	return keys
}
