package proxygroups

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
)

type Page struct {
	*tview.Flex

	groups []config.ProxyGroupEntry

	groupList  *tview.List
	detailView *tview.TextView
	actionBar  *tview.Flex
	status     *tview.TextView

	nameInput    *tview.InputField
	typeSelect   *tview.DropDown
	proxiesInput *tview.InputField
	useInput     *tview.InputField
	urlInput     *tview.InputField
	filterInput  *tview.InputField

	groupForm     *tview.Form
	selectedIndex int
	showInput     bool
	editingGroup  string
	mu            sync.Mutex
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
	p.setupGroupList()

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	p.detailView = tview.NewTextView()
	p.detailView.SetBorder(true)
	p.detailView.SetTitle(" 代理组详情 ")
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

	p.groupForm = tview.NewForm()
	p.groupForm.SetBorder(true)
	p.groupForm.SetTitle(" 添加代理组 ")
	p.nameInput = tview.NewInputField()
	p.nameInput.SetLabel("名称 ")
	p.nameInput.SetFieldWidth(30)
	p.typeSelect = tview.NewDropDown()
	p.typeSelect.SetLabel("类型 ")
	for _, t := range config.ValidGroupTypes {
		p.typeSelect.AddOption(t, nil)
	}
	p.typeSelect.SetCurrentOption(0)
	p.proxiesInput = tview.NewInputField()
	p.proxiesInput.SetLabel("代理节点(逗号分隔) ")
	p.proxiesInput.SetFieldWidth(40)
	p.useInput = tview.NewInputField()
	p.useInput.SetLabel("提供者(逗号分隔) ")
	p.useInput.SetFieldWidth(40)
	p.urlInput = tview.NewInputField()
	p.urlInput.SetLabel("测试URL ")
	p.urlInput.SetFieldWidth(40)
	p.filterInput = tview.NewInputField()
	p.filterInput.SetLabel("过滤规则 ")
	p.filterInput.SetFieldWidth(30)

	p.groupForm.AddFormItem(p.nameInput)
	p.groupForm.AddFormItem(p.typeSelect)
	p.groupForm.AddFormItem(p.proxiesInput)
	p.groupForm.AddFormItem(p.useInput)
	p.groupForm.AddFormItem(p.urlInput)
	p.groupForm.AddFormItem(p.filterInput)
	p.groupForm.AddButton(" 保存 ", p.onSaveClicked)
	p.groupForm.AddButton(" 取消 ", p.onCancelClicked)
	p.groupForm.SetButtonsAlign(tview.AlignCenter)

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.groupList, 0, 1, true)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainFlex.AddItem(leftPanel, 0, 2, true)
	mainFlex.AddItem(rightPanel, 0, 3, false)

	p.SetDirection(tview.FlexRow)
	p.AddItem(mainFlex, 0, 1, true)
	p.SetBorder(true)
	p.SetTitle(" 代理组管理 ")
}

func (p *Page) setupGroupList() {
	p.groupList = tview.NewList()
	p.groupList.SetBorder(true)
	p.groupList.SetTitle(" 代理组 ")
	p.groupList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyCtrlD:
			go p.deleteSelected()
			return nil
		case tcell.KeyEnter:
			p.showSelectedDetail()
			return nil
		}
		switch ev.Rune() {
		case 'a', 'A':
			p.showAddForm()
			return nil
		case 'e', 'E':
			p.showEditForm()
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
	p.groupList.SetChangedFunc(func(idx int, _, _ string, _ rune) {
		p.mu.Lock()
		p.selectedIndex = idx
		var g *config.ProxyGroupEntry
		if idx >= 0 && idx < len(p.groups) {
			g = &p.groups[idx]
		}
		p.mu.Unlock()
		if g != nil {
			p.showDetail(*g)
		}
	})
}

func (p *Page) setupActionBar() {
	refreshBtn := tview.NewButton("[R] 刷新")
	refreshBtn.SetSelectedFunc(func() { go p.refresh() })
	addBtn := tview.NewButton("[A] 添加")
	addBtn.SetSelectedFunc(p.showAddForm)
	editBtn := tview.NewButton("[E] 编辑")
	editBtn.SetSelectedFunc(p.showEditForm)
	deleteBtn := tview.NewButton("[D] 删除")
	deleteBtn.SetSelectedFunc(func() { go p.deleteSelected() })

	p.actionBar.AddItem(refreshBtn, 0, 1, false)
	p.actionBar.AddItem(addBtn, 0, 1, false)
	p.actionBar.AddItem(editBtn, 0, 1, false)
	p.actionBar.AddItem(deleteBtn, 0, 1, false)
}

