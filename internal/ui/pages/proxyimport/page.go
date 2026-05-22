package proxyimport

import (
	"fmt"
	"log"

	"mihomoTui/internal/merge"
	"mihomoTui/internal/proxygroup"

	"github.com/rivo/tview"
)

// Page provides the UI for importing proxy groups from subscriptions or config files.
type Page struct {
	*tview.Flex

	pgMgr    *proxygroup.Manager
	mergeEng *merge.Engine

	// Components
	sourceView  *tview.TextView
	groupList   *tview.List
	dependencyView *tview.TextView
	status      *tview.TextView
	modeSelect  *tview.DropDown

	// Data
	sourceGroups []string
}

// NewPage creates a new proxy group import page.
func NewPage(pgMgr *proxygroup.Manager, mergeEng *merge.Engine) *Page {
	page := &Page{
		Flex:    tview.NewFlex(),
		pgMgr:   pgMgr,
		mergeEng: mergeEng,
	}

	page.setupUI()
	return page
}

func (p *Page) setupUI() {
	// Left panel: source config display
	p.sourceView = tview.NewTextView()
	p.sourceView.SetBorder(true)
	p.sourceView.SetTitle(" 导入源 ")
	p.sourceView.SetDynamicColors(true)
	p.sourceView.SetWordWrap(true)

	// Middle panel: extracted groups
	p.groupList = tview.NewList()
	p.groupList.SetBorder(true)
	p.groupList.SetTitle(" 检测到的代理组 ")

	// Right panel: dependency analysis
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	p.dependencyView = tview.NewTextView()
	p.dependencyView.SetBorder(true)
	p.dependencyView.SetTitle(" 依赖分析 ")
	p.dependencyView.SetDynamicColors(true)

	p.modeSelect = tview.NewDropDown()
	p.modeSelect.SetLabel("导入模式: ")
	p.modeSelect.AddOption("合并 (Merge)", nil)
	p.modeSelect.AddOption("替换 (Replace)", nil)
	p.modeSelect.AddOption("跳过已有 (Skip)", nil)
	p.modeSelect.AddOption("重命名 (Rename)", nil)
	p.modeSelect.SetCurrentOption(0)

	p.status = tview.NewTextView()
	p.status.SetBorder(true)
	p.status.SetTitle(" 状态 ")
	p.status.SetDynamicColors(true)
	p.status.SetText("[green]就绪 — 使用配置文件或订阅作为导入源[white]")

	importBtn := tview.NewButton(" 导入选中组 ")
	importBtn.SetSelectedFunc(p.importSelectedGroups)

	rightPanel.AddItem(p.dependencyView, 0, 3, false)
	rightPanel.AddItem(p.modeSelect, 1, 0, false)
	rightPanel.AddItem(importBtn, 1, 0, false)
	rightPanel.AddItem(p.status, 3, 0, false)

	// Main layout
	leftMid := tview.NewFlex().SetDirection(tview.FlexColumn)
	leftMid.AddItem(p.sourceView, 0, 2, false)
	leftMid.AddItem(p.groupList, 0, 1, true)

	p.SetDirection(tview.FlexRow)
	p.AddItem(leftMid, 0, 1, true)
	p.AddItem(rightPanel, 0, 2, false)

	p.SetBorder(true)
	p.SetTitle(" 代理组导入 ")
}

// Activate loads data when the page becomes visible.
func (p *Page) Activate() {
	log.Printf("Activating proxy import page")
	p.refresh()
}

// Deactivate cleans up.
func (p *Page) Deactivate() {
	log.Printf("Deactivating proxy import page")
}

func (p *Page) refresh() {
	p.sourceView.SetText("[gray]此处显示导入源的代理组列表。\n从订阅管理页面选择一个已解析的配置文件，\n或使用当前运行配置作为源。[white]")
	p.groupList.Clear()
	p.groupList.AddItem("(暂无数据)", "请先导入订阅或选择源", 0, nil)
}

func (p *Page) importSelectedGroups() {
	_, modeStr := p.modeSelect.GetCurrentOption()
	mode := parseImportMode(modeStr)

	p.showStatus(fmt.Sprintf("[yellow]正在导入代理组 (模式: %s)...[white]", modeStr))

	// Placeholder — in production, this would:
	// 1. Get the selected source config
	// 2. Extract groups
	// 3. Run dependency analysis
	// 4. Merge into current config
	// 5. Reload mihomo

	go func() {
		_ = mode
		p.showSuccess("代理组导入功能就绪 (需集成当前运行配置)")
	}()
}

func (p *Page) showStatus(msg string) {
	if p.status != nil {
		p.status.SetText(msg)
	}
}

func (p *Page) showSuccess(msg string) {
	log.Printf("[proxyimport] %s", msg)
	if p.status != nil {
		p.status.SetText(fmt.Sprintf("[green]%s[white]", msg))
	}
}

func parseImportMode(modeStr string) string {
	switch {
	case contains(modeStr, "Merge"):
		return "merge"
	case contains(modeStr, "Replace"):
		return "replace"
	case contains(modeStr, "Skip"):
		return "skip"
	case contains(modeStr, "Rename"):
		return "rename"
	default:
		return "merge"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
