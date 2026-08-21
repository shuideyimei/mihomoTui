package models

// RuleDisplay represents a parsed or UI-facing rule.
type RuleDisplay struct {
	Type    string `json:"type" yaml:"type"`
	Payload string `json:"payload" yaml:"payload"`
	Proxy   string `json:"proxy" yaml:"proxy"`
}

// ProxyGroupEntry is a proxy group definition in config.
type ProxyGroupEntry struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	Proxies  []string `json:"proxies,omitempty" yaml:"proxies,omitempty"`
	Use      []string `json:"use,omitempty" yaml:"use,omitempty"`
	URL      string   `json:"url,omitempty" yaml:"url,omitempty"`
	Interval int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Filter   string   `json:"filter,omitempty" yaml:"filter,omitempty"`
}

// ProxyProviderEntry is a proxy provider definition in config.
type ProxyProviderEntry struct {
	Type        string `json:"type" yaml:"type"`
	URL         string `json:"url" yaml:"url"`
	Interval    int    `json:"interval" yaml:"interval"`
	HealthCheck *struct {
		Enable   bool   `json:"enable" yaml:"enable"`
		URL      string `json:"url" yaml:"url"`
		Interval int    `json:"interval" yaml:"interval"`
	} `json:"health-check,omitempty" yaml:"health-check,omitempty"`
}

// RuleProviderEntry is a rule provider definition in config.
type RuleProviderEntry struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Behavior string `json:"behavior" yaml:"behavior"`
	URL      string `json:"url" yaml:"url"`
	Path     string `json:"path" yaml:"path"`
	Interval int    `json:"interval" yaml:"interval"`
	Format   string `json:"format" yaml:"format"`
}

// ProviderInfo holds metadata about a proxy provider.
type ProviderInfo struct {
	Name     string `json:"name" yaml:"name"`
	URL      string `json:"url" yaml:"url"`
	Interval int    `json:"interval" yaml:"interval"`
}
