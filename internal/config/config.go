package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppConfig represents the application configuration
type AppConfig struct {
	// API settings
	API APIConfig `json:"api"`
}

// GetValue returns configuration value by label
func (c *AppConfig) GetValue(label string) string {
	switch label {
	case "API地址":
		return c.API.BaseURL
	case "API密钥":
		return c.API.Secret
	default:
		return ""
	}
}

// SetValue sets configuration value by label
func (c *AppConfig) SetValue(label, value string) {
	switch label {
	case "API地址":
		c.API.BaseURL = value
	case "API密钥":
		c.API.Secret = value
	default:
		return
	}
}

// APIConfig represents API configuration
type APIConfig struct {
	BaseURL string `json:"base_url"`
	Secret  string `json:"secret"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		API: APIConfig{
			BaseURL: "http://127.0.0.1:9090",
			Secret:  "",
		},
	}
}

// Manager handles application configuration
type Manager struct {
	config     *AppConfig
	configPath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "mihomoTui")
	configPath := filepath.Join(configDir, "config.json")

	return &Manager{
		config:     DefaultConfig(),
		configPath: configPath,
	}
}

// Load loads configuration from file
func (m *Manager) Load() error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, create with default values
		return m.Save()
	}

	// Read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// Save saves configuration to file
func (m *Manager) Save() error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *AppConfig {
	return m.config
}

// Set updates the configuration
func (m *Manager) Set(config *AppConfig) {
	m.config = config
}

// GetAPI returns API configuration
func (m *Manager) GetAPI() APIConfig {
	return m.config.API
}

// SetAPI updates API configuration
func (m *Manager) SetAPI(config APIConfig) error {
	m.config.API = config
	return m.Save()
}

// Reset resets configuration to defaults
func (m *Manager) Reset() error {
	m.config = DefaultConfig()
	return m.Save()
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// Backup creates a backup of the current configuration
func (m *Manager) Backup() error {
	backupPath := m.configPath + ".backup"

	// Read current config
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// Restore restores configuration from backup
func (m *Manager) Restore() error {
	backupPath := m.configPath + ".backup"

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found")
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Parse backup
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse backup: %w", err)
	}

	// Update current config
	m.config = &config

	// Save restored config
	return m.Save()
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	config := m.config

	// Validate API config
	if config.API.BaseURL == "" {
		return fmt.Errorf("API base URL cannot be empty")
	}

	return nil
}

// Export exports configuration to a specified file
func (m *Manager) Export(path string) error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// Import imports configuration from a specified file
func (m *Manager) Import(path string) error {
	// Read import file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse imported config
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	// Create temporary manager to validate
	tempManager := &Manager{config: &config}
	if err := tempManager.Validate(); err != nil {
		return fmt.Errorf("imported config is invalid: %w", err)
	}

	// Update current config
	m.config = &config

	// Save imported config
	return m.Save()
}

// GetEndpoint returns the full API endpoint with secret
func (m *Manager) GetEndpoint() (string, string) {
	return m.config.API.BaseURL, m.config.API.Secret
}

// SetEndpoint updates API endpoint and secret
func (m *Manager) SetEndpoint(baseURL, secret string) error {
	m.config.API.BaseURL = baseURL
	m.config.API.Secret = secret
	return m.Save()
}

// GetDataDir returns a default data directory
func (m *Manager) GetDataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "mihomoTui")
}

// EnsureDataDir ensures the data directory exists
func (m *Manager) EnsureDataDir() error {
	return os.MkdirAll(m.GetDataDir(), 0755)
}
