package pages

import (
	"strings"
	"testing"

	"mihomoTui/internal/i18n"
	"mihomoTui/internal/ui"
)

func TestBuildHelpTextHasNoFormattingErrors(t *testing.T) {
	original := i18n.GetLanguage()
	defer i18n.SetLanguage(original)

	for _, language := range []i18n.Language{i18n.LangChinese, i18n.LangEnglish} {
		i18n.SetLanguage(language)
		text := buildHelpText(ui.DefaultPageInfo())
		if strings.Contains(text, "%!") {
			t.Fatalf("help text contains formatting error for %s: %s", language, text)
		}
		for _, page := range ui.DefaultPageInfo() {
			if !strings.Contains(text, page.Label()) {
				t.Fatalf("help text for %s does not contain page %q", language, page.Label())
			}
		}
	}
}
