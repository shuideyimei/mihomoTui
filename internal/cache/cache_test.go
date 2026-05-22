package cache

import (
	"errors"
	"testing"
	"time"
)

func TestManager_GetSet(t *testing.T) {
	m := NewManager()

	m.Set("key1", "value1", 0)
	val, ok := m.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}
}

func TestManager_GetNonExistent(t *testing.T) {
	m := NewManager()
	_, ok := m.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent key")
	}
}

func TestManager_Delete(t *testing.T) {
	m := NewManager()
	m.Set("key", "val", 0)
	m.Delete("key")
	_, ok := m.Get("key")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestManager_Clear(t *testing.T) {
	m := NewManager()
	m.Set("a", 1, 0)
	m.Set("b", 2, 0)
	m.Clear()
	if m.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", m.Size())
	}
}

func TestManager_TTL(t *testing.T) {
	m := NewManager()
	m.Set("expires", "value", 10*time.Millisecond)

	_, ok := m.Get("expires")
	if !ok {
		t.Fatal("expected key to exist before expiry")
	}

	time.Sleep(20 * time.Millisecond)

	_, ok = m.Get("expires")
	if ok {
		t.Error("expected key to be expired")
	}
}

func TestManager_GetOrSet_Existing(t *testing.T) {
	m := NewManager()
	m.Set("key", "existing", 0)

	called := false
	val, err := m.GetOrSet("key", time.Minute, func() (interface{}, error) {
		called = true
		return "new", nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if val != "existing" {
		t.Errorf("expected 'existing', got %v", val)
	}
	if called {
		t.Error("factory should not have been called for existing key")
	}
}

func TestManager_GetOrSet_New(t *testing.T) {
	m := NewManager()

	val, err := m.GetOrSet("new-key", time.Minute, func() (interface{}, error) {
		return "computed", nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if val != "computed" {
		t.Errorf("expected 'computed', got %v", val)
	}

	cached, ok := m.Get("new-key")
	if !ok {
		t.Fatal("expected computed value to be cached")
	}
	if cached != "computed" {
		t.Errorf("expected cached 'computed', got %v", cached)
	}
}

func TestManager_GetOrSet_Error(t *testing.T) {
	m := NewManager()

	_, err := m.GetOrSet("key", time.Minute, func() (interface{}, error) {
		return nil, errors.New("factory error")
	})

	if err == nil {
		t.Fatal("expected error from factory")
	}
	_, ok := m.Get("key")
	if ok {
		t.Error("should not cache on error")
	}
}

func TestManager_Size(t *testing.T) {
	m := NewManager()
	if m.Size() != 0 {
		t.Error("expected size 0 initially")
	}

	m.Set("a", 1, 0)
	m.Set("b", 2, 0)
	if m.Size() != 2 {
		t.Errorf("expected size 2, got %d", m.Size())
	}

	m.Delete("a")
	if m.Size() != 1 {
		t.Errorf("expected size 1, got %d", m.Size())
	}
}

func TestManager_Keys(t *testing.T) {
	m := NewManager()
	m.Set("alpha", 1, 0)
	m.Set("beta", 2, 0)

	keys := m.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}
