package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/testapi"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/pages"

	"github.com/gdamore/tcell/v2"
)

type CellExport struct {
	Char string `json:"c"`
	Fg   string `json:"fg"`
	Bg   string `json:"bg"`
	Bold bool   `json:"b"`
	Dim  bool   `json:"d"`
}

type ScreenExport struct {
	Width  int            `json:"width"`
	Height int            `json:"height"`
	Title  string         `json:"title"`
	Name   string         `json:"name"`
	Cells  [][]CellExport `json:"cells"`
}

func colorToHex(c tcell.Color) string {
	if c == tcell.ColorDefault {
		return ""
	}
	r, g, b := c.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func captureScreen(simScreen tcell.SimulationScreen, name, title string) ScreenExport {
	cells, w, h := simScreen.GetContents()
	grid := make([][]CellExport, h)
	for y := 0; y < h; y++ {
		row := make([]CellExport, w)
		for x := 0; x < w; x++ {
			idx := y*w + x
			var c CellExport
			if idx < len(cells) {
				sim := cells[idx]
				if len(sim.Runes) > 0 {
					c.Char = string(sim.Runes)
				} else {
					c.Char = " "
				}
				fg, bg, attr := sim.Style.Decompose()
				c.Fg = colorToHex(fg)
				c.Bg = colorToHex(bg)
				c.Bold = (attr & tcell.AttrBold) != 0
				c.Dim = (attr & tcell.AttrDim) != 0
			} else {
				c.Char = " "
			}
			row[x] = c
		}
		grid[y] = row
	}
	return ScreenExport{
		Width:  w,
		Height: h,
		Title:  title,
		Name:   name,
		Cells:  grid,
	}
}

func TestGenerateScreenshotsV3(t *testing.T) {
	if os.Getenv("GEN_SCREENSHOTS") != "1" {
		t.Skip("Skipping screenshot generation unless GEN_SCREENSHOTS=1")
	}

	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	api.InitClient(srv.URL, "")

	outDir := "/tmp/mihomo_screens"
	_ = os.MkdirAll(outDir, 0755)

	targetW, targetH := 118, 32

	pagesToCapture := []struct {
		id       string
		title    string
		pageIdx  int
		subtab   int
		modalKey rune
		waitMs   int
	}{
		{id: "01-dashboard", title: "Dashboard 仪表盘", pageIdx: 0, waitMs: 800},
		{id: "02-proxies", title: "Proxies 代理节点", pageIdx: 1, waitMs: 800},
		{id: "03-connections", title: "Connections 活跃连接", pageIdx: 2, waitMs: 600},
		{id: "04-logs", title: "Logs 实时日志", pageIdx: 3, waitMs: 600},
		{id: "05-profiles", title: "Profiles 配置文件管理", pageIdx: 5, waitMs: 600},
		{id: "06-subscriptions", title: "Subscriptions 订阅管理", pageIdx: 6, waitMs: 600},
		{id: "07-editor-groups", title: "Proxy Groups 代理组配置", pageIdx: 7, subtab: 0, waitMs: 600},
		{id: "08-editor-rules", title: "Rules 路由规则管理", pageIdx: 7, subtab: 1, waitMs: 600},
		{id: "09-editor-providers", title: "Rule Providers 规则提供者", pageIdx: 7, subtab: 2, waitMs: 600},
		{id: "10-settings", title: "Settings 系统与API设置", pageIdx: 8, waitMs: 600},
		{id: "11-palette", title: "Command Palette 命令面板", pageIdx: 0, modalKey: ':', waitMs: 500},
		{id: "12-help", title: "Help 帮助与快捷键", pageIdx: 0, modalKey: '?', waitMs: 500},
	}

	for _, item := range pagesToCapture {
		simScreen := tcell.NewSimulationScreen("UTF-8")
		_ = simScreen.Init()
		simScreen.SetSize(targetW, targetH)

		appInstance := NewApp("mihomoTui", "v0.0-Alpha")
		appInstance.app.SetScreen(simScreen)
		if err := appInstance.Initialize(); err != nil {
			t.Fatalf("Init error: %v", err)
		}

		go appInstance.Run()

		time.Sleep(200 * time.Millisecond)

		// Resize event
		simScreen.SetSize(targetW, targetH)
		appInstance.app.QueueEvent(tcell.NewEventResize(targetW, targetH))
		time.Sleep(200 * time.Millisecond)

		done := make(chan bool)
		ui.Updater.PostUi(func() {
			if item.pageIdx != 0 {
				appInstance.switchPage(item.pageIdx)
			}
			if item.pageIdx == 7 {
				if ed, ok := appInstance.pageLifecycle["editor"].(*pages.EditorPage); ok {
					ed.SwitchTab(item.subtab)
				}
			}
			if item.modalKey != 0 {
				appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, item.modalKey, tcell.ModNone))
			}
			done <- true
		})
		<-done

		waitTime := 400 * time.Millisecond
		if item.waitMs > 0 {
			waitTime = time.Duration(item.waitMs) * time.Millisecond
		}
		time.Sleep(waitTime)

		// Force redraw
		ui.Updater.PostUi(func() {
			appInstance.app.Draw()
		})
		time.Sleep(200 * time.Millisecond)

		exp := captureScreen(simScreen, item.id, item.title)
		b, _ := json.MarshalIndent(exp, "", "  ")
		_ = os.WriteFile(filepath.Join(outDir, item.id+".json"), b, 0644)

		appInstance.Stop()
		time.Sleep(100 * time.Millisecond)
	}
}
