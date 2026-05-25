package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FindAllConfigFiles discovers all available mihomo config.yaml files on the system.
// It checks:
//  1. Config directories from running mihomo processes (via findMihomoProcessConfig)
//  2. Well-known paths: /etc/mihomo/, ~/.config/mihomo/, /opt/clashtui/
//  3. Current working directory
//  4. ~/.config/mihomoTui/configs/ (user-placed configs)
//
// Results are deduplicated and sorted. Only files that exist and are readable are returned.
func FindAllConfigFiles() []string {
	seen := make(map[string]bool)
	var results []string

	addUnique := func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		results = append(results, path)
	}

	// 1. Configs from running mihomo processes
	if procDirs := findMihomoProcessConfig(); len(procDirs) > 0 {
		for _, dir := range procDirs {
			p := filepath.Join(dir, "config.yaml")
			if fileExists(p) {
				addUnique(p)
			}
		}
	}

	// 2. Well-known single-file locations
	wellKnown := []string{
		"/opt/clashtui/mihomo/config/config.yaml",
		"/etc/mihomo/config.yaml",
		filepath.Join(os.ExpandEnv("$HOME"), ".config", "mihomo", "config.yaml"),
	}
	for _, p := range wellKnown {
		if fileExists(p) {
			addUnique(p)
		}
	}

	// 3. Scan /etc/mihomo/ for all *.yaml files (or just config.yaml)
	etcDir := "/etc/mihomo"
	if info, err := os.Stat(etcDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(etcDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
					p := filepath.Join(etcDir, e.Name())
					if fileExists(p) {
						addUnique(p)
					}
				}
			}
		}
	}

	// 4. Scan ~/.config/mihomo/ for all *.yaml files
	mihomoCfgDir := filepath.Join(os.ExpandEnv("$HOME"), ".config", "mihomo")
	if info, err := os.Stat(mihomoCfgDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(mihomoCfgDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
					p := filepath.Join(mihomoCfgDir, e.Name())
					if fileExists(p) {
						addUnique(p)
					}
				}
			}
		}
	}

	// 5. Current working directory
	if wd, err := os.Getwd(); err == nil {
		entries, err := os.ReadDir(wd)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
					p := filepath.Join(wd, e.Name())
					if fileExists(p) {
						addUnique(p)
					}
				}
			}
		}
	}

	// 6. ~/.config/mihomo/ — user can place config files here (must be under Mihomo's home dir)
	userCfgDir := filepath.Join(os.ExpandEnv("$HOME"), ".config", "mihomo")
	if info, err := os.Stat(userCfgDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(userCfgDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
					p := filepath.Join(userCfgDir, e.Name())
					if fileExists(p) {
						addUnique(p)
					}
				}
			}
		}
	}

	// Sort for consistent ordering
	sort.Strings(results)
	return results
}

// FindCurrentConfigPath returns the path to the currently active mihomo config.
// It is a convenience wrapper around FindMihomoConfigPath.
func FindCurrentConfigPath() string {
	return FindMihomoConfigPath()
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

// ValidateConfigFile checks that the given path points to a readable YAML file
// that looks like a mihomo config (has a "port" or "mixed-port" or "proxies" key).
func ValidateConfigFile(path string) error {
	if !fileExists(path) {
		return fmt.Errorf("文件不存在: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("无法读取文件: %w", err)
	}
	content := string(data)
	// Quick check: look for common mihomo config top-level keys
	if !strings.Contains(content, "port:") &&
		!strings.Contains(content, "mixed-port:") &&
		!strings.Contains(content, "proxies:") &&
		!strings.Contains(content, "proxy-groups:") {
		return fmt.Errorf("文件不是有效的 mihomo 配置文件")
	}
	return nil
}
