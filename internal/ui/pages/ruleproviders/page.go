package ruleproviders

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

// QuickPreset is a pre-configured rule-provider from Loyalsoldier/clash-rules
type QuickPreset struct {
	Name     string
	URL      string
	Behavior string
}

var quickPresets = []QuickPreset{
	{"reject", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/reject.txt", "domain"},
	{"proxy", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/proxy.txt", "domain"},
	{"direct", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/direct.txt", "domain"},
	{"private", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/private.txt", "domain"},
	{"icloud", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/icloud.txt", "domain"},
	{"apple", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/apple.txt", "domain"},
	{"google", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/google.txt", "domain"},
	{"gfw", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/gfw.txt", "domain"},
	{"tld-not-cn", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/tld-not-cn.txt", "domain"},
	{"telegramcidr", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/telegramcidr.txt", "ipcidr"},
	{"cncidr", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/cncidr.txt", "ipcidr"},
	{"lancidr", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/lancidr.txt", "ipcidr"},
	{"applications", "https://raw.githubusercontent.com/Loyalsoldier/clash-rules/release/applications.txt", "classical"},
}

type Page struct {
	*tview.Flex

	providers []config.RuleProviderEntry

	subList    *tview.List
	detailView *tview.TextView
	actionBar  *tview.Flex
	status     *tview.TextView

	importForm     *tview.Form
	nameInput      *tview.InputField
	urlInput       *tview.InputField
	behaviorSelect *tview.DropDown
	intervalInput  *tview.InputField

	selectedIndex int
	showInput     bool
	mu            sync.Mutex

	pasteTextArea   *tview.TextArea
	pasteImportForm *tview.Flex
}

func NewPage() *Page {
	page := &Page{
		Flex:          tview.NewFlex(),
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
	p.detailView.SetTitle(" 规则提供者详情 ")
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
	p.importForm.SetTitle(" 导入规则提供者 ")
	p.nameInput = tview.NewInputField()
	p.nameInput.SetLabel("名称 ")
	p.nameInput.SetFieldWidth(30)
	p.urlInput = tview.NewInputField()
	p.urlInput.SetLabel("URL ")
	p.urlInput.SetFieldWidth(50)
	p.behaviorSelect = tview.NewDropDown()
	p.behaviorSelect.SetLabel("行为 ")
	p.behaviorSelect.AddOption("domain", nil)
	p.behaviorSelect.AddOption("ipcidr", nil)
	p.behaviorSelect.AddOption("classical", nil)
	p.behaviorSelect.SetCurrentOption(0)
	p.intervalInput = tview.NewInputField()
	p.intervalInput.SetLabel("更新间隔(秒) ")
	p.intervalInput.SetFieldWidth(10)
	p.intervalInput.SetText("3600")

	p.importForm.AddFormItem(p.nameInput)
	p.importForm.AddFormItem(p.urlInput)
	p.importForm.AddFormItem(p.behaviorSelect)
	p.importForm.AddFormItem(p.intervalInput)
	p.importForm.AddButton(" 导入 ", p.onImportClicked)
	p.importForm.AddButton(" 取消 ", p.onCancelImport)
	p.importForm.SetButtonsAlign(tview.AlignCenter)

	p.setupPasteImport()

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.subList, 0, 1, true)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainFlex.AddItem(leftPanel, 0, 2, true)
	mainFlex.AddItem(rightPanel, 0, 3, false)

	p.SetDirection(tview.FlexRow)
	p.AddItem(mainFlex, 0, 1, true)
	p.SetBorder(true)
	p.SetTitle(" 规则提供者管理 ")
}

func (p *Page) setupSubList() {
	p.subList = tview.NewList()
	p.subList.SetBorder(true)
	p.subList.SetTitle(" 规则提供者 ")
	p.subList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEnter:
			p.showSelectedDetail()
			return nil
		}
		switch ev.Rune() {
		case 'r', 'R':
			go p.refresh()
			return nil
		case 'd', 'D':
			go p.deleteSelected()
			return nil
		case 'i', 'I':
			p.showImportForm()
			return nil
		case 'p', 'P':
			p.showPasteImport()
			return nil
		}
		return ev
	})
	p.subList.SetChangedFunc(func(idx int, _ string, _ string, _ rune) {
		p.mu.Lock()
		p.selectedIndex = idx
		var rp *config.RuleProviderEntry
		if idx >= 0 && idx < len(p.providers) {
			rp = &p.providers[idx]
		}
		p.mu.Unlock()
		if rp != nil {
			p.showDetail(*rp)
		}
	})
}

func (p *Page) setupActionBar() {
	refreshBtn := tview.NewButton("[R] 刷新")
	refreshBtn.SetSelectedFunc(func() { go p.refresh() })
	importBtn := tview.NewButton("[I] 导入")
	importBtn.SetSelectedFunc(p.showImportForm)
	quickImportBtn := tview.NewButton("[Q] 快速导入")
	quickImportBtn.SetSelectedFunc(p.showQuickImportPage)
	pasteBtn := tview.NewButton("[P] 粘贴导入")
	pasteBtn.SetSelectedFunc(p.showPasteImport)
	deleteBtn := tview.NewButton("[D] 删除")
	deleteBtn.SetSelectedFunc(func() { go p.deleteSelected() })

	p.actionBar.AddItem(refreshBtn, 0, 1, false)
	p.actionBar.AddItem(importBtn, 0, 1, false)
	p.actionBar.AddItem(quickImportBtn, 0, 1, false)
	p.actionBar.AddItem(pasteBtn, 0, 1, false)
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

	p.importForm.SetTitle(" 导入规则提供者 ")
	p.nameInput.SetText("")
	p.urlInput.SetText("")
	p.behaviorSelect.SetCurrentOption(0)
	p.intervalInput.SetText("3600")
	p.AddItem(p.importForm, 0, 1, true)
}

func (p *Page) onCancelImport() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.mu.Unlock()
	p.RemoveItem(p.importForm)
}

func (p *Page) onImportClicked() {
	name := strings.TrimSpace(p.nameInput.GetText())
	url := strings.TrimSpace(p.urlInput.GetText())
	_, behavior := p.behaviorSelect.GetCurrentOption()
	if behavior == "" {
		behavior = "domain"
	}
	intervalStr := strings.TrimSpace(p.intervalInput.GetText())
	interval := 3600
	if intervalStr != "" {
		fmt.Sscanf(intervalStr, "%d", &interval)
	}

	if name == "" {
		p.showError("名称不能为空")
		return
	}
	if url == "" {
		p.showError("URL 不能为空")
		return
	}

	p.showStatus(fmt.Sprintf("[yellow]正在导入规则提供者 %s...[white]", name))
	go p.importFromURL(name, url, behavior, interval)
}

func (p *Page) showQuickImportPage() {
	p.mu.Lock()
	if p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = true
	p.mu.Unlock()

	quickList := tview.NewList()
	quickList.SetBorder(true)
	quickList.SetTitle(" 快速导入 - Loyalsoldier/clash-rules ")
	quickList.SetTitleAlign(tview.AlignLeft)

	for _, preset := range quickPresets {
		desc := fmt.Sprintf("%s | %s", preset.Behavior, preset.URL)
		quickList.AddItem(preset.Name, desc, 0, nil)
	}

	quickList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			p.hideQuickImport()
			return nil
		}
		if ev.Key() == tcell.KeyEnter {
			idx := quickList.GetCurrentItem()
			if idx >= 0 && idx < len(quickPresets) {
				preset := quickPresets[idx]
				p.hideQuickImport()
				go p.importFromURL(preset.Name, preset.URL, preset.Behavior, 3600)
			}
			return nil
		}
		return ev
	})

	p.showStatus(fmt.Sprintf("[yellow]按 Enter 选择预设，Esc 返回[white]"))

	p.AddItem(quickList, 0, 1, true)
}

func (p *Page) hideQuickImport() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.mu.Unlock()
	go ui.Updater.UpdateUi(func() {
		items := p.GetItemCount()
		if items > 1 {
			p.RemoveItem(p.GetItem(items - 1))
		}
		p.status.SetText("[green]就绪[white]")
	})
}

func (p *Page) importFromURL(name, url, behavior string, interval int) {
	configPath, err := config.AddRuleProviderToConfig(name, url, behavior, interval)
	if err != nil {
		p.showError(fmt.Sprintf("导入失败: %v", err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
	}

	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
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
		p.updateSubList()
		p.status.SetText(fmt.Sprintf("[green]规则提供者 %s 已导入[white]", name))
	})
}

func (p *Page) deleteSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError("请先选择一个规则提供者")
		return
	}
	rp := p.providers[p.selectedIndex]
	p.mu.Unlock()

	configPath, err := config.DeleteRuleProviderFromConfig(rp.Name)
	if err != nil {
		p.showError(fmt.Sprintf("删除失败: %v", err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
	}

	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf("[green]已删除规则提供者 %s[white]", rp.Name))
	})
}

func (p *Page) refresh() {
	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	apiProviders, apiErr := api.Client.GetRuleProviders()
	if apiErr != nil {
		log.Printf("[ruleproviders] API GetRuleProviders failed: %v", apiErr)
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		_ = apiProviders
		p.status.SetText("[green]刷新完成[white]")
	})
}

