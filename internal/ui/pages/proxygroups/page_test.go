package proxygroups

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/models"
	"mihomoTui/internal/service"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type mockGroupRepo struct {
	mu     sync.Mutex
	groups []models.ProxyGroupEntry
}

func (m *mockGroupRepo) GetGroups() ([]models.ProxyGroupEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]models.ProxyGroupEntry, len(m.groups))
	copy(res, m.groups)
	return res, nil
}
func (m *mockGroupRepo) AddGroup(name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups = append(m.groups, models.ProxyGroupEntry{Name: name, Type: groupType, Proxies: proxies, Use: use})
	return "", nil
}
func (m *mockGroupRepo) UpdateGroup(oldName, name, groupType, filter, testURL string, use, proxies []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, g := range m.groups {
		if g.Name == oldName {
			m.groups[i] = models.ProxyGroupEntry{Name: name, Type: groupType, Proxies: proxies, Use: use, URL: testURL, Filter: filter}
			break
		}
	}
	return "", nil
}
func (m *mockGroupRepo) DeleteGroup(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, g := range m.groups {
		if g.Name == name {
			m.groups = append(m.groups[:i], m.groups[i+1:]...)
			break
		}
	}
	return "", nil
}
func (m *mockGroupRepo) GetGroupNames() ([]string, error) {
	return nil, nil
}
func (m *mockGroupRepo) GetGroupDef(groupName string) ([]string, []string, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// TestNewPage_NonNil
// ---------------------------------------------------------------------------

func TestNewPage_NonNil(t *testing.T) {
	p := NewPage()
	if p == nil {
		t.Fatal("NewPage() returned nil")
	}

	// Verify Flex layout exists (embedded via Page)
	if p.Flex == nil {
		t.Fatal("Flex layout is nil after NewPage()")
	}

	// Verify all layout sub-components are non-nil
	if p.groupList == nil {
		t.Error("groupList should not be nil")
	}
	if p.detailView == nil {
		t.Error("detailView should not be nil")
	}
	if p.actionBar == nil {
		t.Error("actionBar should not be nil")
	}
	if p.status == nil {
		t.Error("status should not be nil")
	}
	if p.nameInput == nil {
		t.Error("nameInput should not be nil")
	}
	if p.typeSelect == nil {
		t.Error("typeSelect should not be nil")
	}
	if p.proxiesInput == nil {
		t.Error("proxiesInput should not be nil")
	}
	if p.useInput == nil {
		t.Error("useInput should not be nil")
	}
	if p.urlInput == nil {
		t.Error("urlInput should not be nil")
	}
	if p.filterInput == nil {
		t.Error("filterInput should not be nil")
	}
	if p.groupForm == nil {
		t.Error("groupForm should not be nil")
	}

	// Verify title is set (from setupUI)
	if title := p.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

func TestPage_ServiceInjection(t *testing.T) {
	mockRepo := &mockGroupRepo{
		groups: []models.ProxyGroupEntry{
			{Name: "Proxy", Type: "select", Proxies: []string{"node1"}},
		},
	}
	svc := service.NewProxyGroupService(mockRepo)

	p := NewPage(svc)
	if p == nil {
		t.Fatal("NewPage(svc) returned nil")
	}
	if p.groupService != svc {
		t.Errorf("expected groupService to be injected")
	}

	p2 := NewPage()
	p2.SetGroupService(svc)
	if p2.groupService != svc {
		t.Errorf("expected groupService to be set via SetGroupService")
	}

	p3 := NewPage()
	if p3.getGroupService() == nil {
		t.Errorf("expected non-nil fallback groupService")
	}
}

func TestPage_OperationsWithService(t *testing.T) {
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)
	go app.Run()
	defer app.Stop()

	mockRepo := &mockGroupRepo{
		groups: []models.ProxyGroupEntry{
			{Name: "Auto", Type: "url-test", Proxies: []string{"node1", "node2"}},
		},
	}
	svc := service.NewProxyGroupService(mockRepo)
	p := NewPage(svc)

	// Test add group
	p.saveGroup("MyGroup", "select", "", "", nil, []string{"node1"}, "")
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count := len(mockRepo.groups)
	mockRepo.mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2 groups after saveGroup, got %d", count)
	}

	// Test update group
	p.saveGroup("MyGroupRenamed", "select", "", "", nil, []string{"node1", "node2"}, "MyGroup")
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.groups)
	name := ""
	if count > 1 {
		name = mockRepo.groups[1].Name
	}
	mockRepo.mu.Unlock()
	if count != 2 || name != "MyGroupRenamed" {
		t.Errorf("expected group updated to MyGroupRenamed, got count=%d name=%s", count, name)
	}

	// Test delete group
	p.deleteGroup(models.ProxyGroupEntry{Name: "MyGroupRenamed"})
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.groups)
	firstName := ""
	if count > 0 {
		firstName = mockRepo.groups[0].Name
	}
	mockRepo.mu.Unlock()
	if count != 1 || firstName != "Auto" {
		t.Errorf("expected only Auto group remaining, got count=%d firstName=%s", count, firstName)
	}
}

// ---------------------------------------------------------------------------
// TestPage_ActivateDeactivate_GoroutineLeak
// ---------------------------------------------------------------------------

func TestPage_ActivateDeactivate_GoroutineLeak(t *testing.T) {
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

	p := NewPage()
	before := runtime.NumGoroutine()

	// Activate starts: go p.refresh() which calls config.GetAllProxyGroupsFromConfig()
	// (will fail quickly — no config) then queues a UI update via ui.Updater.UpdateUi.
	p.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate is a no-op for this page, but call it to exercise the method.
	p.Deactivate()
	time.Sleep(100 * time.Millisecond) // allow goroutines to settle

	// Stop the application event loop so Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

// ---------------------------------------------------------------------------
// TestPage_DeactivateNoPanic
// ---------------------------------------------------------------------------

func TestPage_DeactivateNoPanic(t *testing.T) {
	p := NewPage()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	p.Deactivate()
}
