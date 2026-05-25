package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"

	"mihomoTui/internal/types"
)

// Format represents the detected format of subscription data.
type Format int

const (
	FormatUnknown Format = iota
	FormatClashYAML
	FormatSurge
	FormatBase64
	FormatV2Ray
	FormatShadowsocks
	FormatTrojan
	FormatHysteria2
	FormatProxiesText // newline-separated proxy URIs
)

func (f Format) String() string {
	switch f {
	case FormatClashYAML:
		return "Clash/Mihomo YAML"
	case FormatSurge:
		return "Surge"
	case FormatBase64:
		return "Base64"
	case FormatV2Ray:
		return "V2Ray"
	case FormatShadowsocks:
		return "Shadowsocks"
	case FormatTrojan:
		return "Trojan"
	case FormatHysteria2:
		return "Hysteria2"
	case FormatProxiesText:
		return "Plain Text Proxies"
	default:
		return "Unknown"
	}
}

// DetectFormat identifies the format of subscription data.
func DetectFormat(data []byte) Format {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return FormatUnknown
	}

	// Try YAML first (Clash config)
	if trimmed[0] == '{' || trimmed[0] == '[' {
		// Could be JSON — try to see if it has Clash structure
		if isClashJSON(trimmed) {
			return FormatClashYAML // treat as clash, will convert
		}
	}

	// Check for Clash YAML
	if bytes.HasPrefix(trimmed, []byte("port:")) ||
		bytes.HasPrefix(trimmed, []byte("socks-port:")) ||
		bytes.HasPrefix(trimmed, []byte("mixed-port:")) ||
		bytes.HasPrefix(trimmed, []byte("proxies:")) ||
		bytes.Contains(trimmed, []byte("\nproxies:")) {
		return FormatClashYAML
	}

	// Check for Surge config
	if bytes.HasPrefix(trimmed, []byte("[General]")) ||
		bytes.HasPrefix(trimmed, []byte("[Proxy]")) ||
		bytes.HasPrefix(trimmed, []byte("[Proxy Group]")) ||
		bytes.HasPrefix(trimmed, []byte("[Rule]")) {
		return FormatSurge
	}

	// Check for newline-separated proxy URIs (plain text) before individual protocols
	// to handle mixed protocol types correctly.
	str := string(trimmed)
	lines := strings.Split(str, "\n")
	proxyCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "ss://") ||
			strings.HasPrefix(line, "vmess://") ||
			strings.HasPrefix(line, "trojan://") ||
			strings.HasPrefix(line, "hysteria2://") ||
			strings.HasPrefix(line, "hy2://")) {
			proxyCount++
		}
	}
	if proxyCount > 1 {
		// Multiple proxy URIs → treat as plain text (handles mixed protocols)
		return FormatProxiesText
	}

	// Check for single V2Ray URI
	if bytes.Contains(trimmed, []byte("vmess://")) {
		return FormatV2Ray
	}

	// Check for single Shadowsocks URI
	if bytes.Contains(trimmed, []byte("ss://")) {
		return FormatShadowsocks
	}

	// Check for single Trojan URI
	if bytes.Contains(trimmed, []byte("trojan://")) {
		return FormatTrojan
	}

	// Check for single Hysteria2 URI
	if bytes.Contains(trimmed, []byte("hysteria2://")) || bytes.Contains(trimmed, []byte("hy2://")) {
		return FormatHysteria2
	}

	// Check if it's Base64 encoded (try to decode and check URL patterns)
	if isLikelyBase64(trimmed) {
		return FormatBase64
	}

	return FormatUnknown
}

