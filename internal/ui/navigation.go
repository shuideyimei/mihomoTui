package ui

import "mihomoTui/internal/i18n"

// PageInfo is the shared source of truth for page navigation and help text.
type PageInfo struct {
	ID       string
	LabelKey string
	Function string
	Alt      string
}

func DefaultPageInfo() []PageInfo {
	return []PageInfo{
		{ID: "dashboard", LabelKey: "nav.dashboard", Function: "F1", Alt: "Alt+D"},
		{ID: "proxies", LabelKey: "nav.proxies", Function: "F2", Alt: "Alt+P"},
		{ID: "connections", LabelKey: "nav.connections", Function: "F3", Alt: "Alt+R"},
		{ID: "config", LabelKey: "nav.config", Function: "F4", Alt: "Alt+C"},
		{ID: "logs", LabelKey: "nav.logs", Function: "F5", Alt: "Alt+L"},
		{ID: "subscriptions", LabelKey: "nav.subscriptions", Function: "F6", Alt: "Alt+S"},
		{ID: "proxygroups", LabelKey: "nav.proxygroups", Function: "F7", Alt: "Alt+G"},
		{ID: "rules", LabelKey: "nav.rules", Function: "F8", Alt: "Alt+U"},
		{ID: "ruleproviders", LabelKey: "nav.ruleproviders", Function: "F9", Alt: "Alt+T"},
		{ID: "settings", LabelKey: "nav.settings", Function: "F10", Alt: "Alt+K"},
		{ID: "configmgr", LabelKey: "nav.configmgr", Function: "F11", Alt: "Alt+M"},
	}
}

func (p PageInfo) Label() string {
	return i18n.T(p.LabelKey)
}
