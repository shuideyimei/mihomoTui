package models

import "time"

// Config represents the mihomo configuration
type Config struct {
	Port               int                    `json:"port"`
	SocksPort          int                    `json:"socks-port"`
	RedirPort          int                    `json:"redir-port"`
	TProxyPort         int                    `json:"tproxy-port"`
	MixedPort          int                    `json:"mixed-port"`
	Authentication     []string               `json:"authentication"`
	AllowLan           bool                   `json:"allow-lan"`
	BindAddress        string                 `json:"bind-address"`
	Mode               string                 `json:"mode"`
	Tun                map[string]interface{} `json:"tun"`
	LogLevel           string                 `json:"log-level"`
	ExternalController string                 `json:"external-controller"`
	ExternalUI         string                 `json:"external-ui"`
	Secret             string                 `json:"secret"`
	Interface          string                 `json:"interface-name"`
	RoutingMark        int                    `json:"routing-mark"`
}

// Proxy represents a proxy node
type Proxy struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	UDP     bool                   `json:"udp"`
	History []ProxyHistory         `json:"history"`
	All     []string               `json:"all,omitempty"`
	Now     string                 `json:"now,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// ProxyHistory represents proxy delay history
type ProxyHistory struct {
	Time  time.Time `json:"time"`
	Delay int       `json:"delay"`
}

// ProxyGroup represents a proxy group
type ProxyGroup struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now"`
	All     []string       `json:"all"`
	History []ProxyHistory `json:"history"`
}

// Rule represents a proxy rule
type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

// Connection represents an active connection
type Connection struct {
	ID          string             `json:"id"`
	Metadata    ConnectionMetadata `json:"metadata"`
	Upload      int64              `json:"upload"`
	Download    int64              `json:"download"`
	Start       time.Time          `json:"start"`
	Chains      []string           `json:"chains"`
	Rule        string             `json:"rule"`
	RulePayload string             `json:"rulePayload"`
}

// ConnectionMetadata represents connection metadata
type ConnectionMetadata struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	DNSMode         string `json:"dnsMode"`
	ProcessPath     string `json:"processPath"`
	SpecialProxy    string `json:"specialProxy"`
}

// Traffic represents real-time traffic
type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// Log represents a log entry
type Log struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ProxyProvider represents a proxy provider with its proxies
type ProxyProvider struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	VehicleType    string   `json:"vehicleType"`
	Proxies        []*Proxy `json:"proxies"`
	TestUrl        string   `json:"testUrl"`
	ExpectedStatus string   `json:"expectedStatus"`
	UpdatedAt      string   `json:"updatedAt"`
}

// ProvidersResponse represents the response from /providers/proxies API
type ProvidersResponse struct {
	Providers map[string]*ProxyProvider `json:"providers"`
}

type MemoryUsage struct {
	Inuse   int64 `json:"inuse"`
	Oslimit int64 `json:"oslimit"`
}

// RuleProvider represents a rule provider from the mihomo /providers/rules API
type RuleProvider struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	VehicleType string `json:"vehicleType"`
	Behavior    string `json:"behavior"`
	RuleCount   int    `json:"ruleCount"`
	UpdatedAt   string `json:"updatedAt"`
}

// RuleProvidersResponse wraps the response from GET /providers/rules
type RuleProvidersResponse struct {
	Providers map[string]*RuleProvider `json:"providers"`
}

type Version struct {
	Meta    bool   `json:"meta"`
	Version string `json:"version"`
}
