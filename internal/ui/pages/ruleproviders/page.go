package ruleproviders

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
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
	p.detailView.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.detail_title")))
	p.detailView.SetDynamicColors(true)
	p.detailView.SetWordWrap(true)

	p.actionBar = tview.NewFlex()
	p.actionBar.SetBorder(true)
	p.actionBar.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.actions")))
	p.setupActionBar()

	p.status = tview.NewTextView()
	p.status.SetBorder(true)
	p.status.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.status")))
	p.status.SetDynamicColors(true)
	p.status.SetText(i18n.T("rp.status_ready"))

	rightPanel.AddItem(p.detailView, 0, 3, false)
	rightPanel.AddItem(p.actionBar, 3, 0, false)
	rightPanel.AddItem(p.status, 3, 0, false)

	p.importForm = tview.NewForm()
	p.importForm.SetBorder(true)
	p.importForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.import_title")))
	p.importForm.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.nameInput = tview.NewInputField()
	p.nameInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.nameInput.SetLabel(i18n.T("rp.name_label"))
	p.nameInput.SetFieldWidth(30)
	p.urlInput = tview.NewInputField()
	p.urlInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.urlInput.SetLabel("URL ")
	p.urlInput.SetFieldWidth(50)
	p.behaviorSelect = tview.NewDropDown()
	p.behaviorSelect.SetLabel(i18n.T("rp.behavior_label"))
	p.behaviorSelect.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.behaviorSelect.AddOption("domain", nil)
	p.behaviorSelect.AddOption("ipcidr", nil)
	p.behaviorSelect.AddOption("classical", nil)
	p.behaviorSelect.SetCurrentOption(0)
	p.behaviorSelect.SetListStyles(
		tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite),
		tcell.StyleDefault.Background(ui.ThemeHighlightBg).Foreground(tcell.ColorBlack),
	)
	p.intervalInput = tview.NewInputField()
	p.intervalInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.intervalInput.SetLabel(i18n.T("rp.interval_label"))
	p.intervalInput.SetFieldWidth(10)
	p.intervalInput.SetText("3600")

	p.importForm.AddFormItem(p.nameInput)
	p.importForm.AddFormItem(p.urlInput)
	p.importForm.AddFormItem(p.behaviorSelect)
	p.importForm.AddFormItem(p.intervalInput)
	p.importForm.AddButton(i18n.T("rp.import_btn"), p.onImportClicked)
	p.importForm.AddButton(i18n.T("rp.cancel_btn"), p.onCancelImport)
	p.importForm.SetButtonsAlign(tview.AlignCenter)
	p.importForm.SetButtonStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	p.importForm.SetButtonActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))

	p.setupPasteImport()

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.subList, 0, 1, true)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainFlex.AddItem(leftPanel, 0, 2, true)
	mainFlex.AddItem(rightPanel, 0, 3, false)

	p.SetDirection(tview.FlexRow)
	p.AddItem(mainFlex, 0, 1, true)
	p.SetBorder(true)
	p.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.title")))
}

func (p *Page) setupSubList() {
	p.subList = tview.NewList()
	p.subList.SetBorder(true)
	p.subList.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.sub_list_title")))
	p.subList.SetMainTextColor(tcell.ColorWhite)
	p.subList.SetSelectedBackgroundColor(ui.ThemeHighlightBg)
	p.subList.SetSelectedTextColor(tcell.ColorBlack)
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
	refreshBtn := tview.NewButton(i18n.T("rp.refresh_action"))
	refreshBtn.SetSelectedFunc(func() { go p.refresh() })
	importBtn := tview.NewButton(i18n.T("rp.import_action"))
	importBtn.SetSelectedFunc(p.showImportForm)
	quickImportBtn := tview.NewButton(i18n.T("rp.quick_import_action"))
	quickImportBtn.SetSelectedFunc(p.showQuickImportPage)
	quickImportBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	quickImportBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	pasteBtn := tview.NewButton(i18n.T("rp.paste_action"))
	pasteBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	pasteBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	pasteBtn.SetSelectedFunc(p.showPasteImport)
	deleteBtn := tview.NewButton(i18n.T("rp.del_action"))
	deleteBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	deleteBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	deleteBtn.SetSelectedFunc(func() { go p.deleteSelected() })

	refreshBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	refreshBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	importBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	importBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))

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

	p.importForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.import_title")))
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
		p.showError(i18n.T("rp.name_empty"))
		return
	}
	if url == "" {
		p.showError(i18n.T("rp.url_empty"))
		return
	}

	p.showStatus(fmt.Sprintf(i18n.T("rp.importing"), name))
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
	quickList.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.quick_import_title")))
	quickList.SetTitleAlign(tview.AlignLeft)
	quickList.SetMainTextColor(tcell.ColorWhite)
	quickList.SetSelectedBackgroundColor(ui.ThemeHighlightBg)
	quickList.SetSelectedTextColor(tcell.ColorBlack)

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

	p.showStatus(i18n.T("rp.quick_import_hint"))

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
		p.status.SetText(i18n.T("rp.status_ready"))
	})
}

