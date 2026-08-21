package events

// Event topics
const (
	TopicLanguageChanged  = "language.changed"
	TopicConfigSwitched   = "config.switched"
	TopicProxyModeChanged = "proxymode.changed"
)

// LanguageChangedEvent is triggered when the UI language changes.
type LanguageChangedEvent struct {
	Language string
}

// ConfigSwitchedEvent is triggered when Mihomo config profile is switched.
type ConfigSwitchedEvent struct {
	Path string
}

// ProxyModeChangedEvent is triggered when proxy mode (rule/global/direct) changes.
type ProxyModeChangedEvent struct {
	Mode string
}
