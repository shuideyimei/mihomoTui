package pages

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"

	"mihomoTui/internal/api"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/models"
	"mihomoTui/internal/repository"
	"mihomoTui/internal/service"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RulesPage represents the rules management page
type RulesPage struct {
	*tview.Flex

	ruleService *service.RuleService

	table       *tview.Table
	searchInput *tview.InputField
	statusText  *tview.TextView
	actionBar   *tview.Flex

	rules      []models.RuleDisplay
	filterText string

	addForm        *tview.Form
	typeDrop       *tview.DropDown
	payloadInput   *tview.InputField
	proxyInput     *tview.InputField
	addFormVisible bool

	pasteTextArea    *tview.TextArea
	pasteImportForm  *tview.Flex
	pasteFormVisible bool

	moveToForm        *tview.Form
	moveToInput       *tview.InputField
	moveToFormVisible bool

	refreshId int64 // incremented on each refresh() call to discard stale responses
}

var ruleTypes = []string{
	"DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD",
	"IP-CIDR", "IP-CIDR6", "IP-SUFFIX",
	"GEOIP", "GEOSITE",
	"RULE-SET", "PROCESS-NAME", "PROCESS-PATH",
	"MATCH", "DST-PORT", "SRC-PORT",
}

// NewRulesPage creates a new rules page
func NewRulesPage(ruleService ...*service.RuleService) *RulesPage {
	var svc *service.RuleService
	if len(ruleService) > 0 {
		svc = ruleService[0]
	}
	page := &RulesPage{
		Flex:        tview.NewFlex(),
		ruleService: svc,
	}
	page.setupUI()
	return page
}

func (r *RulesPage) SetRuleService(ruleService *service.RuleService) {
	r.ruleService = ruleService
}

func (r *RulesPage) getRuleService() *service.RuleService {
	if r.ruleService != nil {
		return r.ruleService
	}
	return service.NewRuleService(repository.NewLocalYAMLDriver())
}

func (r *RulesPage) Activate() {
	log.Printf("Activating rules page")
	r.refresh()
}

func (r *RulesPage) Deactivate() {
	log.Printf("Deactivating rules page")
	// Runs on the page-lifecycle goroutine; widget mutations must be
	// marshalled onto the tview UI goroutine via PostUi.
	ui.Updater.PostUi(func() {
		r.table.Clear()
		r.rules = nil
	})
}

func (r *RulesPage) setupUI() {
	r.setupAddForm()
	r.setupPasteImport()
	r.setupMoveToForm()
	r.setupSearchInput()
	r.setupTable()
	r.setupActionBar()
	r.setupStatusText()
	r.setupEventHandlers()

	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.AddItem(r.searchInput, 1, 0, false)
	content.AddItem(r.table, 0, 1, true)
	content.AddItem(r.actionBar, 1, 0, false)
	content.AddItem(r.statusText, 1, 0, false)

	r.SetDirection(tview.FlexRow)
	r.AddItem(content, 0, 1, true)

	r.SetBorder(false)
	r.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.title")))
}

func (r *RulesPage) setupAddForm() {
	r.addForm = tview.NewForm()
	r.addForm.SetBorder(true)
	r.addForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.add_title")))
	components.StyleFocusBorder(r.addForm)
	r.addForm.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.typeDrop = tview.NewDropDown()
	r.typeDrop.SetLabel(i18n.T("rule.type_label"))
	r.typeDrop.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.typeDrop.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(ui.ThemeHighlightBg).Foreground(tcell.ColorBlack),
	)
	for _, t := range ruleTypes {
		r.typeDrop.AddOption(t, nil)
	}
	r.typeDrop.SetCurrentOption(0)
	r.payloadInput = tview.NewInputField()
	r.payloadInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.payloadInput.SetLabel(i18n.T("rule.content_label"))
	r.payloadInput.SetFieldWidth(0)
	r.proxyInput = tview.NewInputField()
	r.proxyInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.proxyInput.SetLabel(i18n.T("rule.proxy_label"))
	r.proxyInput.SetFieldWidth(0)

	r.addForm.AddFormItem(r.typeDrop)
	r.addForm.AddFormItem(r.payloadInput)
	r.addForm.AddFormItem(r.proxyInput)
	r.addForm.AddButton(i18n.T("rule.save_btn"), r.onAdd)
	r.addForm.AddButton(i18n.T("rule.cancel_btn"), r.onCancelAdd)
	r.addForm.SetButtonsAlign(tview.AlignCenter)
	r.addForm.SetButtonStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	r.addForm.SetButtonActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorBlack))
}

