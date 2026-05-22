package pages

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RulesPage represents the rules management page
type RulesPage struct {
	*tview.Flex

	table       *tview.Table
	searchInput *tview.InputField
	statusText  *tview.TextView
	actionBar   *tview.Flex

	rules      []RuleDisplay
	filterText string

	addForm       *tview.Form
	typeDrop      *tview.DropDown
	payloadInput  *tview.InputField
	proxyInput    *tview.InputField
	addFormVisible bool

	pasteTextArea    *tview.TextArea
	pasteImportForm  *tview.Flex
	pasteFormVisible bool
}

// RuleDisplay is a display-ready rule with formatted fields
type RuleDisplay struct {
	Type    string
	Payload string
	Proxy   string
}

var ruleTypes = []string{
	"DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD",
	"IP-CIDR", "IP-CIDR6", "IP-SUFFIX",
	"GEOIP", "GEOSITE",
	"RULE-SET", "PROCESS-NAME", "PROCESS-PATH",
	"MATCH", "DST-PORT", "SRC-PORT",
}

// NewRulesPage creates a new rules page
func NewRulesPage() *RulesPage {
	page := &RulesPage{
		Flex: tview.NewFlex(),
	}
	page.setupUI()
	return page
}

func (r *RulesPage) Activate() {
	log.Printf("Activating rules page")
	r.refresh()
}

func (r *RulesPage) Deactivate() {
	log.Printf("Deactivating rules page")
	r.table.Clear()
	r.rules = nil
}

func (r *RulesPage) setupUI() {
	r.setupAddForm()
	r.setupPasteImport()
	r.setupSearchInput()
	r.setupTable()
	r.setupActionBar()
	r.setupStatusText()
	r.setupEventHandlers()

	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.AddItem(r.searchInput, 3, 0, false)
	content.AddItem(r.table, 0, 1, true)
	content.AddItem(r.actionBar, 3, 0, false)
	content.AddItem(r.statusText, 3, 0, false)

	r.SetDirection(tview.FlexRow)
	r.AddItem(content, 0, 1, true)

	r.SetBorder(true)
	r.SetTitle(" 规则管理 ")
}

func (r *RulesPage) setupAddForm() {
	r.addForm = tview.NewForm()
	r.addForm.SetBorder(true)
	r.addForm.SetTitle(" 添加规则 ")
	r.typeDrop = tview.NewDropDown()
	r.typeDrop.SetLabel("类型 ")
	for _, t := range ruleTypes {
		r.typeDrop.AddOption(t, nil)
	}
	r.typeDrop.SetCurrentOption(0)
	r.payloadInput = tview.NewInputField()
	r.payloadInput.SetLabel("内容 ")
	r.payloadInput.SetFieldWidth(40)
	r.proxyInput = tview.NewInputField()
	r.proxyInput.SetLabel("代理策略 ")
	r.proxyInput.SetFieldWidth(20)

	r.addForm.AddFormItem(r.typeDrop)
	r.addForm.AddFormItem(r.payloadInput)
	r.addForm.AddFormItem(r.proxyInput)
	r.addForm.AddButton(" 保存 ", r.onAdd)
	r.addForm.AddButton(" 取消 ", r.onCancelAdd)
	r.addForm.SetButtonsAlign(tview.AlignCenter)
}

func (r *RulesPage) setupSearchInput() {
	r.searchInput = tview.NewInputField()
	r.searchInput.SetLabel("搜索: ")
	r.searchInput.SetPlaceholder("输入关键词过滤规则...")
	r.searchInput.SetChangedFunc(func(text string) {
		r.filterText = strings.TrimSpace(text)
		r.updateTable()
	})
	r.searchInput.SetBorder(true)
	r.searchInput.SetTitle(" 过滤 ")
}

func (r *RulesPage) setupTable() {
	r.table = tview.NewTable().SetFixed(1, 0)
	r.table.SetSelectable(true, false)
	r.table.SetBorder(true)
	r.table.SetTitle(" 规则列表 ")

	headers := []string{"类型", "内容", "代理"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		r.table.SetCell(0, i, cell)
	}
}

