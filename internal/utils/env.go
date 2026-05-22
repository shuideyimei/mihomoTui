package utils

import "os"

// GetEnvWithDefault gets environment variable with default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
