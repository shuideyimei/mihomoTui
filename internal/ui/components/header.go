package components

import (
	"fmt"
	"log"
	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
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
	h.SetBorder(true)
	h.SetBorderPadding(0, 0, 1, 1)
	h.SetTextAlign(tview.AlignLeft)
	h.SetDynamicColors(true)
	h.SetWrap(false)
	// h.SetBackgroundColor(tcell.ColorCadetBlue)
}

// updateContent updates the header content
func (h *Header) updateContent() {
	status := "[red]○[white]"
	if h.connected {
		status = "[green]●[white]"
	}

	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("%s %s", h.appName, h.appVersion))
	content.WriteString(fmt.Sprintf(" | coreVer: %s | Status: %s", h.coreVersion, status))

	h.SetText(content.String())
}

// FetchBasicInfo retrieves basic information about the application
func (h *Header) SetHeaderInfo() {
	var connected bool
	var version *models.Version
	err := api.Client.HealthCheck()
	if err != nil {
		log.Printf("Failed to check health: %v", err)
	}
	connected = err == nil
	version, err = api.Client.GetVersion()
	if err != nil {
		log.Printf("Failed to get version: %v", err)
	}
	if version != nil {
		h.coreVersion = version.Version
	} else {
		h.coreVersion = "unknown"
	}
	h.connected = connected
	h.updateContent()
}
