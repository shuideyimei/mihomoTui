package pages

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"mihomoTui/internal/api"
	"mihomoTui/internal/config"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type ConfigManager struct {
	*tview.Pages

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
		Pages:         tview.NewPages(),
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
	mainFlex.SetTitle(" 配置管理 ")

	cm.configList = tview.NewList()
	cm.configList.SetBorder(true)
	cm.configList.SetTitle(" 配置文件列表 ")
	cm.configList.SetInputCapture(cm.handleConfigListInput)
	cm.configList.SetChangedFunc(func(idx int, _ string, _ string, _ rune) {
		cm.selectedIdx = idx
	})

	cm.editArea = tview.NewTextArea()
	cm.editArea.SetWrap(false)
	cm.editArea.SetWordWrap(false)

	cm.statusText = tview.NewTextView()
	cm.statusText.SetBorder(true)
	cm.statusText.SetTitle(" 状态 ")
	cm.statusText.SetDynamicColors(true)
	cm.statusText.SetText("[green]就绪[-]")

	// Build edit dialog page (reusable, shown/hidden as a page overlay)
	editSaveBtn := tview.NewButton("[white::b] 保存(Ctrl+S) ")
	editSaveBtn.SetBackgroundColor(tcell.ColorGrey)
	editSaveBtn.SetLabelColor(tcell.ColorWhite)
	editCancelBtn := tview.NewButton("[white::b] 取消(Esc) ")
	editCancelBtn.SetBackgroundColor(tcell.ColorGrey)
	editCancelBtn.SetLabelColor(tcell.ColorWhite)

	editBtnFlex := tview.NewFlex()
	editBtnFlex.AddItem(nil, 0, 1, false)
	editBtnFlex.AddItem(editSaveBtn, 0, 1, true)
	editBtnFlex.AddItem(nil, 1, 0, false)
	editBtnFlex.AddItem(editCancelBtn, 0, 1, false)
	editBtnFlex.AddItem(nil, 0, 1, false)

	editDialog := tview.NewFlex()
	editDialog.SetDirection(tview.FlexRow)
	editDialog.SetBorder(true)
	editDialog.SetTitle(" 编辑配置 ")
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
		cm.statusText.SetText("[yellow]正在保存配置...[-]")
		text := cm.editArea.GetText()
		go cm.saveEdit(text)
	})
	editCancelBtn.SetSelectedFunc(func() {
		cm.closeEditDialog()
	})
	cm.editFlex = editFlex

	editBtn := tview.NewButton("[white::b] 预览 ")
	editBtn.SetBackgroundColor(tcell.ColorGrey)
	editBtn.SetLabelColor(tcell.ColorWhite)
	editBtn.SetSelectedFunc(func() { cm.showEditDialog() })

	deleteBtn := tview.NewButton("[white::b] 删除 ")
	deleteBtn.SetBackgroundColor(tcell.ColorGrey)
	deleteBtn.SetLabelColor(tcell.ColorWhite)
	deleteBtn.SetLabelColorActivated(tcell.ColorBlack)
	deleteBtn.SetBackgroundColorActivated(tcell.ColorWhite)
	deleteBtn.SetSelectedFunc(func() { cm.showDeleteConfirm() })

	btnFlex := tview.NewFlex()
	btnFlex.AddItem(nil, 0, 1, false)
	btnFlex.AddItem(editBtn, 8, 0, false)
	btnFlex.AddItem(nil, 1, 0, false)
	btnFlex.AddItem(deleteBtn, 8, 0, false)
	btnFlex.AddItem(nil, 0, 1, false)

	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText("[gray]E 编辑/预览 | D 删除 | R 刷新 | Tab 导航[-]")
	helpText.SetTextAlign(tview.AlignCenter)

	mainFlex.AddItem(cm.configList, 0, 1, true)
	mainFlex.AddItem(cm.statusText, 3, 0, false)
	mainFlex.AddItem(btnFlex, 1, 0, false)
	mainFlex.AddItem(helpText, 1, 0, false)

	cm.AddPage("main", mainFlex, true, true)
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

	ui.Updater.UpdateUi(func() {
		cm.configList.Clear()
		for _, f := range files {
			label := config.FriendlyPath(f)
			if f == current {
				label = "[当前] " + label
			}
			cm.configList.AddItem(label, "", 0, nil)
		}
		if idx >= 0 {
			cm.configList.SetCurrentItem(idx)
		}
		cm.statusText.SetText("[green]已刷新列表[-]")
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
				label = "[当前] " + label
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
		cm.statusText.SetText("[red]请先选择一个配置文件[-]")
		return
	}

	path := files[idx]
	go func() {
		data, err := os.ReadFile(path)
		if err != nil {
			ui.Updater.PostUi(func() {
				cm.statusText.SetText(fmt.Sprintf("[red]读取失败: %s[-]", tview.Escape(err.Error())))
			})
			return
		}
		content := string(data)

		ui.Updater.PostUi(func() {
			cm.editMode = true
			cm.editArea.SetTitle(fmt.Sprintf(" 编辑: %s ", config.FriendlyPath(path)))
			cm.editArea.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
				if ev.Key() == tcell.KeyCtrlS || (ev.Modifiers()&tcell.ModCtrl != 0 && (ev.Rune() == 's' || ev.Rune() == 'S')) {
					cm.statusText.SetText("[yellow]正在保存配置...[-]")
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
			cm.AddPage("edit", cm.editFlex, true, true)
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
	cm.RemovePage("edit")
	cm.SwitchToPage("main")
	cm.statusText.SetText("[green]已退出编辑模式[-]")
}

func (cm *ConfigManager) saveEdit(text string) {
	if !cm.editMode {
		return
	}

	if text == "" {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText("[red]配置内容不能为空[-]")
		})
		return
	}

	var temp interface{}
	if err := yaml.Unmarshal([]byte(text), &temp); err != nil {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText(fmt.Sprintf("[red]YAML 格式错误: %s[-]", tview.Escape(err.Error())))
		})
		return
	}

	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		ui.Updater.PostUi(func() {
			cm.statusText.SetText("[red]未选择配置文件[-]")
		})
		return
	}

	configPath := files[idx]

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
			cm.statusText.SetText(fmt.Sprintf("[red]保存失败: %s[-]", tview.Escape(saveErr.Error())))
		} else {
			cm.statusText.SetText("[green]配置已保存并重载成功[-]")
		}
		cm.closeEditDialog()
		log.Printf("[configmgr] PostUi callback done")
	})
}

