package parser

import "strings"

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func startsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func toLower(s string) string {
	return strings.ToLower(s)
}