func (r *RulesPage) setupActionBar() {
	r.actionBar = tview.NewFlex()
	r.actionBar.SetBorder(true)
	r.actionBar.SetTitle(" 操作 ")

	addBtn := tview.NewButton("[A] 添加")
	addBtn.SetSelectedFunc(r.showAddDialog)
	deleteBtn := tview.NewButton("[D] 删除")
	deleteBtn.SetSelectedFunc(func() { go r.deleteSelected() })
	refreshBtn := tview.NewButton("[R] 刷新")
	refreshBtn.SetSelectedFunc(func() { go r.refresh() })
	pasteBtn := tview.NewButton("[P] 粘贴导入")
	pasteBtn.SetSelectedFunc(r.showPasteImport)

	r.actionBar.AddItem(refreshBtn, 0, 1, false)
	r.actionBar.AddItem(addBtn, 0, 1, false)
	r.actionBar.AddItem(pasteBtn, 0, 1, false)
	r.actionBar.AddItem(deleteBtn, 0, 1, false)
}

func (r *RulesPage) setupStatusText() {
	r.statusText = tview.NewTextView()
	r.statusText.SetBorder(true)
	r.statusText.SetTitle(" 状态 ")
	r.statusText.SetDynamicColors(true)
	r.statusText.SetText("[green]就绪[white]")
}

func (r *RulesPage) setupEventHandlers() {
	r.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			r.refresh()
			return nil
		case tcell.KeyEscape:
			if r.searchInput.HasFocus() {
				r.searchInput.SetText("")
			}
			return nil
		}
		switch event.Rune() {
		case 'a', 'A':
			if !r.searchInput.HasFocus() {
				r.showAddDialog()
			}
			return nil
		case 'd', 'D':
			if !r.searchInput.HasFocus() {
				go r.deleteSelected()
			}
			return nil
		case 'p', 'P':
			if !r.searchInput.HasFocus() {
				r.showPasteImport()
			}
			return nil
		case 'r', 'R':
			if !r.searchInput.HasFocus() {
				r.refresh()
			}
			return nil
		case '/':
			go ui.Updater.SetFocus(r.searchInput)
			return nil
		}
		return event
	})
}

func (r *RulesPage) showAddDialog() {
	if r.addFormVisible {
		return
	}
	r.addFormVisible = true
	r.typeDrop.SetCurrentOption(0)
	r.payloadInput.SetText("")
	r.proxyInput.SetText("")
	r.AddItem(r.addForm, 0, 1, true)
}

func (r *RulesPage) hideAddDialog() {
	if !r.addFormVisible {
		return
	}
	r.addFormVisible = false
	r.RemoveItem(r.addForm)
}

