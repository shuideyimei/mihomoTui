package pages

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"

	"github.com/rivo/tview"
)

type Logs struct {
	*LogsPage
}

func (l *Logs) Activate() {
	log.Printf("Activating logs page")
	l.LogsPage.Activate()
}

func (l *Logs) Deactivate() {
	log.Printf("Deactivating logs page")
	l.LogsPage.Deactivate()
}

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
	searchInput   *tview.InputField
	pauseBtn      *tview.Button
	levelDropdown *tview.DropDown
	clearBtn      *tview.Button
}

var levels = []string{"All", "Error", "Warn", "Info", "Debug"}

func levelToAPI(display string) string {
	switch display {
	case "Error":
		return "error"
	case "Warn":
		return "warning"
	case "Info":
		return "info"
	case "Debug":
		return "debug"
	default:
		return ""
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
	p.textView.SetBorder(true)
	p.textView.SetTitle(" 实时日志 ")
	p.textView.SetDynamicColors(true)
	p.textView.SetScrollable(true)

	// Only auto-scroll when not paused
	p.textView.SetChangedFunc(func() {
		p.mu.Lock()
		paused := p.isPaused
		p.mu.Unlock()
		if paused {
			return
		}
		go func() {
			time.Sleep(10 * time.Millisecond)
			ui.Updater.PostUi(func() {
				p.textView.ScrollToEnd()
			})
		}()
	})

	// Level dropdown
	p.levelDropdown = tview.NewDropDown().
		SetOptions(levels, func(text string, index int) {
			p.mu.Lock()
			p.levelFilter = levelToAPI(text)
			p.mu.Unlock()
			p.onLevelChanged()
		})
	p.levelDropdown.SetCurrentOption(0)
	p.levelDropdown.SetLabel("Level: ")

	// Search input
	p.searchInput = tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(20)
	p.searchInput.SetChangedFunc(func(text string) {
		p.mu.Lock()
		p.searchText = text
		p.mu.Unlock()
		p.refreshDisplay()
	})

	// Pause/Resume button
	p.pauseBtn = tview.NewButton("⏸ Pause")
	p.pauseBtn.SetSelectedFunc(func() {
		p.togglePause()
	})

	// Clear button
	p.clearBtn = tview.NewButton("Clear")
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
}

func (p *LogsPage) onLevelChanged() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	wasPaused := p.isPaused
	p.isPaused = false
	p.mu.Unlock()

	p.ctx, p.cancel = context.WithCancel(context.Background())

	if wasPaused {
		p.mu.Lock()
		p.isPaused = true
		p.mu.Unlock()
		p.pauseBtn.SetLabel("▶ Resume")
	} else {
		p.startLogStream()
	}
}

func (p *LogsPage) togglePause() {
	p.mu.Lock()
	wasPaused := p.isPaused
	p.isPaused = !p.isPaused
	p.mu.Unlock()

	if wasPaused {
		if p.cancel != nil {
			p.cancel()
		}
		p.ctx, p.cancel = context.WithCancel(context.Background())
		p.pauseBtn.SetLabel("⏸ Pause")
		p.startLogStream()
	} else {
		if p.cancel != nil {
			p.cancel()
		}
		p.pauseBtn.SetLabel("▶ Resume")
	}
}

func (p *LogsPage) startLogStream() {
	go func() {
		for {
			p.mu.Lock()
			ctx := p.ctx
			level := p.levelFilter
			p.mu.Unlock()

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
			if err != nil && err != context.Canceled {
				p.addLog(fmt.Sprintf("[red]连接错误: %v[white]", err))
				time.Sleep(5 * time.Second)
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

	p.updateTextView()
}

func (p *LogsPage) updateTextView() {
	content := ""
	for _, line := range p.logs {
		content += line + "\n"
	}
	ui.Updater.PostUi(func() {
		p.textView.SetText(content)
	})
}

func (p *LogsPage) refreshDisplay() {
	p.mu.Lock()
	search := strings.ToLower(p.searchText)
	logsCopy := make([]string, len(p.logs))
	copy(logsCopy, p.logs)
	p.mu.Unlock()

	if search == "" {
		content := ""
		for _, line := range logsCopy {
			content += line + "\n"
		}
		ui.Updater.PostUi(func() {
			p.textView.SetText(content)
		})
		return
	}

	content := ""
	for _, line := range logsCopy {
		lower := strings.ToLower(line)
		if strings.Contains(lower, search) {
			content += line + "\n"
		}
	}
	ui.Updater.PostUi(func() {
		p.textView.SetText(content)
	})
}

func (p *LogsPage) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *LogsPage) Activate() {
	p.mu.Lock()
	wasPaused := p.isPaused
	p.mu.Unlock()

	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.addLog("[green]日志流已激活，开始接收实时日志...[white]")

	if !wasPaused {
		p.startLogStream()
	}
}

func (p *LogsPage) Deactivate() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Unlock()
}

func (p *LogsPage) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logs = make([]string, 0)
	p.textView.SetText("")
}