func writeRawConfig(configPath, text string) error {
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

	origStat, _ := os.Stat(configPath)

	cmd := exec.Command("sudo", "cp", tmpPath, configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("写入配置文件失败 (需要 sudo 权限): %s", string(output))
	}

	// Preserve original file ownership
	if origStat != nil {
		if st, ok := origStat.Sys().(*syscall.Stat_t); ok {
			if err := exec.Command("sudo", "chown", fmt.Sprintf("%d:%d", st.Uid, st.Gid), configPath).Run(); err != nil {
				return fmt.Errorf("恢复文件所有权失败: %w", err)
			}
		}
	}
	return nil
}

func yamlHighlight(text string) string {
	lines := strings.Split(text, "\n")
	var buf strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			buf.WriteByte('\n')
			continue
		}
		if trimmed[0] == '#' {
			buf.WriteString("[gray::]" + tview.Escape(line) + "[-::-]\n")
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx >= 0 {
			keyPart := line[:colonIdx]
			valPart := line[colonIdx:]
			buf.WriteString("[yellow]" + tview.Escape(keyPart) + "[-]" + tview.Escape(valPart) + "\n")
			continue
		}
		dashIdx := strings.Index(line, "- ")
		if dashIdx >= 0 {
			beforeDash := line[:dashIdx]
			afterDash := line[dashIdx+2:]
			buf.WriteString(tview.Escape(beforeDash) + "[green]-[-] " + tview.Escape(afterDash) + "\n")
			continue
		}
		buf.WriteString(tview.Escape(line) + "\n")
	}
	return buf.String()
}

func (cm *ConfigManager) showDeleteConfirm() {
	cm.mu.Lock()
	idx := cm.selectedIdx
	files := cm.files
	cm.mu.Unlock()

	if idx < 0 || idx >= len(files) {
		cm.statusText.SetText("[red]请先选择一个配置文件[-]")
		return
	}

	path := files[idx]
	label := config.FriendlyPath(path)

	isCurrent := path == cm.currentConfig

	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetTextAlign(tview.AlignCenter)
	warning := fmt.Sprintf("确定要删除配置文件吗?\n\n[red]%s[-]", tview.Escape(label))
	if isCurrent {
		warning += "\n\n[yellow]警告: 这是当前使用的配置文件![-]"
	}
	textView.SetText(warning)

	deleteBtn := tview.NewButton("删除")
	cancelBtn := tview.NewButton("取消")

	btnFlex := tview.NewFlex()
	btnFlex.AddItem(deleteBtn, 0, 1, true)
	btnFlex.AddItem(cancelBtn, 0, 1, false)

	dialog := tview.NewFlex()
	dialog.SetDirection(tview.FlexRow)
	dialog.SetBorder(true)
	dialog.SetTitle(" 删除配置 ")
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
			cm.statusText.SetText(fmt.Sprintf("[red]删除失败: %s[-]", tview.Escape(err.Error())))
		} else {
			cm.mu.Lock()
			if idx >= 0 && idx < len(cm.files) {
				cm.files = append(cm.files[:idx], cm.files[idx+1:]...)
			}
			cm.selectedIdx = -1
			cm.mu.Unlock()
			cm.rebuildList()
			cm.statusText.SetText(fmt.Sprintf("[green]已删除: %s[-]", tview.Escape(label)))
			log.Printf("[configmgr] deleted config: %s", path)
		}
		cm.SwitchToPage("main")
	})

	cancelBtn.SetSelectedFunc(func() {
		cm.SwitchToPage("main")
	})

	dialog.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			cm.SwitchToPage("main")
			return nil
		}
		return ev
	})

	cm.AddPage("delete", flex, true, true)
	cm.SwitchToPage("delete")
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
