package testapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"mihomoTui/internal/models"
)

// Request records an HTTP request for test assertions.
type Request struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   []byte `json:"body,omitempty"`
}

// Recorder records all requests processed by the mock server.
type Recorder struct {
	mu       sync.Mutex
	requests []Request
}

// record captures a request.
func (r *Recorder) record(method, path string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, Request{Method: method, Path: path, Body: body})
}

// Requests returns a copy of all recorded requests.
func (r *Recorder) Requests() []Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Request, len(r.requests))
	copy(out, r.requests)
	return out
}

// NewMockMihomoServer creates a new httptest.Server with a mock mihomo REST API.
func NewMockMihomoServer() *httptest.Server {
	return NewMockMihomoServerWithRecorderInternal(nil, 0, "")
}

// NewMockMihomoServerWithRecorder creates a mock server and a Recorder for test assertions.
func NewMockMihomoServerWithRecorder() (*httptest.Server, *Recorder) {
	rec := &Recorder{}
	srv := NewMockMihomoServerWithRecorderInternal(rec, 0, "")
	return srv, rec
}

// NewMockMihomoServerWithError creates a mock server that returns the given status
// code and message for ALL endpoints, useful for testing error handling.
func NewMockMihomoServerWithError(statusCode int, message string) *httptest.Server {
	return NewMockMihomoServerWithRecorderInternal(nil, statusCode, message)
}

// NewMockMihomoServerWithRecorderInternal is the shared constructor.
func NewMockMihomoServerWithRecorderInternal(rec *Recorder, errorStatus int, errorMsg string) *httptest.Server {
	isErrorMode := errorStatus != 0

	mux := http.NewServeMux()

	// Shared handler wrapper: records requests and optionally returns error mode.
	handler := func(w http.ResponseWriter, r *http.Request, fn func(w http.ResponseWriter, r *http.Request)) {
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
		if rec != nil {
			rec.record(r.Method, r.URL.Path, bodyBytes)
		}

		if isErrorMode {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(errorStatus)
			json.NewEncoder(w).Encode(models.APIResponse{Error: errorMsg})
			return
		}

		fn(w, r)
	}

	// 1. GET /version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models.Version{Meta: true, Version: "v1.19.25-mock"})
		})
	})

	// 2. GET /configs
	mux.HandleFunc("/configs", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(models.Config{
					Port:               7890,
					SocksPort:          7891,
					MixedPort:          7890,
					AllowLan:           false,
					BindAddress:        "*",
					Mode:               "rule",
					LogLevel:           "info",
					ExternalController: "127.0.0.1:9090",
					Secret:             "",
					Tun: map[string]interface{}{
						"enable": true,
					},
				})
			case http.MethodPatch:
				w.WriteHeader(http.StatusNoContent)
			case http.MethodPut:
				contentType := r.Header.Get("Content-Type")
				if contentType == "application/json" {
					// Try to parse as JSON body; if it has a "path" field, accept it.
					bodyBytes, err := io.ReadAll(r.Body)
					if err != nil || (len(bodyBytes) == 0) {
						// Reject empty/trivial body
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(models.APIResponse{Error: "Body invalid"})
						return
					}
					// Explicit invalid body trigger: use query param ?invalid=1
					if r.URL.Query().Get("invalid") == "1" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(models.APIResponse{Error: "Body invalid"})
						return
					}
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		})
	})

	// 5. GET /proxies — returns map with proxy and proxy-group entries
	mux.HandleFunc("/proxies", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			// Handle /proxies/{name}/delay and /proxies/{groupName} PUT
			path := strings.TrimPrefix(r.URL.Path, "/proxies")
			if path != "" && path != "/" {
				// Delegate to the catch-all handler below (it's registered separately)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			result := struct {
				Proxies map[string]*models.Proxy `json:"proxies"`
			}{
				Proxies: mockProxies(),
			}
			json.NewEncoder(w).Encode(result)
		})
	})

	// 12. GET /proxies/{name}/delay  and  13. PUT /proxies/{groupName}
	mux.HandleFunc("/proxies/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/proxies/")
			if strings.HasSuffix(path, "/delay") {
				// GET /proxies/{name}/delay → {"delay": 50}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(struct {
					Delay int `json:"delay"`
				}{Delay: 50})
				return
			}
			// PUT /proxies/{groupName} → 204 No Content
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// Proxy detail by name
			proxyName := strings.TrimPrefix(r.URL.Path, "/proxies/")
			proxies := mockProxies()
			proxy, ok := proxies[proxyName]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(proxy)
		})
	})

	// 6. GET /providers/proxies
	mux.HandleFunc("/providers/proxies", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			response := models.ProvidersResponse{
				Providers: mockProxyProviders(),
			}
			json.NewEncoder(w).Encode(response)
		})
	})

	// 7. GET /providers/rules
	mux.HandleFunc("/providers/rules", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			response := models.RuleProvidersResponse{
				Providers: mockRuleProviders(),
			}
			json.NewEncoder(w).Encode(response)
		})
	})

	// Refresh rule provider: PUT /providers/rules/{name} → 204
	mux.HandleFunc("/providers/rules/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// 8. GET /rules
	mux.HandleFunc("/rules", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			result := struct {
				Rules []models.Rule `json:"rules"`
			}{
				Rules: mockRules(),
			}
			json.NewEncoder(w).Encode(result)
		})
	})

	// 9. GET /connections  10. DELETE /connections/{id}
	mux.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			result := struct {
				Connections     []models.Connection `json:"connections"`
				ConnectionCount int                 `json:"connectionCount"`
			}{
				Connections:     mockConnections(),
				ConnectionCount: 3,
			}
			json.NewEncoder(w).Encode(result)
		})
	})

	mux.HandleFunc("/connections/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// 11. GET /group/{name}/delay
	mux.HandleFunc("/group/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/group/")
			_ = path // group name is in path; we return a static set
			w.Header().Set("Content-Type", "application/json")
			delays := map[string]int{
				"Proxy": 100,
				"HK":    50,
				"JP":    150,
			}
			json.NewEncoder(w).Encode(delays)
		})
	})

	// 14. GET / — health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "hello")
		})
	})

	return httptest.NewServer(mux)
}

