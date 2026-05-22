package parser

import (
	"strconv"
)

// parseInt safely converts a string to int, returning 0 on error.
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// toInt converts an interface{} to int, handling both float64 (JSON) and string.
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		return parseInt(val)
	default:
		return 0
	}
}
