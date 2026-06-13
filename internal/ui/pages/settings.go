package pages

import (
	"fmt"

	"mihomoTui/internal/config"
	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"
	"mihomoTui/internal/ui/components"

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

	langBtn    *tview.Button
	langStatus *tview.TextView
}

func newSettingsPage(cm *config.Manager, appName, appVersion string) *Settings {
	s := &Settings{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		configManager: cm,
	}

	s.AddItem(s.buildBackupSection(), 0, 1, false)
	s.AddItem(s.buildLangSection(), 0, 1, false)
	s.AddItem(s.buildUpdateSection(), 0, 1, false)
	s.AddItem(s.buildAboutSection(appName, appVersion), 0, 1, false)

	s.SetBorder(false)
	s.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.title")))

	return s
}

func (s *Settings) buildBackupSection() *tview.Flex {
	s.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	s.statusText.SetBorder(false)

	s.backupBtn = tview.NewButton(i18n.T("setting.backup_btn"))
	components.StyleButton(s.backupBtn, components.ButtonPrimary)
	s.backupBtn.SetSelectedFunc(func() {
		if err := s.configManager.Backup(); err != nil {
			s.showStatus(fmt.Sprintf(i18n.T("setting.backup_fail"), err.Error()))
		} else {
			s.showStatus(i18n.T("setting.backup_ok"))
		}
	})

	s.restoreBtn = tview.NewButton(i18n.T("setting.restore_btn"))
	components.StyleButton(s.restoreBtn, components.ButtonDanger)
	s.restoreBtn.SetSelectedFunc(func() {
		ui.Updater.ShowConfirm(i18n.T("setting.restore_btn"), i18n.T("setting.restore_confirm"), func() {
			if err := s.configManager.Restore(); err != nil {
				s.showStatus(fmt.Sprintf(i18n.T("setting.restore_fail"), err.Error()))
			} else {
				s.showStatus(i18n.T("setting.restore_ok"))
			}
		})
	})

	btnFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(s.backupBtn, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(s.restoreBtn, 12, 0, false).
		AddItem(nil, 0, 1, false)

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.backup_section")))
	section.AddItem(btnFlex, 3, 0, false)
	section.AddItem(s.statusText, 1, 0, false)

	return section
}

func (s *Settings) buildLangSection() *tview.Flex {
	currentLang := i18n.GetLanguage()
	langDisplay := i18n.T("setting.lang_zh")
	if currentLang == i18n.LangEnglish {
		langDisplay = i18n.T("setting.lang_en")
	}

	s.langStatus = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	s.langStatus.SetText(fmt.Sprintf(i18n.T("setting.lang_current"), langDisplay))

	s.langBtn = tview.NewButton(i18n.T("setting.lang_switch_btn"))
	components.StyleButton(s.langBtn, components.ButtonNormal)
	s.langBtn.SetSelectedFunc(func() {
		// Toggle language
		currentLang := i18n.GetLanguage()
		var newLang i18n.Language
		if currentLang == i18n.LangChinese {
			newLang = i18n.LangEnglish
		} else {
			newLang = i18n.LangChinese
		}
		i18n.SetLanguage(newLang)
		s.configManager.SetLanguage(string(newLang))

		// Update display
		langDisplay := i18n.T("setting.lang_zh")
		if newLang == i18n.LangEnglish {
			langDisplay = i18n.T("setting.lang_en")
		}
		s.langStatus.SetText(fmt.Sprintf(i18n.T("setting.lang_current"), langDisplay))
		s.showStatus(i18n.T("setting.lang_switched"))

		// Refresh UI texts that can be updated dynamically
		s.refreshTexts()
	})

	btnFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(s.langBtn, 16, 0, false).
		AddItem(nil, 0, 1, false)

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.lang_section")))
	section.AddItem(btnFlex, 3, 0, false)
	section.AddItem(s.langStatus, 1, 0, false)

	return section
}

// refreshTexts updates the UI texts that can change on language switch
func (s *Settings) refreshTexts() {
	s.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.title")))
	s.backupBtn.SetLabel(i18n.T("setting.backup_btn"))
	s.restoreBtn.SetLabel(i18n.T("setting.restore_btn"))
	s.langBtn.SetLabel(i18n.T("setting.lang_switch_btn"))

	// Rebuild sections - we just update section titles
	// Since we can't easily rebuild flex children, we update what we can
}

func (s *Settings) buildUpdateSection() *tview.Flex {
	updateText := tview.NewTextView().
		SetText(i18n.T("setting.update_placeholder")).
		SetTextAlign(tview.AlignCenter)

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.update_section")))
	section.AddItem(nil, 0, 1, false)
	section.AddItem(updateText, 1, 0, false)
	section.AddItem(nil, 0, 1, false)

	return section
}

func (s *Settings) buildAboutSection(appName, appVersion string) *tview.Flex {
	aboutText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter)
	aboutText.SetText(fmt.Sprintf(i18n.T("setting.about_format"), appName, appVersion))

	section := tview.NewFlex().SetDirection(tview.FlexRow)
	section.SetBorder(true)
	section.SetTitle(fmt.Sprintf(" %s ", i18n.T("setting.about_section")))
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