func (r *RulesPage) setupSearchInput() {
	r.searchInput = tview.NewInputField()
	r.searchInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.searchInput.SetLabel(i18n.T("rule.search_label"))
	r.searchInput.SetPlaceholder(i18n.T("rule.search_placeholder"))
	r.searchInput.SetChangedFunc(func(text string) {
		r.filterText = strings.TrimSpace(text)
		r.updateTable()
	})
	r.searchInput.SetBorder(true)
	r.searchInput.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.filter")))
}

func (r *RulesPage) setupTable() {
	r.table = tview.NewTable().SetFixed(1, 0)
	r.table.SetSelectable(true, false)
	r.table.SetBorder(true)
	r.table.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.list_title")))
	components.StyleFocusBorder(r.table)

	headers := []string{i18n.T("rule.col_type"), i18n.T("rule.col_content"), i18n.T("rule.col_proxy")}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		r.table.SetCell(0, i, cell)
	}
}

func (r *RulesPage) setupActionBar() {
	r.actionBar = tview.NewFlex()
	components.StyleActionBar(r.actionBar)

	addBtn := components.NewButton(i18n.T("rule.add_action"), components.ButtonPrimary, r.showAddDialog)
	deleteBtn := components.NewButton(i18n.T("rule.del_action"), components.ButtonDanger, func() { go r.deleteSelected() })
	moveUpBtn := components.NewButton(i18n.T("rule.up_action"), components.ButtonNormal, func() { go r.moveUp() })
	moveDownBtn := components.NewButton(i18n.T("rule.down_action"), components.ButtonNormal, func() { go r.moveDown() })
	moveToBtn := components.NewButton(i18n.T("rule.move_action"), components.ButtonNormal, r.showMoveToDialog)
	refreshBtn := components.NewButton(i18n.T("rule.refresh_action"), components.ButtonNormal, func() { go r.refresh() })
	pasteBtn := components.NewButton(i18n.T("rule.paste_action"), components.ButtonNormal, r.showPasteImport)

	r.actionBar.AddItem(refreshBtn, 0, 1, false)
	r.actionBar.AddItem(addBtn, 0, 1, false)
	r.actionBar.AddItem(moveUpBtn, 0, 1, false)
	r.actionBar.AddItem(moveDownBtn, 0, 1, false)
	r.actionBar.AddItem(moveToBtn, 0, 1, false)
	r.actionBar.AddItem(pasteBtn, 0, 1, false)
	r.actionBar.AddItem(deleteBtn, 0, 1, false)
}

func (r *RulesPage) setupMoveToForm() {
	r.moveToInput = tview.NewInputField()
	r.moveToInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.moveToInput.SetLabel(i18n.T("rule.move_target"))
	r.moveToInput.SetPlaceholder(i18n.T("rule.move_hint"))
	r.moveToInput.SetFieldWidth(0)

	r.moveToForm = tview.NewForm()
	r.moveToForm.SetBorder(true)
	r.moveToForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.move_title")))
	r.moveToForm.SetFieldBackgroundColor(ui.ThemeInputBg)
	r.moveToForm.AddFormItem(r.moveToInput)
	r.moveToForm.AddButton(i18n.T("rule.confirm_btn"), r.onMoveTo)
	r.moveToForm.AddButton(i18n.T("rule.cancel_move_btn"), func() {
		r.hideMoveToDialog()
	})
	r.moveToForm.SetButtonsAlign(tview.AlignCenter)
	r.moveToForm.SetButtonStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	r.moveToForm.SetButtonActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorBlack))
}