func isLikelyBase64(data []byte) bool {
	str := string(bytes.TrimSpace(data))
	// Base64 typically has no spaces, ends with = or ==, and is mostly alphanumeric + +/ or -_
	hasEquals := strings.HasSuffix(str, "=") || strings.HasSuffix(str, "==")
	if !hasEquals && len(str) < 20 {
		return false
	}

	// Try to detect if it's a single-line base64 (no newlines or only at end)
	lines := strings.Split(str, "\n")
	if len(lines) > 5 {
		return false // too many lines for base64
	}

	// Check character set
	validChars := 0
	for _, c := range str {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '+' || c == '/' || c == '=' || c == '-' || c == '_' || c == '\n' || c == '\r' {
			validChars++
		}
	}

	// If we can decode it and it looks like a proxy config...
	if float64(validChars)/float64(len(str)) > 0.85 {
		decoded, err := base64Decode(str)
		if err != nil {
			return false
		}
		// Check if decoded contains proxy URI patterns
		decodedStr := string(decoded)
		return strings.Contains(decodedStr, "://") &&
			(strings.Contains(decodedStr, "ss://") ||
				strings.Contains(decodedStr, "vmess://") ||
				strings.Contains(decodedStr, "trojan://") ||
				strings.Contains(decodedStr, "hysteria2://"))
	}

	return false
}

func isClashJSON(data []byte) bool {
	var probe map[string]interface{}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false
	}
	_, hasProxies := probe["proxies"]
	_, hasGroups := probe["proxy-groups"]
	return hasProxies || hasGroups
}

// base64Decode attempts base64 decoding with various padding/encoding schemes.
func base64Decode(s string) ([]byte, error) {
	// Remove whitespace
	s = strings.TrimSpace(s)

	// Try standard encoding first
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	// Try URL-safe encoding
	if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	// Try with padding fix
	padded := s
	switch len(s) % 4 {
	case 1:
		return nil, fmt.Errorf("invalid base64 length")
	case 2:
		padded += "=="
	case 3:
		padded += "="
	}

	if decoded, err := base64.StdEncoding.DecodeString(padded); err == nil {
		return decoded, nil
	}

	if decoded, err := base64.URLEncoding.DecodeString(padded); err == nil {
		return decoded, nil
	}

	// Try raw variants (no padding)
	if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("cannot decode base64")
}

// Parse parses subscription data of any supported format into a ConfigDocument.
func Parse(data []byte) (*types.ConfigDocument, error) {
	format := DetectFormat(data)

	switch format {
	case FormatClashYAML:
		return parseClashYAML(data)
	case FormatSurge:
		return parseSurge(data)
	case FormatBase64:
		return parseBase64(data)
	case FormatV2Ray:
		return parseV2Ray(data)
	case FormatShadowsocks:
		return parseShadowsocks(data)
	case FormatTrojan:
		return parseTrojan(data)
	case FormatHysteria2:
		return parseHysteria2(data)
	case FormatProxiesText:
		return parseProxyURIs(data)
	default:
		return nil, fmt.Errorf("unsupported subscription format: cannot auto-detect")
	}
}

// parseClashYAML parses a Clash/Mihomo YAML config.
func parseClashYAML(data []byte) (*types.ConfigDocument, error) {
	// First unmarshal into a raw map to detect which scalar keys were explicitly set.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid Clash YAML: %w", err)
	}

	var doc types.ConfigDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid Clash YAML: %w", err)
	}

	// Track explicitly set scalar fields so mergeScalars can distinguish
	// "explicitly set to zero" from "not set".
	doc.SetFields = make(map[string]bool)
	scalarKeys := []string{
		"port", "socks-port", "mixed-port", "redir-port", "tproxy-port",
		"allow-lan", "bind-address", "mode", "log-level", "external-ui",
		"secret", "interface-name", "routing-mark", "ipv6", "tcp-concurrent",
	}
	for _, k := range scalarKeys {
		if _, ok := raw[k]; ok {
			doc.SetFields[k] = true
		}
	}

	return &doc, nil
}

// parseSurge converts a Surge config to a Clash ConfigDocument.
// This is a best-effort conversion.
func parseSurge(data []byte) (*types.ConfigDocument, error) {
	lines := strings.Split(string(data), "\n")
	doc := &types.ConfigDocument{}
	section := ""
	var proxies []map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case line == "" || strings.HasPrefix(line, ";"):
			continue
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		switch section {
		case "proxy":
			if proxy, err := parseSurgeProxyLine(line); err == nil {
				proxies = append(proxies, proxy)
			}
		case "proxy group":
			if group, err := parseSurgeGroupLine(line); err == nil {
				doc.ProxyGroups = append(doc.ProxyGroups, group)
			}
		case "rule":
			if rule := parseSurgeRuleLine(line); rule != "" {
				doc.Rules = append(doc.Rules, rule)
			}
		}
	}

	if len(proxies) > 0 {
		doc.Proxies = proxies
	}

	return doc, nil
}

