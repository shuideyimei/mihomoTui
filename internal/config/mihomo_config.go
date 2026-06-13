package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// configMu protects all config file read+modify+write cycles against
// concurrent access from different pages (rules, rule-providers, proxy-groups).
var configMu sync.RWMutex

// proxy-group types
const (
	GroupTypeSelect      = "select"
	GroupTypeURLTest     = "url-test"
	GroupTypeFallback    = "fallback"
	GroupTypeLoadBalance = "load-balance"
)

var ValidGroupTypes = []string{GroupTypeSelect, GroupTypeURLTest, GroupTypeFallback, GroupTypeLoadBalance}

// MihomoConfig represents the full mihomo YAML config
type MihomoConfig struct {
	ProxyProviders map[string]ProxyProviderEntry `yaml:"proxy-providers,omitempty"`
	ProxyGroups    []ProxyGroupEntry             `yaml:"proxy-groups,omitempty"`
	// Other fields are preserved as-is via the raw document
}

// ProxyProviderEntry is a single proxy-provider definition
type ProxyProviderEntry struct {
	Type        string `yaml:"type"`
	URL         string `yaml:"url"`
	Interval    int    `yaml:"interval"`
	HealthCheck *struct {
		Enable   bool   `yaml:"enable"`
		URL      string `yaml:"url"`
		Interval int    `yaml:"interval"`
	} `yaml:"health-check,omitempty"`
}

// ProxyGroupEntry is a single proxy-group definition
type ProxyGroupEntry struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies,omitempty"`
	Use      []string `yaml:"use,omitempty"`
	URL      string   `yaml:"url,omitempty"`
	Interval int      `yaml:"interval,omitempty"`
	Filter   string   `yaml:"filter,omitempty"`
}

// ProviderInfo holds metadata about a mihomo proxy-provider read from config.
type ProviderInfo struct {
	Name     string
	URL      string
	Interval int
}

