// Package testapp provides helpers for testing tview-based TUI applications.
package testapp

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewSimScreen creates a new tcell.SimulationScreen with the given dimensions.
func NewSimScreen(width, height int) tcell.SimulationScreen {
	s := tcell.NewSimulationScreen("UTF-8")
	s.SetSize(width, height)
	return s
}

// NewTestApp creates a tview.Application backed by a SimulationScreen for testing.
func NewTestApp(width, height int) (*tview.Application, tcell.SimulationScreen) {
	app := tview.NewApplication()
	screen := NewSimScreen(width, height)
	app.SetScreen(screen)
	return app, screen
}

// MustInit initialises the simulation screen, calling t.Fatal on error.
func MustInit(t testing.TB, screen tcell.SimulationScreen) {
	t.Helper()
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
}

// GetScreenText extracts the rendered text content from a SimulationScreen.
// It calls Show() to flush the back buffer, then reads the cell contents
// row by row, converting runes to a string.
func GetScreenText(screen tcell.SimulationScreen) string {
	screen.Show()
	cells, w, h := screen.GetContents()
	var buf strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := cells[y*w+x].Runes
			if len(r) > 0 {
				buf.WriteRune(r[0])
			} else {
				buf.WriteRune(' ')
			}
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}
