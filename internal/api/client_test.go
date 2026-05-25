package api

import (
	"encoding/json"
	"testing"

	"mihomoTui/internal/models"
	"mihomoTui/internal/testapi"
)

// assertRequest verifies that a recorded request with the given method and path exists.
func assertRequest(t *testing.T, reqs []testapi.Request, method, path string) {
	t.Helper()
	for _, r := range reqs {
		if r.Method == method && r.Path == path {
			return
		}
	}
	t.Errorf("expected request %s %s not found among %d requests", method, path, len(reqs))
}

// findRequest returns the first request matching method+path, or nil.
func findRequest(reqs []testapi.Request, method, path string) *testapi.Request {
	for _, r := range reqs {
		if r.Method == method && r.Path == path {
			return &r
		}
	}
	return nil
}

// ============================================================================
// GetVersion
// ============================================================================

func TestGetVersion_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	ver, err := Client.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if !ver.Meta {
		t.Error("expected Meta=true")
	}
	if ver.Version != "v1.19.25-mock" {
		t.Errorf("expected v1.19.25-mock, got %s", ver.Version)
	}
}

func TestGetVersion_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetVersion()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetConfig
// ============================================================================

func TestGetConfig_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	cfg, err := Client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg.Port != 7890 {
		t.Errorf("expected port=7890, got %d", cfg.Port)
	}
	if cfg.SocksPort != 7891 {
		t.Errorf("expected socks-port=7891, got %d", cfg.SocksPort)
	}
	if cfg.Mode != "rule" {
		t.Errorf("expected mode=rule, got %s", cfg.Mode)
	}
	if cfg.AllowLan != false {
		t.Errorf("expected allow-lan=false, got %v", cfg.AllowLan)
	}
	tun, ok := cfg.Tun["enable"]
	if !ok {
		t.Fatal("expected tun.enable to exist")
	}
	if tun != true {
		t.Errorf("expected tun.enable=true, got %v", tun)
	}
}

func TestGetConfig_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// UpdateConfig
// ============================================================================

func TestUpdateConfig_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	cfg := &models.Config{Mode: "global", Port: 7890}
	err := Client.UpdateConfig(cfg)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PATCH", "/configs")

	// Verify the request body contains the config fields we sent
	req := findRequest(reqs, "PATCH", "/configs")
	if req == nil {
		t.Fatal("PATCH /configs request not recorded")
	}
	var sent models.Config
	if err := json.Unmarshal(req.Body, &sent); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if sent.Mode != "global" {
		t.Errorf("expected mode=global in request body, got %s", sent.Mode)
	}
}

func TestUpdateConfig_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.UpdateConfig(&models.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetProxies
// ============================================================================

func TestGetProxies_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	proxies, err := Client.GetProxies()
	if err != nil {
		t.Fatalf("GetProxies failed: %v", err)
	}
	if len(proxies) == 0 {
		t.Fatal("expected non-empty proxies map")
	}

	proxy, ok := proxies["Proxy"]
	if !ok {
		t.Fatal("expected 'Proxy' group to exist")
	}
	if proxy.Type != "Selector" {
		t.Errorf("expected Proxy type=Selector, got %s", proxy.Type)
	}
	if proxy.Now != "HK" {
		t.Errorf("expected Proxy now=HK, got %s", proxy.Now)
	}

	hk, ok := proxies["HK"]
	if !ok {
		t.Fatal("expected 'HK' proxy to exist")
	}
	if hk.Type != "ss" {
		t.Errorf("expected HK type=ss, got %s", hk.Type)
	}
	if !hk.UDP {
		t.Error("expected HK UDP=true")
	}
}

func TestGetProxies_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetProxies()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetProviders
// ============================================================================

func TestGetProviders_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	providers, err := Client.GetProviders()
	if err != nil {
		t.Fatalf("GetProviders failed: %v", err)
	}
	if len(providers.Providers) == 0 {
		t.Fatal("expected non-empty providers map")
	}

	def, ok := providers.Providers["default"]
	if !ok {
		t.Fatal("expected 'default' provider")
	}
	if def.VehicleType != "HTTP" {
		t.Errorf("expected default vehicleType=HTTP, got %s", def.VehicleType)
	}
	if len(def.Proxies) != 5 {
		t.Errorf("expected 5 proxies in default provider, got %d", len(def.Proxies))
	}

	custom, ok := providers.Providers["custom"]
	if !ok {
		t.Fatal("expected 'custom' provider")
	}
	if custom.VehicleType != "File" {
		t.Errorf("expected custom vehicleType=File, got %s", custom.VehicleType)
	}
}

