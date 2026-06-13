package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestResponsiveSplitChangesLayoutAtBreakpoint(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()

	split := NewResponsiveSplit(tview.NewBox(), tview.NewBox(), 90, 2, 3)
	split.SetRect(0, 0, 120, 30)
	split.Draw(screen)
	if split.IsStacked() {
		t.Fatal("wide layout should not be stacked")
	}

	split.SetRect(0, 0, 70, 30)
	split.Draw(screen)
	if !split.IsStacked() {
		t.Fatal("narrow layout should be stacked")
	}
}
