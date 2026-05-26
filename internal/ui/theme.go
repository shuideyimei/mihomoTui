package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme color constants (tcell.Color for component styling)
// Minimalist palette: pure monochrome (no blue).
var (
	ThemeBorderColor  = tcell.ColorGray
	ThemeTitleColor   = tcell.ColorWhite        // Bright white for titles
	ThemeHighlightBg  = tcell.ColorGray         // Selection highlight background
	ThemeSelBgFocus   = tcell.ColorGray         // Focused selection
	ThemeSelBgBlur    = tcell.ColorDarkGray     // Blurred selection
	ThemeHeaderBg     = tcell.ColorDarkGray      // Header background
	ThemeStatusBarBg  = tcell.ColorDarkGray      // Status bar background
	// Component backgrounds — all gray tones
	ThemeButtonBg       = tcell.ColorGray     // Normal button background
	ThemeButtonFocusBg  = tcell.ColorDarkGray // Focused/activated button background
	ThemeInputBg        = tcell.ColorGray     // Input field background (editable area)

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

// Theme tag constants (tview tag strings for inline text coloring)
const (
	ThemeTagError        = "[red]"
	ThemeTagSuccess      = "[green]"
	ThemeTagInfo         = "[yellow]"
	ThemeTagWhite        = "[white]"
	ThemeTagGray         = "[gray]"
	ThemeTagModeRule     = "[white]"   // subtle — default mode
	ThemeTagModeGlobal   = "[yellow]"  // mild attention
	ThemeTagModeDirect   = "[green]"   // calm positive
	ThemeTagConnected    = "[green]"
	ThemeTagDisconnected = "[gray]"
)
