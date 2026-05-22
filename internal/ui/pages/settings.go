package pages

import (
	"mihomoTui/internal/config"

	"github.com/rivo/tview"
)

// Settings represents the settings page
type Settings struct {
	*tview.TextView
	configManager *config.Manager
}
