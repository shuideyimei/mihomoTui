package types

// ConfigDocument is the canonical parsed representation of a mihomo/clash config.
// It is designed for clean serialization to/from YAML and easy merging.
type ConfigDocument struct {
	Port            int                    `yaml:"port,omitempty"`
	SocksPort       int                    `yaml:"socks-port,omitempty"`
	MixedPort       int                    `yaml:"mixed-port,omitempty"`
	RedirPort       int                    `yaml:"redir-port,omitempty"`
	TProxyPort      int                    `yaml:"tproxy-port,omitempty"`
	AllowLan        bool                   `yaml:"allow-lan,omitempty"`
	BindAddress     string                 `yaml:"bind-address,omitempty"`
	Mode            string                 `yaml:"mode,omitempty"`
	LogLevel        string                 `yaml:"log-level,omitempty"`
	ExternalUI      string                 `yaml:"external-ui,omitempty"`
	Secret          string                 `yaml:"secret,omitempty"`
	Interface       string                 `yaml:"interface-name,omitempty"`
	RoutingMark     int                    `yaml:"routing-mark,omitempty"`
	IPv6            bool                   `yaml:"ipv6,omitempty"`
	TCPConcurrent   bool                   `yaml:"tcp-concurrent,omitempty"`

	Proxies        []map[string]interface{} `yaml:"proxies,omitempty,flow"`
	ProxyGroups    []map[string]interface{} `yaml:"proxy-groups,omitempty,flow"`
	Rules          []string                 `yaml:"rules,omitempty"`
	RuleProviders  []map[string]interface{} `yaml:"rule-providers,omitempty,flow"`
	ProxyProviders []map[string]interface{} `yaml:"proxy-providers,omitempty,flow"`

	DNS map[string]interface{} `yaml:"dns,omitempty"`
	Tun map[string]interface{} `yaml:"tun,omitempty"`

	// Extra preserves any top-level keys not covered above.
	Extra map[string]interface{} `yaml:",inline"`
}

// Clone returns a deep copy by round-tripping through map.
func (c *ConfigDocument) Clone() *ConfigDocument {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Proxies = cloneMaps(c.Proxies)
	cp.ProxyGroups = cloneMaps(c.ProxyGroups)
	cp.RuleProviders = cloneMaps(c.RuleProviders)
	cp.ProxyProviders = cloneMaps(c.ProxyProviders)
	cp.Extra = cloneMapSS(c.Extra)
	cp.DNS = cloneMapSS(c.DNS)
	cp.Tun = cloneMapSS(c.Tun)
	if c.Rules != nil {
		cp.Rules = make([]string, len(c.Rules))
		copy(cp.Rules, c.Rules)
	}
	return &cp
}

func cloneMaps(src []map[string]interface{}) []map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make([]map[string]interface{}, len(src))
	for i, m := range src {
		if m != nil {
			dst[i] = cloneMapSS(m)
		}
	}
	return dst
}

func cloneMapSS(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// IsEmpty returns true if the document has no meaningful content.
func (c *ConfigDocument) IsEmpty() bool {
	return len(c.Proxies) == 0 && len(c.ProxyGroups) == 0 && len(c.Rules) == 0
}
