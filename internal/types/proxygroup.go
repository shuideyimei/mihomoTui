package types

// ProxyGroupDef is a single proxy-group definition from a parsed config.
type ProxyGroupDef struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies,omitempty"`
	Use      []string `yaml:"use,omitempty"`
	URL      string   `yaml:"url,omitempty"`
	Interval int      `yaml:"interval,omitempty"`
	Filter   string   `yaml:"filter,omitempty"`
}

// ProxyNode is a single proxy (node) definition.
type ProxyNode struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Server   string                 `yaml:"server,omitempty"`
	Port     int                    `yaml:"port,omitempty"`
	Cipher   string                 `yaml:"cipher,omitempty"`
	Password string                 `yaml:"password,omitempty"`
	UDP      bool                   `yaml:"udp,omitempty"`
	Raw      map[string]interface{} `yaml:",inline"`
}

// RuleProviderDef is a single rule-provider block.
type RuleProviderDef struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior"`
	URL      string `yaml:"url,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Interval int    `yaml:"interval,omitempty"`
}

// DependencyReport describes the result of analyzing proxy-group dependencies.
type DependencyReport struct {
	Groups      []ProxyGroupDef     `json:"groups"`
	Missing     []string            `json:"missing"`    // proxy names referenced but not defined
	Available   []string            `json:"available"`  // proxy names that exist
	Providers   []string            `json:"providers"`  // provider names referenced
	GroupRefs   map[string][]string `json:"group_refs"` // group name -> proxy names it references
	CanAutoFill bool                `json:"can_auto_fill"`
}

// ImportResult contains the outcome of importing proxy groups or a subscription.
type ImportResult struct {
	GroupsAdded   []string        `json:"groups_added"`
	GroupsUpdated []string        `json:"groups_updated"`
	GroupsSkipped []string        `json:"groups_skipped"`
	ProxiesAdded  int             `json:"proxies_added"`
	RulesAdded    int             `json:"rules_added"`
	Conflicts     []ConflictEntry `json:"conflicts,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
}

// ConflictEntry describes a single conflict between two config values.
type ConflictEntry struct {
	Field      string      `json:"field"`
	Local      interface{} `json:"local"`
	Imported   interface{} `json:"imported"`
	Resolution string      `json:"resolution,omitempty"` // "keep_local", "use_imported", "manual"
}

// ConflictResolution represents the user's choice for a conflict.
type ConflictResolution string

const (
	ResolutionKeepLocal   ConflictResolution = "keep_local"
	ResolutionUseImported ConflictResolution = "use_imported"
	ResolutionManual      ConflictResolution = "manual"
	ResolutionMerge       ConflictResolution = "merge"
)

// ProxyGroupImportMode controls how duplicate groups are handled during import.
type ProxyGroupImportMode string

const (
	ImportModeMerge   ProxyGroupImportMode = "merge"   // merge proxies from both sides
	ImportModeReplace ProxyGroupImportMode = "replace" // replace local with imported
	ImportModeSkip    ProxyGroupImportMode = "skip"    // keep local, skip imported
	ImportModeRename  ProxyGroupImportMode = "rename"  // rename imported group
)
