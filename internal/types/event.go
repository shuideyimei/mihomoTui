package types

// EventType categorises events on the internal event bus.
type EventType string

const (
	// Subscription events
	EventSubAdded    EventType = "subscription:added"
	EventSubUpdated  EventType = "subscription:updated"
	EventSubRemoved  EventType = "subscription:removed"
	EventSubError    EventType = "subscription:error"
	EventSubProgress EventType = "subscription:progress"

	// Profile events
	EventProfileActivated   EventType = "profile:activated"
	EventProfileDeactivated EventType = "profile:deactivated"
	EventProfileCreated     EventType = "profile:created"
	EventProfileDeleted     EventType = "profile:deleted"

	// Config events
	EventConfigReloaded   EventType = "config:reloaded"
	EventConfigRolledBack EventType = "config:rolled_back"
	EventConfigError      EventType = "config:error"

	// Proxy group events
	EventProxyGroupImported EventType = "proxygroup:imported"
	EventProxyGroupResolved EventType = "proxygroup:resolved"

	// Update events
	EventUpdateStarted EventType = "update:started"
	EventUpdateDone    EventType = "update:done"
	EventUpdateFailed  EventType = "update:failed"

	// Cache events
	EventCacheCleared     EventType = "cache:cleared"
	EventCacheNodeUpdated EventType = "cache:node_updated"
)

// Event is a generic event on the event bus.
type Event struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// EventHandler is a callback for handling events.
type EventHandler func(Event)

// EventBus defines the pub/sub contract for internal events.
type EventBus interface {
	Publish(event Event)
	Subscribe(eventType EventType, handler EventHandler) func() // returns unsubscribe func
	Unsubscribe(id string)
}

// ProgressPayload is sent during long-running operations.
type ProgressPayload struct {
	SubscriptionID string `json:"subscription_id,omitempty"`
	Progress       int    `json:"progress"` // 0-100
	Message        string `json:"message"`
}
