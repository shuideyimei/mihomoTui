package ui

import "mihomoTui/internal/i18n"

// PageInfo is the shared source of truth for page navigation and help text.
type PageInfo struct {
	ID        string
	LabelKey  string
	Function  string
	Alt       string
	Separator bool
}

func DefaultPageInfo() []PageInfo {
	return []PageInfo{
		{ID: "dashboard", LabelKey: "nav.dashboard", Function: "F1", Alt: "1"},
		{ID: "proxies", LabelKey: "nav.proxies", Function: "F2", Alt: "2"},
		{ID: "connections", LabelKey: "nav.connections", Function: "F3", Alt: "3"},
		{ID: "logs", LabelKey: "nav.logs", Function: "F4", Alt: "4"},
		{Separator: true, LabelKey: "", Function: "", Alt: ""},
		{ID: "configmgr", LabelKey: "nav.profiles", Function: "F5", Alt: "5"}, // Renamed locally
		{ID: "subscriptions", LabelKey: "nav.subscriptions", Function: "F6", Alt: "6"},
		{ID: "editor", LabelKey: "nav.editor", Function: "F7", Alt: "7"},     // Merged Groups, Rules, Providers
		{ID: "settings", LabelKey: "nav.settings", Function: "F8", Alt: "8"}, // Merged Config & Settings
	}
}

func (p PageInfo) Label() string {
	return i18n.T(p.LabelKey)
}
