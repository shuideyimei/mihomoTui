package pages

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestConfigManager_New_NonNil(t *testing.T) {
	cm := NewConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager() returned nil")
	}
}

func TestConfigManagerMarksEditedContentDirty(t *testing.T) {
	cm := NewConfigManager()
	cm.editMode = true
	cm.editPath = "/tmp/config.yaml"
	cm.editOriginal = "mixed-port: 7890\n"
	cm.editArea.SetText("mixed-port: 7891\n", true)
	cm.updateEditTitle()

	if !cm.editDirty {
		t.Fatal("edited content should be marked dirty")
	}
	if !strings.Contains(cm.editArea.GetTitle(), "*") {
		t.Fatalf("dirty editor title should contain *, got %q", cm.editArea.GetTitle())
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
