package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ResponsiveSplit shows two panels side-by-side on wide terminals and stacks
// them vertically when the available width falls below the breakpoint.
type ResponsiveSplit struct {
	*tview.Flex
	first, second         tview.Primitive
	breakpoint            int
	firstProp, secondProp int
	stacked               bool
	initialized           bool
}

func NewResponsiveSplit(first, second tview.Primitive, breakpoint, firstProp, secondProp int) *ResponsiveSplit {
	split := &ResponsiveSplit{
		Flex:       tview.NewFlex(),
		first:      first,
		second:     second,
		breakpoint: breakpoint,
		firstProp:  firstProp,
		secondProp: secondProp,
	}
	split.rebuild(false)
	return split
}

func (s *ResponsiveSplit) rebuild(stacked bool) {
	s.Clear()
	if stacked {
		s.SetDirection(tview.FlexRow)
	} else {
		s.SetDirection(tview.FlexColumn)
	}
	s.AddItem(s.first, 0, s.firstProp, true)
	s.AddItem(s.second, 0, s.secondProp, false)
	s.stacked = stacked
	s.initialized = true
}

func (s *ResponsiveSplit) Draw(screen tcell.Screen) {
	_, _, width, _ := s.GetRect()
	stacked := width > 0 && width < s.breakpoint
	if !s.initialized || stacked != s.stacked {
		s.rebuild(stacked)
	}
	s.Flex.Draw(screen)
}

func (s *ResponsiveSplit) IsStacked() bool {
	return s.stacked
}