func (r *RulesPage) setupStatusText() {
	r.statusText = tview.NewTextView()
	components.StyleStatusLine(r.statusText)
	r.statusText.SetText(i18n.T("rule.status_ready"))

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
				ui.Updater.SetFocus(r.table)
				return nil
			}
			return event
		case tcell.KeyUp:
			if event.Modifiers()&tcell.ModAlt != 0 && !r.searchInput.HasFocus() {
				go r.moveUp()
				return nil
			}
			return event
		case tcell.KeyDown:
			if event.Modifiers()&tcell.ModAlt != 0 && !r.searchInput.HasFocus() {
				go r.moveDown()
				return nil
			}
			return event
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
		case 'm', 'M':
			if !r.searchInput.HasFocus() {
				r.showMoveToDialog()
			}
			return nil
		case '/':
			ui.Updater.SetFocus(r.searchInput)
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
	r.pasteTextArea.SetPlaceholder(i18n.T("rule.paste_placeholder"))
	r.pasteTextArea.SetBorder(true)

	btnBar := tview.NewFlex()
	importBtn := components.NewButton(i18n.T("rule.import_btn"), components.ButtonPrimary, func() {
		if r.pasteFormVisible {
			go r.processPasteImport()
		}
	})
	cancelBtn := components.NewButton(i18n.T("rule.import_cancel"), components.ButtonNormal, func() {
		r.hidePasteImport()
	})
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(importBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)
	btnBar.AddItem(cancelBtn, 0, 1, false)
	btnBar.AddItem(tview.NewBox(), 0, 1, false)

	r.pasteImportForm = tview.NewFlex().SetDirection(tview.FlexRow)
	r.pasteImportForm.SetBorder(true)
	r.pasteImportForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.import_title")))
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

// cleanPasteLine strips YAML formatting from a pasted rule line:
// removes "- " prefix, strips surrounding quotes, removes trailing # comments.
func cleanPasteLine(line string) string {
	s := strings.TrimSpace(line)
	// Skip full-line comments (already handled by caller, but safety check)
	if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
		return ""
	}
	// Remove "- " prefix (YAML list item marker)
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimSpace(s)
	// Strip trailing inline YAML comment (#) BEFORE quote removal,
	// because quotes and comments often appear together:
	//   'GEOSITE,private,DIRECT  # comment'
	// The quote check below looks at first+last chars; a trailing
	// comment after the closing quote would prevent quote detection.
	if idx := strings.Index(s, " #"); idx >= 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "\t#"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	// Remove surrounding single or double quotes
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

func (r *RulesPage) processPasteImport() {
	// All UI reads must run on the UI goroutine
	ui.Updater.PostUi(func() {
		text := r.pasteTextArea.GetText()
		r.hidePasteImport()
		r.showStatus(i18n.T("rule.importing"))
		go r.doProcessPasteImport(text)
	})
}

func (r *RulesPage) doProcessPasteImport(text string) {
	lines := strings.Split(text, "\n")

	var rules []models.RuleDisplay
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		cleaned := cleanPasteLine(line)
		if cleaned == "" {
			continue
		}
		parts := strings.SplitN(cleaned, ",", 3)
		if len(parts) == 2 {
			// Format: MATCH,PROXY (no payload)
			ruleType := strings.TrimSpace(parts[0])
			proxy := strings.TrimSpace(parts[1])
			if ruleType == "" || proxy == "" {
				errs = append(errs, fmt.Sprintf(i18n.T("rule.invalid_line"), line))
				continue
			}
			rules = append(rules, models.RuleDisplay{Type: ruleType, Payload: "", Proxy: proxy})
		} else if len(parts) == 3 {
			ruleType := strings.TrimSpace(parts[0])
			payload := strings.TrimSpace(parts[1])
			proxy := strings.TrimSpace(parts[2])
			if ruleType == "" || proxy == "" {
				errs = append(errs, fmt.Sprintf(i18n.T("rule.invalid_line"), fmt.Sprintf("(missing fields) %s", line)))
				continue
			}
			rules = append(rules, models.RuleDisplay{Type: ruleType, Payload: payload, Proxy: proxy})
		} else {
			errs = append(errs, fmt.Sprintf(i18n.T("rule.invalid_line"), fmt.Sprintf("(format error) %s", line)))
			continue
		}
	}

	if len(rules) == 0 {
		msg := i18n.T("rule.import_no_valid")
		if len(errs) > 0 {
			msg += fmt.Sprintf(i18n.T("rule.import_fail_count"), len(errs))
		}
		r.refresh()
		ui.Updater.PostUi(func() {
			r.showStatus(msg)
		})
		return
	}

	_, err := r.getRuleService().BatchAddRules(rules)
	if err != nil {
		log.Printf("[rules] batch import failed: %v", err)
		ui.Updater.PostUi(func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.import_fail"), err))
		})
		return
	}

	msg := fmt.Sprintf(i18n.T("rule.import_ok"), len(rules))
	if len(errs) > 0 {
		msg += fmt.Sprintf(i18n.T("rule.import_skip"), len(errs))
		log.Printf("[rules] paste import skipped: %v", errs)
	}
	r.refresh()
	ui.Updater.PostUi(func() {
		r.showStatus(msg)
	})
}

