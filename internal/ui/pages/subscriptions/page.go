package subscriptions

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

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
	p.detailView.SetTitle(fmt.Sprintf(" %s ", i18n.T("sub.detail_title")))
	p.detailView.SetDynamicColors(true)
	p.detailView.SetWordWrap(true)

	p.actionBar = tview.NewFlex()
	components.StyleActionBar(p.actionBar)
	p.setupActionBar()

	p.status = tview.NewTextView()
	components.StyleStatusLine(p.status)
	p.status.SetText(i18n.T("sub.status_ready"))

	rightPanel.AddItem(p.detailView, 0, 3, false)
	rightPanel.AddItem(p.actionBar, 1, 0, false)
	rightPanel.AddItem(p.status, 1, 0, false)

	p.importForm = tview.NewForm()
	p.importForm.SetBorder(true)
	p.importForm.SetTitle(fmt.Sprintf(" %s ", i18n.T("sub.import_title")))
	components.StyleFocusBorder(p.importForm)
	p.importForm.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.nameInput = tview.NewInputField()
	p.nameInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.nameInput.SetLabel(i18n.T("sub.name_label"))
	p.nameInput.SetFieldWidth(0)
	p.urlInput = tview.NewInputField()
	p.urlInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.urlInput.SetLabel(i18n.T("sub.url_label"))
	p.urlInput.SetFieldWidth(0)
	p.intervalInput = tview.NewInputField()
	p.intervalInput.SetFieldBackgroundColor(ui.ThemeInputBg)
	p.intervalInput.SetLabel(i18n.T("sub.interval_label"))
	p.intervalInput.SetText("1440")
	p.intervalInput.SetFieldWidth(0)

	p.importForm.AddFormItem(p.nameInput)
	p.importForm.AddFormItem(p.urlInput)
	p.importForm.AddFormItem(p.intervalInput)
	p.importForm.AddButton(i18n.T("sub.from_url_btn"), p.onImportURL)
	p.importForm.AddButton(i18n.T("sub.from_local_btn"), p.onImportLocal)
	p.importForm.AddButton(i18n.T("sub.cancel_btn"), p.onCancelImport)
	p.importForm.SetButtonsAlign(tview.AlignCenter)
	p.importForm.SetButtonStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	p.importForm.SetButtonActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorBlack))

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(p.subList, 0, 1, true)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	mainFlex.AddItem(components.NewResponsiveSplit(leftPanel, rightPanel, 92, 2, 3), 0, 1, true)

	p.SetDirection(tview.FlexRow)
	p.AddItem(mainFlex, 0, 1, true)
	p.SetBorder(false)
	p.SetTitle(fmt.Sprintf(" %s ", i18n.T("sub.title")))
}

func (p *Page) setupSubList() {
	p.subList = tview.NewList()
	p.subList.SetBorder(true)
	p.subList.SetTitle(fmt.Sprintf(" %s ", i18n.T("sub.mihomo_providers")))
	components.StyleSelectableList(p.subList)
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
	refreshBtn := components.NewButton(i18n.T("sub.refresh_action"), components.ButtonNormal, func() { go p.refresh() })
	importBtn := components.NewButton(i18n.T("sub.import_action"), components.ButtonPrimary, p.showImportForm)
	updateBtn := components.NewButton(i18n.T("sub.update_action"), components.ButtonNormal, func() { go p.updateSelected() })
	deleteBtn := components.NewButton(i18n.T("sub.del_action"), components.ButtonDanger, func() { go p.deleteSelected() })

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
		p.showError(i18n.T("sub.name_url_required"))
		return
	}
	p.showStatus(fmt.Sprintf(i18n.T("sub.importing_url"), name))
	go p.importFromURL(name, url)
}

