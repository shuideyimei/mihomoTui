package updater

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"mihomoTui/internal/types"
)

// SubUpdater defines the contract for updating a single subscription.
type SubUpdater interface {
	Update(ctx context.Context, id string) (*types.ConfigDocument, error)
}

// ConfigApplier defines the contract for applying a parsed config.
type ConfigApplier interface {
	Apply(doc *types.ConfigDocument) error
}

// Manager handles background subscription updates.
type Manager struct {
	mu         sync.RWMutex
	subUpdater SubUpdater
	applier    ConfigApplier
	eventBus   types.EventBus

	tasks     map[string]*types.UpdateTask
	running   bool
	cancel    context.CancelFunc
	ticker    *time.Ticker

	// Scheduling
	interval     time.Duration
	autoSubs     []string // subscription IDs with auto-update enabled
}

// NewManager creates a new background updater manager.
func NewManager(subUpdater SubUpdater, applier ConfigApplier, eventBus types.EventBus) *Manager {
	return &Manager{
		subUpdater: subUpdater,
		applier:    applier,
		eventBus:   eventBus,
		tasks:      make(map[string]*types.UpdateTask),
		interval:   60 * time.Minute,
	}
}

// Start begins the background update loop.
// It does not block — runs in its own goroutine.
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.mu.Unlock()

	log.Printf("[updater] started background update loop (interval: %v)", m.interval)

	go m.loop(ctx)
}

// Stop stops the background update loop.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	if m.cancel != nil {
		m.cancel()
	}
	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.running = false
	log.Printf("[updater] stopped")
}

// UpdateOne triggers an immediate update for a single subscription.
func (m *Manager) UpdateOne(ctx context.Context, id string) error {
	m.mu.Lock()
	task := &types.UpdateTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		SubscriptionID: id,
		Status:         types.UpdateStatusRunning,
		StartTime:      time.Now(),
		Attempt:        1,
		MaxAttempts:    3,
	}
	m.tasks[id] = task
	m.mu.Unlock()

	m.emitProgress(id, 10, "开始更新...")

	doc, err := m.subUpdater.Update(ctx, id)

	m.mu.Lock()
	t, exists := m.tasks[id]
	if !exists {
		t = task
	}
	t.EndTime = time.Now()

	if err != nil {
		t.Status = types.UpdateStatusFailed
		t.Error = err.Error()
	} else if doc == nil {
		t.Status = types.UpdateStatusSuccess
	} else {
		t.Status = types.UpdateStatusSuccess
	}
	m.tasks[id] = t
	m.mu.Unlock()

	// Publish events outside lock to avoid deadlocks
	if err != nil {
		m.emitProgress(id, 100, "更新失败")
		m.publishEvent(types.EventUpdateFailed, map[string]interface{}{
			"subscription_id": id,
			"error":           err.Error(),
		})
	} else if doc == nil {
		m.emitProgress(id, 100, "无需更新")
		m.publishEvent(types.EventUpdateDone, map[string]interface{}{
			"subscription_id": id,
			"changed":         false,
		})
	} else {
		m.emitProgress(id, 50, "配置解析完成，正在应用...")
		if appErr := m.applier.Apply(doc); appErr != nil {
			m.mu.Lock()
			t.Status = types.UpdateStatusFailed
			t.Error = appErr.Error()
			m.mu.Unlock()
			m.emitProgress(id, 100, "应用失败")
			m.publishEvent(types.EventUpdateFailed, map[string]interface{}{
				"subscription_id": id,
				"error":           appErr.Error(),
			})
		} else {
			m.emitProgress(id, 100, "更新完成")
			m.publishEvent(types.EventUpdateDone, map[string]interface{}{
				"subscription_id": id,
				"changed":         true,
			})
		}
	}

	return err
}

// UpdateAll triggers updates for all auto-update subscriptions in parallel.
func (m *Manager) UpdateAll(ctx context.Context) error {
	m.mu.RLock()
	subs := make([]string, len(m.autoSubs))
	copy(subs, m.autoSubs)
	m.mu.RUnlock()

	if len(subs) == 0 {
		return nil
	}

	m.publishEvent(types.EventUpdateStarted, map[string]interface{}{
		"count": len(subs),
	})

	var wg sync.WaitGroup
	errCh := make(chan error, len(subs))

	for _, id := range subs {
		wg.Add(1)
		go func(subID string) {
			defer wg.Done()
			if err := m.UpdateOne(ctx, subID); err != nil {
				errCh <- err
			}
		}(id)
	}

	wg.Wait()
	close(errCh)

	var lastErr error
	for err := range errCh {
		lastErr = err
	}

	return lastErr
}

// Status returns the current updater status.
func (m *Manager) Status() types.UpdaterStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := types.UpdaterStatus{
		IsRunning: m.running,
		Queued:    0,
		Running:   0,
		Failed:    0,
	}

	for _, t := range m.tasks {
		switch t.Status {
		case types.UpdateStatusRunning:
			status.Running++
		case types.UpdateStatusFailed:
			status.Failed++
		default:
			status.Queued++
		}
		status.Tasks = append(status.Tasks, t)
	}

	return status
}

// SetInterval sets the update interval for the background loop.
func (m *Manager) SetInterval(d time.Duration) {
	m.mu.Lock()
	m.interval = d
	m.mu.Unlock()
}

// SetAutoSubscriptions sets which subscriptions should be auto-updated.
func (m *Manager) SetAutoSubscriptions(ids []string) {
	m.mu.Lock()
	m.autoSubs = make([]string, len(ids))
	copy(m.autoSubs, ids)
	m.mu.Unlock()
}

// CancelUpdate cancels a running update for a subscription.
func (m *Manager) CancelUpdate(id string) {
	m.mu.Lock()
	if task, ok := m.tasks[id]; ok && task.Status == types.UpdateStatusRunning {
		task.Status = types.UpdateStatusCancelled
	}
	m.mu.Unlock()
}

// IsRunning reports whether the updater loop is active.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// loop is the main background ticker loop.
func (m *Manager) loop(ctx context.Context) {
	m.mu.Lock()
	m.ticker = time.NewTicker(m.interval)
	m.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.ticker.C:
			log.Printf("[updater] ticker fired, starting auto-update")
			if err := m.UpdateAll(ctx); err != nil {
				log.Printf("[updater] auto-update error: %v", err)
			}
		}
	}
}

func (m *Manager) emitProgress(subID string, progress int, message string) {
	m.publishEvent(types.EventSubProgress, types.ProgressPayload{
		SubscriptionID: subID,
		Progress:       progress,
		Message:        message,
	})
}

func (m *Manager) publishEvent(eventType types.EventType, payload interface{}) {
	if m.eventBus != nil {
		m.eventBus.Publish(types.Event{
			Type:    eventType,
			Payload: payload,
		})
	}
}
