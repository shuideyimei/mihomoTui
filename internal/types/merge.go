package types

// MergeMode controls how two ConfigDocuments are merged for a given top-level key.
type MergeMode string

const (
	MergeModeReplace MergeMode = "replace" // overlay entirely replaces base
	MergeModeAppend  MergeMode = "append"  // overlay items appended to base
	MergeModeMerge   MergeMode = "merge"   // deep-merge by key (default for maps)
	MergeModePrepend MergeMode = "prepend" // overlay items prepended to base
	MergeModeSkip    MergeMode = "skip"    // keep base, skip overlay
)

// MergeStrategy defines per-field merge behavior.
type MergeStrategy struct {
	Proxies        MergeMode `json:"proxies"`
	ProxyGroups    MergeMode `json:"proxy_groups"`
	Rules          MergeMode `json:"rules"`
	RuleProviders  MergeMode `json:"rule_providers"`
	ProxyProviders MergeMode `json:"proxy_providers"`
	DNS            MergeMode `json:"dns"`
	Tun            MergeMode `json:"tun"`
	Others         MergeMode `json:"others"` // extra/top-level scalars
}

// DefaultMergeStrategy returns a sane default strategy.
// - Proxy providers: replace (subscription is authoritative)
// - Proxy groups: merge (preserve user customization, add new)
// - Rules: append (subscription rules go after user rules)
// - DNS, Tun: skip (user's DNS/Tun config takes priority)
func DefaultMergeStrategy() MergeStrategy {
	return MergeStrategy{
		Proxies:        MergeModeReplace,
		ProxyGroups:    MergeModeMerge,
		Rules:          MergeModeAppend,
		RuleProviders:  MergeModeMerge,
		ProxyProviders: MergeModeReplace,
		DNS:            MergeModeSkip,
		Tun:            MergeModeSkip,
		Others:         MergeModeReplace,
	}
}

// MergeResult holds the outcome of a merge operation.
type MergeResult struct {
	Document  *ConfigDocument `json:"document"`
	Conflicts []ConflictEntry `json:"conflicts,omitempty"`
	Strategy  MergeStrategy   `json:"strategy"`
	Warning   []string        `json:"warnings,omitempty"`
}

// MergePreview shows what would change without applying it.
type MergePreview struct {
	Added     []string        `json:"added"`
	Removed   []string        `json:"removed"`
	Modified  []string        `json:"modified"`
	Conflicts []ConflictEntry `json:"conflicts,omitempty"`
	Strategy  MergeStrategy   `json:"strategy"`
}