func (p *Page) updateSubList() {
	p.subList.Clear()
	p.mu.Lock()
	providers := p.providers
	p.mu.Unlock()

	if len(providers) == 0 {
		p.subList.AddItem("(暂无规则提供者)", "按 I 键导入新规则提供者", 0, nil)
		return
	}

	for _, rp := range providers {
		sec := rp.Behavior
		if rp.URL != "" {
			urlDisplay := rp.URL
			if len(urlDisplay) > 50 {
				urlDisplay = urlDisplay[:50] + "..."
			}
			sec += " | " + urlDisplay
		}
		if rp.Interval > 0 {
			sec += fmt.Sprintf(" | %ds", rp.Interval)
		}
		p.subList.AddItem(rp.Name, sec, 0, nil)
	}
}

func (p *Page) showDetail(rp config.RuleProviderEntry) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[yellow]名称:[white] %s\n", rp.Name))
	b.WriteString(fmt.Sprintf("[yellow]行为:[white] %s\n", rp.Behavior))
	if rp.URL != "" {
		b.WriteString(fmt.Sprintf("[yellow]URL:[white] %s\n", rp.URL))
	}
	if rp.Path != "" {
		b.WriteString(fmt.Sprintf("[yellow]路径:[white] %s\n", rp.Path))
	}
	if rp.Type != "" {
		b.WriteString(fmt.Sprintf("[yellow]类型:[white] %s\n", rp.Type))
	}
	if rp.Interval > 0 {
		b.WriteString(fmt.Sprintf("[yellow]更新间隔:[white] %ds\n", rp.Interval))
	}
	b.WriteString("\n[gray]配置文件中的规则提供者定义[white]")
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
	log.Printf("[ruleproviders] error: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[red]%s[white]", msg)) })
}

