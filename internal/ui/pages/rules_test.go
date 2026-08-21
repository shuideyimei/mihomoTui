package pages

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/service"
	"mihomoTui/internal/ui"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Compile-time check: verify RulesPage implements ActivatablePage.
var _ ActivatablePage = (*RulesPage)(nil)

type mockRuleRepo struct {
	mu    sync.Mutex
	rules []models.RuleDisplay
}

func (m *mockRuleRepo) GetRules() ([]models.RuleDisplay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]models.RuleDisplay, len(m.rules))
	copy(res, m.rules)
	return res, nil
}
func (m *mockRuleRepo) AddRule(ruleType, payload, proxy string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, models.RuleDisplay{Type: ruleType, Payload: payload, Proxy: proxy})
	return "", nil
}
func (m *mockRuleRepo) DeleteRule(index int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index >= 0 && index < len(m.rules) {
		m.rules = append(m.rules[:index], m.rules[index+1:]...)
	}
	return "", nil
}
func (m *mockRuleRepo) DeleteRuleByContent(ruleType, payload, proxy string) (string, error) {
	return "", nil
}
func (m *mockRuleRepo) MoveRule(fromIdx, toIdx int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fromIdx < 0 || fromIdx >= len(m.rules) || toIdx < 0 || toIdx >= len(m.rules) {
		return "", nil
	}
	r := m.rules[fromIdx]
	m.rules = append(m.rules[:fromIdx], m.rules[fromIdx+1:]...)
	m.rules = append(m.rules[:toIdx], append([]models.RuleDisplay{r}, m.rules[toIdx:]...)...)
	return "", nil
}
func (m *mockRuleRepo) UpdateRule(index int, ruleType, payload, proxy string) (string, error) {
	return "", nil
}
func (m *mockRuleRepo) BatchAddRules(rules []models.RuleDisplay) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, rules...)
	return "", nil
}

func TestNewRulesPage_NonNil(t *testing.T) {
	r := NewRulesPage()
	if r == nil {
		t.Fatal("NewRulesPage() returned nil")
	}
	// Verify embedded Flex layout exists
	if r.Flex == nil {
		t.Fatal("Flex layout is nil after NewRulesPage()")
	}

	// Verify sub-components are non-nil
	if r.table == nil {
		t.Error("table should not be nil")
	}
	if r.searchInput == nil {
		t.Error("searchInput should not be nil")
	}
	if r.statusText == nil {
		t.Error("statusText should not be nil")
	}
	if r.actionBar == nil {
		t.Error("actionBar should not be nil")
	}

	// Verify title is set (from setupUI)
	if title := r.GetTitle(); title == "" {
		t.Error("page title should not be empty")
	}
}

func TestRules_ServiceInjection(t *testing.T) {
	mockRepo := &mockRuleRepo{
		rules: []models.RuleDisplay{
			{Type: "DOMAIN-SUFFIX", Payload: "google.com", Proxy: "PROXY"},
		},
	}
	svc := service.NewRuleService(mockRepo)

	r := NewRulesPage(svc)
	if r == nil {
		t.Fatal("NewRulesPage(svc) returned nil")
	}
	if r.ruleService != svc {
		t.Errorf("expected ruleService to be injected")
	}

	// Test SetRuleService
	r2 := NewRulesPage()
	r2.SetRuleService(svc)
	if r2.ruleService != svc {
		t.Errorf("expected ruleService to be set via SetRuleService")
	}

	// Test fallback when nil
	r3 := NewRulesPage()
	if r3.getRuleService() == nil {
		t.Errorf("expected non-nil fallback ruleService")
	}
}

func TestRules_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	// Initialize API client so api.Client.GetRules doesn't panic.
	api.InitClient("http://127.0.0.1:1", "")

	// Need a running tview application so QueueUpdateDraw calls do not hang.
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)

	// Start the application event loop in the background.
	go app.Run()
	time.Sleep(10 * time.Millisecond)

	r := NewRulesPage()
	before := runtime.NumGoroutine()

	// Activate starts refresh() which launches goroutines (UpdateUi + API call).
	r.Activate()
	time.Sleep(20 * time.Millisecond)

	// Deactivate clears internal state.
	r.Deactivate()
	time.Sleep(100 * time.Millisecond)

	// Stop the application event loop so Run/poll goroutines exit.
	app.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()

	if diff := after - before; diff > 2 {
		t.Errorf("potential goroutine leak: before=%d, after=%d (diff=%d)", before, after, diff)
	}
}

func TestRules_OperationsWithService(t *testing.T) {
	api.InitClient("http://127.0.0.1:1", "")
	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)
	go app.Run()
	defer app.Stop()

	mockRepo := &mockRuleRepo{
		rules: []models.RuleDisplay{
			{Type: "DOMAIN-SUFFIX", Payload: "google.com", Proxy: "PROXY"},
			{Type: "GEOIP", Payload: "CN", Proxy: "DIRECT"},
		},
	}
	svc := service.NewRuleService(mockRepo)
	r := NewRulesPage(svc)

	// Test saveRule
	r.saveRule("DOMAIN-KEYWORD", "youtube", "PROXY")
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count := len(mockRepo.rules)
	mockRepo.mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 rules after saveRule, got %d", count)
	}

	// Test doDelete
	r.doDelete(0)
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.rules)
	firstPayload := ""
	if count > 0 {
		firstPayload = mockRepo.rules[0].Payload
	}
	mockRepo.mu.Unlock()
	if count != 2 || firstPayload != "CN" {
		t.Errorf("expected CN as first rule after delete, got count=%d payload=%s", count, firstPayload)
	}

	// Test doMoveRule
	r.doMoveRule(0, 1)
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	p0, p1 := "", ""
	if len(mockRepo.rules) >= 2 {
		p0, p1 = mockRepo.rules[0].Payload, mockRepo.rules[1].Payload
	}
	mockRepo.mu.Unlock()
	if p0 != "youtube" || p1 != "CN" {
		t.Errorf("expected rules swapped after move, got p0=%s p1=%s", p0, p1)
	}

	// Test doProcessPasteImport
	pasteText := "DOMAIN,github.com,PROXY\nMATCH,DIRECT\n"
	r.doProcessPasteImport(pasteText)
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.rules)
	mockRepo.mu.Unlock()
	if count != 4 {
		t.Errorf("expected 4 rules after paste import, got %d", count)
	}
}

func TestRules_DeactivateNoPanic(t *testing.T) {
	r := NewRulesPage()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Deactivate panicked: %v", r)
		}
	}()

	// Deactivate without calling Activate first — should not panic.
	r.Deactivate()
}
