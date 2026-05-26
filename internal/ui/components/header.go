package components

import (
	"fmt"
	"log"
	"mihomoTui/internal/api"
	"mihomoTui/internal/ui"
	"strings"

	"github.com/rivo/tview"
)

// Header represents the top header bar
type Header struct {
	*tview.TextView
	appName     string
	appVersion  string
	coreVersion string
	connected   bool
}

// NewHeader creates a new header component
func NewHeader(appName, version string) *Header {
	header := &Header{
		TextView:   tview.NewTextView(),
		appName:    appName,
		appVersion: version,
		connected:  false,
	}

	header.setupStyle()
	header.updateContent()
	return header
}

// setupStyle configures the header appearance
func (h *Header) setupStyle() {
	h.SetBorderPadding(0, 0, 1, 1)
	h.SetTextAlign(tview.AlignLeft)
	h.SetDynamicColors(true)
	h.SetWrap(false)
	h.SetBackgroundColor(ui.ThemeHeaderBg)
}

// updateContent updates the header content
func (h *Header) updateContent() {
	status := ui.ThemeTagDisconnected + "○" + ui.ThemeTagWhite
	if h.connected {
		status = ui.ThemeTagConnected + "●" + ui.ThemeTagWhite
	}

	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("%s %s", h.appName, h.appVersion))
	content.WriteString(fmt.Sprintf(" | core: %s | %s", h.coreVersion, status))

	h.SetText(content.String())
}

// FetchBasicInfo retrieves basic information about the application.
// All tview UI updates must run on the UI goroutine via UpdateUi.
func (h *Header) SetHeaderInfo() {
	var coreVersion string
	var connected bool
	err := api.Client.HealthCheck()
	if err != nil {
		log.Printf("Failed to check health: %v", err)
	}
	connected = err == nil
	version, err := api.Client.GetVersion()
	if err != nil {
		log.Printf("Failed to get version: %v", err)
	}
	if version != nil {
		coreVersion = version.Version
	} else {
		coreVersion = "unknown"
	}
	ui.Updater.UpdateUi(func() {
		h.connected = connected
		h.coreVersion = coreVersion
		h.updateContent()
	})
}
