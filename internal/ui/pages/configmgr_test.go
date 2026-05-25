package pages

import (
	"runtime"
	"testing"
	"time"
)

func TestConfigManager_New_NonNil(t *testing.T) {
	cm := NewConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager() returned nil")
	}
}

func TestConfigManager_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	cm := NewConfigManager()
	cm.Activate()
	cm.Deactivate()
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - before
	if leaked > 2 {
		t.Errorf("possible goroutine leak: %d goroutines before, %d after (diff: %d)", before, after, leaked)
	}
}

func TestConfigManager_DeactivateNoPanic(t *testing.T) {
	cm := NewConfigManager()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	cm.Deactivate()
}