func (r *RulesPage) onAdd() {
	_, ruleType := r.typeDrop.GetCurrentOption()
	payload := strings.TrimSpace(r.payloadInput.GetText())
	proxy := strings.TrimSpace(r.proxyInput.GetText())

	// MATCH and certain rule types have no payload concept
	needsPayload := ruleType != "MATCH"
	if needsPayload && (payload == "") {
		r.showStatus(i18n.T("rule.type_required"))
		return
	}
	if proxy == "" {
		r.showStatus(i18n.T("rule.proxy_required"))
		return
	}

	r.showStatus(fmt.Sprintf(i18n.T("rule.adding"), ruleType, payload, proxy))
	r.hideAddDialog()
	go r.saveRule(ruleType, payload, proxy)
}

func (r *RulesPage) onCancelAdd() {
	r.hideAddDialog()
}

func (r *RulesPage) saveRule(ruleType, payload, proxy string) {
	err := r.getRuleService().AddRule(ruleType, payload, proxy)
	if err != nil {
		ui.Updater.PostUi(func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.add_fail"), err))
		})
		return
	}

	r.refresh()
}

func (r *RulesPage) deleteSelected() {
	// All UI reads must run on the UI goroutine
	ui.Updater.PostUi(func() {
		row, _ := r.table.GetSelection()
		if row <= 0 {
			r.showStatus(i18n.T("rule.no_selection"))
			return
		}

		// Find the actual rule index accounting for filter
		ruleIdx := r.findRuleIndex(row)
		if ruleIdx < 0 || ruleIdx >= len(r.rules) {
			return
		}
		rule := r.rules[ruleIdx]

		label := fmt.Sprintf("%s, %s, %s", rule.Type, rule.Payload, rule.Proxy)
		ui.Updater.ConfirmDelete(label, func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.deleting"), rule.Type, rule.Payload, rule.Proxy))
			go r.doDelete(ruleIdx)
		})
	})
}

func (r *RulesPage) doDelete(ruleIdx int) {
	// Use index-based deletion: API returns rules in config order,
	// so ruleIdx directly maps to the config file's rules list index.
	configRules, err := r.getRuleService().GetRules()
	if err != nil {
		log.Printf("[rules] GetRules failed: %v", err)
		ui.Updater.PostUi(func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.read_fail"), err))
		})
		return
	}

	if ruleIdx >= len(configRules) {
		// Rule might be from a rule-provider, not in the config file's rules list.
		ui.Updater.PostUi(func() {
			r.showStatus(i18n.T("rule.delete_from_provider"))
		})
		return
	}

	err = r.getRuleService().DeleteRule(ruleIdx)
	if err != nil {
		log.Printf("[rules] DeleteRule failed: %v", err)
		ui.Updater.PostUi(func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.delete_fail"), err))
		})
		return
	}

	r.refresh()
}

func (r *RulesPage) moveUp() {
	ui.Updater.PostUi(func() {
		row, _ := r.table.GetSelection()
		if row <= 1 {
			r.showStatus(i18n.T("rule.first_rule"))
			return
		}
		ruleIdx := r.findRuleIndex(row)
		if ruleIdx < 0 || ruleIdx >= len(r.rules) {
			return
		}
		r.showStatus(fmt.Sprintf("[yellow]%s...[white]", i18n.T("rule.moving_up")))
		go r.doMoveRule(ruleIdx, ruleIdx-1)
	})
}

