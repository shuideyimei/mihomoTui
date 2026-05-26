package pages

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	*tview.Flex

	pages         *tview.Pages
	mu            sync.Mutex
	configList    *tview.List
	editArea      *tview.TextArea
	statusText    *tview.TextView
	files         []string
	currentConfig string
	selectedIdx   int
	editMode      bool
	editFlex      *tview.Flex // saved reference for AddPage/RemovePage
}

func newConfigManagerPage() *ConfigManager {
	cm := &ConfigManager{
		Flex:          tview.NewFlex(),
		pages:         tview.NewPages(),
		selectedIdx:   -1,
		currentConfig: config.FindCurrentConfigPath(),
	}
	cm.setupUI()
	return cm
}

func (cm *ConfigManager) setupUI() {
	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.SetBorder(true)
	mainFlex.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.title")))

	cm.configList = tview.NewList()
	cm.configList.SetBorder(true)
	cm.configList.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.file_list")))
	cm.configList.SetMainTextColor(tcell.ColorWhite)
	cm.configList.SetSelectedBackgroundColor(ui.ThemeHighlightBg)
	cm.configList.SetSelectedTextColor(tcell.ColorBlack)
	cm.configList.SetInputCapture(cm.handleConfigListInput)
	cm.configList.SetChangedFunc(func(idx int, _ string, _ string, _ rune) {
		cm.selectedIdx = idx
	})
	cm.configList.SetFocusFunc(func() {
		cm.configList.SetSelectedBackgroundColor(ui.ThemeHighlightBg)
	})
	cm.configList.SetBlurFunc(func() {
		cm.configList.SetSelectedBackgroundColor(ui.ThemeSelBgBlur)
	})

	cm.editArea = tview.NewTextArea()
	cm.editArea.SetWrap(false)
	cm.editArea.SetWordWrap(false)

	cm.statusText = tview.NewTextView()
	cm.statusText.SetBorder(true)
	cm.statusText.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.status")))
	cm.statusText.SetDynamicColors(true)
	cm.statusText.SetText(i18n.T("cmgr.status_ready"))

	// Build edit dialog page (reusable, shown/hidden as a page overlay)
	editSaveBtn := tview.NewButton(i18n.T("cmgr.save_btn"))
	editSaveBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	editSaveBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	editCancelBtn := tview.NewButton(i18n.T("cmgr.cancel_btn"))
	editCancelBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	editCancelBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))

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
		cm.closeEditDialog()
	})
	cm.editFlex = editFlex

	editBtn := tview.NewButton(i18n.T("cmgr.preview_btn"))
	editBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	editBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	editBtn.SetSelectedFunc(func() { cm.showEditDialog() })

	deleteBtn := tview.NewButton(i18n.T("cmgr.delete_btn"))
	deleteBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	deleteBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	deleteBtn.SetLabelColorActivated(tcell.ColorWhite)
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
	mainFlex.AddItem(cm.statusText, 3, 0, false)
	mainFlex.AddItem(btnFlex, 1, 0, false)
	mainFlex.AddItem(helpText, 1, 0, false)

	cm.pages.AddPage("main", mainFlex, true, true)
	cm.AddItem(cm.pages, 0, 1, true)
}