func (p *Page) showSuccess(msg string) {
	log.Printf("[ruleproviders] success: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[green]%s[white]", msg)) })
}

func (p *Page) showStatus(msg string) {
	go ui.Updater.UpdateUi(func() { p.status.SetText(msg) })
}

type pasteProviderDef struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
	Interval int    `yaml:"interval"`
}

func (p *Page) setupPasteImport() {
	p.pasteTextArea = tview.NewTextArea()
	p.pasteTextArea.SetPlaceholder(`粘贴规则提供者定义 (YAML 格式)，例如:

  reject:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/.../reject.txt"
    path: ./ruleset/reject.yaml
    interval: 86400

  google:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/.../google.txt"
    path: ./ruleset/google.yaml
    interval: 86400`)
	p.pasteTextArea.SetBorder(true)

	btnBar := tview.NewFlex()
	importBtn := tview.NewButton(" 导入 ")
	importBtn.SetSelectedFunc(func() {
		if p.showInput {
			go p.processPasteImportYAML()
		}
	})
	cancelBtn := tview.NewButton(" 取消 ")
	cancelBtn.SetSelectedFunc(func() {
		p.hidePasteImport()
	})
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(importBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(cancelBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)

	p.pasteImportForm = tview.NewFlex().SetDirection(tview.FlexRow)
	p.pasteImportForm.SetBorder(true)
	p.pasteImportForm.SetTitle(" 粘贴导入规则提供者 ")
	p.pasteImportForm.AddItem(p.pasteTextArea, 0, 1, true)
	p.pasteImportForm.AddItem(btnBar, 3, 0, false)
}

func (p *Page) showPasteImport() {
	p.mu.Lock()
	if p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = true
	p.mu.Unlock()

	p.pasteTextArea.SetText("", true)
	p.AddItem(p.pasteImportForm, 0, 3, true)
	p.showStatus("[yellow]粘贴 YAML 格式的规则提供者定义，按导入按钮开始[white]")
}

func (p *Page) hidePasteImport() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.mu.Unlock()
	p.RemoveItem(p.pasteImportForm)
	p.showStatus("[green]就绪[white]")
}

