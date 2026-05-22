package subscriptions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Page struct {
	*tview.Flex

	subMgr    *subscription.Manager
	providers []config.ProviderInfo

	subList    *tview.List
	detailView *tview.TextView
	actionBar  *tview.Flex
	status     *tview.TextView

	nameInput     *tview.InputField
	urlInput      *tview.InputField
	intervalInput *tview.InputField

	selectedIndex int
	showInput     bool
	importForm    *tview.Form
	mu            sync.Mutex
}

func NewPage(subMgr *subscription.Manager) *Page {
	page := &Page{
		Flex:          tview.NewFlex(),
		subMgr:        subMgr,
		selectedIndex: -1,
	}
	page.setupUI()
	return page
}

func (p *Page) setupUI() {
	p.setupSubList()

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	p.detailView = tview.NewTextView()
	p.detailView.SetBorder(true)
	p.detailView.SetTitle(" 订阅详情 ")
	p.detailView.SetDynamicColors(true)
	p.detailView.SetWordWrap(true)

	p.actionBar = tview.NewFlex()
	p.actionBar.SetBorder(true)
	p.actionBar.SetTitle(" 操作 ")
	p.setupActionBar()

	p.status = tview.NewTextView()
	p.status.SetBorder(true)
	p.status.SetTitle(" 状态 ")
	p.status.SetDynamicColors(true)
	p.status.SetText("[green]就绪[white]")

	rightPanel.AddItem(p.detailView, 0, 3, false)
	rightPanel.AddItem(p.actionBar, 3, 0, false)
	rightPanel.AddItem(p.status, 3, 0, false)

	p.importForm = tview.NewForm()
	p.importForm.SetBorder(true)
	p.importForm.SetTitle(" 导入订阅 ")
	p.nameInput = tview.NewInputField()
	p.nameInput.SetLabel("名称 ")
	p.nameInput.SetFieldWidth(40)
	p.urlInput = tview.NewInputField()
	p.urlInput.SetLabel("URL/路径 ")
	p.urlInput.SetFieldWidth(60)
	p.intervalInput = tview.NewInputField()
	p.intervalInput.SetLabel("更新间隔(分) ")
	p.intervalInput.SetText("1440")
	p.intervalInput.SetFieldWidth(10)

	p.importForm.AddFormItem(p.nameInput)
	p.importForm.AddFormItem(p.urlInput)
	p.importForm.AddFormItem(p.intervalInput)
	p.importForm.AddButton(" 从URL导入 ", p.onImportURL)
	p.importForm.AddButton(" 从本地导入 ", p.onImportLocal)
	p.importForm.AddButton("  取消  ", p.onCancelImport)
	p.importForm.SetButtonsAlign(tview.AlignCenter)

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.subList, 0, 1, true)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainFlex.AddItem(leftPanel, 0, 2, true)
	mainFlex.AddItem(rightPanel, 0, 3, false)

	p.SetDirection(tview.FlexRow)
	p.AddItem(mainFlex, 0, 1, true)
	p.SetBorder(true)
	p.SetTitle(" 订阅管理 ")
}

func (p *Page) setupSubList() {
	p.subList = tview.NewList()
	p.subList.SetBorder(true)
	p.subList.SetTitle(" mihomo 配置中的提供者 ")
	p.subList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyCtrlD:
			go p.deleteSelected()
			return nil
		case tcell.KeyCtrlU:
			go p.updateSelected()
			return nil
		case tcell.KeyEnter:
			p.showSelectedDetail()
			return nil
		}
		switch ev.Rune() {
		case 'i', 'I':
			p.showImportForm()
			return nil
		case 'u', 'U':
			go p.updateSelected()
			return nil
		case 'd', 'D':
			go p.deleteSelected()
			return nil
		case 'r', 'R':
			go p.refresh()
			return nil
		}
		return ev
	})
	p.subList.SetChangedFunc(func(idx int, _, _ string, _ rune) {
		p.mu.Lock()
		p.selectedIndex = idx
		var pv *config.ProviderInfo
		if idx >= 0 && idx < len(p.providers) {
			pv = &p.providers[idx]
		}
		p.mu.Unlock()
		if pv != nil {
			p.showDetail(*pv)
		}
	})
}