func parseSurgeProxyLine(line string) (map[string]interface{}, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid surge proxy line")
	}
	name := strings.TrimSpace(parts[0])
	attrs := strings.TrimSpace(parts[1])

	proxy := map[string]interface{}{
		"name": name,
	}

	tokens := strings.Split(attrs, ",")
	if len(tokens) < 2 {
		return nil, fmt.Errorf("invalid surge proxy attrs")
	}

	proxyType := strings.TrimSpace(tokens[0])
	switch strings.ToLower(proxyType) {
	case "ss":
		proxy["type"] = "ss"
		if len(tokens) >= 4 {
			proxy["server"] = strings.TrimSpace(tokens[1])
			proxy["port"] = parseInt(strings.TrimSpace(tokens[2]))
			proxy["cipher"] = strings.TrimSpace(tokens[3])
		}
		if len(tokens) >= 5 {
			proxy["password"] = strings.TrimSpace(tokens[4])
		}
	case "trojan":
		proxy["type"] = "trojan"
		if len(tokens) >= 3 {
			proxy["server"] = strings.TrimSpace(tokens[1])
			proxy["port"] = parseInt(strings.TrimSpace(tokens[2]))
		}
		if len(tokens) >= 4 {
			proxy["password"] = strings.TrimSpace(tokens[3])
		}
	default:
		return nil, fmt.Errorf("unsupported surge proxy type: %s", proxyType)
	}

	return proxy, nil
}

func parseSurgeGroupLine(line string) (map[string]interface{}, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid surge group line")
	}
	name := strings.TrimSpace(parts[0])
	attrs := strings.TrimSpace(parts[1])

	tokens := strings.Split(attrs, ",")
	if len(tokens) < 2 {
		return nil, fmt.Errorf("invalid surge group attrs")
	}

	group := map[string]interface{}{
		"name": name,
		"type": "select",
	}

	if len(tokens) > 0 {
		group["type"] = strings.TrimSpace(tokens[0])
	}
	var proxies []string
	for _, t := range tokens[1:] {
		p := strings.TrimSpace(t)
		if p != "" {
			proxies = append(proxies, p)
		}
	}
	if len(proxies) > 0 {
		group["proxies"] = proxies
	}

	return group, nil
}

func parseSurgeRuleLine(line string) string {
	// Convert Surge rule format to Clash format (rough conversion)
	parts := strings.SplitN(line, ",", 3)
	if len(parts) < 2 {
		return ""
	}
	ruleType := strings.TrimSpace(parts[0])
	ruleValue := strings.TrimSpace(parts[1])
	ruleProxy := "DIRECT"
	if len(parts) >= 3 {
		ruleProxy = strings.TrimSpace(parts[2])
	}

	switch strings.ToUpper(ruleType) {
	case "DOMAIN-SUFFIX":
		return fmt.Sprintf("DOMAIN-SUFFIX,%s,%s", ruleValue, ruleProxy)
	case "DOMAIN-KEYWORD":
		return fmt.Sprintf("DOMAIN-KEYWORD,%s,%s", ruleValue, ruleProxy)
	case "DOMAIN":
		return fmt.Sprintf("DOMAIN,%s,%s", ruleValue, ruleProxy)
	case "IP-CIDR", "IP-CIDR6":
		return fmt.Sprintf("%s,%s,%s,no-resolve", strings.ToUpper(ruleType), ruleValue, ruleProxy)
	case "GEOIP":
		return fmt.Sprintf("GEOIP,%s,%s", ruleValue, ruleProxy)
	case "FINAL":
		return fmt.Sprintf("MATCH,%s", ruleProxy)
	default:
		return fmt.Sprintf("%s,%s,%s", ruleType, ruleValue, ruleProxy)
	}
}

