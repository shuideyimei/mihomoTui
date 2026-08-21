package pages

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/events"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/service"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	*tview.Flex

	pages          *tview.Pages
	profileService *service.ProfileService
	bus            *events.Bus
	mu             sync.Mutex
	configList     *tview.List
	editArea       *tview.TextArea
	statusText     *tview.TextView
	files          []string
	currentConfig  string
	selectedIdx    int
	editMode       bool
	editFlex       *tview.Flex // saved reference for AddPage/RemovePage
	editPath       string
	editOriginal   string
	editDirty      bool
}

func newConfigManagerPage(profileService *service.ProfileService, bus *events.Bus) *ConfigManager {
	cm := &ConfigManager{
		Flex:           tview.NewFlex(),
		pages:          tview.NewPages(),
		profileService: profileService,
		bus:            bus,
		selectedIdx:    -1,
		currentConfig:  config.FindCurrentConfigPath(),
	}
	cm.setupUI()
	return cm
}

func (cm *ConfigManager) setupUI() {
	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.SetBorder(false)

	cm.configList = tview.NewList()
	cm.configList.SetBorder(true)
	cm.configList.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.file_list")))
	components.StyleSelectableList(cm.configList)
	cm.configList.SetInputCapture(cm.handleConfigListInput)
	cm.configList.SetSelectedFunc(func(_ int, _ string, _ string, _ rune) {
		cm.activateSelectedConfig()
	})
	cm.configList.SetChangedFunc(func(idx int, _ string, _ string, _ rune) {
		cm.selectedIdx = idx
	})

	cm.editArea = tview.NewTextArea()
	cm.editArea.SetWrap(false)
	cm.editArea.SetWordWrap(false)
	cm.editArea.SetBorder(true)
	components.StyleFocusBorder(cm.editArea)
	cm.editArea.SetChangedFunc(cm.updateEditTitle)
	cm.editArea.SetMovedFunc(cm.updateEditTitle)

	cm.statusText = tview.NewTextView()
	components.StyleStatusLine(cm.statusText)
	cm.statusText.SetText(i18n.T("cmgr.status_ready"))

	// Build edit dialog page (reusable, shown/hidden as a page overlay)
	editSaveBtn := components.NewButton(i18n.T("cmgr.save_btn"), components.ButtonPrimary, nil)
	editCancelBtn := components.NewButton(i18n.T("cmgr.cancel_btn"), components.ButtonNormal, nil)

	editBtnFlex := tview.NewFlex()
	editBtnFlex.AddItem(nil, 0, 1, false)
	editBtnFlex.AddItem(editSaveBtn, 0, 1, true)
	editBtnFlex.AddItem(nil, 1, 0, false)
	editBtnFlex.AddItem(editCancelBtn, 0, 1, false)
	editBtnFlex.AddItem(nil, 0, 1, false)

	editDialog := tview.NewFlex()
	editDialog.SetDirection(tview.FlexRow)
	editDialog.SetBorder(true)
	editDialog.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.edit_title")))
	editDialog.AddItem(cm.editArea, 0, 1, true)
	editDialog.AddItem(editBtnFlex, 1, 0, false)

	editFlex := tview.NewFlex()
	editFlex.AddItem(nil, 0, 1, false)
	editFlex.AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(editDialog, 0, 2, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	editFlex.AddItem(nil, 0, 1, false)

	editSaveBtn.SetSelectedFunc(func() {
		cm.statusText.SetText(i18n.T("cmgr.saving"))
		text := cm.editArea.GetText()
		go cm.saveEdit(text)
	})
	editCancelBtn.SetSelectedFunc(func() {
		cm.requestCloseEditDialog()
	})
	cm.editFlex = editFlex

	editBtn := components.NewButton(i18n.T("cmgr.preview_btn"), components.ButtonNormal, nil)
	editBtn.SetSelectedFunc(func() { cm.showEditDialog() })

	deleteBtn := components.NewButton(i18n.T("cmgr.delete_btn"), components.ButtonDanger, nil)
	deleteBtn.SetSelectedFunc(func() { cm.showDeleteConfirm() })

	btnFlex := tview.NewFlex()
	btnFlex.AddItem(nil, 0, 1, false)
	btnFlex.AddItem(editBtn, 8, 0, false)
	btnFlex.AddItem(nil, 1, 0, false)
	btnFlex.AddItem(deleteBtn, 8, 0, false)
	btnFlex.AddItem(nil, 0, 1, false)

	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText(i18n.T("cmgr.nav_hint"))
	helpText.SetTextAlign(tview.AlignCenter)

	mainFlex.AddItem(cm.configList, 0, 1, true)
	mainFlex.AddItem(cm.statusText, 1, 0, false)
	mainFlex.AddItem(btnFlex, 1, 0, false)
	mainFlex.AddItem(helpText, 1, 0, false)

	cm.pages.AddPage("main", mainFlex, true, true)
	cm.AddItem(cm.pages, 0, 1, true)
}

func (cm *ConfigManager) handleConfigListInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		cm.activateSelectedConfig()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'e', 'E':
			cm.showEditDialog()
			return nil
		case 'd', 'D':
			cm.showDeleteConfirm()
			return nil
		case 'r', 'R':
			go cm.refresh()
			return nil
		}
	}
	return event
}