func (r *RulesPage) setupPasteImport() {
	r.pasteTextArea = tview.NewTextArea()
	r.pasteTextArea.SetPlaceholder(`粘贴多条规则，每行一条，格式:
  - TYPE,payload,proxy

示例:
  - DOMAIN-SUFFIX,google.com,PROXY
  - DOMAIN-KEYWORD,google,PROXY
  - GEOIP,CN,DIRECT
  - MATCH,PROXY

支持 # 注释行，"- " 前缀可选。`)
	r.pasteTextArea.SetBorder(true)

	btnBar := tview.NewFlex()
	importBtn := tview.NewButton(" 导入 ")
	importBtn.SetSelectedFunc(func() {
		if r.pasteFormVisible {
			go r.processPasteImport()
		}
	})
	cancelBtn := tview.NewButton(" 取消 ")
	cancelBtn.SetSelectedFunc(func() {
		r.hidePasteImport()
	})
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(importBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(cancelBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)

	r.pasteImportForm = tview.NewFlex().SetDirection(tview.FlexRow)
	r.pasteImportForm.SetBorder(true)
	r.pasteImportForm.SetTitle(" 粘贴导入规则 ")
	r.pasteImportForm.AddItem(r.pasteTextArea, 0, 1, true)
	r.pasteImportForm.AddItem(btnBar, 3, 0, false)
}

func (r *RulesPage) showPasteImport() {
	if r.pasteFormVisible {
		return
	}
	if r.addFormVisible {
		r.hideAddDialog()
	}
	r.pasteFormVisible = true
	r.pasteTextArea.SetText("", true)
	r.AddItem(r.pasteImportForm, 0, 3, true)
}

func (r *RulesPage) hidePasteImport() {
	if !r.pasteFormVisible {
		return
	}
	r.pasteFormVisible = false
	r.RemoveItem(r.pasteImportForm)
}

func (r *RulesPage) processPasteImport() {
	// Capture text BEFORE hiding the form — removing from UI tree may clear buffer
	text := r.pasteTextArea.GetText()

	r.hidePasteImport()
	r.showStatus("[yellow]正在导入规则...[white]")

	lines := strings.Split(text, "\n")

	var rules []config.RuleEntry
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		cleaned := strings.TrimPrefix(line, "- ")
		parts := strings.SplitN(cleaned, ",", 3)
		if len(parts) == 2 {
			// Format: MATCH,PROXY (no payload)
			ruleType := strings.TrimSpace(parts[0])
			proxy := strings.TrimSpace(parts[1])
			if ruleType == "" || proxy == "" {
				errs = append(errs, fmt.Sprintf("无效行: %s", line))
				continue
			}
			rules = append(rules, config.RuleEntry{Type: ruleType, Payload: "", Proxy: proxy})
		} else if len(parts) == 3 {
			ruleType := strings.TrimSpace(parts[0])
			payload := strings.TrimSpace(parts[1])
			proxy := strings.TrimSpace(parts[2])
			if ruleType == "" || proxy == "" {
				errs = append(errs, fmt.Sprintf("无效行 (缺少必要字段): %s", line))
				continue
			}
			rules = append(rules, config.RuleEntry{Type: ruleType, Payload: payload, Proxy: proxy})
		} else {
			errs = append(errs, fmt.Sprintf("无效行 (格式错误): %s", line))
			continue
		}
	}

	if len(rules) == 0 {
		msg := "[red]没有有效的规则可导入[white]"
		if len(errs) > 0 {
			msg += fmt.Sprintf(" | %d 条失败", len(errs))
		}
		r.refresh()
		go ui.Updater.UpdateUi(func() {
			r.showStatus(msg)
		})
		return
	}

	cp, err := config.BatchAddRulesToConfig(rules)
	if err != nil {
		log.Printf("[rules] batch import failed: %v", err)
		go ui.Updater.UpdateUi(func() {
			r.showStatus(fmt.Sprintf("[red]批量导入失败: %v[white]", err))
		})
		return
	}

	if err := api.Client.ReloadConfig(cp); err != nil {
		log.Printf("[rules] hot-reload failed: %v", err)
		r.refresh()
		go ui.Updater.UpdateUi(func() {
			r.showStatus(fmt.Sprintf("[red]配置重载失败: %v[white]", err))
		})
		return
	}

	msg := fmt.Sprintf("[green]成功导入 %d 条规则[white]", len(rules))
	if len(errs) > 0 {
		msg += fmt.Sprintf(" | [red]%d 条跳过[white]", len(errs))
		log.Printf("[rules] paste import skipped: %v", errs)
	}
	r.refresh()
	go ui.Updater.UpdateUi(func() {
		r.showStatus(msg)
	})
}

func (r *RulesPage) onAdd() {
	_, ruleType := r.typeDrop.GetCurrentOption()
	payload := strings.TrimSpace(r.payloadInput.GetText())
	proxy := strings.TrimSpace(r.proxyInput.GetText())

	if payload == "" || proxy == "" {
		r.showStatus("[red]内容和代理策略不能为空[white]")
		return
	}

	r.showStatus(fmt.Sprintf("[yellow]正在添加规则 %s,%s,%s...[white]", ruleType, payload, proxy))
	r.hideAddDialog()
	go r.saveRule(ruleType, payload, proxy)
}

func (r *RulesPage) onCancelAdd() {
	r.hideAddDialog()
}

func (r *RulesPage) saveRule(ruleType, payload, proxy string) {
	configPath, err := config.AddRuleToConfig(ruleType, payload, proxy)
	if err != nil {
		go ui.Updater.UpdateUi(func() {
			r.showStatus(fmt.Sprintf("[red]添加失败: %v[white]", err))
		})
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[rules] hot-reload failed: %v", err)
	}

	r.refresh()
}