func (cm *ConfigManager) handleConfigListInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
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
			cm.editArea.SetTitle(fmt.Sprintf(i18n.T("cmgr.edit_file"), config.FriendlyPath(path)))
			cm.editArea.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
				if ev.Key() == tcell.KeyCtrlS || (ev.Modifiers()&tcell.ModCtrl != 0 && (ev.Rune() == 's' || ev.Rune() == 'S')) {
					cm.statusText.SetText(i18n.T("cmgr.saving"))
					text := cm.editArea.GetText()
					go cm.saveEdit(text)
					return nil
				}
				if ev.Key() == tcell.KeyEscape {
					cm.closeEditDialog()
					return nil
				}
				return ev
			})
			cm.editArea.SetText(content, true)
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
	cm.editArea.SetText("", true)
	cm.pages.RemovePage("edit")
	cm.pages.SwitchToPage("main")
	cm.statusText.SetText(i18n.T("cmgr.edit_exited"))
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
	// Resolve symlinks to prevent sudo cp from following a malicious symlink
	realPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		return fmt.Errorf("解析配置文件路径失败: %w", err)
	}

	// Write to a temp file first, then use sudo cp to handle files
	// owned by a different user (e.g. mihomo:mihomo at /opt/clashtui/...)
	tmpFile, err := os.CreateTemp("", "mihomo_config_*.yaml")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := os.WriteFile(tmpPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// Verify the temp file is still a regular file (prevent TOCTOU swap)
	tmpStat, err := os.Stat(tmpPath)
	if err != nil || !tmpStat.Mode().IsRegular() {
		return fmt.Errorf("临时文件状态异常: %v", err)
	}

	origStat, _ := os.Stat(realPath)

	cmd := exec.Command("sudo", "cp", tmpPath, realPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("写入配置文件失败 (需要 sudo 权限): %s", string(output))
	}

	// Preserve original file ownership
	if origStat != nil {
		if st, ok := origStat.Sys().(*syscall.Stat_t); ok {
			if err := exec.Command("sudo", "chown", fmt.Sprintf("%d:%d", st.Uid, st.Gid), realPath).Run(); err != nil {
				return fmt.Errorf("恢复文件所有权失败: %w", err)
			}
		}
	}
	return nil
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

	isCurrent := path == cm.currentConfig

	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetTextAlign(tview.AlignCenter)
	warning := fmt.Sprintf(i18n.T("cmgr.delete_confirm"), tview.Escape(label))
	if isCurrent {
		warning += i18n.T("cmgr.delete_warn")
	}
	textView.SetText(warning)

	deleteBtn := tview.NewButton(i18n.T("cmgr.delete_confirm_btn"))
	deleteBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	deleteBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))
	cancelBtn := tview.NewButton(i18n.T("cmgr.delete_cancel_btn"))
	cancelBtn.SetStyle(tcell.StyleDefault.Background(ui.ThemeButtonBg).Foreground(tcell.ColorWhite))
	cancelBtn.SetActivatedStyle(tcell.StyleDefault.Background(ui.ThemeButtonFocusBg).Foreground(tcell.ColorWhite))

	btnFlex := tview.NewFlex()
	btnFlex.AddItem(deleteBtn, 0, 1, true)
	btnFlex.AddItem(cancelBtn, 0, 1, false)

	dialog := tview.NewFlex()
	dialog.SetDirection(tview.FlexRow)
	dialog.SetBorder(true)
	dialog.SetTitle(fmt.Sprintf(" %s ", i18n.T("cmgr.delete_title")))
	dialog.AddItem(textView, 3, 0, true)
	dialog.AddItem(btnFlex, 1, 0, false)

	flex := tview.NewFlex()
	flex.AddItem(nil, 0, 1, false)
	flex.AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(dialog, 6, 0, true).
		AddItem(nil, 0, 1, false), 50, 0, true)
	flex.AddItem(nil, 0, 1, false)

	deleteBtn.SetSelectedFunc(func() {
		if err := os.Remove(path); err != nil {
			cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.delete_fail"), tview.Escape(err.Error())))
		} else {
			cm.mu.Lock()
			if idx >= 0 && idx < len(cm.files) {
				cm.files = append(cm.files[:idx], cm.files[idx+1:]...)
			}
			cm.selectedIdx = -1
			cm.mu.Unlock()
			cm.rebuildList()
			cm.statusText.SetText(fmt.Sprintf(i18n.T("cmgr.deleted"), tview.Escape(label)))
			log.Printf("[configmgr] deleted config: %s", path)
		}
		cm.pages.SwitchToPage("main")
	})

	cancelBtn.SetSelectedFunc(func() {
		cm.pages.SwitchToPage("main")
	})

	dialog.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			cm.pages.SwitchToPage("main")
			return nil
		}
		return ev
	})

	cm.pages.AddPage("delete", flex, true, true)
	cm.pages.SwitchToPage("delete")
}

// ensureSafePaths adds the user's config directory to Mihomo's SAFE_PATHS so that
// path-based config reloads work without validation errors. It modifies the config
// file (for persistence) and patches the running config via API (for immediate effect).
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

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Printf("[configmgr] ensureSafePaths: read failed: %v", err)
		return
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		log.Printf("[configmgr] ensureSafePaths: parse failed: %v", err)
		return
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return
	}

	allPaths := []string{cfgDir}
	alreadyPresent := false
	var pathsNode *yaml.Node

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == "safe-paths" {
			pathsNode = mapping.Content[i+1]
			for j := 0; j < len(pathsNode.Content); j++ {
				v := pathsNode.Content[j].Value
				if v == cfgDir {
					alreadyPresent = true
				}
				// Collect all existing paths for the PATCH call
				if v != "" && v != cfgDir {
					allPaths = append(allPaths, v)
				}
			}
			break
		}
	}

	if alreadyPresent {
		return
	}

	if pathsNode != nil {
		pathsNode.Content = append(pathsNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: cfgDir,
			Style: yaml.DoubleQuotedStyle,
		})
	} else {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "safe-paths"},
			&yaml.Node{
				Kind: yaml.SequenceNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: cfgDir, Style: yaml.DoubleQuotedStyle},
				},
			},
		)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		log.Printf("[configmgr] ensureSafePaths: marshal failed: %v", err)
		return
	}

	// Use writeRawConfig which handles sudo cp for files owned by mihomo user
	if err := writeRawConfig(cfgPath, string(out)); err != nil {
		log.Printf("[configmgr] ensureSafePaths: write failed: %v", err)
		return
	}

	// Apply at runtime via PATCH /configs (send the complete list to avoid
	// overwriting existing entries)
	if err := api.Client.PatchConfig(map[string][]string{
		"safe-paths": allPaths,
	}); err != nil {
		log.Printf("[configmgr] ensureSafePaths: patch failed: %v", err)
	}
	log.Printf("[configmgr] ensureSafePaths: added %s to safe-paths", cfgDir)
}
