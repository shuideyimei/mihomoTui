package ruleproviders

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

type mockRuleProviderRepo struct {
	mu            sync.Mutex
	ruleProviders []models.RuleProviderEntry
}

func (m *mockRuleProviderRepo) GetProxyProviders() ([]models.ProviderInfo, error) {
	return nil, nil
}
func (m *mockRuleProviderRepo) AddProxyProvider(name, url string) (string, error) {
	return "", nil
}
func (m *mockRuleProviderRepo) AddLocalProxyProvider(name, localPath string) (string, error) {
	return "", nil
}
func (m *mockRuleProviderRepo) RemoveProxyProvider(name string) error {
	return nil
}
func (m *mockRuleProviderRepo) GetRuleProviders() ([]models.RuleProviderEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]models.RuleProviderEntry, len(m.ruleProviders))
	copy(res, m.ruleProviders)
	return res, nil
}
func (m *mockRuleProviderRepo) AddRuleProvider(name, url, behavior string, interval int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ruleProviders = append(m.ruleProviders, models.RuleProviderEntry{Name: name, URL: url, Behavior: behavior, Interval: interval})
	return "", nil
}
func (m *mockRuleProviderRepo) UpdateRuleProvider(name, url, behavior string, interval int) (string, error) {
	return "", nil
}
func (m *mockRuleProviderRepo) DeleteRuleProvider(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, rp := range m.ruleProviders {
		if rp.Name == name {
			m.ruleProviders = append(m.ruleProviders[:i], m.ruleProviders[i+1:]...)
			break
		}
	}
	return "", nil
}
func (m *mockRuleProviderRepo) BatchAddRuleProviders(providers []models.RuleProviderEntry) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ruleProviders = append(m.ruleProviders, providers...)
	return "", nil
}

// initRuleProvidersTestOnce ensures test setup runs only once.
var initRuleProvidersTestOnce sync.Once

func initRuleProvidersTest() {
	initRuleProvidersTestOnce.Do(func() {
		// Init API client so api.Client is non-nil (used by refresh via Activate).
		api.InitClient("http://127.0.0.1:1", "")

		// Init UI updater so showError / showStatus calls don't panic.
		// tview.NewApplication() is safe in tests — it only allocates internal queues.
		ui.InitUpdater(tview.NewApplication())
	})
}

// ---------------------------------------------------------------------------
// TestNewPage_NonNil
// ---------------------------------------------------------------------------

func TestNewPage_NonNil(t *testing.T) {
	initRuleProvidersTest()

	page := NewPage()
	if page == nil {
		t.Fatal("NewPage() returned nil")
	}
}

func TestPage_ServiceInjection(t *testing.T) {
	initRuleProvidersTest()

	mockRepo := &mockRuleProviderRepo{
		ruleProviders: []models.RuleProviderEntry{
			{Name: "reject", Behavior: "domain", URL: "https://example.com/reject.txt"},
		},
	}
	svc := service.NewProviderService(mockRepo, nil)

	p := NewPage(svc)
	if p == nil {
		t.Fatal("NewPage(svc) returned nil")
	}
	if p.providerService != svc {
		t.Errorf("expected providerService to be injected")
	}

	p2 := NewPage()
	p2.SetProviderService(svc)
	if p2.providerService != svc {
		t.Errorf("expected providerService to be set via SetProviderService")
	}

	p3 := NewPage()
	if p3.getProviderService() == nil {
		t.Errorf("expected non-nil fallback providerService")
	}
}

func TestPage_OperationsWithService(t *testing.T) {
	initRuleProvidersTest()

	app := tview.NewApplication()
	simScreen := tcell.NewSimulationScreen("")
	if err := simScreen.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	app.SetScreen(simScreen)
	ui.InitUpdater(app)
	go app.Run()
	defer app.Stop()

	mockRepo := &mockRuleProviderRepo{
		ruleProviders: []models.RuleProviderEntry{
			{Name: "direct", Behavior: "domain", URL: "https://example.com/direct.txt"},
		},
	}
	svc := service.NewProviderService(mockRepo, nil)
	p := NewPage(svc)

	// Test importFromURL
	p.importFromURL("custom-rules", "https://example.com/custom.txt", "classical", 3600)
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count := len(mockRepo.ruleProviders)
	mockRepo.mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2 rule providers after importFromURL, got %d", count)
	}

	// Test deleteProvider
	p.deleteProvider(models.RuleProviderEntry{Name: "direct"})
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.ruleProviders)
	firstName := ""
	if count > 0 {
		firstName = mockRepo.ruleProviders[0].Name
	}
	mockRepo.mu.Unlock()
	if count != 1 || firstName != "custom-rules" {
		t.Errorf("expected custom-rules as remaining provider, got count=%d firstName=%s", count, firstName)
	}

	// Test doProcessPasteImportYAML
	yamlText := `
provider1:
  type: http
  behavior: domain
  url: https://example.com/p1.yaml
  interval: 3600
`
	p.doProcessPasteImportYAML(yamlText)
	time.Sleep(50 * time.Millisecond)
	mockRepo.mu.Lock()
	count = len(mockRepo.ruleProviders)
	mockRepo.mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2 rule providers after paste import, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// TestPage_ActivateDeactivate_GoroutineLeak
// ---------------------------------------------------------------------------

func TestPage_ActivateDeactivate_GoroutineLeak(t *testing.T) {
	initRuleProvidersTest()

	page := NewPage()
	baseline := runtime.NumGoroutine()

	// Activate spawns a goroutine via go p.refresh().  refresh() will try to
	// read the mihomo config (which does not exist in this test environment)
	// and fail, calling showError which queues a UI update in a goroutine.
	// Both goroutines should terminate quickly.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Activate panicked: %v", r)
			}
		}()
		page.Activate()
	}()

	// Give spawned goroutines a moment to start and finish.
	time.Sleep(50 * time.Millisecond)

	// Deactivate is a no-op but should not panic.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Deactivate panicked: %v", r)
			}
		}()
		page.Deactivate()
	}()

	// Wait for all goroutines to settle back to baseline.
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			got := runtime.NumGoroutine()
			t.Fatalf(
				"goroutine count did not return to baseline within deadline: initial=%d, current=%d",
				baseline, got,
			)
		case <-tick.C:
			if runtime.NumGoroutine() <= baseline+5 {
				return
			}
		}
	}
}
