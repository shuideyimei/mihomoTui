package hotreload

import (
	"fmt"
	"os"
	"sync"
	"time"

	"mihomoTui/internal/api"
	"mihomoTui/internal/types"
)

// Manager handles hot-reloading the mihomo configuration with rollback support.
type Manager struct {
	mu         sync.RWMutex
	apiClient  apiReloader
	history    []types.ReloadEvent
	lastBackup string // path to the last known good config backup
}

type apiReloader interface {
	ReloadConfig(path string) error
}

// NewManager creates a hot-reload manager.
func NewManager(client apiReloader) *Manager {
	return &Manager{
		apiClient: client,
		history:   make([]types.ReloadEvent, 0, 10),
	}
}

// Reload pushes a config to mihomo via the API.
// Returns a reload event with success/failure status.
func (m *Manager) Reload(configPath string) (*types.ReloadEvent, error) {
	event := types.ReloadEvent{
		ID:         fmt.Sprintf("reload_%d", time.Now().UnixNano()),
		ConfigPath: configPath,
		Time:       time.Now(),
	}

	err := m.apiClient.ReloadConfig(configPath)
	event.Success = err == nil
	if err != nil {
		event.Error = err.Error()
	}

	m.mu.Lock()
	m.history = append(m.history, event)
	if len(m.history) > 50 {
		m.history = m.history[len(m.history)-50:]
	}
	m.mu.Unlock()

	return &event, err
}

// ReloadWithBackup creates a backup, reloads, and rolls back on failure.
func (m *Manager) ReloadWithBackup(configPath, backupPath string) (*types.ReloadEvent, error) {
	// Create backup of current config if not already provided
	if backupPath == "" {
		backupPath = configPath + ".hotreload.bak"
	}

	if err := m.createBackup(configPath, backupPath); err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	event, err := m.Reload(configPath)
	if err != nil {
		// Attempt rollback
		rollbackErr := m.Rollback(backupPath)
		if rollbackErr != nil {
			event.RolledBack = true
			event.Error = fmt.Sprintf("reload: %v; rollback also failed: %v", err, rollbackErr)
		} else {
			event.RolledBack = true
			event.Error = fmt.Sprintf("reload failed, rolled back: %v", err)
		}
		return event, err
	}

	m.mu.Lock()
	m.lastBackup = backupPath
	m.mu.Unlock()

	return event, nil
}

// Rollback restores a previously backed-up config and reloads it.
func (m *Manager) Rollback(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	return m.apiClient.ReloadConfig(backupPath)
}

// RollbackToLast restores the last known good configuration.
func (m *Manager) RollbackToLast() error {
	m.mu.RLock()
	backup := m.lastBackup
	m.mu.RUnlock()

	if backup == "" {
		return fmt.Errorf("no backup available")
	}
	return m.Rollback(backup)
}

// GetHistory returns the reload event history.
func (m *Manager) GetHistory() []types.ReloadEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history := make([]types.ReloadEvent, len(m.history))
	copy(history, m.history)
	return history
}

// LastReload returns the most recent reload event, if any.
func (m *Manager) LastReload() *types.ReloadEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.history) == 0 {
		return nil
	}
	last := m.history[len(m.history)-1]
	return &last
}

// UpdateAPIClient updates the API client reference (e.g., after reconnection).
func (m *Manager) UpdateAPIClient(client apiReloader) {
	m.mu.Lock()
	m.apiClient = client
	m.mu.Unlock()
}

func (m *Manager) createBackup(sourcePath, backupPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, data, 0644)
}

// EnsureCompatibleAPIVersion checks that the API client wraps the expected function.
// This is a compile-time check of the interface.
var _ apiReloader = (*api.HttpClient)(nil)