// parseBase64 handles base64-encoded subscription data.
func parseBase64(data []byte) (*types.ConfigDocument, error) {
	decoded, err := base64Decode(string(data))
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	// After decoding, re-detect the format
	return Parse(decoded)
}

// parseV2Ray handles vmess:// URIs.
func parseV2Ray(data []byte) (*types.ConfigDocument, error) {
	proxies, err := extractURIs(string(data), "vmess://", parseVmessURI)
	if err != nil {
		return nil, err
	}
	return &types.ConfigDocument{Proxies: proxies}, nil
}

func parseVmessURI(uri string) (map[string]interface{}, error) {
	// Remove vmess:// prefix
	encoded := strings.TrimPrefix(uri, "vmess://")

	// Try base64 decode
	decoded, err := base64Decode(encoded)
	if err != nil {
		// Try parsing as query parameters
		return parseVmessQuery(uri)
	}

	var vmess struct {
		Add  string      `json:"add"`
		Aid  interface{} `json:"aid"`
		Host string      `json:"host"`
		ID   string      `json:"id"`
		Port interface{} `json:"port"`
		PS   string      `json:"ps"`
		Net  string      `json:"net"`
		Path string      `json:"path"`
		Type string      `json:"type"`
		TLS  string      `json:"tls"`
		Scy  string      `json:"scy"`
		SNI  string      `json:"sni"`
		Alpn string      `json:"alpn"`
		FP   string      `json:"fp"`
	}

	if err := json.Unmarshal(decoded, &vmess); err != nil {
		return nil, fmt.Errorf("invalid vmess JSON: %w", err)
	}

	proxy := map[string]interface{}{
		"name":     vmess.PS,
		"type":     "vmess",
		"server":   vmess.Add,
		"port":     toInt(vmess.Port),
		"uuid":     vmess.ID,
		"alterId":  toInt(vmess.Aid),
		"cipher":   "auto",
		"tls":      vmess.TLS != "",
		"network":  vmess.Net,
		"ws-path":  vmess.Path,
		"ws-headers": map[string]string{},
	}

	if vmess.Scy != "" {
		proxy["cipher"] = vmess.Scy
	}
	if vmess.Host != "" && vmess.Net == "ws" {
		proxy["ws-headers"] = map[string]string{"Host": vmess.Host}
	}

	return proxy, nil
}

func parseVmessQuery(uri string) (map[string]interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	return map[string]interface{}{
		"name":   query.Get("remarks"),
		"type":   "vmess",
		"server": u.Hostname(),
		"port":   parseInt(u.Port()),
		"uuid":   query.Get("id"),
		"alterId": parseInt(query.Get("aid")),
		"cipher": "auto",
		"tls":    query.Get("security") == "tls",
	}, nil
}

// parseShadowsocks handles ss:// URIs.
func parseShadowsocks(data []byte) (*types.ConfigDocument, error) {
	proxies, err := extractURIs(string(data), "ss://", parseSSURI)
	if err != nil {
		return nil, err
	}
	return &types.ConfigDocument{Proxies: proxies}, nil
}

func parseSSURI(uri string) (map[string]interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	proxy := map[string]interface{}{
		"type": "ss",
		"name": u.Fragment,
	}

	// Try SIP002 format: ss://base64(method:password)@server:port
	if u.User != nil {
		userInfo := u.User.String()
		if decoded, err := base64Decode(userInfo); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				proxy["cipher"] = parts[0]
				proxy["password"] = parts[1]
			}
		} else {
			// Try directly as method:password
			parts := strings.SplitN(userInfo, ":", 2)
			if len(parts) == 2 {
				proxy["cipher"] = parts[0]
				proxy["password"] = parts[1]
			}
		}
	}

	proxy["server"] = u.Hostname()
	proxy["port"] = parseInt(u.Port())

	// Query parameters
	query := u.Query()
	if plugin := query.Get("plugin"); plugin != "" {
		proxy["plugin"] = plugin
	}
	if udp := query.Get("udp-over-tcp"); udp != "" {
		proxy["udp-over-tcp"] = udp == "true"
	}

	return proxy, nil
}