func TestGetProviders_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetProviders()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// TestGroupDelay
// ============================================================================

func TestTestGroupDelay_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	delays, err := Client.TestGroupDelay("Proxy", "http://www.gstatic.com/generate_204", 5000)
	if err != nil {
		t.Fatalf("TestGroupDelay failed: %v", err)
	}
	if delays["Proxy"] != 100 {
		t.Errorf("expected Proxy delay=100, got %d", delays["Proxy"])
	}
	if delays["HK"] != 50 {
		t.Errorf("expected HK delay=50, got %d", delays["HK"])
	}
	if delays["JP"] != 150 {
		t.Errorf("expected JP delay=150, got %d", delays["JP"])
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "GET", "/group/Proxy/delay")
}

func TestTestGroupDelay_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.TestGroupDelay("Proxy", "http://test.com", 5000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetProxy
// ============================================================================

func TestGetProxy_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	proxy, err := Client.GetProxy("HK")
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}
	if proxy.Name != "HK" {
		t.Errorf("expected name=HK, got %s", proxy.Name)
	}
	if proxy.Type != "ss" {
		t.Errorf("expected type=ss, got %s", proxy.Type)
	}
	if !proxy.UDP {
		t.Error("expected UDP=true")
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "GET", "/proxies/HK")
}

func TestGetProxy_NotFound(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetProxy("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent proxy, got nil")
	}
}

func TestGetProxy_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetProxy("HK")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// SelectProxy
// ============================================================================

func TestSelectProxy_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.SelectProxy("Proxy", "HK")
	if err != nil {
		t.Fatalf("SelectProxy failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PUT", "/proxies/Proxy")

	req := findRequest(reqs, "PUT", "/proxies/Proxy")
	if req == nil {
		t.Fatal("PUT /proxies/Proxy not recorded")
	}
	var body map[string]string
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to unmarshal PUT body: %v", err)
	}
	if body["name"] != "HK" {
		t.Errorf("expected body.name=HK, got %s", body["name"])
	}
}

func TestSelectProxy_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.SelectProxy("Proxy", "HK")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// TestProxyDelay
// ============================================================================

func TestTestProxyDelay_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	delay, err := Client.TestProxyDelay("HK", "http://www.gstatic.com/generate_204", 5000)
	if err != nil {
		t.Fatalf("TestProxyDelay failed: %v", err)
	}
	if delay != 50 {
		t.Errorf("expected delay=50, got %d", delay)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "GET", "/proxies/HK/delay")
}

func TestTestProxyDelay_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.TestProxyDelay("HK", "http://test.com", 5000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetRules
// ============================================================================

func TestGetRules_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	rules, err := Client.GetRules()
	if err != nil {
		t.Fatalf("GetRules failed: %v", err)
	}
	if len(rules) != 5 {
		t.Errorf("expected 5 rules, got %d", len(rules))
	}
	if rules[0].Type != "DOMAIN-SUFFIX" || rules[0].Payload != "google.com" || rules[0].Proxy != "Proxy" {
		t.Errorf("unexpected first rule: %+v", rules[0])
	}
	if rules[4].Type != "MATCH" || rules[4].Proxy != "Proxy" {
		t.Errorf("unexpected last rule: %+v", rules[4])
	}
}

func TestGetRules_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetRules()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetRuleProviders
// ============================================================================

func TestGetRuleProviders_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	providers, err := Client.GetRuleProviders()
	if err != nil {
		t.Fatalf("GetRuleProviders failed: %v", err)
	}
	if len(providers.Providers) == 0 {
		t.Fatal("expected non-empty rule providers map")
	}

	reject, ok := providers.Providers["reject"]
	if !ok {
		t.Fatal("expected 'reject' rule provider")
	}
	if reject.Behavior != "domain" {
		t.Errorf("expected reject behavior=domain, got %s", reject.Behavior)
	}
	if reject.RuleCount != 1500 {
		t.Errorf("expected reject ruleCount=1500, got %d", reject.RuleCount)
	}

	apple, ok := providers.Providers["apple"]
	if !ok {
		t.Fatal("expected 'apple' rule provider")
	}
	if apple.RuleCount != 500 {
		t.Errorf("expected apple ruleCount=500, got %d", apple.RuleCount)
	}
}

