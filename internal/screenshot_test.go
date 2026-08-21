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

func TestGenerateScreenshots(t *testing.T) {
	if os.Getenv("GEN_SCREENSHOTS") != "1" {
		t.Skip("Skipping screenshot generation unless GEN_SCREENSHOTS=1")
	}

	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	api.InitClient(srv.URL, "")

	simScreen := tcell.NewSimulationScreen("UTF-8")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simScreen: %v", err)
	}

	appInstance := NewApp("mihomoTui", "v0.0-Alpha")
	appInstance.app.SetScreen(simScreen)

	if err := appInstance.Initialize(); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	// Lock size to 118x32
	simScreen.SetSize(118, 32)

	go appInstance.Run()
	defer appInstance.Stop()

	time.Sleep(300 * time.Millisecond)

	outDir := "/tmp/mihomo_screens"
	_ = os.MkdirAll(outDir, 0755)

	savePage := func(filename, title string) {
		time.Sleep(300 * time.Millisecond)
		ui.Updater.PostUi(func() {
			simScreen.SetSize(118, 32)
			appInstance.app.Draw()
		})
		time.Sleep(200 * time.Millisecond)
		exp := captureScreen(simScreen, filename, title)
		b, _ := json.MarshalIndent(exp, "", "  ")
		_ = os.WriteFile(filepath.Join(outDir, filename+".json"), b, 0644)
		t.Logf("Saved %s (%s) [%dx%d]", filename, title, exp.Width, exp.Height)
	}

	// 1. Dashboard (F1)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(0)
	})
	savePage("01-dashboard", "Dashboard 仪表盘")

	// 2. Proxies (F2)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(1)
	})
	savePage("02-proxies", "Proxies 代理节点")

	// 3. Connections (F3)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(2)
	})
	savePage("03-connections", "Connections 活跃连接")

	// 4. Logs (F4)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(3)
	})
	savePage("04-logs", "Logs 实时日志")

	// 5. ConfigMgr / Profiles (F5)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(5)
	})
	savePage("05-profiles", "Profiles 配置文件管理")

	// 6. Subscriptions (F6)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(6)
	})
	savePage("06-subscriptions", "Subscriptions 订阅管理")

	// 7. Editor - Proxy Groups (F7 subtab 0)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(7)
	})
	savePage("07-editor-groups", "Proxy Groups 代理组配置")

	// 8. Editor - Rules (F7 subtab 1)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(7)
		if ed, ok := appInstance.pageLifecycle["editor"].(*pages.EditorPage); ok {
			_ = ed
		}
		appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	})
	savePage("08-editor-rules", "Rules 路由规则管理")

	// 9. Editor - Rule Providers (F7 subtab 2)
	ui.Updater.PostUi(func() {
		appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	})
	savePage("09-editor-providers", "Rule Providers 规则提供者")

	// 10. Settings (F8)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(8)
	})
	savePage("10-settings", "Settings 系统与API设置")

	// 11. Command Palette (:)
	ui.Updater.PostUi(func() {
		appInstance.switchPage(0)
		appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, ':', tcell.ModNone))
	})
	savePage("11-palette", "Command Palette 命令面板")
	ui.Updater.PostUi(func() {
		appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	})

	// 12. Help Modal (?)
	ui.Updater.PostUi(func() {
		appInstance.handleGlobalKeys(tcell.NewEventKey(tcell.KeyRune, '?', tcell.ModNone))
	})
	savePage("12-help", "Help 帮助与快捷键")
}