func (p *Page) importFromURL(name, url, behavior string, interval int) {
	configPath, err := config.AddRuleProviderToConfig(name, url, behavior, interval)
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("rp.import_fail"), err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
	}

	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("rp.config_read_fail"), err))
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
		p.status.SetText(fmt.Sprintf(i18n.T("rp.import_ok"), name))
	})
}

func (p *Page) deleteSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError(i18n.T("rp.select_first"))
		return
	}
	rp := p.providers[p.selectedIndex]
	p.mu.Unlock()

	configPath, err := config.DeleteRuleProviderFromConfig(rp.Name)
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("rp.del_fail"), err))
		return
	}

	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
	}

	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("rp.config_read_fail"), err))
		return
	}

	go ui.Updater.UpdateUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf(i18n.T("rp.del_ok"), rp.Name))
	})
}

func (p *Page) refresh() {
	providers, err := config.GetRuleProvidersFromConfig()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("rp.config_read_fail"), err))
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
		p.status.SetText(i18n.T("rp.refreshed"))
	})
}

func (p *Page) updateSubList() {
	p.subList.Clear()
	p.mu.Lock()
	providers := p.providers
	p.mu.Unlock()

	if len(providers) == 0 {
		p.subList.AddItem(i18n.T("rp.empty_list"), i18n.T("rp.add_hint"), 0, nil)
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
		p.subList.AddItem(ui.StripFlagEmoji(rp.Name), sec, 0, nil)
	}
}

func (p *Page) showDetail(rp config.RuleProviderEntry) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(i18n.T("rp.detail_name"), ui.StripFlagEmoji(rp.Name)))
	b.WriteString(fmt.Sprintf(i18n.T("rp.detail_behavior"), rp.Behavior))
	if rp.URL != "" {
		b.WriteString(fmt.Sprintf("[yellow]URL:[white] %s\n", rp.URL))
	}
	if rp.Path != "" {
		b.WriteString(fmt.Sprintf(i18n.T("rp.detail_url"), rp.Path))
	}
	if rp.Type != "" {
		b.WriteString(fmt.Sprintf(i18n.T("rp.detail_type"), rp.Type))
	}
	if rp.Interval > 0 {
		b.WriteString(fmt.Sprintf(i18n.T("rp.detail_interval"), rp.Interval))
	}
	b.WriteString(i18n.T("rp.detail_in_config"))
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
p.pasteTextArea.SetPlaceholder(i18n.T("rp.paste_placeholder"))
	p.pasteTextArea.SetBorder(true)

	btnBar := tview.NewFlex()
	importBtn := tview.NewButton(i18n.T("rp.import_btn"))
	importBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	importBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	importBtn.SetSelectedFunc(func() {
		if p.showInput {
			go p.processPasteImportYAML()
		}
	})
	cancelBtn := tview.NewButton(i18n.T("rp.cancel_btn"))
	cancelBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	cancelBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
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
	p.pasteImportForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("rp.import_title")))
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
	p.showStatus(i18n.T("rp.paste_hint"))
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
	p.showStatus(i18n.T("rp.status_ready"))
}

func (p *Page) processPasteImportYAML() {
	// Capture text BEFORE hiding the form — removing from UI tree may clear buffer
	text := p.pasteTextArea.GetText()

	p.hidePasteImport()

	if strings.TrimSpace(text) == "" {
		p.showStatus(i18n.T("rp.paste_empty"))
		return
	}

	var rawProviders map[string]pasteProviderDef
	if err := yaml.Unmarshal([]byte(text), &rawProviders); err != nil {
		p.showStatus(fmt.Sprintf(i18n.T("rp.yaml_parse_fail"), err))
		return
	}

	if len(rawProviders) == 0 {
		p.showStatus(i18n.T("rp.no_providers_found"))
		return
	}

	p.showStatus(fmt.Sprintf(i18n.T("rp.importing_count"), len(rawProviders)))

	var batch []config.RuleProviderBatchEntry
	var errs []string

	for name, def := range rawProviders {
		if name == "" || def.URL == "" || def.Behavior == "" {
			errs = append(errs, fmt.Sprintf(i18n.T("rp.missing_fields"), name))
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
		msg := i18n.T("rp.no_providers_found")
		if len(errs) > 0 {
			msg += fmt.Sprintf(i18n.T("rp.skip_count"), len(errs))
		}
		p.showStatus(msg)
		return
	}

	cp, err := config.BatchAddRuleProvidersToConfig(batch)
	if err != nil {
		log.Printf("[ruleproviders] batch import failed: %v", err)
		p.showStatus(fmt.Sprintf(i18n.T("rp.import_fail"), err))
		return
	}

	if err := api.Client.ReloadConfig(cp); err != nil {
		log.Printf("[ruleproviders] hot-reload failed: %v", err)
		p.showStatus(fmt.Sprintf(i18n.T("rp.reload_fail"), err))
		return
	}

	msg := fmt.Sprintf(i18n.T("rp.import_ok_count"), len(batch))
	if len(errs) > 0 {
		msg += fmt.Sprintf(i18n.T("rp.skip_count"), len(errs))
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