func (r *RulesPage) moveDown() {
	ui.Updater.PostUi(func() {
		row, _ := r.table.GetSelection()
		if row <= 0 {
			r.showStatus(i18n.T("rule.no_selection"))
			return
		}
		ruleIdx := r.findRuleIndex(row)
		if ruleIdx < 0 || ruleIdx >= len(r.rules) {
			return
		}
		if ruleIdx >= len(r.rules)-1 {
			r.showStatus(i18n.T("rule.last_rule"))
			return
		}
		r.showStatus(fmt.Sprintf("[yellow]%s...[white]", i18n.T("rule.moving_down")))
		go r.doMoveRule(ruleIdx, ruleIdx+1)
	})
}

func (r *RulesPage) showMoveToDialog() {
	if r.moveToFormVisible || r.addFormVisible || r.pasteFormVisible {
		return
	}
	// Read selected rule info on UI goroutine
	row, _ := r.table.GetSelection()
	if row <= 0 {
		r.showStatus(i18n.T("rule.no_selection"))
		return
	}
	ruleIdx := r.findRuleIndex(row)
	if ruleIdx < 0 || ruleIdx >= len(r.rules) {
		return
	}
	rule := r.rules[ruleIdx]

	r.moveToFormVisible = true
	r.moveToInput.SetText("")
	r.moveToInput.SetPlaceholder(fmt.Sprintf("1-%d", len(r.rules)))
	// Show current position as hint
	r.moveToForm.SetTitle(fmt.Sprintf(i18n.T("rule.move_to_title"), row, rule.Type, rule.Payload, rule.Proxy))
	r.AddItem(r.moveToForm, 0, 1, true)
}

func (r *RulesPage) hideMoveToDialog() {
	if !r.moveToFormVisible {
		return
	}
	r.moveToFormVisible = false
	r.RemoveItem(r.moveToForm)
}

func (r *RulesPage) onMoveTo() {
	ui.Updater.PostUi(func() {
		row, _ := r.table.GetSelection()
		if row <= 0 {
			r.showStatus(i18n.T("rule.no_selection"))
			r.hideMoveToDialog()
			return
		}
		fromIdx := r.findRuleIndex(row)
		if fromIdx < 0 || fromIdx >= len(r.rules) {
			r.hideMoveToDialog()
			return
		}

		text := strings.TrimSpace(r.moveToInput.GetText())
		targetNum, err := strconv.Atoi(text)
		if err != nil || targetNum < 1 || targetNum > len(r.rules) {
			r.showStatus(fmt.Sprintf(i18n.T("rule.invalid_position"), len(r.rules)))
			return
		}
		toIdx := targetNum - 1 // convert to 0-based

		if fromIdx == toIdx {
			r.showStatus(i18n.T("rule.move_same"))
			r.hideMoveToDialog()
			return
		}

		r.showStatus(fmt.Sprintf(i18n.T("rule.moving"), fromIdx+1, toIdx+1))
		r.hideMoveToDialog()

		go func() {
			err := r.getRuleService().MoveRule(fromIdx, toIdx)
			if err != nil {
				log.Printf("[rules] MoveRule failed: %v", err)
				ui.Updater.PostUi(func() {
					r.showStatus(fmt.Sprintf(i18n.T("rule.move_fail"), err))
				})
				return
			}

			r.refresh()
		}()
	})
}

