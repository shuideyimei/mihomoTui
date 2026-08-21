package components

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/events"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/utils"

	"github.com/rivo/tview"
)

// StatusBar represents the bottom status bar
type StatusBar struct {
	*tview.TextView
	mu       sync.RWMutex
	traffic  *models.Traffic
	config   *models.Config
	bus      *events.Bus
	busUnsub func()

	ctx    context.Context
	cancel context.CancelFunc
}

// NewStatusBar creates a new status bar component
func NewStatusBar(bus ...*events.Bus) *StatusBar {
	statusBar := &StatusBar{
		TextView: tview.NewTextView(),
	}

	statusBar.setupStyle()
	statusBar.updateContent()
	if len(bus) > 0 && bus[0] != nil {
		statusBar.SetBus(bus[0])
	}
	return statusBar
}

// SetBus sets the event bus instance for the status bar and subscribes to proxy mode changes.
func (s *StatusBar) SetBus(bus *events.Bus) {
	s.mu.Lock()
	if s.busUnsub != nil {
		s.busUnsub()
		s.busUnsub = nil
	}
	s.bus = bus
	s.mu.Unlock()

	if bus != nil {
		unsub := bus.Subscribe(events.TopicProxyModeChanged, func(ev interface{}) {
			if event, ok := ev.(events.ProxyModeChangedEvent); ok {
				ui.Updater.PostUi(func() {
					s.mu.Lock()
					if s.config == nil {
						s.config = &models.Config{}
					}
					s.config.Mode = event.Mode
					s.updateContent()
					s.mu.Unlock()
				})
			}
		})
		s.mu.Lock()
		s.busUnsub = unsub
		s.mu.Unlock()
	}
}

// setupStyle configures the status bar appearance
func (s *StatusBar) setupStyle() {
	s.SetBorderPadding(0, 0, 1, 1)
	s.SetTextAlign(tview.AlignLeft)
	s.SetDynamicColors(true)
	s.SetWrap(false)
	s.SetBackgroundColor(ui.ThemeStatusBarBg)
}

func (s *StatusBar) Active() {
	s.Deactivate()
	s.getConfigData()
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	ctx := s.ctx
	s.mu.Unlock()
	go s.startTrafficStream(ctx)
}

func (s *StatusBar) Deactivate() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.mu.Unlock()
}

// getConfigData retrieves and updates the configuration data
func (s *StatusBar) getConfigData() {
	if config, err := api.Client.GetConfig(); err == nil {
		ui.Updater.UpdateUi(func() {
			s.updateConfig(config)
		})
	}
}

// startTrafficStream starts streaming traffic data
func (s *StatusBar) startTrafficStream(ctx context.Context) {
	for {
		if ctx == nil || ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			err := api.StreamClient.StreamTraffic(ctx, func(traffic *models.Traffic) {
				ui.Updater.UpdateUi(func() {
					s.updateTraffic(traffic)
				})
			})

			if ctx.Err() != nil {
				return
			}

			if err != nil && err != context.Canceled {
				ui.Updater.UpdateUi(func() {
					s.updateTraffic(nil)
				})
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
			}
		}
	}
}

// updateTraffic updates traffic information
func (s *StatusBar) updateTraffic(traffic *models.Traffic) {
	s.mu.Lock()
	s.traffic = traffic
	s.updateContent()
	s.mu.Unlock()
}

// updateConfig updates configuration information and publishes mode change event if mode is non-empty
func (s *StatusBar) updateConfig(config *models.Config) {
	s.mu.Lock()
	s.config = config
	s.updateContent()
	bus := s.bus
	var mode string
	if config != nil {
		mode = config.Mode
	}
	s.mu.Unlock()

	if mode != "" && bus != nil {
		bus.Publish(events.TopicProxyModeChanged, events.ProxyModeChangedEvent{
			Mode: mode,
		})
	}
}

// GetCurrentMode returns the current proxy mode
func (s *StatusBar) GetCurrentMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.config != nil {
		return s.config.Mode
	}
	return "Unknown"
}

// UpdateConfig updates the configuration and refreshes the status bar
func (s *StatusBar) UpdateConfig(config interface{}) {
	if config, ok := config.(*models.Config); ok {
		s.updateConfig(config)
	}
}

// updateContent updates the status bar content
func (s *StatusBar) updateContent() {
	var content string

	// Proxy mode
	mode := i18n.T("status.unknown")
	if s.config != nil {
		mode = s.config.Mode
	}

	// Mode color tag
	var modeTag string
	switch mode {
	case "rule":
		modeTag = ui.ThemeTagModeRule
	case "global":
		modeTag = ui.ThemeTagModeGlobal
	case "direct":
		modeTag = ui.ThemeTagModeDirect
	default:
		modeTag = "[white]"
	}

	// Traffic
	var upSpeed, downSpeed string
	if s.traffic != nil {
		upSpeed = fmt.Sprintf("%s/s", utils.FormatBytes(s.traffic.Up))
		downSpeed = fmt.Sprintf("%s/s", utils.FormatBytes(s.traffic.Down))
	} else {
		upSpeed = "- B/s"
		downSpeed = "- B/s"
	}

	content = fmt.Sprintf("%s%s[white] | ↑ [green]%s[white]  ↓ [green]%s[white]  [gray]%s[-]",
		modeTag, mode, upSpeed, downSpeed, i18n.T("status.shortcuts"))

	s.SetText(content)
}

// UpdateTexts refreshes the status bar text after a language change
func (s *StatusBar) UpdateTexts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateContent()
}
