package mergepreview

import (
	"fmt"
	"log"

	"mihomoTui/internal/merge"

	"github.com/rivo/tview"
)

// Page provides a preview of what would change when merging configs.
type Page struct {
	*tview.Flex

	mergeEng *merge.Engine

	// Components
	diffView  *tview.TextView
	summary   *tview.TextView
	status    *tview.TextView
	applyBtn  *tview.Button
	cancelBtn *tview.Button
}

// NewPage creates a new merge preview page.
func NewPage(mergeEng *merge.Engine) *Page {
	page := &Page{
		Flex:     tview.NewFlex(),
		mergeEng: mergeEng,
	}

	page.setupUI()
	return page
}

func (p *Page) setupUI() {
	// Main diff/content area
	p.diffView = tview.NewTextView()
	p.diffView.SetBorder(true)
	p.diffView.SetTitle(" 合并预览 ")
	p.diffView.SetDynamicColors(true)
	p.diffView.SetWordWrap(true)

	// Summary
	p.summary = tview.NewTextView()
	p.summary.SetBorder(true)
	p.summary.SetTitle(" 变更摘要 ")
	p.summary.SetDynamicColors(true)
	p.summary.SetText("[gray]无待处理的合并操作[white]")

	// Action buttons
	actionBar := tview.NewFlex()
	p.applyBtn = tview.NewButton("  [green]应用合并[white]  ")
	p.cancelBtn = tview.NewButton("  [red]取消[white]  ")
	actionBar.AddItem(p.applyBtn, 0, 1, false)
	actionBar.AddItem(p.cancelBtn, 0, 1, false)

	// Status
	p.status = tview.NewTextView()
	p.status.SetBorder(true)
	p.status.SetTitle(" 状态 ")
	p.status.SetDynamicColors(true)
	p.status.SetText("[green]就绪[white]")

	// Layout
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(p.summary, 0, 2, false)
	rightPanel.AddItem(actionBar, 1, 0, false)
	rightPanel.AddItem(p.status, 3, 0, false)

	p.SetDirection(tview.FlexColumn)
	p.AddItem(p.diffView, 0, 3, true)
	p.AddItem(rightPanel, 0, 1, false)

	p.SetBorder(true)
	p.SetTitle(" 合并预览 ")
}

// Activate loads data when the page becomes visible.
func (p *Page) Activate() {
	log.Printf("Activating merge preview page")
}

// Deactivate cleans up.
func (p *Page) Deactivate() {
	log.Printf("Deactivating merge preview page")
}

// ShowPreview displays a merge preview result.
func (p *Page) ShowPreview(added, removed, modified []string) {
	content := "[yellow]=== 变更预览 ===[white]\n\n"

	if len(added) > 0 {
		content += "[green]新增:[white]\n"
		for _, item := range added {
			content += fmt.Sprintf("  + %s\n", item)
		}
		content += "\n"
	}

	if len(removed) > 0 {
		content += "[red]移除:[white]\n"
		for _, item := range removed {
			content += fmt.Sprintf("  - %s\n", item)
		}
		content += "\n"
	}

	if len(modified) > 0 {
		content += "[yellow]修改:[white]\n"
		for _, item := range modified {
			content += fmt.Sprintf("  ~ %s\n", item)
		}
		content += "\n"
	}

	if len(added) == 0 && len(removed) == 0 && len(modified) == 0 {
		content += "[gray]无变更[white]"
	}

	p.diffView.SetText(content)

	summaryText := fmt.Sprintf(
		"[green]+%d 新增[white]  [red]-%d 移除[white]  [yellow]~%d 修改[white]",
		len(added), len(removed), len(modified),
	)
	p.summary.SetText(summaryText)
}
