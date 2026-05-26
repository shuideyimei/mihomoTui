package parser

import (
	"fmt"
	"strings"
	"time"

	"mihomoTui/internal/types"
)

// ParseWithRetry tries parsing with format hints and fallback.
func ParseWithRetry(data []byte, hint string) (*types.ConfigDocument, error) {
	doc, err := Parse(data)
	if err != nil && hint != "" {
		// If auto-detect fails, try the hinted format
		doc, err = parseWithHint(data, hint)
	}
	return doc, err
}

func parseWithHint(data []byte, hint string) (*types.ConfigDocument, error) {
	switch hint {
	case "clash", "yaml", "mihomo":
		return parseClashYAML(data)
	case "surge":
		return parseSurge(data)
	case "base64":
		return parseBase64(data)
	case "v2ray":
		return parseV2Ray(data)
	case "ss", "shadowsocks":
		return parseShadowsocks(data)
	case "trojan":
		return parseTrojan(data)
	case "hysteria2":
		return parseHysteria2(data)
	default:
		return nil, fmt.Errorf("unknown format hint: %s", hint)
	}
}

// ValidateConfig performs basic validation on a parsed config document.
func ValidateConfig(doc *types.ConfigDocument) error {
	if doc == nil {
		return fmt.Errorf("config document is nil")
	}

	for i, proxy := range doc.Proxies {
		name, _ := proxy["name"].(string)
		if name == "" {
			return fmt.Errorf("proxy at index %d has no name", i)
		}
		proxyType, _ := proxy["type"].(string)
		if proxyType == "" {
			return fmt.Errorf("proxy %q has no type", name)
		}
	}

	for i, group := range doc.ProxyGroups {
		name, _ := group["name"].(string)
		if name == "" {
			return fmt.Errorf("proxy-group at index %d has no name", i)
		}
		groupType, _ := group["type"].(string)
		if groupType == "" {
			return fmt.Errorf("proxy-group %q has no type", name)
		}
	}

	return nil
}

// ExtractProxiesFromText extracts proxy names from a mixed text containing proxy references.
func ExtractProxiesFromText(text string) []string {
	return extractProxyNames(text)
}

// extractProxyNames is a placeholder for regex-based extraction.
func extractProxyNames(text string) []string {
	// Simple line-by-line extraction; in production would use regex
	// This is used by proxy group import for reading proxy lists from user input
	lines := strings.Split(text, "\n")
	var names []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			names = append(names, line)
		}
	}
	return names
}

// Timing info helpers
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
