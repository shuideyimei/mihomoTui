package pages

import (
	"fmt"

	"mihomoTui/internal/config"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Settings is the application settings page.
type Settings struct {
	*tview.Flex
	configManager *config.Manager

	backupBtn  *tview.Button
	restoreBtn *tview.Button
	statusText *tview.TextView
}

func newSettingsPage(cm *config.Manager, appName, appVersion string) *Settings {
	s := &Settings{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		configManager: cm,
	}

	s.AddItem(s.buildBackupSection(), 0, 1, false)
	s.AddItem(s.buildUpdateSection(), 0, 1, false)
	s.AddItem(s.buildAboutSection(appName, appVersion), 0, 1, false)

	return s
}

func (s *Settings) buildBackupSection() *tview.Flex {
	s.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	s.statusText.SetBorder(false)

	s.backupBtn = tview.NewButton("备份配置")
	s.backupBtn.SetBorder(true)
	s.backupBtn.SetSelectedFunc(func() {
		if err := s.configManager.Backup(); err != nil {
			s.showStatus(fmt.Sprintf("[red]备份失败: %s[-]", err.Error()))
		} else {
			s.showStatus("[green]配置已备份成功[-]")
		}
	})

	s.restoreBtn = tview.NewButton("恢复配置")
	s.restoreBtn.SetBorder(true)
	s.restoreBtn.SetSelectedFunc(func() {
		if err := s.configManager.Restore(); err != nil {
			s.showStatus(fmt.Sprintf("[red]恢复失败: %s[-]", err.Error()))
		} else {
			s.showStatus("[green]配置已恢复成功[-]")
		}
	})

	btnFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(s.backupBtn, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(s.restoreBtn, 12, 0, false).
		AddItem(nil, 0, 1, false)

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(" 配置备份/恢复 ")
	section.AddItem(btnFlex, 3, 0, false)
	section.AddItem(s.statusText, 1, 0, false)

	return section
}

func (s *Settings) buildUpdateSection() *tview.Flex {
	updateText := tview.NewTextView().
		SetText("更新间隔: 即将推出...").
		SetTextAlign(tview.AlignCenter)

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(" 更新设置 ")
	section.AddItem(nil, 0, 1, false)
	section.AddItem(updateText, 1, 0, false)
	section.AddItem(nil, 0, 1, false)

	return section
}

func (s *Settings) buildAboutSection(appName, appVersion string) *tview.Flex {
	aboutText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter)
	aboutText.SetText(fmt.Sprintf("应用: %s\n版本: %s", appName, appVersion))

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(" 关于 ")
	section.AddItem(nil, 0, 1, false)
	section.AddItem(aboutText, 3, 0, false)
	section.AddItem(nil, 0, 1, false)

	return section
}

func (s *Settings) showStatus(message string) {
	s.statusText.SetText(message)
}

func (s *Settings) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if s.backupBtn.HasFocus() {
				s.restoreBtn.Focus(nil)
			} else {
				s.backupBtn.Focus(nil)
			}
			return nil
		}
		return event
	}
}