func (cm *ConfigManager) activateSelectedConfig() {
	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		cm.statusText.SetText(i18n.T("cmgr.select_first"))
		return
	}

	path := files[idx]
	cm.statusText.SetText(i18n.T("cmgr.saving"))

	go func() {
		var err error
		if cm.profileService != nil {
			err = cm.profileService.SwitchProfile(path)
		} else {
			err = api.Client.ReloadConfig(path)
		}

		ui.Updater.PostUi(func() {
			if err != nil {
				cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.save_fail"), tview.Escape(err.Error())))
				return
			}
			cm.mu.Lock()
			cm.currentConfig = path
			cm.mu.Unlock()
			cm.rebuildList()
			cm.statusText.SetText(fmt.Sprintf("%s %s", i18n.T("cmgr.current_tag"), config.FriendlyPath(path)))
		})
	}()
}

func (cm *ConfigManager) Activate() {
	cm.mu.Lock()
	cm.currentConfig = config.FindCurrentConfigPath()
	cm.mu.Unlock()
	ensureSafePaths()
	go cm.refresh()
}

func (cm *ConfigManager) Deactivate() {
	if cm.editMode {
		ui.Updater.PostUi(func() {
			cm.closeEditDialog()
		})
	}
}

func (cm *ConfigManager) RequestDeactivate(continueNavigation func()) bool {
	if !cm.editMode || !cm.editDirty {
		return true
	}
	ui.Updater.ShowConfirm(i18n.T("cmgr.unsaved_title"), i18n.T("cmgr.unsaved_confirm"), func() {
		cm.closeEditDialog()
		continueNavigation()
	})
	return false
}

func (cm *ConfigManager) refresh() {
	cm.mu.Lock()
	cm.currentConfig = config.FindCurrentConfigPath()
	cm.files = config.FindAllConfigFiles()
	idx := -1
	for i, f := range cm.files {
		if f == cm.currentConfig {
			idx = i
			break
		}
	}
	if idx == -1 && len(cm.files) > 0 {
		idx = 0
	}
	cm.selectedIdx = idx
	// capture copies to avoid closure data races
	files := make([]string, len(cm.files))
	copy(files, cm.files)
	current := cm.currentConfig
	cm.mu.Unlock()

	ui.Updater.PostUi(func() {
		cm.configList.Clear()
		for _, f := range files {
			label := config.FriendlyPath(f)
			if f == current {
				label = i18n.T("cmgr.current_tag") + label
			}
			cm.configList.AddItem(label, "", 0, nil)
		}
		if idx >= 0 {
			cm.configList.SetCurrentItem(idx)
		}
		cm.statusText.SetText(i18n.T("cmgr.refreshed"))
	})
}

