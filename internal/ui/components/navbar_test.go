package components

import (
	"strings"
	"testing"

	"mihomoTui/internal/ui"
)

func TestNavBarKeepsActivePageMarker(t *testing.T) {
	nav := NewNavBar(ui.DefaultPageInfo())
	nav.SelectItem(3)

	index, _ := nav.GetCurrentPage()
	if index != 3 {
		t.Fatalf("expected active page 3, got %d", index)
	}
	main, _ := nav.GetItemText(3)
	if !strings.Contains(main, "●") {
		t.Fatalf("active item should retain marker, got %q", main)
	}
}
