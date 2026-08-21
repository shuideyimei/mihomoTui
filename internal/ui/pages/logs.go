package pages

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogsPage struct {
	*tview.Flex
	textView *tview.TextView

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	logs   []string

	maxLines int

	levelFilter   string
	isPaused      bool
	searchText    string
	renderTimer   *time.Timer
	searchInput   *tview.InputField
	pauseBtn      *tview.Button
	levelDropdown *tview.DropDown
	clearBtn      *tview.Button
}

// levelMapping associates display text with API filter value
type levelOption struct {
	display string
	api     string
}

func logLevelOptions() []levelOption {
	return []levelOption{
		{i18n.T("log.level_all"), ""},
		{i18n.T("log.level_error"), "error"},
		{i18n.T("log.level_warn"), "warning"},
		{i18n.T("log.level_info"), "info"},
		{i18n.T("log.level_debug"), "debug"},
	}
}

func NewLogsPage() *LogsPage {
	page := &LogsPage{
		Flex:     tview.NewFlex(),
		textView: tview.NewTextView(),
		logs:     make([]string, 0),
		maxLines: 1000,
	}

	page.setupUI()
	return page
}

func (p *LogsPage) setupUI() {
	p.textView.SetDynamicColors(true)
	p.textView.SetScrollable(true)

	options := logLevelOptions()
	displayOpts := make([]string, len(options))
	for i, opt := range options {
		displayOpts[i] = opt.display
	}

	// Level dropdown
	p.levelDropdown = tview.NewDropDown().
		SetOptions(displayOpts, func(text string, index int) {
			p.mu.Lock()
			if index >= 0 && index < len(options) {
				p.levelFilter = options[index].api
			}
			p.mu.Unlock()
			p.onLevelChanged()
		})
	p.levelDropdown.SetCurrentOption(0)
	p.levelDropdown.SetLabel(i18n.T("log.level_label"))
	p.levelDropdown.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.levelDropdown.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(ui.ThemeHighlightBg).Foreground(tcell.ColorBlack),
	)

	// Search input
	p.searchInput = tview.NewInputField().
		SetLabel(i18n.T("log.search_label")).
		SetFieldWidth(0)
	p.searchInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.searchInput.SetChangedFunc(func(text string) {
		p.mu.Lock()
		p.searchText = text
		p.mu.Unlock()
		p.refreshDisplay()
	})

	// Pause/Resume button
	p.pauseBtn = tview.NewButton(i18n.T("log.pause"))
	components.StyleButton(p.pauseBtn, components.ButtonNormal)
	p.pauseBtn.SetSelectedFunc(func() {
		p.togglePause()
	})

	// Clear button
	p.clearBtn = tview.NewButton(i18n.T("log.clear"))
	components.StyleButton(p.clearBtn, components.ButtonDanger)
	p.clearBtn.SetSelectedFunc(func() {
		p.Clear()
	})

	// Control bar
	controlBar := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(p.levelDropdown, 16, 0, false).
		AddItem(p.searchInput, 24, 0, false).
		AddItem(p.pauseBtn, 12, 0, false).
		AddItem(p.clearBtn, 8, 0, false)

	// Main layout
	p.SetDirection(tview.FlexRow)
	p.AddItem(controlBar, 1, 0, false)
	p.AddItem(p.textView, 0, 1, true)

	p.SetBorder(false)
	p.SetTitle(fmt.Sprintf(" %s ", i18n.T("log.title")))
	p.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if p.searchInput.HasFocus() {
			if event.Key() == tcell.KeyEscape {
				p.searchInput.SetText("")
				return nil
			}
			return event
		}
		switch event.Rune() {
		case '/':
			ui.Updater.SetFocus(p.searchInput)
			return nil
		case 'c', 'C':
			p.Clear()
			return nil
		case ' ':
			p.togglePause()
			return nil
		}
		return event
	})
}

func (p *LogsPage) onLevelChanged() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	wasPaused := p.isPaused
	p.isPaused = false
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.mu.Unlock()

	if wasPaused {
		p.mu.Lock()
		p.isPaused = true
		p.mu.Unlock()
		p.pauseBtn.SetLabel(i18n.T("log.resume"))
	} else {
		p.startLogStream()
	}
}

func (p *LogsPage) togglePause() {
	p.mu.Lock()
	wasPaused := p.isPaused
	p.isPaused = !p.isPaused
	if wasPaused {
		// Resuming: cancel the old stream context and allocate a fresh one.
		if p.cancel != nil {
			p.cancel()
		}
		p.ctx, p.cancel = context.WithCancel(context.Background())
	} else {
		// Pausing: cancel the active stream context immediately.
		if p.cancel != nil {
			p.cancel()
			p.cancel = nil
		}
	}
	p.mu.Unlock()

	if wasPaused {
		p.pauseBtn.SetLabel(i18n.T("log.pause"))
		p.startLogStream()
	} else {
		p.pauseBtn.SetLabel(i18n.T("log.resume"))
	}
}