// rebuildList updates the list widget from cm.files/cm.currentConfig.
// Safe to call from any goroutine.
func (cm *ConfigManager) rebuildList() {
	cm.mu.Lock()
	files := make([]string, len(cm.files))
	copy(files, cm.files)
	current := cm.currentConfig
	cm.mu.Unlock()

	ui.Updater.PostUi(func() {
		cm.configList.Clear()
		for _, f := range files {
			label := config.FriendlyPath(f)
			if f == current {
				label = i18n.T("cmgr.current_tag") + label
			}
			cm.configList.AddItem(label, "", 0, nil)
		}
		idx := 0
		if current != "" {
			for i, f := range files {
				if f == current {
					idx = i
					break
				}
			}
		}
		cm.configList.SetCurrentItem(idx)
		cm.selectedIdx = idx
	})
}

func (cm *ConfigManager) showEditDialog() {
	if cm.editMode {
		return
	}
	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		cm.statusText.SetText(i18n.T("cmgr.select_first"))
		return
	}

	path := files[idx]
	go func() {
		data, err := os.ReadFile(path)
		if err != nil {
			ui.Updater.PostUi(func() {
				cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.read_fail"), tview.Escape(err.Error())))
			})
			return
		}
		content := string(data)

		ui.Updater.PostUi(func() {
			cm.editMode = true
			cm.editPath = path
			cm.editOriginal = content
			cm.editDirty = false
			cm.editArea.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
				if ev.Key() == tcell.KeyCtrlS || (ev.Modifiers()&tcell.ModCtrl != 0 && (ev.Rune() == 's' || ev.Rune() == 'S')) {
					cm.statusText.SetText(i18n.T("cmgr.saving"))
					text := cm.editArea.GetText()
					go cm.saveEdit(text)
					return nil
				}
				if ev.Key() == tcell.KeyEscape {
					cm.requestCloseEditDialog()
					return nil
				}
				return ev
			})
			cm.editArea.SetText(content, true)
			cm.updateEditTitle()
			cm.pages.AddPage("edit", cm.editFlex, true, true)
			ui.Updater.SetFocus(cm.editArea)
		})
	}()
}

func (cm *ConfigManager) closeEditDialog() {
	if !cm.editMode {
		return
	}
	cm.editMode = false
	cm.editPath = ""
	cm.editOriginal = ""
	cm.editDirty = false
	cm.editArea.SetText("", true)
	cm.pages.RemovePage("edit")
	cm.pages.SwitchToPage("main")
	cm.statusText.SetText(i18n.T("cmgr.edit_exited"))
}

func (cm *ConfigManager) requestCloseEditDialog() {
	if !cm.editDirty {
		cm.closeEditDialog()
		return
	}
	ui.Updater.ShowConfirm(i18n.T("cmgr.unsaved_title"), i18n.T("cmgr.unsaved_confirm"), cm.closeEditDialog)
}

func (cm *ConfigManager) updateEditTitle() {
	if !cm.editMode {
		return
	}
	cm.editDirty = cm.editArea.GetText() != cm.editOriginal
	row, column, _, _ := cm.editArea.GetCursor()
	dirty := ""
	if cm.editDirty {
		dirty = " *"
	}
	cm.editArea.SetTitle(fmt.Sprintf(i18n.T("cmgr.edit_position"),
		config.FriendlyPath(cm.editPath), row+1, column+1, dirty))
}