func (p *Page) setupActionBar() {
	refreshBtn := tview.NewButton("[R] 刷新")
	refreshBtn.SetSelectedFunc(func() { go p.refresh() })
	importBtn := tview.NewButton("[I] 导入")
	importBtn.SetSelectedFunc(p.showImportForm)
	updateBtn := tview.NewButton("[U] 更新")
	updateBtn.SetSelectedFunc(func() { go p.updateSelected() })
	deleteBtn := tview.NewButton("[D] 删除")
	deleteBtn.SetSelectedFunc(func() { go p.deleteSelected() })

	p.actionBar.AddItem(refreshBtn, 0, 1, false)
	p.actionBar.AddItem(importBtn, 0, 1, false)
	p.actionBar.AddItem(updateBtn, 0, 1, false)
	p.actionBar.AddItem(deleteBtn, 0, 1, false)
}

func (p *Page) Activate() {
	go p.refresh()
}

func (p *Page) Deactivate() {}

func (p *Page) showImportForm() {
	p.mu.Lock()
	if p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = true
	p.mu.Unlock()
	p.AddItem(p.importForm, 0, 1, true)
}

func (p *Page) hideImportForm() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.mu.Unlock()
	p.RemoveItem(p.importForm)
}

func (p *Page) onImportURL() {
	name := strings.TrimSpace(p.nameInput.GetText())
	url := strings.TrimSpace(p.urlInput.GetText())
	if name == "" || url == "" {
		p.showError("请输入名称和URL")
		return
	}
	p.showStatus(fmt.Sprintf("[yellow]正在从URL导入 %s...[white]", name))
	go p.importFromURL(name, url)
}

func (p *Page) onImportLocal() {
	name := strings.TrimSpace(p.nameInput.GetText())
	path := strings.TrimSpace(p.urlInput.GetText())
	if name == "" || path == "" {
		p.showError("请输入名称和文件路径")
		return
	}
	p.showStatus(fmt.Sprintf("[yellow]正在从本地导入 %s...[white]", name))
	go p.importFromLocal(name, path)
}

func (p *Page) onCancelImport() {
	p.hideImportForm()
	p.nameInput.SetText("")
	p.urlInput.SetText("")
	p.intervalInput.SetText("1440")
}

func (p *Page) importFromURL(name, subURL string) {
	doc, err := p.subMgr.ImportFromURL(context.Background(), subURL)
	if err != nil {
		p.showError(fmt.Sprintf("导入失败: %v", err))
		return
	}

	configPath, cfgErr := config.AddProviderToConfig(name, subURL)
	if cfgErr != nil {
		p.showError(fmt.Sprintf("写入 mihomo 配置失败: %v", cfgErr))
		return
	}
	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[subs] hot-reload failed (will auto-load): %v", err)
	}

	log.Printf("[subs] imported %d proxies to mihomo as %s", len(doc.Proxies), name)

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf("读取 mihomo 配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		formWasShown := p.showInput
		p.showInput = false
		p.mu.Unlock()
		if formWasShown {
			p.RemoveItem(p.importForm)
		}
		p.nameInput.SetText("")
		p.urlInput.SetText("")
		p.intervalInput.SetText("1440")
		p.updateSubList()
		p.status.SetText(fmt.Sprintf("[green]已导入 %s (%d 个节点)[white]", name, len(doc.Proxies)))
	})
}

