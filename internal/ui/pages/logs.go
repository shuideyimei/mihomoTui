package pages

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"

	"github.com/rivo/tview"
)

// Logs represents the logs page
type Logs struct {
	*LogsPage
}

// Activate activates the logs page and starts streaming
func (l *Logs) Activate() {
	log.Printf("Activating logs page")
	l.LogsPage.Activate()
}

// Deactivate deactivates the logs page and stops streaming
func (l *Logs) Deactivate() {
	log.Printf("Deactivating logs page")
	l.LogsPage.Deactivate()
}

// LogsPage represents the logs page with streaming support
type LogsPage struct {
	*tview.Flex
	textView *tview.TextView

	// Streaming control
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	logs   []string

	// Settings
	maxLines int
}

// NewLogsPage creates a new logs page with streaming support
func NewLogsPage() *LogsPage {
	page := &LogsPage{
		Flex:     tview.NewFlex(),
		textView: tview.NewTextView(),
		logs:     make([]string, 0),
		maxLines: 1000, // Keep last 1000 lines
	}

	page.setupUI()
	return page
}

// setupUI initializes the logs page UI
func (p *LogsPage) setupUI() {
	// Configure text view
	p.textView.SetBorder(true)
	p.textView.SetTitle(" 实时日志 ")
	p.textView.SetDynamicColors(true)
	p.textView.SetScrollable(true)
	p.textView.SetChangedFunc(func() {
		// Auto-scroll to bottom
		go func() {
			time.Sleep(10 * time.Millisecond)
			p.textView.ScrollToEnd()
		}()
	})

	// Add text view to flex container
	p.SetDirection(tview.FlexRow)
	p.AddItem(p.textView, 0, 1, true)
}

// startLogStream starts streaming logs from the API
func (p *LogsPage) startLogStream() {
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			default:
				err := api.StreamClient.StreamLogs(p.ctx, p.onLogReceived)
				if err != nil && err != context.Canceled {
					// Connection lost, show error and retry
					p.addLog(fmt.Sprintf("[red]连接错误: %v[white]", err))
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

// onLogReceived handles incoming log messages
func (p *LogsPage) onLogReceived(log *models.Log) {
	logText := p.formatLog(log)
	p.addLog(logText)
}

// formatLog formats a log entry for display
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

// addLog adds a new log line to the display
func (p *LogsPage) addLog(logText string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add to logs slice
	p.logs = append(p.logs, logText)

	// Trim if exceeding max lines
	if len(p.logs) > p.maxLines {
		p.logs = p.logs[len(p.logs)-p.maxLines:]
	}

	// Update text view
	content := ""
	for _, line := range p.logs {
		content += line + "\n"
	}

	// Update UI in main goroutine
	go func() {
		p.textView.SetText(content)
	}()
}

// Stop stops the log streaming
func (p *LogsPage) Stop() {
	p.cancel()
}

// Activate starts log streaming when page becomes active
func (p *LogsPage) Activate() {
	// Create new context for this activation
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Add activation message
	p.addLog("[green]日志流已激活，开始接收实时日志...[white]")

	// Start streaming
	p.startLogStream()
}

// Deactivate stops log streaming when page becomes inactive
func (p *LogsPage) Deactivate() {
	// Cancel streaming context
	if p.cancel != nil {
		p.cancel()
	}
	// Clear logs
	p.Clear()
}

// Clear clears all logs
func (p *LogsPage) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logs = make([]string, 0)
	p.textView.SetText("")
}