// mockProxies returns a realistic proxy/group map.
func mockProxies() map[string]*models.Proxy {
	return map[string]*models.Proxy{
		"Proxy": {
			Name: "Proxy",
			Type: "Selector",
			Now:  "HK",
			All:  []string{"HK", "JP", "SG", "TW", "US", "Auto"},
		},
		"GLOBAL": {
			Name: "GLOBAL",
			Type: "Selector",
			Now:  "Proxy",
			All:  []string{"Proxy", "DIRECT", "REJECT"},
		},
		"Auto": {
			Name: "Auto",
			Type: "URLTest",
			Now:  "HK",
			All:  []string{"HK", "JP", "SG", "TW", "US"},
		},
		"HK": {
			Name: "HK",
			Type: "ss",
			UDP:  true,
		},
		"JP": {
			Name: "JP",
			Type: "vmess",
			UDP:  true,
		},
		"SG": {
			Name: "SG",
			Type: "trojan",
			UDP:  true,
		},
		"TW": {
			Name: "TW",
			Type: "ss",
			UDP:  true,
		},
		"US": {
			Name: "US",
			Type: "vmess",
			UDP:  false,
		},
	}
}

// mockProxyProviders returns 2 mock proxy providers.
func mockProxyProviders() map[string]*models.ProxyProvider {
	return map[string]*models.ProxyProvider{
		"default": {
			Name:        "default",
			Type:        "Proxy",
			VehicleType: "HTTP",
			Proxies: []*models.Proxy{
				{Name: "HK", Type: "ss", UDP: true},
				{Name: "JP", Type: "vmess", UDP: true},
				{Name: "SG", Type: "trojan", UDP: true},
				{Name: "US", Type: "vmess", UDP: false},
				{Name: "TW", Type: "ss", UDP: true},
			},
			TestUrl:        "http://www.gstatic.com/generate_204",
			ExpectedStatus: "204",
			UpdatedAt:      "2025-01-15T10:30:00Z",
		},
		"custom": {
			Name:        "custom",
			Type:        "Proxy",
			VehicleType: "File",
			Proxies: []*models.Proxy{
				{Name: "Auto", Type: "URLTest", Now: "HK", All: []string{"HK", "JP"}},
				{Name: "Proxy", Type: "Selector", Now: "Auto", All: []string{"Auto", "HK", "JP", "DIRECT"}},
			},
			TestUrl:        "http://www.gstatic.com/generate_204",
			ExpectedStatus: "204",
			UpdatedAt:      "2025-01-15T09:00:00Z",
		},
	}
}