func (p *Page) onImportLocal() {
	name := strings.TrimSpace(p.nameInput.GetText())
	path := strings.TrimSpace(p.urlInput.GetText())
	if name == "" || path == "" {
		p.showError(i18n.T("sub.name_path_required"))
		return
	}
	p.showStatus(fmt.Sprintf(i18n.T("sub.importing_local"), name))
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
		p.showError(fmt.Sprintf(i18n.T("sub.import_fail"), err))
		return
	}

	configPath, cfgErr := config.AddProviderToConfig(name, subURL)
	if cfgErr != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_write_fail"), cfgErr))
		return
	}
	if err := api.Client.ReloadConfig(configPath); err != nil {
		log.Printf("[subs] hot-reload failed (will auto-load): %v", err)
	}

	log.Printf("[subs] imported %d proxies to mihomo as %s", len(doc.Proxies), name)

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_read_fail"), err))
		return
	}

	ui.Updater.PostUi(func() {
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
		p.status.SetText(fmt.Sprintf(i18n.T("sub.import_ok"), name, len(doc.Proxies)))
	})
}

func (p *Page) importFromLocal(name, path string) {
	doc, err := p.subMgr.ImportFromLocal(path)
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.import_fail"), err))
		return
	}

	log.Printf("[subs] parsed %d proxies from %s", len(doc.Proxies), name)

	ui.Updater.PostUi(func() {
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
		p.status.SetText(fmt.Sprintf(i18n.T("sub.import_ok_local"), name, len(doc.Proxies)))
	})
}

func (p *Page) updateSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError(i18n.T("sub.select_first"))
		return
	}
	pv := p.providers[p.selectedIndex]
	p.mu.Unlock()

	if pv.URL == "" {
		p.showError(i18n.T("sub.no_url"))
		return
	}

	p.showStatus(fmt.Sprintf(i18n.T("sub.updating"), pv.Name))

	doc, err := p.subMgr.ImportFromURL(context.Background(), pv.URL)
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.update_fail"), err))
		return
	}

	_, cfgErr := config.AddProviderToConfig(pv.Name, pv.URL)
	if cfgErr != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_write_fail"), cfgErr))
		return
	}

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_read_fail"), err))
		return
	}

	ui.Updater.PostUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf(i18n.T("sub.update_ok"), pv.Name, len(doc.Proxies)))
	})
}

func (p *Page) deleteSelected() {
	p.mu.Lock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.providers) {
		p.mu.Unlock()
		p.showError(i18n.T("sub.select_first"))
		return
	}
	pv := p.providers[p.selectedIndex]
	p.mu.Unlock()

	ui.Updater.ConfirmDelete(pv.Name, func() {
		go p.deleteProvider(pv)
	})
}

func (p *Page) deleteProvider(pv config.ProviderInfo) {
	if err := config.RemoveProviderFromConfig(pv.Name); err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.delete_fail"), err))
		return
	}

	configPath := config.FindMihomoConfigPath()
	if configPath != "" {
		api.Client.ReloadConfig(configPath)
	}

	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_read_fail"), err))
		return
	}

	ui.Updater.PostUi(func() {
		p.mu.Lock()
		p.providers = providers
		p.mu.Unlock()
		p.updateSubList()
		p.status.SetText(fmt.Sprintf(i18n.T("sub.del_ok"), pv.Name))
	})
}

func (p *Page) refresh() {
	providers, err := config.GetMihomoProxyProviders()
	if err != nil {
		p.showError(fmt.Sprintf(i18n.T("sub.config_read_fail"), err))
		return
	}
	ui.Updater.PostUi(func() {
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
		p.subList.AddItem(i18n.T("sub.empty_list"), i18n.T("sub.add_hint"), 0, nil)
		return
	}

	for _, pv := range providers {
		sec := pv.URL
		if sec == "" {
			sec = i18n.T("sub.local_tag")
		}
		if pv.Interval > 0 {
			sec += fmt.Sprintf(i18n.T("sub.interval_min"), pv.Interval)
		}
		p.subList.AddItem(ui.StripFlagEmoji(pv.Name), sec, 0, nil)
	}
}

func (p *Page) showDetail(pv config.ProviderInfo) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(i18n.T("sub.detail_name"), ui.StripFlagEmoji(pv.Name)))
	b.WriteString(fmt.Sprintf("[yellow]URL:[white] %s\n", pv.URL))
	b.WriteString(fmt.Sprintf(i18n.T("sub.detail_interval"), pv.Interval))
	b.WriteString(i18n.T("sub.detail_in_config"))
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
	ui.Updater.PostUi(func() { p.status.SetText(fmt.Sprintf("[red]%s[white]", msg)) })
}

func (p *Page) showStatus(msg string) {
	ui.Updater.PostUi(func() { p.status.SetText(msg) })
}