// GetMihomoProxyProviders reads proxy-providers from the running mihomo config.yaml.
func GetMihomoProxyProviders() ([]ProviderInfo, error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg struct {
		ProxyProviders map[string]ProxyProviderEntry `yaml:"proxy-providers"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	var providers []ProviderInfo
	for name, entry := range cfg.ProxyProviders {
		if entry.URL == "" {
			continue
		}
		providers = append(providers, ProviderInfo{
			Name:     name,
			URL:      entry.URL,
			Interval: entry.Interval,
		})
	}
	return providers, nil
}

// RemoveProviderFromConfig removes a proxy-provider from mihomo config.yaml.
func RemoveProviderFromConfig(name string) error {
	configMu.Lock()
	defer configMu.Unlock()

	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-providers")
	idx, ok := indices["proxy-providers"]
	if !ok {
		return fmt.Errorf("配置文件中没有 proxy-providers 字段")
	}

	providersMap := root.Content[idx+1]
	if providersMap.Kind != yaml.MappingNode {
		return fmt.Errorf("proxy-providers 格式错误")
	}

	found := false
	for j := 0; j < len(providersMap.Content); j += 2 {
		if providersMap.Content[j].Value == name {
			providersMap.Content = append(providersMap.Content[:j], providersMap.Content[j+2:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("未找到 proxy-provider %q", name)
	}

	return writeConfig(doc, configPath)
}

// FindMihomoConfigPath tries to locate the mihomo config.yaml
func FindMihomoConfigPath() string {
	if path := explicitMihomoConfigPath(); path != "" {
		return path
	}

	// Try to find from running mihomo process
	paths := findMihomoProcessConfig()
	for _, p := range paths {
		configPath := filepath.Join(p, "config.yaml")
		if fileExists(configPath) {
			return configPath
		}
	}
	// Fallback: check common locations
	commonPaths := []string{
		"/opt/clashtui/mihomo/config/config.yaml",
		"/etc/mihomo/config.yaml",
		os.ExpandEnv("$HOME/.config/mihomo/config.yaml"),
	}
	for _, p := range commonPaths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func explicitMihomoConfigPath() string {
	path := strings.TrimSpace(os.Getenv("MIHOMO_CONFIG_PATH"))
	if path == "" {
		return ""
	}
	path = filepath.Clean(os.ExpandEnv(path))
	if !fileExists(path) {
		return ""
	}
	return path
}

// findMihomoProcessConfig returns config dirs from running mihomo processes
func findMihomoProcessConfig() []string {
	return findMihomoProcessConfigAt("/proc")
}

func findMihomoProcessConfigAt(procRoot string) []string {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil
	}

	var paths []string
	seen := make(map[string]bool)
	addPath := func(raw string) {
		if strings.TrimSpace(raw) == "" {
			return
		}
		dir := filepath.Clean(raw)
		if !seen[dir] {
			seen[dir] = true
			paths = append(paths, dir)
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		processDir := filepath.Join(procRoot, entry.Name())
		if !isMihomoProcess(processDir) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(processDir, "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(string(data), "\x00")
		for i, arg := range args {
			if (arg == "-d" || arg == "--directory") && i+1 < len(args) {
				addPath(args[i+1])
			}
			if strings.HasPrefix(arg, "-d=") || strings.HasPrefix(arg, "--directory=") {
				addPath(strings.SplitN(arg, "=", 2)[1])
			}
		}
	}
	return paths
}

func isMihomoProcess(processDir string) bool {
	if comm, err := os.ReadFile(filepath.Join(processDir, "comm")); err == nil {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(comm))), "mihomo") {
			return true
		}
	}
	if exe, err := os.Readlink(filepath.Join(processDir, "exe")); err == nil {
		name := strings.TrimSuffix(filepath.Base(exe), " (deleted)")
		return strings.HasPrefix(strings.ToLower(name), "mihomo")
	}
	return false
}

// writeConfig writes the YAML document to the config file atomically.
func writeConfig(doc *yaml.Node, configPath string) error {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	return WriteFileAtomic(configPath, out, 0644)
}

// parseConfig reads and parses the mihomo config file
func parseConfig(configPath string) (*yaml.Node, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// Guard against empty/minimal YAML files to avoid index panic on doc.Content[0]
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0] == nil {
		return nil, fmt.Errorf("配置文件为空或格式无效: %s", configPath)
	}

	return &doc, nil
}

// findKeysInMapping finds indices of keys in a mapping node
func findKeysInMapping(root *yaml.Node, keys ...string) map[string]int {
	result := make(map[string]int)
	for i := 0; i < len(root.Content); i += 2 {
		for _, key := range keys {
			if root.Content[i].Value == key {
				result[key] = i
				break
			}
		}
	}
	return result
}

// GetProxyGroupsFromConfig returns the list of proxy group names from the config file
func GetProxyGroupsFromConfig() ([]string, error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	doc, err := parseConfig(configPath)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return nil, nil
	}

	groupsList := root.Content[groupsIdx+1]
	var names []string
	for _, groupNode := range groupsList.Content {
		if groupNode.Kind == yaml.MappingNode {
			for j := 0; j < len(groupNode.Content); j += 2 {
				if groupNode.Content[j].Value == "name" {
					names = append(names, groupNode.Content[j+1].Value)
					break
				}
			}
		}
	}
	return names, nil
}

// AddGroupToConfig adds a new proxy group to the config file
func AddGroupToConfig(name, groupType, filter, testURL string, use, proxies []string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 proxy-groups 字段")
	}

	groupsList := root.Content[groupsIdx+1]

	// Check for duplicate name
	for _, groupNode := range groupsList.Content {
		if groupNode.Kind == yaml.MappingNode {
			for j := 0; j < len(groupNode.Content); j += 2 {
				if groupNode.Content[j].Value == "name" && groupNode.Content[j+1].Value == name {
					return "", fmt.Errorf("代理组 %q 已存在", name)
				}
			}
		}
	}

	// Build the new group mapping node
	newGroup := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	// name
	newGroup.Content = append(newGroup.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
	)

	// type
	newGroup.Content = append(newGroup.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "type", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: groupType, Tag: "!!str"},
	)

	// test-url for url-test/fallback types
	if testURL != "" && (groupType == GroupTypeURLTest || groupType == GroupTypeFallback) {
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "url", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: testURL, Tag: "!!str"},
		)
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "interval", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "300", Tag: "!!int"},
		)
	}

	// use (providers)
	if len(use) > 0 {
		useNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, u := range use {
			useNode.Content = append(useNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: strings.TrimSpace(u), Tag: "!!str"},
			)
		}
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "use", Tag: "!!str"},
			useNode,
		)
	}

	// proxies (direct node references)
	if len(proxies) > 0 {
		proxiesNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, p := range proxies {
			proxiesNode.Content = append(proxiesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: strings.TrimSpace(p), Tag: "!!str"},
			)
		}
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "proxies", Tag: "!!str"},
			proxiesNode,
		)
	}

	// filter
	if filter != "" {
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "filter", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: filter, Tag: "!!str"},
		)
	}

	groupsList.Content = append(groupsList.Content, newGroup)

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func UpdateGroupInConfig(oldName, name, groupType, filter, testURL string, use, proxies []string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 proxy-groups 字段")
	}

	groupsList := root.Content[groupsIdx+1]

	var foundGroup *yaml.Node
	foundIndex := -1
	for idx, groupNode := range groupsList.Content {
		if groupNode.Kind == yaml.MappingNode {
			for j := 0; j < len(groupNode.Content); j += 2 {
				if groupNode.Content[j].Value == "name" && groupNode.Content[j+1].Value == oldName {
					foundGroup = groupNode
					foundIndex = idx
					break
				}
			}
			if foundGroup != nil {
				break
			}
		}
	}

	if foundGroup == nil {
		return "", fmt.Errorf("未找到代理组 %q", oldName)
	}

	groupsList.Content = append(groupsList.Content[:foundIndex], groupsList.Content[foundIndex+1:]...)

	newGroup := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	newGroup.Content = append(newGroup.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
	)
	newGroup.Content = append(newGroup.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "type", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: groupType, Tag: "!!str"},
	)

	if testURL != "" && (groupType == GroupTypeURLTest || groupType == GroupTypeFallback || groupType == GroupTypeLoadBalance) {
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "url", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: testURL, Tag: "!!str"},
		)
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "interval", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "300", Tag: "!!int"},
		)
	}

	if len(use) > 0 {
		useNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, u := range use {
			useNode.Content = append(useNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: strings.TrimSpace(u), Tag: "!!str"},
			)
		}
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "use", Tag: "!!str"},
			useNode,
		)
	}

	if len(proxies) > 0 {
		proxiesNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, p := range proxies {
			proxiesNode.Content = append(proxiesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: strings.TrimSpace(p), Tag: "!!str"},
			)
		}
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "proxies", Tag: "!!str"},
			proxiesNode,
		)
	}

	if filter != "" {
		newGroup.Content = append(newGroup.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "filter", Tag: "!!str"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: filter, Tag: "!!str"},
		)
	}

	if foundIndex >= len(groupsList.Content) {
		groupsList.Content = append(groupsList.Content, newGroup)
	} else {
		groupsList.Content = append(groupsList.Content, nil)
		copy(groupsList.Content[foundIndex+1:], groupsList.Content[foundIndex:])
		groupsList.Content[foundIndex] = newGroup
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func GetAllProxyGroupsFromConfig() ([]ProxyGroupEntry, error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	doc, err := parseConfig(configPath)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return nil, nil
	}

	groupsList := root.Content[groupsIdx+1]
	var groups []ProxyGroupEntry

	for _, groupNode := range groupsList.Content {
		if groupNode.Kind != yaml.MappingNode {
			continue
		}
		var entry ProxyGroupEntry
		var useNode, proxiesNode *yaml.Node
		for j := 0; j < len(groupNode.Content); j += 2 {
			key := groupNode.Content[j].Value
			val := groupNode.Content[j+1]
			switch key {
			case "name":
				entry.Name = val.Value
			case "type":
				entry.Type = val.Value
			case "url":
				entry.URL = val.Value
			case "interval":
				if v, err := strconv.Atoi(val.Value); err == nil {
					entry.Interval = v
				}
			case "filter":
				entry.Filter = val.Value
			case "use":
				useNode = val
			case "proxies":
				proxiesNode = val
			}
		}
		if useNode != nil && useNode.Kind == yaml.SequenceNode {
			for _, item := range useNode.Content {
				n := strings.TrimSpace(item.Value)
				if n != "" {
					entry.Use = append(entry.Use, n)
				}
			}
		}
		if proxiesNode != nil && proxiesNode.Kind == yaml.SequenceNode {
			for _, item := range proxiesNode.Content {
				n := strings.TrimSpace(item.Value)
				if n != "" {
					entry.Proxies = append(entry.Proxies, n)
				}
			}
		}
		if entry.Name != "" {
			groups = append(groups, entry)
		}
	}

	return groups, nil
}

// DeleteGroupFromConfig removes a proxy group by name from the config file
func DeleteGroupFromConfig(name string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 proxy-groups 字段")
	}

	groupsList := root.Content[groupsIdx+1]
	found := false
	for idx, groupNode := range groupsList.Content {
		if groupNode.Kind == yaml.MappingNode {
			for j := 0; j < len(groupNode.Content); j += 2 {
				if groupNode.Content[j].Value == "name" && groupNode.Content[j+1].Value == name {
					// Remove this element from the sequence
					groupsList.Content = append(groupsList.Content[:idx], groupsList.Content[idx+1:]...)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		return "", fmt.Errorf("未找到代理组 %q", name)
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// AddProviderToConfig adds a new proxy-provider and wires it into the first select-type
// proxy group with a "use" field. If no suitable group exists, a new select group is created.
func AddProviderToConfig(name, url string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件，请确认 mihomo 正在运行且配置文件存在")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	modified := false
	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-providers", "proxy-groups")
	providersIdx, hasProviders := indices["proxy-providers"]
	groupsIdx, hasGroups := indices["proxy-groups"]

	if hasProviders {
		providersMap := root.Content[providersIdx+1]
		if providersMap.Kind == yaml.MappingNode {
			found := false
			for j := 0; j < len(providersMap.Content); j += 2 {
				if providersMap.Content[j].Value == name {
					providersMap.Content[j+1] = createProviderValue(name, url)
					found = true
					modified = true
					break
				}
			}
			if !found {
				providersMap.Content = append(providersMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
					createProviderValue(name, url),
				)
				modified = true
			}
		}
	} else {
		// proxy-providers section doesn't exist — create it
		providersValue := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		providersValue.Content = append(providersValue.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
			createProviderValue(name, url),
		)
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "proxy-providers", Tag: "!!str"},
			providersValue,
		)
		modified = true
	}

	// Wire the provider into a select-type proxy group's "use" list.
	// Prefer an existing select group; create one if none exists.
	if wireProviderIntoSelectGroup(root, name, hasGroups, groupsIdx, &modified) {
		// modified flag already set by the helper
	} else if !hasGroups {
		// No proxy-groups section at all - create one with a new select group
		groupsList := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		groupsList.Content = append(groupsList.Content, buildSelectGroup(name))
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "proxy-groups", Tag: "!!str"},
			groupsList,
		)
		modified = true
	}

	if !modified {
		return configPath, nil
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// wireProviderIntoSelectGroup finds the first select-type group with a "use" field
// in the config and adds the provider to its "use" list. Returns true if the
// provider was wired into an existing group (modified may still be false if it
// was already present).
func wireProviderIntoSelectGroup(root *yaml.Node, providerName string, hasGroups bool, groupsIdx int, modified *bool) bool {
	if !hasGroups {
		return false
	}

	groupsList := root.Content[groupsIdx+1]
	if groupsList.Kind != yaml.SequenceNode {
		return false
	}

	var targetGroup *yaml.Node
	for _, groupNode := range groupsList.Content {
		if groupNode.Kind != yaml.MappingNode {
			continue
		}

		groupType := ""
		hasUseField := false
		for j := 0; j < len(groupNode.Content); j += 2 {
			switch groupNode.Content[j].Value {
			case "type":
				groupType = groupNode.Content[j+1].Value
			case "use":
				hasUseField = true
			}
		}

		if groupType == GroupTypeSelect && hasUseField {
			targetGroup = groupNode
			break
		}
	}

	if targetGroup == nil {
		// No suitable group exists - create one
		newGroup := buildSelectGroup(providerName)
		groupsList.Content = append(groupsList.Content, newGroup)
		*modified = true
		return true
	}

	// Wire provider into the existing group's "use" list
	for k := 0; k < len(targetGroup.Content); k += 2 {
		if targetGroup.Content[k].Value == "use" {
			useList := targetGroup.Content[k+1]
			if useList.Kind == yaml.SequenceNode {
				for _, item := range useList.Content {
					if item.Value == providerName {
						return true // already present
					}
				}
				useList.Content = append(useList.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: providerName, Tag: "!!str"},
				)
				*modified = true
			}
			break
		}
	}
	return true
}

// buildSelectGroup creates a new select-type proxy group with the given provider in its "use" list.
func buildSelectGroup(providerName string) *yaml.Node {
	useNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	useNode.Content = append(useNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: providerName, Tag: "!!str"},
	)

	group := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	group.Content = append(group.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: providerName, Tag: "!!str"},
	)
	group.Content = append(group.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "type", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: GroupTypeSelect, Tag: "!!str"},
	)
	group.Content = append(group.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "use", Tag: "!!str"},
		useNode,
	)

	return group
}

func createProviderValue(name, url string) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "http", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "url", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: url, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "interval", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "3600", Tag: "!!int"},
			{Kind: yaml.ScalarNode, Value: "health-check", Tag: "!!str"},
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "enable", Tag: "!!str"},
					{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
					{Kind: yaml.ScalarNode, Value: "url", Tag: "!!str"},
					{Kind: yaml.ScalarNode, Value: "https://www.gstatic.com/generate_204", Tag: "!!str"},
					{Kind: yaml.ScalarNode, Value: "interval", Tag: "!!str"},
					{Kind: yaml.ScalarNode, Value: "300", Tag: "!!int"},
				},
			},
		},
	}
}

// RuleProviderEntry is a single rule-provider definition
type RuleProviderEntry struct {
	Name     string
	Type     string
	Behavior string
	URL      string
	Path     string
	Interval int
}

// createRuleProviderValue builds a yaml.MappingNode for a rule-provider entry.
func createRuleProviderValue(name, url, behavior string, interval int) *yaml.Node {
	// Sanitize name to prevent path traversal (e.g. "../../etc/passwd")
	safeName := strings.ReplaceAll(name, "..", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	path := fmt.Sprintf("./ruleset/%s.yaml", safeName)
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "http", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "behavior", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: behavior, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "url", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: url, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "path", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: path, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "interval", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", interval), Tag: "!!int"},
		},
	}
}

// GetRuleProvidersFromConfig reads rule-providers from the mihomo config.yaml.
func GetRuleProvidersFromConfig() ([]RuleProviderEntry, error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	doc, err := parseConfig(configPath)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rule-providers")
	rpIdx, ok := indices["rule-providers"]
	if !ok {
		return nil, nil
	}

	providersMap := root.Content[rpIdx+1]
	if providersMap.Kind != yaml.MappingNode {
		return nil, nil
	}

	var entries []RuleProviderEntry
	for j := 0; j < len(providersMap.Content); j += 2 {
		name := providersMap.Content[j].Value
		val := providersMap.Content[j+1]
		if val.Kind != yaml.MappingNode {
			continue
		}
		entry := RuleProviderEntry{Name: name}
		for k := 0; k < len(val.Content); k += 2 {
			switch val.Content[k].Value {
			case "type":
				entry.Type = val.Content[k+1].Value
			case "behavior":
				entry.Behavior = val.Content[k+1].Value
			case "url":
				entry.URL = val.Content[k+1].Value
			case "path":
				entry.Path = val.Content[k+1].Value
			case "interval":
				if v, err := strconv.Atoi(val.Content[k+1].Value); err == nil {
					entry.Interval = v
				}
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// AddRuleProviderToConfig adds a new rule-provider and a corresponding RULE-SET rule.
func AddRuleProviderToConfig(name, url, behavior string, interval int) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rule-providers", "rules")
	rpIdx, hasRP := indices["rule-providers"]
	rulesIdx, hasRules := indices["rules"]

	if hasRP {
		providersMap := root.Content[rpIdx+1]
		if providersMap.Kind == yaml.MappingNode {
			for j := 0; j < len(providersMap.Content); j += 2 {
				if providersMap.Content[j].Value == name {
					return "", fmt.Errorf("rule-provider %q 已存在", name)
				}
			}
			providersMap.Content = append(providersMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
				createRuleProviderValue(name, url, behavior, interval),
			)
		}
	} else {
		providersValue := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		providersValue.Content = append(providersValue.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: name, Tag: "!!str"},
			createRuleProviderValue(name, url, behavior, interval),
		)
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "rule-providers", Tag: "!!str"},
			providersValue,
		)
	}

	// Add RULE-SET entry in rules section if it doesn't already exist
	ruleEntry := fmt.Sprintf("RULE-SET,%s,DIRECT", name)
	if hasRules {
		rulesList := root.Content[rulesIdx+1]
		if rulesList.Kind == yaml.SequenceNode {
			exists := false
			for _, ruleNode := range rulesList.Content {
				if ruleNode.Value == ruleEntry {
					exists = true
					break
				}
			}
			if !exists {
				rulesList.Content = append(rulesList.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: ruleEntry, Tag: "!!str"},
				)
			}
		}
	} else {
		rulesList := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		rulesList.Content = append(rulesList.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: ruleEntry, Tag: "!!str"},
		)
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "rules", Tag: "!!str"},
			rulesList,
		)
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// DeleteRuleProviderFromConfig removes a rule-provider and its associated RULE-SET rule.
func DeleteRuleProviderFromConfig(name string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rule-providers", "rules")
	rpIdx, hasRP := indices["rule-providers"]
	rulesIdx, hasRules := indices["rules"]

	if !hasRP {
		return "", fmt.Errorf("配置文件中没有 rule-providers 字段")
	}

	providersMap := root.Content[rpIdx+1]
	if providersMap.Kind != yaml.MappingNode {
		return "", fmt.Errorf("rule-providers 格式错误")
	}

	found := false
	for j := 0; j < len(providersMap.Content); j += 2 {
		if providersMap.Content[j].Value == name {
			providersMap.Content = append(providersMap.Content[:j], providersMap.Content[j+2:]...)
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("未找到 rule-provider %q", name)
	}

	// Remove associated RULE-SET rule
	if hasRules {
		rulesList := root.Content[rulesIdx+1]
		if rulesList.Kind == yaml.SequenceNode {
			ruleEntry := fmt.Sprintf("RULE-SET,%s,", name)
			for idx, ruleNode := range rulesList.Content {
				if strings.Contains(ruleNode.Value, ruleEntry) {
					rulesList.Content = append(rulesList.Content[:idx], rulesList.Content[idx+1:]...)
					break
				}
			}
		}
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// UpdateRuleProviderInConfig updates an existing rule-provider.
func UpdateRuleProviderInConfig(name, url, behavior string, interval int) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rule-providers")
	rpIdx, ok := indices["rule-providers"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rule-providers 字段")
	}

	providersMap := root.Content[rpIdx+1]
	if providersMap.Kind != yaml.MappingNode {
		return "", fmt.Errorf("rule-providers 格式错误")
	}

	found := false
	for j := 0; j < len(providersMap.Content); j += 2 {
		if providersMap.Content[j].Value == name {
			providersMap.Content[j+1] = createRuleProviderValue(name, url, behavior, interval)
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("未找到 rule-provider %q", name)
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// formatRuleLine formats a rule line for the config file.
// When Payload is empty, uses 2-part format (TYPE,PROXY) for compatibility
// with Mihomo rule parser, which doesn't accept empty-payload 3-part entries.
func formatRuleLine(ruleType, payload, proxy string) string {
	if payload == "" {
		return fmt.Sprintf("%s,%s", ruleType, proxy)
	}
	return fmt.Sprintf("%s,%s,%s", ruleType, payload, proxy)
}

// FormatRuleLine is the exported version of formatRuleLine.
func FormatRuleLine(ruleType, payload, proxy string) string {
	return formatRuleLine(ruleType, payload, proxy)
}

// RuleEntry is a single rule to be added in batch.
type RuleEntry struct {
	Type    string
	Payload string
	Proxy   string
}

// BatchAddRulesToConfig adds multiple rules in a single read-modify-write cycle.
// Duplicate rules (identical line) are silently skipped to prevent config bloat.
func BatchAddRulesToConfig(rules []RuleEntry) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}
	if len(rules) == 0 {
		return configPath, nil
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, hasRules := indices["rules"]

	var rulesList *yaml.Node
	if !hasRules {
		rulesList = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "rules", Tag: "!!str"},
			rulesList,
		)
	} else {
		rulesList = root.Content[rulesIdx+1]
		if rulesList.Kind != yaml.SequenceNode {
			return "", fmt.Errorf("rules 格式错误")
		}
	}

	// Build a set of existing rule lines for dedup
	existing := make(map[string]struct{}, len(rulesList.Content))
	for _, item := range rulesList.Content {
		existing[item.Value] = struct{}{}
	}

	for _, rule := range rules {
		ruleLine := formatRuleLine(rule.Type, rule.Payload, rule.Proxy)
		if _, dup := existing[ruleLine]; dup {
			continue // skip duplicate
		}
		rulesList.Content = append(rulesList.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: ruleLine, Tag: "!!str"},
		)
		existing[ruleLine] = struct{}{}
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// RuleProviderBatchEntry is a single rule-provider to be added in batch.
type RuleProviderBatchEntry struct {
	Name     string
	URL      string
	Behavior string
	Interval int
}

// BatchAddRuleProvidersToConfig adds multiple rule-providers in a single read-modify-write cycle.
func BatchAddRuleProvidersToConfig(providers []RuleProviderBatchEntry) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}
	if len(providers) == 0 {
		return configPath, nil
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rule-providers", "rules")
	rpIdx, hasRP := indices["rule-providers"]
	rulesIdx, hasRules := indices["rules"]

	for _, p := range providers {
		if hasRP {
			providersMap := root.Content[rpIdx+1]
			if providersMap.Kind == yaml.MappingNode {
				providersMap.Content = append(providersMap.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: p.Name, Tag: "!!str"},
					createRuleProviderValue(p.Name, p.URL, p.Behavior, p.Interval),
				)
			}
		} else {
			providersValue := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			providersValue.Content = append(providersValue.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: p.Name, Tag: "!!str"},
				createRuleProviderValue(p.Name, p.URL, p.Behavior, p.Interval),
			)
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "rule-providers", Tag: "!!str"},
				providersValue,
			)
			hasRP = true
			rpIdx = len(root.Content) - 2
		}

		ruleEntry := fmt.Sprintf("RULE-SET,%s,PROXY", p.Name)
		if hasRules {
			rulesList := root.Content[rulesIdx+1]
			if rulesList.Kind == yaml.SequenceNode {
				rulesList.Content = append(rulesList.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: ruleEntry, Tag: "!!str"},
				)
			}
		} else {
			rulesList := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			rulesList.Content = append(rulesList.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: ruleEntry, Tag: "!!str"},
			)
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "rules", Tag: "!!str"},
				rulesList,
			)
			hasRules = true
			rulesIdx = len(root.Content) - 2
		}
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// GetGroupDef returns the 'proxies' and 'use' field values for a named proxy group.
func GetGroupDef(groupName string) (proxies []string, use []string, err error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	doc, err := parseConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "proxy-groups")
	groupsIdx, ok := indices["proxy-groups"]
	if !ok {
		return nil, nil, nil
	}

	groupsList := root.Content[groupsIdx+1]
	for _, groupNode := range groupsList.Content {
		if groupNode.Kind != yaml.MappingNode {
			continue
		}
		var nameVal string
		var useNode, proxiesNode *yaml.Node
		for j := 0; j < len(groupNode.Content); j += 2 {
			key := groupNode.Content[j].Value
			val := groupNode.Content[j+1]
			switch key {
			case "name":
				nameVal = val.Value
			case "use":
				useNode = val
			case "proxies":
				proxiesNode = val
			}
		}
		if nameVal != groupName {
			continue
		}

		if proxiesNode != nil && proxiesNode.Kind == yaml.SequenceNode {
			for _, item := range proxiesNode.Content {
				n := strings.TrimSpace(item.Value)
				if n != "" {
					proxies = append(proxies, n)
				}
			}
		}
		if useNode != nil && useNode.Kind == yaml.SequenceNode {
			for _, item := range useNode.Content {
				n := strings.TrimSpace(item.Value)
				if n != "" {
					use = append(use, n)
				}
			}
		}
		break
	}
	return proxies, use, nil
}

// GetRulesFromConfig returns all rules from the config file as strings
func GetRulesFromConfig() ([]string, error) {
	configPath := FindMihomoConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("找不到 mihomo 配置文件")
	}

	configMu.RLock()
	defer configMu.RUnlock()

	doc, err := parseConfig(configPath)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return nil, nil
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return nil, nil
	}

	var rules []string
	for _, item := range rulesList.Content {
		rules = append(rules, item.Value)
	}
	return rules, nil
}

// AddRuleToConfig appends a new rule to the config file
func AddRuleToConfig(ruleType, payload, proxy string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, hasRules := indices["rules"]

	ruleLine := formatRuleLine(ruleType, payload, proxy)
	if !hasRules {
		rulesList := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		rulesList.Content = append(rulesList.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: ruleLine, Tag: "!!str"},
		)
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "rules", Tag: "!!str"},
			rulesList,
		)
	} else {
		rulesList := root.Content[rulesIdx+1]
		if rulesList.Kind != yaml.SequenceNode {
			return "", fmt.Errorf("rules 格式错误")
		}
		rulesList.Content = append(rulesList.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: ruleLine, Tag: "!!str"},
		)
	}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// DeleteRuleFromConfig removes a rule at the given index from the config file
func DeleteRuleFromConfig(index int) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rules 字段")
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("rules 格式错误")
	}

	if index < 0 || index >= len(rulesList.Content) {
		return "", fmt.Errorf("规则索引 %d 超出范围", index)
	}

	rulesList.Content = append(rulesList.Content[:index], rulesList.Content[index+1:]...)

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// DeleteRuleByContent finds a rule by its type/payload/proxy content and deletes it
// from the config file. Returns the config path on success.
// Uses case-insensitive comparison to handle API (PascalCase) vs config (UPPERCASE) differences.
func DeleteRuleByContent(ruleType, payload, proxy string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rules 字段")
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("rules 格式错误")
	}

	target := formatRuleLine(ruleType, payload, proxy)
	targetUpper := strings.ToUpper(target)
	found := -1
	for i, item := range rulesList.Content {
		if strings.ToUpper(item.Value) == targetUpper {
			found = i
			break
		}
	}

	if found < 0 {
		return "", fmt.Errorf("未找到匹配的规则: %s", target)
	}

	rulesList.Content = append(rulesList.Content[:found], rulesList.Content[found+1:]...)

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// MoveRuleToPosition removes the rule at fromIdx and inserts it at toIdx,
// shifting all intervening rules. This supports arbitrary repositioning:
//   - toIdx < fromIdx: move up (insert before current position)
//   - toIdx > fromIdx: move down (insert after current position)
func MoveRuleToPosition(fromIdx, toIdx int) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rules 字段")
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("rules 格式错误")
	}

	n := len(rulesList.Content)
	if fromIdx < 0 || fromIdx >= n {
		return "", fmt.Errorf("来源索引 %d 超出范围 (0-%d)", fromIdx, n-1)
	}
	if toIdx < 0 || toIdx >= n {
		return "", fmt.Errorf("目标索引 %d 超出范围 (0-%d)", toIdx, n-1)
	}
	if fromIdx == toIdx {
		return configPath, nil // no-op
	}

	rulesList.Content = moveNodesToPosition(rulesList.Content, fromIdx, toIdx)

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func moveNodesToPosition(nodes []*yaml.Node, fromIdx, toIdx int) []*yaml.Node {
	node := nodes[fromIdx]
	nodes = append(nodes[:fromIdx], nodes[fromIdx+1:]...)
	if toIdx == len(nodes) {
		return append(nodes, node)
	}
	nodes = append(nodes, nil)
	copy(nodes[toIdx+1:], nodes[toIdx:])
	nodes[toIdx] = node
	return nodes
}

// MoveRuleInConfig swaps the rules at two indices, effectively moving ruleIdx to targetIdx.
// For move-up: targetIdx = ruleIdx - 1
// For move-down: targetIdx = ruleIdx + 1
func MoveRuleInConfig(ruleIdx, targetIdx int) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rules 字段")
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("rules 格式错误")
	}

	if ruleIdx < 0 || ruleIdx >= len(rulesList.Content) {
		return "", fmt.Errorf("规则索引 %d 超出范围", ruleIdx)
	}
	if targetIdx < 0 || targetIdx >= len(rulesList.Content) {
		return "", fmt.Errorf("目标索引 %d 超出范围", targetIdx)
	}

	// Swap the two rule nodes in-place
	rulesList.Content[ruleIdx], rulesList.Content[targetIdx] = rulesList.Content[targetIdx], rulesList.Content[ruleIdx]

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

// UpdateRuleInConfig updates a rule at the given index
func UpdateRuleInConfig(index int, ruleType, payload, proxy string) (configPath string, err error) {
	configMu.Lock()
	defer configMu.Unlock()

	configPath = FindMihomoConfigPath()
	if configPath == "" {
		return "", fmt.Errorf("找不到 mihomo 配置文件")
	}

	doc, err := parseConfig(configPath)
	if err != nil {
		return "", err
	}

	root := doc.Content[0]
	indices := findKeysInMapping(root, "rules")
	rulesIdx, ok := indices["rules"]
	if !ok {
		return "", fmt.Errorf("配置文件中没有 rules 字段")
	}

	rulesList := root.Content[rulesIdx+1]
	if rulesList.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("rules 格式错误")
	}

	if index < 0 || index >= len(rulesList.Content) {
		return "", fmt.Errorf("规则索引 %d 超出范围", index)
	}

	ruleLine := formatRuleLine(ruleType, payload, proxy)
	rulesList.Content[index] = &yaml.Node{Kind: yaml.ScalarNode, Value: ruleLine, Tag: "!!str"}

	if err := writeConfig(doc, configPath); err != nil {
		return "", err
	}

	return configPath, nil
}