func (p *Page) importFromLocal(name, path string) {
	doc, err := p.subMgr.ImportFromLocal(path)
	if err != nil {
		p.showError(fmt.Sprintf("导入失败: %v", err))
		return
	}

	log.Printf("[subs] parsed %d proxies from %s", len(doc.Proxies), name)

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		formWasShown := p.showInput
		p.showInput = false
		p.mu.Unlock()
		if formWasShown {
			p.RemoveItem(p.importForm)
		}
		p.nameInput.SetText("")
		p.urlInput.SetText("")
		p.intervalInput.SetText("1440")
		p.status.SetText(fmt.Sprintf("[yellow]本地文件 %s 解析成功 (%d 个节点)[white]\n[gray]本地导入不会写入 mihomo 配置，请手动编辑配置或使用 URL 导入[white]", name, len(doc.Proxies)))
	})
}

func (p *Page) updateSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError("请先选择一个提供者")
		return
	}
	pv := p.providers[p.selectedIndex]
	p.mu.Unlock()

	if pv.URL == "" {
		p.showError("该提供者没有 URL，无法更新")
		return
	}

	p.showStatus(fmt.Sprintf("[yellow]正在更新 %s...[white]", pv.Name))

	doc, err := p.subMgr.ImportFromURL(context.Background(), pv.URL)
	if err != nil {
		p.showError(fmt.Sprintf("更新失败: %v", err))
		return
	}

	_, cfgErr := config.AddProviderToConfig(pv.Name, pv.URL)
	if cfgErr != nil {
		p.showError(fmt.Sprintf("更新 mihomo 配置失败: %v", cfgErr))
		return
	}

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf("[green]%s 已更新 (%d 个节点)[white]", pv.Name, len(doc.Proxies)))
	})
}

func (p *Page) deleteSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError("请先选择一个提供者")
		return
	}
	pv := p.providers[p.selectedIndex]
	p.mu.Unlock()

	if err := config.RemoveProviderFromConfig(pv.Name); err != nil {
		p.showError(fmt.Sprintf("删除失败: %v", err))
		return
	}

	configPath := config.FindMihomoConfigPath()
	if configPath != "" {
		api.Client.ReloadConfig(configPath)
	}

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf("[green]已从 mihomo 配置中移除 %s[white]", pv.Name))
	})
}

func (p *Page) refresh() {
	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf("读取 mihomo 配置失败: %v", err))
		return
	}
	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
	})
}

func (p *Page) updateSubList() {
	p.subList.Clear()
	p.mu.Lock()
	providers := p.providers
	p.mu.Unlock()

	if len(providers) == 0 {
		p.subList.AddItem("(暂无配置)", "按 I 键导入订阅到 mihomo 配置", 0, nil)
		return
	}

	for _, pv := range providers {
		sec := pv.URL
		if sec == "" {
			sec = "本地"
		}
		if pv.Interval > 0 {
			sec += fmt.Sprintf(" | 更新间隔: %d分", pv.Interval)
		}
		p.subList.AddItem(pv.Name, sec, 0, nil)
	}
}

func (p *Page) showDetail(pv config.ProviderInfo) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[yellow]名称:[white] %s\n", pv.Name))
	b.WriteString(fmt.Sprintf("[yellow]URL:[white] %s\n", pv.URL))
	b.WriteString(fmt.Sprintf("[yellow]更新间隔:[white] %d 分钟\n", pv.Interval))
	b.WriteString(fmt.Sprintf("\n[gray]此提供者存在于 mihomo 配置文件中[white]\n"))
	p.detailView.SetText(b.String())
}

func (p *Page) showSelectedDetail() {
	p.mu.Lock()
	idx, providers := p.selectedIndex, p.providers
	p.mu.Unlock()
	if idx >= 0 && idx < len(providers) {
		p.showDetail(providers[idx])
	}
}

func (p *Page) showError(msg string) {
	log.Printf("[subs] error: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[red]%s[white]", msg)) })
}

func (p *Page) showSuccess(msg string) {
	log.Printf("[subs] success: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[green]%s[white]", msg)) })
}

func (p *Page) showStatus(msg string) {
	go ui.Updater.UpdateUi(func() { p.status.SetText(msg) })
}