func TestGetRuleProviders_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetRuleProviders()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// RefreshRuleProvider
// ============================================================================

func TestRefreshRuleProvider_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.RefreshRuleProvider("reject")
	if err != nil {
		t.Fatalf("RefreshRuleProvider failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PUT", "/providers/rules/reject")
}

func TestRefreshRuleProvider_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.RefreshRuleProvider("reject")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// GetConnections
// ============================================================================

func TestGetConnections_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	conns, err := Client.GetConnections()
	if err != nil {
		t.Fatalf("GetConnections failed: %v", err)
	}
	if len(conns) != 3 {
		t.Errorf("expected 3 connections, got %d", len(conns))
	}
	if conns[0].ID != "conn-001" {
		t.Errorf("expected first connection ID=conn-001, got %s", conns[0].ID)
	}
	if conns[0].Metadata.Host != "www.google.com" {
		t.Errorf("expected host=www.google.com, got %s", conns[0].Metadata.Host)
	}
	if conns[0].Upload != 1024 {
		t.Errorf("expected upload=1024, got %d", conns[0].Upload)
	}
	if conns[2].Metadata.Network != "udp" {
		t.Errorf("expected conn-003 network=udp, got %s", conns[2].Metadata.Network)
	}
}

func TestGetConnections_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	_, err := Client.GetConnections()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// CloseConnection
// ============================================================================

func TestCloseConnection_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.CloseConnection("conn-001")
	if err != nil {
		t.Fatalf("CloseConnection failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "DELETE", "/connections/conn-001")
}

func TestCloseConnection_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.CloseConnection("conn-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// ReloadConfig
// ============================================================================

func TestReloadConfig_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.ReloadConfig("/etc/mihomo/config.yaml")
	if err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PUT", "/configs")

	req := findRequest(reqs, "PUT", "/configs")
	if req == nil {
		t.Fatal("PUT /configs request not recorded")
	}
	var body map[string]string
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to unmarshal PUT body: %v", err)
	}
	if body["path"] != "/etc/mihomo/config.yaml" {
		t.Errorf("expected body.path=/etc/mihomo/config.yaml, got %s", body["path"])
	}
}

func TestReloadConfig_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.ReloadConfig("/etc/mihomo/config.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// ReloadConfigData
// ============================================================================

func TestReloadConfigData_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	data := []byte("port: 7890\nmode: rule\n")
	err := Client.ReloadConfigData(data)
	if err != nil {
		t.Fatalf("ReloadConfigData failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PUT", "/configs")

	req := findRequest(reqs, "PUT", "/configs")
	if req == nil {
		t.Fatal("PUT /configs request not recorded")
	}
	if string(req.Body) != "port: 7890\nmode: rule\n" {
		t.Errorf("unexpected request body: %s", string(req.Body))
	}
}

func TestReloadConfigData_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.ReloadConfigData([]byte("port: 7890"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// PatchConfig
// ============================================================================

func TestPatchConfig_Success(t *testing.T) {
	srv, rec := testapi.NewMockMihomoServerWithRecorder()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	partial := map[string]interface{}{
		"mode": "direct",
	}
	err := Client.PatchConfig(partial)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	reqs := rec.Requests()
	assertRequest(t, reqs, "PATCH", "/configs")

	req := findRequest(reqs, "PATCH", "/configs")
	if req == nil {
		t.Fatal("PATCH /configs request not recorded")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to unmarshal PATCH body: %v", err)
	}
	if body["mode"] != "direct" {
		t.Errorf("expected body.mode=direct, got %v", body["mode"])
	}
}

func TestPatchConfig_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.PatchConfig(map[string]interface{}{"mode": "direct"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ============================================================================
// HealthCheck
// ============================================================================

func TestHealthCheck_Success(t *testing.T) {
	srv := testapi.NewMockMihomoServer()
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestHealthCheck_Error(t *testing.T) {
	srv := testapi.NewMockMihomoServerWithError(500, "internal error")
	defer srv.Close()

	InitClient(srv.URL, "test-secret")

	err := Client.HealthCheck()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
