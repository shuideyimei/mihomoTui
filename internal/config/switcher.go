package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FindAllConfigFiles discovers valid Mihomo configs from explicit, active, and
// well-known locations. It deliberately does not scan the current directory.
func FindAllConfigFiles() []string {
	seen := make(map[string]bool)
	var results []string

	addUnique := func(path string) {
		path = filepath.Clean(path)
		if seen[path] || ValidateConfigFile(path) != nil {
			return
		}
		seen[path] = true
		results = append(results, path)
	}

	if path := explicitMihomoConfigPath(); path != "" {
		addUnique(path)
	}

	for _, dir := range findMihomoProcessConfig() {
		addUnique(filepath.Join(dir, "config.yaml"))
	}

	wellKnown := []string{
		"/opt/clashtui/mihomo/config/config.yaml",
		"/etc/mihomo/config.yaml",
		filepath.Join(os.ExpandEnv("$HOME"), ".config", "mihomo", "config.yaml"),
	}
	for _, p := range wellKnown {
		addUnique(p)
	}

	dirs := []string{
		"/etc/mihomo",
		filepath.Join(os.ExpandEnv("$HOME"), ".config", "mihomo"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				addUnique(filepath.Join(dir, entry.Name()))
			}
		}
	}

	sort.Strings(results)
	return results
}

// FindCurrentConfigPath returns the path to the currently active mihomo config.
// It is a convenience wrapper around FindMihomoConfigPath.
func FindCurrentConfigPath() string {
	return FindMihomoConfigPath()
}

// SameConfigFile reports whether two paths refer to the same config file.
func SameConfigFile(first, second string) bool {
	if first == "" || second == "" {
		return false
	}
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	if firstErr == nil && secondErr == nil {
		return os.SameFile(firstInfo, secondInfo)
	}
	firstAbs, firstAbsErr := filepath.Abs(first)
	secondAbs, secondAbsErr := filepath.Abs(second)
	return firstAbsErr == nil && secondAbsErr == nil && filepath.Clean(firstAbs) == filepath.Clean(secondAbs)
}

// fileExists checks whether a file exists and is readable.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FriendlyPath returns a shortened, human-friendly version of a config path.
// It replaces $HOME with "~" for readability.
func FriendlyPath(path string) string {
	home := os.ExpandEnv("$HOME")
	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// ValidateConfigFile checks that the path is a readable YAML mapping with at
// least one known Mihomo top-level key.
func ValidateConfigFile(path string) error {
	if !fileExists(path) {
		return fmt.Errorf("文件不存在: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取文件: %w", err)
	}

	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	knownKeys := []string{
		"port", "socks-port", "redir-port", "tproxy-port", "mixed-port",
		"allow-lan", "bind-address", "mode", "log-level", "ipv6",
		"external-controller", "external-ui", "secret", "authentication",
		"interface-name", "routing-mark", "find-process-mode",
		"dns", "hosts", "profile", "tun", "sniffer",
		"proxies", "proxy-groups", "proxy-providers", "rule-providers", "rules",
		"geodata-mode", "geox-url", "global-client-fingerprint",
	}
	for _, key := range knownKeys {
		if _, ok := root[key]; ok {
			return nil
		}
	}
	return fmt.Errorf("文件不是有效的 mihomo 配置文件")
}