func (p *LogsPage) startLogStream() {
	go func() {
		for {
			p.mu.Lock()
			ctx := p.ctx
			level := p.levelFilter
			p.mu.Unlock()

			if ctx == nil || ctx.Err() != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			var err error
			if level == "" {
				err = api.StreamClient.StreamLogs(ctx, p.onLogReceived)
			} else {
				err = api.StreamClient.StreamLogsWithLevel(ctx, level, p.onLogReceived)
			}

			if ctx.Err() != nil {
				return
			}

			if err != nil && err != context.Canceled {
				p.addLog(fmt.Sprintf(i18n.T("log.conn_err"), err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
}

func (p *LogsPage) onLogReceived(log *models.Log) {
	p.mu.Lock()
	if p.isPaused {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	logText := p.formatLog(log)
	p.addLog(logText)
}

func (p *LogsPage) formatLog(log *models.Log) string {
	timestamp := time.Now().Format("15:04:05")
	if log.Time != "" {
		if t, err := time.Parse(time.RFC3339, log.Time); err == nil {
			timestamp = t.Format("15:04:05")
		}
	}

	var color string
	switch log.Type {
	case "error", "ERROR":
		color = "[red]"
	case "warn", "WARN", "warning", "WARNING":
		color = "[yellow]"
	case "info", "INFO":
		color = "[green]"
	case "debug", "DEBUG":
		color = "[blue]"
	default:
		color = "[white]"
	}

	return fmt.Sprintf("%s[gray]%s[white] %s%-5s[white] %s",
		color, timestamp, color, log.Type, log.Payload)
}

func (p *LogsPage) addLog(logText string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logs = append(p.logs, logText)
	if len(p.logs) > p.maxLines {
		p.logs = p.logs[len(p.logs)-p.maxLines:]
	}

	if p.renderTimer == nil {
		p.renderTimer = time.AfterFunc(50*time.Millisecond, func() {
			p.mu.Lock()
			p.renderTimer = nil
			p.mu.Unlock()
			p.refreshDisplay()
		})
	}
}

func (p *LogsPage) refreshDisplay() {
	p.mu.Lock()
	search := strings.ToLower(p.searchText)
	logsCopy := make([]string, len(p.logs))
	copy(logsCopy, p.logs)
	p.mu.Unlock()

	if search == "" {
		var b strings.Builder
		for _, line := range logsCopy {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		content := b.String()
		ui.Updater.PostUi(func() {
			p.textView.SetText(content)
			p.mu.Lock()
			paused := p.isPaused
			p.mu.Unlock()
			if !paused {
				p.textView.ScrollToEnd()
			}
		})
		return
	}

	var b strings.Builder
	for _, line := range logsCopy {
		lower := strings.ToLower(line)
		if strings.Contains(lower, search) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	content := b.String()
	ui.Updater.PostUi(func() {
		p.textView.SetText(content)
		p.mu.Lock()
		paused := p.isPaused
		p.mu.Unlock()
		if !paused {
			p.textView.ScrollToEnd()
		}
	})
}

func (p *LogsPage) Stop() {
	p.Deactivate()
}

func (p *LogsPage) Activate() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	wasPaused := p.isPaused
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.mu.Unlock()

	p.addLog(i18n.T("log.stream_active"))

	if !wasPaused {
		p.startLogStream()
	}
}

func (p *LogsPage) Deactivate() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.renderTimer != nil {
		p.renderTimer.Stop()
		p.renderTimer = nil
	}
	p.mu.Unlock()
}

// UpdateTexts refreshes all localized labels on the logs page
func (p *LogsPage) UpdateTexts() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.SetTitle(fmt.Sprintf(" %s ", i18n.T("log.title")))
	if p.levelDropdown != nil {
		p.levelDropdown.SetLabel(i18n.T("log.level_label"))
	}
	if p.searchInput != nil {
		p.searchInput.SetLabel(i18n.T("log.search_label"))
	}
	if p.pauseBtn != nil {
		if p.isPaused {
			p.pauseBtn.SetLabel(i18n.T("log.resume"))
		} else {
			p.pauseBtn.SetLabel(i18n.T("log.pause"))
		}
	}
	if p.clearBtn != nil {
		p.clearBtn.SetLabel(i18n.T("log.clear"))
	}
}

func (p *LogsPage) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logs = make([]string, 0)
	p.textView.SetText("")
}
