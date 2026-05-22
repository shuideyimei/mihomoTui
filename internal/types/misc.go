package types

import "time"

// UpdateTask represents a single background update task.
type UpdateTask struct {
	ID            string        `json:"id"`
	SubscriptionID string       `json:"subscription_id"`
	Status        UpdateStatus  `json:"status"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time,omitempty"`
	Error         string        `json:"error,omitempty"`
	Attempt       int           `json:"attempt"`
	MaxAttempts   int           `json:"max_attempts"`
}

// UpdateStatus represents the status of an update task.
type UpdateStatus string

const (
	UpdateStatusPending   UpdateStatus = "pending"
	UpdateStatusRunning   UpdateStatus = "running"
	UpdateStatusSuccess   UpdateStatus = "success"
	UpdateStatusFailed    UpdateStatus = "failed"
	UpdateStatusCancelled UpdateStatus = "cancelled"
)

// UpdaterStatus reflects the overall state of the background updater.
type UpdaterStatus struct {
	IsRunning bool          `json:"is_running"`
	Queued    int           `json:"queued"`
	Running   int           `json:"running"`
	Failed    int           `json:"failed"`
	LastRun   time.Time     `json:"last_run,omitempty"`
	NextRun   time.Time     `json:"next_run,omitempty"`
	Tasks     []*UpdateTask `json:"tasks,omitempty"`
}

// CacheEntry is a single item in the node cache.
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpireAt  time.Time   `json:"expire_at,omitempty"`
}

// ReloadEvent records a single config reload attempt.
type ReloadEvent struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	ConfigPath string   `json:"config_path"`
	Time      time.Time `json:"time"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	RolledBack bool     `json:"rolled_back,omitempty"`
}
