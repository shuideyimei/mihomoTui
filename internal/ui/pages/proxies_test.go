package pages

import (
	"runtime"
	"testing"
	"time"

	"mihomoTui/internal/models"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Compile-time check: verify ProxiesPage implements ActivatablePage.
var _ ActivatablePage = (*ProxiesPage)(nil)

func TestNewProxies_NonNil(t *testing.T) {
	p := NewProxies()
	if p == nil {
		t.Fatal("NewProxies() returned nil")
	}
	// Verify Flex layout exists (embedded via NewProxiesPage now returns *ProxiesPage)
	if p.Flex == nil {
		t.Fatal("Flex layout is nil after NewProxiesPage()")
	}

	// Verify all layout sub-components are non-nil
	if p.groupsList == nil {
		t.Error("groupsList should not be nil")
	}
	if p.switchButtons == nil {
		t.Error("switchButtons should not be nil")
	}
	if p.nodesList == nil {
		t.Error("nodesList should not be nil")
	}
	if p.statusText == nil {
		t.Error("statusText should not be nil")
	}
	if p.searchInput == nil {
		t.Error("searchInput should not be nil")
	}

	// Verify title is set (from setupLayout)
	if title := p.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

func TestProxies_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	// We need a running tview application so that QueueUpdateDraw calls
	// (which block until the event loop processes them) do not hang.
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)

	// Start the application event loop in the background so that
	// ui.Updater.UpdateUi → app.QueueUpdateDraw is processed.
	go app.Run()

	p := NewProxies()
	before := runtime.NumGoroutine()

	// Activate starts:
	//   1. loadProvidersData goroutine (calls UpdateUi → unblocked by event loop)
	//   2. startAutoRefresh goroutine (ticker + context)
	//   3. syncModeFromStatusBar goroutine (returns quickly, statBar is nil)
	p.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate stops the auto-refresh goroutine and cancels the context.
	p.Deactivate()
	time.Sleep(100 * time.Millisecond) // allow goroutines to settle

	// Stop the application event loop so the Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

func TestProxies_DeactivateNoPanic(t *testing.T) {
	p := NewProxies()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	p.Deactivate()
}

func TestProxies_UpdateGroupsListPreservesSelection(t *testing.T) {
	p := NewProxies()
	p.proxiesData = map[string]*models.Proxy{
		"Proxy": {Name: "Proxy", Type: "Selector", All: []string{"A", "B"}, Now: "A"},
		"Auto":  {Name: "Auto", Type: "URLTest", All: []string{"A", "B"}, Now: "B"},
	}
	p.selectedGroup = "Auto"

	p.updateGroupsList()

	if got := p.selectedGroup; got != "Auto" {
		t.Fatalf("selectedGroup = %q, want Auto", got)
	}
	if got := p.groupsList.GetCurrentItem(); got != 1 {
		t.Fatalf("current group index = %d, want 1", got)
	}
}

func TestProxies_UpdateGroupsListPreservesOffset(t *testing.T) {
	p := NewProxies()
	p.proxiesData = map[string]*models.Proxy{
		"Proxy": {Name: "Proxy", Type: "Selector", All: []string{"A"}, Now: "A"},
		"Auto":  {Name: "Auto", Type: "URLTest", All: []string{"A"}, Now: "A"},
		"Group": {Name: "Group", Type: "Selector", All: []string{"A"}, Now: "A"},
	}
	p.selectedGroup = "Group"
	p.groupsList.SetOffset(2, 1)

	p.updateGroupsList()

	itemOffset, horizontalOffset := p.groupsList.GetOffset()
	if itemOffset != 2 || horizontalOffset != 1 {
		t.Fatalf("group offset = (%d, %d), want (2, 1)", itemOffset, horizontalOffset)
	}
}

func TestProxies_UpdateGroupsListMovesSelectionToScrolledViewport(t *testing.T) {
	p := NewProxies()
	p.proxiesData = map[string]*models.Proxy{
		"Proxy": {Name: "Proxy", Type: "Selector", All: []string{"A"}, Now: "A"},
		"Auto":  {Name: "Auto", Type: "URLTest", All: []string{"A"}, Now: "A"},
		"Beta":  {Name: "Beta", Type: "Selector", All: []string{"A"}, Now: "A"},
	}
	p.selectedGroup = "Proxy"
	p.groupsList.SetOffset(2, 0)

	p.updateGroupsList()

	if got := p.groupsList.GetCurrentItem(); got != 2 {
		t.Fatalf("current group index = %d, want first visible index 2", got)
	}
	if got := p.selectedGroup; got != "Beta" {
		t.Fatalf("selectedGroup = %q, want Beta", got)
	}
}

func TestProxies_UpdateNodesListContentPreservesSelection(t *testing.T) {
	p := NewProxies()
	p.proxyGroups = []*models.Proxy{
		{Name: "Proxy", Type: "Selector", All: []string{"A", "B", "C"}, Now: "A"},
	}
	p.proxiesData = map[string]*models.Proxy{
		"A": {Name: "A", Type: "ss"},
		"B": {Name: "B", Type: "ss"},
		"C": {Name: "C", Type: "ss"},
	}
	p.selectedGroup = "Proxy"
	p.selectedNode = "C"

	p.updateNodesListContent()

	row, _ := p.nodesList.GetSelection()
	if row != 3 {
		t.Fatalf("selected node row = %d, want 3", row)
	}
	if got := p.selectedNode; got != "C" {
		t.Fatalf("selectedNode = %q, want C", got)
	}
}

func TestProxies_UpdateNodesListContentPreservesOffset(t *testing.T) {
	p := NewProxies()
	p.proxyGroups = []*models.Proxy{
		{Name: "Proxy", Type: "Selector", All: []string{"A", "B", "C", "D"}, Now: "A"},
	}
	p.proxiesData = map[string]*models.Proxy{
		"A": {Name: "A", Type: "ss"},
		"B": {Name: "B", Type: "ss"},
		"C": {Name: "C", Type: "ss"},
		"D": {Name: "D", Type: "ss"},
	}
	p.selectedGroup = "Proxy"
	p.selectedNode = "C"
	p.nodesList.SetOffset(2, 1)

	p.updateNodesListContent()

	rowOffset, columnOffset := p.nodesList.GetOffset()
	if rowOffset != 2 || columnOffset != 1 {
		t.Fatalf("node offset = (%d, %d), want (2, 1)", rowOffset, columnOffset)
	}
}

func TestProxies_UpdateNodesListContentMovesSelectionToScrolledViewport(t *testing.T) {
	p := NewProxies()
	p.proxyGroups = []*models.Proxy{
		{Name: "Proxy", Type: "Selector", All: []string{"A", "B", "C", "D"}, Now: "A"},
	}
	p.proxiesData = map[string]*models.Proxy{
		"A": {Name: "A", Type: "ss"},
		"B": {Name: "B", Type: "ss"},
		"C": {Name: "C", Type: "ss"},
		"D": {Name: "D", Type: "ss"},
	}
	p.selectedGroup = "Proxy"
	p.selectedNode = "A"
	p.nodesList.Select(1, 0)
	p.nodesList.SetOffset(2, 0)

	p.updateNodesListContent()

	row, _ := p.nodesList.GetSelection()
	if row != 3 {
		t.Fatalf("selected node row = %d, want first visible row 3", row)
	}
	if got := p.selectedNode; got != "C" {
		t.Fatalf("selectedNode = %q, want C", got)
	}
	rowOffset, columnOffset := p.nodesList.GetOffset()
	if rowOffset != 2 || columnOffset != 0 {
		t.Fatalf("node offset = (%d, %d), want (2, 0)", rowOffset, columnOffset)
	}
}