// mockRuleProviders returns 3 mock rule providers.
func mockRuleProviders() map[string]*models.RuleProvider {
	return map[string]*models.RuleProvider{
		"reject": {
			Name:        "reject",
			Type:        "Rule",
			VehicleType: "HTTP",
			Behavior:    "domain",
			RuleCount:   1500,
			UpdatedAt:   "2025-01-15T10:00:00Z",
		},
		"icloud": {
			Name:        "icloud",
			Type:        "Rule",
			VehicleType: "HTTP",
			Behavior:    "domain",
			RuleCount:   200,
			UpdatedAt:   "2025-01-15T09:30:00Z",
		},
		"apple": {
			Name:        "apple",
			Type:        "Rule",
			VehicleType: "HTTP",
			Behavior:    "domain",
			RuleCount:   500,
			UpdatedAt:   "2025-01-15T09:45:00Z",
		},
	}
}

// mockRules returns 5 mock rules.
func mockRules() []models.Rule {
	return []models.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "google.com", Proxy: "Proxy"},
		{Type: "DOMAIN-KEYWORD", Payload: "youtube", Proxy: "Proxy"},
		{Type: "DOMAIN-SUFFIX", Payload: "twitter.com", Proxy: "Proxy"},
		{Type: "DOMAIN-SUFFIX", Payload: "baidu.com", Proxy: "DIRECT"},
		{Type: "MATCH", Payload: "", Proxy: "Proxy"},
	}
}

// mockConnections returns 3 active mock connections.
func mockConnections() []models.Connection {
	return []models.Connection{
		{
			ID: "conn-001",
			Metadata: models.ConnectionMetadata{
				Network:         "tcp",
				Type:            "HTTP",
				SourceIP:        "192.168.1.100",
				DestinationIP:   "142.250.80.46",
				SourcePort:      "54321",
				DestinationPort: "443",
				Host:            "www.google.com",
				DNSMode:         "normal",
				ProcessPath:     "/usr/bin/curl",
			},
			Upload:      1024,
			Download:    20480,
			Chains:      []string{"Proxy", "HK"},
			Rule:        "DOMAIN-SUFFIX",
			RulePayload: "google.com",
		},
		{
			ID: "conn-002",
			Metadata: models.ConnectionMetadata{
				Network:         "tcp",
				Type:            "HTTPS",
				SourceIP:        "192.168.1.100",
				DestinationIP:   "172.217.24.14",
				SourcePort:      "54322",
				DestinationPort: "443",
				Host:            "www.youtube.com",
				DNSMode:         "normal",
				ProcessPath:     "/usr/bin/firefox",
			},
			Upload:      512,
			Download:    102400,
			Chains:      []string{"Proxy", "JP"},
			Rule:        "DOMAIN-KEYWORD",
			RulePayload: "youtube",
		},
		{
			ID: "conn-003",
			Metadata: models.ConnectionMetadata{
				Network:         "udp",
				Type:            "DNS",
				SourceIP:        "192.168.1.100",
				DestinationIP:   "8.8.8.8",
				SourcePort:      "54323",
				DestinationPort: "53",
				Host:            "",
				DNSMode:         "normal",
				ProcessPath:     "/usr/lib/systemd/systemd-resolved",
			},
			Upload:      128,
			Download:    256,
			Chains:      []string{"DIRECT"},
			Rule:        "MATCH",
			RulePayload: "",
		},
	}
}
