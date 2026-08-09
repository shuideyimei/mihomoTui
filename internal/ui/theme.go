package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme color constants (tcell.Color for component styling)
// Modern palette with a mix of blues and grays for a polished look.
var (
	ThemeBorderColor = tcell.ColorDimGray
	ThemeTitleColor  = tcell.ColorAqua     // Cyan for titles
	ThemeHighlightBg = tcell.ColorSteelBlue
	ThemeSelBgFocus  = tcell.ColorRoyalBlue
	ThemeSelBgBlur   = tcell.ColorDarkSlateGray
	ThemeHeaderBg    = tcell.ColorDarkSlateGray // Header background
	ThemeStatusBarBg = tcell.ColorDarkSlateGray // Status bar background
	ThemeFocusColor  = tcell.ColorGoldenrod
	ThemeActiveColor = tcell.ColorMediumSpringGreen
	ThemeDangerColor = tcell.ColorIndianRed

	ThemeSelectionColor     = tcell.ColorRoyalBlue
	ThemeSelectionBlurColor = tcell.ColorDarkSlateGray
	// Component backgrounds
	ThemeButtonBg      = tcell.ColorDarkSlateGray // Normal button background
	ThemeButtonFocusBg = tcell.ColorGoldenrod     // Focused/activated button background
	ThemeInputBg       = tcell.ColorDarkSlateGray // Input field background (editable area)

	// ApplyThemeOnce guards the one-time application of global tview styles
	applyThemeOnce bool
)

// InitTheme applies the global theme defaults to tview's style system.
// Call once at startup before creating any UI components.
func InitTheme() {
	if applyThemeOnce {
		return
	}
	applyThemeOnce = true

	// Border color (default: white → gray)
	tview.Styles.BorderColor = ThemeBorderColor

	// Title color (default: white → dark cyan)
	tview.Styles.TitleColor = ThemeTitleColor

	// Graphics (default: white → gray)
	tview.Styles.GraphicsColor = ThemeBorderColor

	// Input field text area background (default: BLUE → gray)
	tview.Styles.ContrastBackgroundColor = ThemeInputBg

	// More contrasting background (default: green → dark gray)
	tview.Styles.MoreContrastBackgroundColor = ThemeButtonFocusBg

	// Text colors
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorGray
	tview.Styles.TertiaryTextColor = tcell.ColorDarkGray

	// Inverse text (text on primary-colored backgrounds)
	tview.Styles.InverseTextColor = tcell.ColorBlack

	// Text on ContrastBackgroundColor-colored backgrounds (default: navy)
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorGray
}

// StripFlagEmoji removes Regional Indicator characters (U+1F1E6-U+1F1FF)
// from display strings. These characters form flag emojis (e.g. 🇭🇰) but
// their terminal rendering width often mismatches the TUI library's width
// calculation, causing screen ghosting artifacts when content changes.
func StripFlagEmoji(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 0x1F1E6 && r <= 0x1F1FF {
			continue
		}
		result = append(result, r)
	}
	return string(result)
}

// Theme tag constants (tview tag strings for inline text coloring)
const (
	ThemeTagError        = "[red]"
	ThemeTagSuccess      = "[green]"
	ThemeTagInfo         = "[yellow]"
	ThemeTagWhite        = "[white]"
	ThemeTagGray         = "[gray]"
	ThemeTagModeRule     = "[white]"  // subtle — default mode
	ThemeTagModeGlobal   = "[yellow]" // mild attention
	ThemeTagModeDirect   = "[green]"  // calm positive
	ThemeTagConnected    = "[green]"
	ThemeTagDisconnected = "[gray]"
)
