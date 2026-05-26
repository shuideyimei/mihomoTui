package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"mihomoTui/internal/models"
)

var (
	stopLogSync context.CancelFunc
	logFile     *os.File
)

// InitLogging sets up the application log file. Should be called once at startup.
// Returns a shutdown function that flushes and closes the log.
func InitLogging() func() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("warning: cannot determine home dir: %v, logging to stderr", err)
		return func() {}
	}
	logDir := filepath.Join(homeDir, ".config", "mihomoTui")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("warning: cannot create log dir %s: %v, logging to stderr", logDir, err)
		return func() {}
	}
	logPath := filepath.Join(logDir, "mihomo.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("warning: cannot open log file %s: %v, logging to stderr", logPath, err)
		return func() {}
	}
	logFile = f

	ctx, cancel := context.WithCancel(context.Background())
	stopLogSync = cancel
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		defer logFile.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logFile.Sync()
			}
		}
	}()

	log.SetOutput(logFile)

	return func() {
		cancel()
		log.SetOutput(os.Stderr)
	}
}

var (
	Client       *HttpClient
	StreamClient *HttpClient
)

// HttpClient represents the API client
type HttpClient struct {
	mu         sync.RWMutex
	baseURL    string
	secret     string
	httpClient *http.Client
}

func InitClient(baseURL, secret string) {
	Client = &HttpClient{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	StreamClient = &HttpClient{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

func UpdateClient(baseURL, secret string) {
	Client.mu.Lock()
	Client.baseURL = baseURL
	Client.secret = secret
	Client.mu.Unlock()

	StreamClient.mu.Lock()
	StreamClient.baseURL = baseURL
	StreamClient.secret = secret
	StreamClient.mu.Unlock()

	log.Printf("Updated API client: %s", baseURL)
}

// drainAndClose drains the response body and closes it.
// Draining is required for HTTP connection reuse by Go's transport.
func drainAndClose(resp *http.Response) {
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// SetTimeout sets the HTTP client timeout
func (c *HttpClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// makeRequest makes an HTTP request to the API.
func (c *HttpClient) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	return c.makeRequestWithContext(context.Background(), method, endpoint, body)
}

// makeRequestWithContext is like makeRequest but uses the given context for
// the HTTP request, enabling context-based cancellation of in-flight requests
// and response body reads.
func (c *HttpClient) makeRequestWithContext(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	c.mu.RLock()
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	secret := c.secret
	c.mu.RUnlock()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

func (c *HttpClient) GetVersion() (*models.Version, error) {
	resp, err := c.makeRequest("GET", "/version", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var version models.Version
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode version: %w", err)
	}

	return &version, nil
}

// GetConfig retrieves the current configuration
func (c *HttpClient) GetConfig() (*models.Config, error) {
	resp, err := c.makeRequest("GET", "/configs", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var config models.Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// UpdateConfig updates the configuration
func (c *HttpClient) UpdateConfig(config *models.Config) error {
	resp, err := c.makeRequest("PATCH", "/configs", config)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to update config, status: %d", resp.StatusCode)
	}

	return nil
}

// GetProxies retrieves all proxies
func (c *HttpClient) GetProxies() (map[string]*models.Proxy, error) {
	resp, err := c.makeRequest("GET", "/proxies", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Proxies map[string]*models.Proxy `json:"proxies"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode proxies: %w", err)
	}

	return result.Proxies, nil
}

// GetProviders retrieves all proxy providers
func (c *HttpClient) GetProviders() (*models.ProvidersResponse, error) {
	resp, err := c.makeRequest("GET", "/providers/proxies", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.ProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode providers: %w", err)
	}

	return &result, nil
}

// TestGroupDelay tests the delay of all proxies in a group and returns delay data
func (c *HttpClient) TestGroupDelay(groupName string, testURL string, timeout int) (map[string]int, error) {
	endpoint := fmt.Sprintf("/group/%s/delay", groupName)

	query := url.Values{}
	if testURL != "" {
		query.Set("url", testURL)
	}
	query.Set("timeout", strconv.Itoa(timeout))
	params := "?" + query.Encode()

	resp, err := c.makeRequest("GET", endpoint+params, nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to test group delay, status: %d", resp.StatusCode)
	}

	var delays map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&delays); err != nil {
		return nil, fmt.Errorf("failed to decode group delay: %w", err)
	}

	return delays, nil
}

// GetProxy retrieves a specific proxy
func (c *HttpClient) GetProxy(name string) (*models.Proxy, error) {
	endpoint := fmt.Sprintf("/proxies/%s", name)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var proxy models.Proxy
	if err := json.NewDecoder(resp.Body).Decode(&proxy); err != nil {
		return nil, fmt.Errorf("failed to decode proxy: %w", err)
	}

	return &proxy, nil
}

// SelectProxy selects a proxy for a group
func (c *HttpClient) SelectProxy(groupName, proxyName string) error {
	endpoint := fmt.Sprintf("/proxies/%s", groupName)
	body := map[string]string{"name": proxyName}

	resp, err := c.makeRequest("PUT", endpoint, body)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to select proxy, status: %d", resp.StatusCode)
	}

	return nil
}

// TestProxyDelay tests the delay of a proxy
func (c *HttpClient) TestProxyDelay(name string, testURL string, timeout int) (int, error) {
	endpoint := fmt.Sprintf("/proxies/%s/delay", name)

	query := url.Values{}
	if testURL != "" {
		query.Set("url", testURL)
	}
	query.Set("timeout", strconv.Itoa(timeout))
	params := "?" + query.Encode()

	resp, err := c.makeRequest("GET", endpoint+params, nil)
	if err != nil {
		return 0, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to test delay, status: %d", resp.StatusCode)
	}

	var result struct {
		Delay int `json:"delay"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode delay: %w", err)
	}

	return result.Delay, nil
}

// GetRules retrieves all rules
func (c *HttpClient) GetRules() ([]models.Rule, error) {
	resp, err := c.makeRequest("GET", "/rules", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Rules []models.Rule `json:"rules"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode rules: %w", err)
	}

	return result.Rules, nil
}

// GetRuleProviders retrieves all rule providers from mihomo
func (c *HttpClient) GetRuleProviders() (*models.RuleProvidersResponse, error) {
	resp, err := c.makeRequest("GET", "/providers/rules", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.RuleProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode rule providers: %w", err)
	}

	return &result, nil
}

// RefreshRuleProvider triggers an update for a specific rule provider
func (c *HttpClient) RefreshRuleProvider(name string) error {
	endpoint := fmt.Sprintf("/providers/rules/%s", name)
	resp, err := c.makeRequest("PUT", endpoint, nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to refresh rule provider, status: %d", resp.StatusCode)
	}

	return nil
}

// GetConnections retrieves all active connections
func (c *HttpClient) GetConnections() ([]models.Connection, error) {
	resp, err := c.makeRequest("GET", "/connections", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Connections []models.Connection `json:"connections"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode connections: %w", err)
	}

	return result.Connections, nil
}

// CloseConnection closes a specific connection
func (c *HttpClient) CloseConnection(id string) error {
	endpoint := fmt.Sprintf("/connections/%s", id)
	resp, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to close connection, status: %d", resp.StatusCode)
	}

	return nil
}

// ReloadConfig triggers mihomo to re-parse its config by sending a PUT /configs
// request with the config file path in JSON format. The config file must already
// have been written by writeConfig before calling this function.
func (c *HttpClient) ReloadConfig(path string) error {
	c.mu.RLock()
	url := fmt.Sprintf("%s%s", c.baseURL, "/configs")
	secret := c.secret
	c.mu.RUnlock()

	body := map[string]string{"path": path}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReloadConfigData sends raw YAML content to PUT /configs, bypassing SAFE_PATHS
// validation by sending the config content as the request body.
func (c *HttpClient) ReloadConfigData(data []byte) error {
	c.mu.RLock()
	url := fmt.Sprintf("%s%s", c.baseURL, "/configs")
	secret := c.secret
	c.mu.RUnlock()

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PatchConfig merges the given partial config into the running configuration
// via PATCH /configs. This is used for runtime changes like adding safe-paths.
func (c *HttpClient) PatchConfig(partial interface{}) error {
	resp, err := c.makeRequest("PATCH", "/configs", partial)
	if err != nil {
		return fmt.Errorf("patch config request failed: %w", err)
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("patch config failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HealthCheck checks if the API is accessible
func (c *HttpClient) HealthCheck() error {
	resp, err := c.makeRequest("GET", "/", nil)
	if err != nil {
		return err
	}
	defer drainAndClose(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed, status: %d", resp.StatusCode)
	}

	return nil
}