// parseTrojan handles trojan:// URIs.
func parseTrojan(data []byte) (*types.ConfigDocument, error) {
	proxies, err := extractURIs(string(data), "trojan://", parseTrojanURI)
	if err != nil {
		return nil, err
	}
	return &types.ConfigDocument{Proxies: proxies}, nil
}

func parseTrojanURI(uri string) (map[string]interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	proxy := map[string]interface{}{
		"type":     "trojan",
		"name":     u.Fragment,
		"server":   u.Hostname(),
		"port":     parseInt(u.Port()),
		"password": u.User.String(),
	}

	query := u.Query()
	if sni := query.Get("sni"); sni != "" {
		proxy["sni"] = sni
	}
	if alpn := query.Get("alpn"); alpn != "" {
		proxy["alpn"] = alpn
	}
	if fp := query.Get("fp"); fp != "" {
		proxy["fingerprint"] = fp
	}
	if allowInsecure := query.Get("allowInsecure"); allowInsecure == "1" {
		proxy["skip-cert-verify"] = true
	}
	if ws := query.Get("ws"); ws == "1" {
		proxy["network"] = "ws"
		if path := query.Get("wspath"); path != "" {
			proxy["ws-path"] = path
		}
	}

	return proxy, nil
}

// parseHysteria2 handles hysteria2:// and hy2:// URIs.
func parseHysteria2(data []byte) (*types.ConfigDocument, error) {
	proxies, err := extractURIs(string(data), "hysteria2://", parseHysteria2URI)
	if err != nil {
		// Try hy2:// prefix
		proxies, err = extractURIs(string(data), "hy2://", parseHysteria2URI)
		if err != nil {
			return nil, err
		}
	}
	return &types.ConfigDocument{Proxies: proxies}, nil
}

func parseHysteria2URI(uri string) (map[string]interface{}, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	proxy := map[string]interface{}{
		"type":     "hysteria2",
		"name":     u.Fragment,
		"server":   u.Hostname(),
		"port":     parseInt(u.Port()),
		"password": u.User.String(),
	}

	query := u.Query()
	if sni := query.Get("sni"); sni != "" {
		proxy["sni"] = sni
	}
	if insecure := query.Get("insecure"); insecure == "1" {
		proxy["skip-cert-verify"] = true
	}
	if obfs := query.Get("obfs"); obfs != "" {
		proxy["obfs"] = obfs
	}
	if obfsPassword := query.Get("obfs-password"); obfsPassword != "" {
		proxy["obfs-password"] = obfsPassword
	}
	if alpn := query.Get("alpn"); alpn != "" {
		proxy["alpn"] = alpn
	}

	return proxy, nil
}

// parseProxyURIs handles newline-separated proxy URIs of any supported type.
func parseProxyURIs(data []byte) (*types.ConfigDocument, error) {
	lines := strings.Split(string(data), "\n")
	var proxies []map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var proxy map[string]interface{}
		var err error

		switch {
		case strings.HasPrefix(line, "vmess://"):
			proxy, err = parseVmessURI(line)
		case strings.HasPrefix(line, "ss://"):
			proxy, err = parseSSURI(line)
		case strings.HasPrefix(line, "trojan://"):
			proxy, err = parseTrojanURI(line)
		case strings.HasPrefix(line, "hysteria2://"):
			proxy, err = parseHysteria2URI(line)
		case strings.HasPrefix(line, "hy2://"):
			proxy, err = parseHysteria2URI(line)
		default:
			continue // skip unrecognized lines
		}

		if err == nil && proxy != nil {
			proxies = append(proxies, proxy)
		}
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no valid proxy URIs found")
	}

	return &types.ConfigDocument{Proxies: proxies}, nil
}

// extractURIs extracts URIs with a given prefix from text and parses them.
func extractURIs(text, prefix string, parser func(string) (map[string]interface{}, error)) ([]map[string]interface{}, error) {
	lines := strings.Split(text, "\n")
	var proxies []map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		proxy, err := parser(line)
		if err != nil {
			continue // skip invalid entries
		}
		proxies = append(proxies, proxy)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no valid %s URIs found", prefix)
	}

	return proxies, nil
}