func (r *RulesPage) deleteSelected() {
	row, _ := r.table.GetSelection()
	if row <= 0 {
		go ui.Updater.UpdateUi(func() {
			r.showStatus("[red]请先选择一条规则[white]")
		})
		return
	}

	// Find the actual rule index accounting for filter
	ruleIdx := r.findRuleIndex(row)
	if ruleIdx < 0 || ruleIdx >= len(r.rules) {
		return
	}
	rule := r.rules[ruleIdx]

	r.showStatus(fmt.Sprintf("[yellow]正在删除规则 %s,%s,%s...[white]", rule.Type, rule.Payload, rule.Proxy))

	configPath, err := config.DeleteRuleFromConfig(ruleIdx)
	if err != nil {
		go ui.Updater.UpdateUi(func() {
			r.showStatus(fmt.Sprintf("[red]删除失败: %v[white]", err))
		})
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[rules] hot-reload failed: %v", err)
	}

	r.refresh()
}

func (r *RulesPage) findRuleIndex(tableRow int) int {
	idx := 0
	row := 1
	for _, rule := range r.rules {
		if r.filterText != "" {
			searchLower := strings.ToLower(r.filterText)
			if !strings.Contains(strings.ToLower(rule.Type), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Payload), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Proxy), searchLower) {
				idx++
				continue
			}
		}
		if row == tableRow {
			return idx
		}
		row++
		idx++
	}
	return -1
}

func (r *RulesPage) refresh() {
	go ui.Updater.UpdateUi(func() {
		r.showStatus("[yellow]加载规则中...[white]")
	})
	go func() {
		rulesData, err := api.Client.GetRules()
		if err != nil {
			go ui.Updater.UpdateUi(func() {
				r.showStatus(fmt.Sprintf("[red]获取规则失败: %v[white]", err))
			})
			return
		}

		var displayRules []RuleDisplay
		for _, rule := range rulesData {
			displayRules = append(displayRules, RuleDisplay{
				Type:    rule.Type,
				Payload: rule.Payload,
				Proxy:   rule.Proxy,
			})
		}

		go ui.Updater.UpdateUi(func() {
			r.rules = displayRules
			r.updateTable()
			r.showStatus(fmt.Sprintf("[green]%d[white] 条规则", len(rulesData)))
		})
	}()
}

func (r *RulesPage) updateTable() {
	r.table.Clear()

	headers := []string{"#", "类型", "内容", "代理"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		r.table.SetCell(0, i, cell)
	}

	row := 1
	totalCount := len(r.rules)
	visibleCount := 0
	displayIdx := 0
	for idx, rule := range r.rules {
		if r.filterText != "" {
			searchLower := strings.ToLower(r.filterText)
			if !strings.Contains(strings.ToLower(rule.Type), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Payload), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Proxy), searchLower) {
				displayIdx++
				continue
			}
		}

		idxCell := tview.NewTableCell(strconv.Itoa(idx + 1)).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter)
		typeCell := tview.NewTableCell(rule.Type).
			SetTextColor(tcell.ColorBlue).
			SetAlign(tview.AlignCenter)
		payloadCell := tview.NewTableCell(rule.Payload).
			SetTextColor(tcell.ColorWhite)
		proxyCell := tview.NewTableCell(rule.Proxy).
			SetTextColor(tcell.ColorGreen).
			SetAlign(tview.AlignCenter)

		r.table.SetCell(row, 0, idxCell)
		r.table.SetCell(row, 1, typeCell)
		r.table.SetCell(row, 2, payloadCell)
		r.table.SetCell(row, 3, proxyCell)
		row++
		visibleCount++
		displayIdx++
	}

	if visibleCount == 0 {
		if totalCount == 0 {
			r.showStatus("[yellow]暂无规则数据[white]")
		} else {
			r.showStatus(fmt.Sprintf("[yellow]未找到匹配的规则（共 %d 条）[white]", totalCount))
		}
	} else {
		statusMsg := fmt.Sprintf("[green]%d[white] 条规则", totalCount)
		if r.filterText != "" {
			statusMsg = fmt.Sprintf("[green]%d/%d[white] 条规则", visibleCount, totalCount)
		}
		r.showStatus(statusMsg)
	}
}

func (r *RulesPage) showStatus(message string) {
	r.statusText.SetText(message)
}