func (p *Page) Activate() {
	go p.refresh()
}

func (p *Page) Deactivate() {}

// showAddForm runs on the UI goroutine; tview ops are direct, no QueueUpdateDraw.
func (p *Page) showAddForm() {
	p.mu.Lock()
	if p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = true
	p.editingGroup = ""
	p.mu.Unlock()

	p.groupForm.SetTitle(" 添加代理组 ")
	p.nameInput.SetText("")
	p.typeSelect.SetCurrentOption(0)
	p.proxiesInput.SetText("")
	p.useInput.SetText("")
	p.urlInput.SetText("")
	p.filterInput.SetText("")
	p.AddItem(p.groupForm, 0, 1, true)
}

// showEditForm runs on the UI goroutine; tview ops are direct, no QueueUpdateDraw.
func (p *Page) showEditForm() {
	p.mu.Lock()
	idx, groups := p.selectedIndex, p.groups
	p.mu.Unlock()

	if idx < 0 || idx >= len(groups) {
		p.showError("请先选择一个代理组")
		return
	}
	g := groups[idx]
	if g.Name == "Proxy" || g.Name == "Auto" || g.Name == "GLOBAL" {
		p.showError(fmt.Sprintf("内置组 %q 不可编辑", g.Name))
		return
	}

	p.mu.Lock()
	if p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = true
	p.editingGroup = g.Name
	p.mu.Unlock()

	p.groupForm.SetTitle(fmt.Sprintf(" 编辑代理组: %s ", g.Name))
	p.nameInput.SetText(g.Name)
	typeIndex := 0
	for i, vt := range config.ValidGroupTypes {
		if vt == g.Type {
			typeIndex = i
			break
		}
	}
	p.typeSelect.SetCurrentOption(typeIndex)
	p.proxiesInput.SetText(strings.Join(g.Proxies, ", "))
	p.useInput.SetText(strings.Join(g.Use, ", "))
	p.urlInput.SetText(g.URL)
	p.filterInput.SetText(g.Filter)
	p.AddItem(p.groupForm, 0, 1, true)
}

func (p *Page) hideForm() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.editingGroup = ""
	p.mu.Unlock()
	go ui.Updater.UpdateUi(func() {
		p.RemoveItem(p.groupForm)
	})
}

func (p *Page) onSaveClicked() {
	name := strings.TrimSpace(p.nameInput.GetText())
	if name == "" {
		p.showError("名称不能为空")
		return
	}
	_, typeStr := p.typeSelect.GetCurrentOption()
	if typeStr == "" {
		typeStr = config.ValidGroupTypes[0]
	}
	proxiesStr := strings.TrimSpace(p.proxiesInput.GetText())
	useStr := strings.TrimSpace(p.useInput.GetText())
	testURL := strings.TrimSpace(p.urlInput.GetText())
	filter := strings.TrimSpace(p.filterInput.GetText())

	var proxies, use []string
	if proxiesStr != "" {
		for _, s := range strings.Split(proxiesStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				proxies = append(proxies, s)
			}
		}
	}
	if useStr != "" {
		for _, s := range strings.Split(useStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				use = append(use, s)
			}
		}
	}

	p.mu.Lock()
	editing := p.editingGroup
	p.mu.Unlock()

	p.showStatus(fmt.Sprintf("[yellow]正在保存代理组 %s...[white]", name))
	go p.saveGroup(name, typeStr, filter, testURL, use, proxies, editing)
}

func (p *Page) onCancelClicked() {
	p.mu.Lock()
	if !p.showInput {
		p.mu.Unlock()
		return
	}
	p.showInput = false
	p.editingGroup = ""
	p.mu.Unlock()
	p.RemoveItem(p.groupForm)
}