func (r *RulesPage) doMoveRule(fromIdx, toIdx int) {
	err := r.getRuleService().MoveRule(fromIdx, toIdx)
	if err != nil {
		log.Printf("[rules] MoveRule failed: %v", err)
		ui.Updater.PostUi(func() {
			r.showStatus(fmt.Sprintf(i18n.T("rule.move_fail"), err))
		})
		return
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
	id := atomic.AddInt64(&r.refreshId, 1)

	ui.Updater.PostUi(func() {
		r.showStatus(i18n.T("rule.loading"))
	})
	go func() {
		// Note: no ReloadConfig here — config writes (add/delete/move/import)
		// already trigger their own hot reload, and a plain page refresh must
		// not force mihomo to re-parse its config.
		rulesData, err := api.Client.GetRules()
		if err != nil {
			// Only show error if no newer refresh is pending
			if atomic.LoadInt64(&r.refreshId) == id {
				ui.Updater.PostUi(func() {
					r.showStatus(fmt.Sprintf(i18n.T("rule.fetch_fail"), err))
				})
			}
			return
		}

		var displayRules []models.RuleDisplay
		for _, rule := range rulesData {
			displayRules = append(displayRules, models.RuleDisplay{
				Type:    rule.Type,
				Payload: rule.Payload,
				Proxy:   rule.Proxy,
			})
		}

		// Only apply the result if no newer refresh was started
		if atomic.LoadInt64(&r.refreshId) == id {
			ui.Updater.PostUi(func() {
				r.rules = displayRules
				r.updateTable()
				r.showStatus(fmt.Sprintf("[green]%d[-] %s", len(rulesData), i18n.T("rule.rows")))
			})
		}
	}()
}

func (r *RulesPage) updateTable() {
	r.table.Clear()

	headers := []string{i18n.T("rule.col_number"), i18n.T("rule.col_type"), i18n.T("rule.col_content"), i18n.T("rule.col_proxy")}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		r.table.SetCell(0, i, cell)
	}

	const maxDisplayRows = 500
	row := 1
	totalCount := len(r.rules)
	visibleCount := 0
	for idx, rule := range r.rules {
		if r.filterText != "" {
			searchLower := strings.ToLower(r.filterText)
			if !strings.Contains(strings.ToLower(rule.Type), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Payload), searchLower) &&
				!strings.Contains(strings.ToLower(rule.Proxy), searchLower) {
				continue
			}
		}

		visibleCount++
		if row <= maxDisplayRows {
			idxCell := tview.NewTableCell(strconv.Itoa(idx + 1)).
				SetTextColor(tcell.ColorGray).
				SetAlign(tview.AlignCenter)
			typeCell := tview.NewTableCell(rule.Type).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter)
			payloadCell := tview.NewTableCell(rule.Payload).
				SetTextColor(tcell.ColorWhite)
			proxyCell := tview.NewTableCell(rule.Proxy).
				SetTextColor(tcell.ColorGray).
				SetAlign(tview.AlignCenter)

			r.table.SetCell(row, 0, idxCell)
			r.table.SetCell(row, 1, typeCell)
			r.table.SetCell(row, 2, payloadCell)
			r.table.SetCell(row, 3, proxyCell)
			row++
		}
	}

	if visibleCount == 0 {
		if totalCount == 0 {
			r.showStatus(i18n.T("rule.no_rules"))
		} else {
			r.showStatus(fmt.Sprintf(i18n.T("rule.filter_match"), totalCount))
		}
	} else {
		statusMsg := fmt.Sprintf("[green]%d[-] %s", totalCount, i18n.T("rule.rows"))
		if r.filterText != "" {
			statusMsg = fmt.Sprintf("[green]%d/%d[-] %s", visibleCount, totalCount, i18n.T("rule.rows"))
		}
		if visibleCount > maxDisplayRows {
			statusMsg += fmt.Sprintf(" [gray]("+i18n.T("common.showing_top")+")[-]", maxDisplayRows)
		}
		r.showStatus(statusMsg)
	}
}

func (r *RulesPage) showStatus(message string) {
	r.statusText.SetText(message)
}

// UpdateTexts refreshes all localized labels on the rules page
func (r *RulesPage) UpdateTexts() {
	ui.Updater.PostUi(func() {
		r.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.title")))
		if r.statusText != nil {
			r.statusText.SetText(i18n.T("rule.status_ready"))
		}
		if r.searchInput != nil {
			r.searchInput.SetLabel(i18n.T("rule.search_label"))
			r.searchInput.SetPlaceholder(i18n.T("rule.search_placeholder"))
			r.searchInput.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.filter")))
		}
		if r.table != nil {
			r.table.SetTitle(fmt.Sprintf(" %s ", i18n.T("rule.list_title")))
			headers := []string{i18n.T("rule.col_type"), i18n.T("rule.col_content"), i18n.T("rule.col_proxy")}
			for i, header := range headers {
				cell := r.table.GetCell(0, i)
				if cell != nil {
					cell.SetText(header)
				}
			}
		}
	})
}
