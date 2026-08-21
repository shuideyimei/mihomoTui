package pages

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"mihomoTui/internal/events"
	"mihomoTui/internal/service"
)

type mockProfileRepo struct {
	profiles []string
	current  string
}

func (m *mockProfileRepo) FindAllProfiles() []string {
	return m.profiles
}
func (m *mockProfileRepo) GetCurrentProfilePath() string {
	return m.current
}
func (m *mockProfileRepo) ReadProfile(path string) (string, error) {
	return "mode: rule", nil
}
func (m *mockProfileRepo) WriteProfile(path, content string) error {
	return nil
}
func (m *mockProfileRepo) DeleteProfile(path string) error {
	return nil
}
func (m *mockProfileRepo) ValidateProfile(path string) error {
	return nil
}
func (m *mockProfileRepo) EnsureSafePath(cfgPath, path string) ([]string, error) {
	return nil, nil
}

func newTestConfigManager() *ConfigManager {
	bus := events.NewBus()
	repo := &mockProfileRepo{
		profiles: []string{"/tmp/config1.yaml", "/tmp/config2.yaml"},
		current:  "/tmp/config1.yaml",
	}
	profileService := service.NewProfileService(repo, bus)
	return NewConfigManager(profileService, bus)
}

func TestConfigManager_New_NonNil(t *testing.T) {
	cm := newTestConfigManager()
	if cm == nil {
		t.Fatal("NewConfigManager() returned nil")
	}
}

func TestConfigManagerMarksEditedContentDirty(t *testing.T) {
	cm := newTestConfigManager()
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

	cm := newTestConfigManager()
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
	cm := newTestConfigManager()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	cm.Deactivate()
}