func (p *Page) saveGroup(name, groupType, filter, testURL string, use, proxies []string, editing string) {
	var configPath string
	var err error

	if editing != "" {
		configPath, err = config.UpdateGroupInConfig(editing, name, groupType, filter, testURL, use, proxies)
	} else {
		configPath, err = config.AddGroupToConfig(name, groupType, filter, testURL, use, proxies)
	}

	if err != nil {
		p.showError(fmt.Sprintf("保存失败: %v", err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[proxygroups] hot-reload failed: %v", err)
	}

	groups, err := config.GetAllProxyGroupsFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.groups = groups
		formWasShown := p.showInput
		p.showInput = false
		p.editingGroup = ""
		p.mu.Unlock()
		if formWasShown {
			p.RemoveItem(p.groupForm)
		}
		p.updateGroupList()
		p.status.SetText(fmt.Sprintf("[green]代理组 %s 已保存[white]", name))
	})
}

func (p *Page) deleteSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.groups) {
		p.mu.Unlock()
		p.showError("请先选择一个代理组")
		return
	}
	g := p.groups[p.selectedIndex]
	p.mu.Unlock()

	if g.Name == "Proxy" || g.Name == "Auto" || g.Name == "GLOBAL" {
		p.showError(fmt.Sprintf("内置组 %q 不能删除", g.Name))
		return
	}

	configPath, err := config.DeleteGroupFromConfig(g.Name)
	if err != nil {
		p.showError(fmt.Sprintf("删除失败: %v", err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[proxygroups] hot-reload failed: %v", err)
	}

	groups, err := config.GetAllProxyGroupsFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.groups = groups
		p.mu.Unlock()
		p.updateGroupList()
		p.status.SetText(fmt.Sprintf("[green]已删除代理组 %s[white]", g.Name))
	})
}

func (p *Page) refresh() {
	groups, err := config.GetAllProxyGroupsFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf("读取配置失败: %v", err))
		return
	}
	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.groups = groups
		p.mu.Unlock()
		p.updateGroupList()
	})
}

func (p *Page) updateGroupList() {
	p.groupList.Clear()
	p.mu.Lock()
	groups := p.groups
	p.mu.Unlock()

	if len(groups) == 0 {
		p.groupList.AddItem("(暂无代理组)", "按 A 键添加新代理组", 0, nil)
		return
	}

	for _, g := range groups {
		sec := fmt.Sprintf("%s", g.Type)
		if len(g.Proxies) > 0 {
			sec += fmt.Sprintf(" | 节点: %d", len(g.Proxies))
		}
		if len(g.Use) > 0 {
			sec += fmt.Sprintf(" | 提供者: %d", len(g.Use))
		}
		p.groupList.AddItem(g.Name, sec, 0, nil)
	}
}

// showDetail is always called from the UI goroutine (SetChangedFunc, keyboard handler).
func (p *Page) showDetail(g config.ProxyGroupEntry) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[yellow]名称:[white] %s\n", g.Name))
	b.WriteString(fmt.Sprintf("[yellow]类型:[white] %s\n", g.Type))
	if len(g.Proxies) > 0 {
		b.WriteString("[yellow]代理节点:[white]\n")
		for _, pr := range g.Proxies {
			b.WriteString(fmt.Sprintf("  - %s\n", pr))
		}
	}
	if len(g.Use) > 0 {
		b.WriteString("[yellow]代理提供者:[white]\n")
		for _, u := range g.Use {
			b.WriteString(fmt.Sprintf("  - %s\n", u))
		}
	}
	if g.URL != "" {
		b.WriteString(fmt.Sprintf("[yellow]测试URL:[white] %s\n", g.URL))
	}
	if g.Interval > 0 {
		b.WriteString(fmt.Sprintf("[yellow]测试间隔:[white] %ds\n", g.Interval))
	}
	if g.Filter != "" {
		b.WriteString(fmt.Sprintf("[yellow]过滤规则:[white] %s\n", g.Filter))
	}
	b.WriteString("\n[gray]配置文件中的代理组定义[white]")
	p.detailView.SetText(b.String())
}

func (p *Page) showSelectedDetail() {
	p.mu.Lock()
	idx, groups := p.selectedIndex, p.groups
	p.mu.Unlock()
	if idx >= 0 && idx < len(groups) {
		p.showDetail(groups[idx])
	}
}

func (p *Page) showError(msg string) {
	log.Printf("[proxygroups] error: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[red]%s[white]", msg)) })
}

func (p *Page) showSuccess(msg string) {
	log.Printf("[proxygroups] success: %s", msg)
	go ui.Updater.UpdateUi(func() { p.status.SetText(fmt.Sprintf("[green]%s[white]", msg)) })
}

func (p *Page) showStatus(msg string) {
	go ui.Updater.UpdateUi(func() { p.status.SetText(msg) })
}
