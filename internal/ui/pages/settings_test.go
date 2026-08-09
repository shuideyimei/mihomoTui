package pages

import (
	"testing"

	"mihomoTui/internal/config"
)

func TestNewSettings_NonNil(t *testing.T) {
	s := newSettingsPage(config.NewManager(), "test", "1.0")
	if s == nil {
		t.Fatal("newSettingsPage(config.NewManager(), \"test\", \"1.0\") returned nil")
	}
}

func TestSettings_BasicLayout(t *testing.T) {
	s := newSettingsPage(config.NewManager(), "test", "1.0")
	if s == nil {
		t.Fatal("newSettingsPage returned nil")
	}

	if s.Flex == nil {
		t.Error("Flex is nil — expected *tview.Flex")
	}
	if s.backupBtn == nil {
		t.Error("backupBtn is nil — expected *tview.Button")
	}
	if s.restoreBtn == nil {
		t.Error("restoreBtn is nil — expected *tview.Button")
	}
	if s.statusText == nil {
		t.Error("statusText is nil — expected *tview.TextView")
	}
}