func (p *Page) processPasteImportYAML() {
	// Capture text BEFORE hiding the form — removing from UI tree may clear buffer
	text := p.pasteTextArea.GetText()

	p.hidePasteImport()

	if strings.TrimSpace(text) == "" {
		p.showStatus("[red]粘贴内容为空[white]")
		return
	}

	var rawProviders map[string]pasteProviderDef
	if err := yaml.Unmarshal([]byte(text), &rawProviders); err != nil {
		p.showStatus(fmt.Sprintf("[red]YAML 解析失败: %v[white]", err))
		return
	}

	if len(rawProviders) == 0 {
		p.showStatus("[red]未解析到任何规则提供者[white]")
		return
	}

	p.showStatus(fmt.Sprintf("[yellow]正在导入 %d 个规则提供者...[white]", len(rawProviders)))

	var batch []config.RuleProviderBatchEntry
	var errs []string

	for name, def := range rawProviders {
		if name == "" || def.URL == "" || def.Behavior == "" {
			errs = append(errs, fmt.Sprintf("提供者 %q 缺少必要字段 (url/behavior)", name))
			continue
		}
		interval := def.Interval
		if interval <= 0 {
			interval = 86400
		}
		batch = append(batch, config.RuleProviderBatchEntry{
			Name:     name,
			URL:      def.URL,
			Behavior: def.Behavior,
			Interval: interval,
		})
	}

	if len(batch) == 0 {
		msg := "[red]没有有效的规则提供者可导入[white]"
		if len(errs) > 0 {
			msg += fmt.Sprintf(" | %d 个跳过", len(errs))
		}
		p.showStatus(msg)
		return
	}

	cp, err := config.BatchAddRuleProvidersToConfig(batch)
	if err != nil {
		log.Printf("[ruleproviders] batch import failed: %v", err)
		p.showStatus(fmt.Sprintf("[red]批量导入失败: %v[white]", err))
		return
	}

	if err := api.Client.ReloadConfig(cp); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
		p.showStatus(fmt.Sprintf("[red]配置重载失败: %v[white]", err))
		return
	}

	msg := fmt.Sprintf("[green]成功导入 %d 个规则提供者[white]", len(batch))
	if len(errs) > 0 {
		msg += fmt.Sprintf(" | [red]%d 个跳过[white]", len(errs))
		log.Printf("[ruleproviders] paste import skipped: %v", errs)
	}

	providersList, _ := config.GetRuleProvidersFromConfig()
	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providersList
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(msg)
	})
}