func (cm *ConfigManager) saveEdit(text string) {
	if !cm.editMode {
		return
	}

	if text == "" {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(i18n.T("cmgr.content_empty"))
		})
		return
	}

	var temp interface{}
	if err := yaml.Unmarshal([]byte(text), &temp); err != nil {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.yaml_invalid"), tview.Escape(err.Error())))
		})
		return
	}

	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(i18n.T("cmgr.no_selection"))
		})
		return
	}

	configPath := files[idx]

	if api.Client == nil {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(i18n.T("cmgr.no_api_client"))
			cm.closeEditDialog()
		})
		return
	}

	log.Printf("[configmgr] writing config to %s", configPath)
	saveErr := writeRawConfig(configPath, text)
	if saveErr != nil {
		log.Printf("[configmgr] write failed: %v", saveErr)
	} else {
		// Try raw YAML reload first (content-type: application/yaml)
		log.Printf("[configmgr] calling ReloadConfigData")
		saveErr = api.Client.ReloadConfigData([]byte(text))
		log.Printf("[configmgr] ReloadConfigData done: %v", saveErr)

		// If raw reload fails, fall back to file-path based reload
		if saveErr != nil {
			log.Printf("[configmgr] ReloadConfigData failed, trying ReloadConfig as fallback")
			saveErr = api.Client.ReloadConfig(configPath)
			log.Printf("[configmgr] ReloadConfig fallback done: %v", saveErr)
		}
	}

	log.Printf("[configmgr] PostUi closeEditDialog start")
	// Single UI update: show result and close dialog (non-blocking)
	ui.Updater.PostUi(func() {
		log.Printf("[configmgr] PostUi callback running")
		if saveErr != nil {
			cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.save_fail"), tview.Escape(saveErr.Error())))
		} else {
			cm.statusText.SetText(i18n.T("cmgr.save_ok"))
		}
		cm.closeEditDialog()
		log.Printf("[configmgr] PostUi callback done")
	})
}

func writeRawConfig(configPath, text string) error {
	return config.WriteFileAtomic(configPath, []byte(text), 0644)
}

func (cm *ConfigManager) showDeleteConfirm() {
	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		cm.statusText.SetText(i18n.T("cmgr.select_first"))
		return
	}

	path := files[idx]
	label := config.FriendlyPath(path)

	if config.SameConfigFile(path, config.FindCurrentConfigPath()) {
		cm.statusText.SetText(i18n.T("cmgr.delete_active_forbidden"))
		return
	}

	ui.Updater.ConfirmDelete(label, func() {
		go cm.deleteConfig(path, label)
	})
}

func (cm *ConfigManager) deleteConfig(path, label string) {
	if config.SameConfigFile(path, config.FindCurrentConfigPath()) {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(i18n.T("cmgr.delete_active_forbidden"))
		})
		return
	}
	err := os.Remove(path)
	ui.Updater.PostUi(func() {
		if err != nil {
			cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.delete_fail"), tview.Escape(err.Error())))
			return
		}
		cm.mu.Lock()
		for index, file := range cm.files {
			if file == path {
				cm.files = append(cm.files[:index], cm.files[index+1:]...)
				break
			}
		}
		cm.selectedIdx = -1
		cm.mu.Unlock()
		cm.rebuildList()
		cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.deleted"), tview.Escape(label)))
		log.Printf("[configmgr] deleted config: %s", path)
	})
}

// UpdateTexts refreshes all localized labels on the config manager page
func (cm *ConfigManager) UpdateTexts() {
	ui.Updater.PostUi(func() {
		if cm.configList != nil {
			cm.configList.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.file_list")))
		}
		if cm.statusText != nil {
			cm.statusText.SetText(i18n.T("cmgr.status_ready"))
		}
		cm.rebuildList()
	})
}

// ensureSafePaths adds the user's config directory to Mihomo's SAFE_PATHS so that
// path-based config reloads work without validation errors. The config-file part
// runs under configMu via config.EnsureSafePathInConfig; the running config is
// then patched via API for immediate effect.
func ensureSafePaths() {
	cfgPath := config.FindCurrentConfigPath()
	if cfgPath == "" {
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cfgDir := filepath.Join(homeDir, ".config", "mihomo")

	paths, err := config.EnsureSafePathInConfig(cfgPath, cfgDir)
	if err != nil {
		log.Printf("[configmgr] ensureSafePaths: %v", err)
		return
	}
	if paths == nil {
		return // already present
	}

	// Apply at runtime via PATCH /configs (send the complete list to avoid
	// overwriting existing entries)
	if err := api.Client.PatchConfig(map[string][]string{
		"safe-paths": paths,
	}); err != nil {
		log.Printf("[configmgr] ensureSafePaths: patch failed: %v", err)
	}
	log.Printf("[configmgr] ensureSafePaths: added %s to safe-paths", cfgDir)
}
