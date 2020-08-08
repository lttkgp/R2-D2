package utils

import "os"

// GetEnv Get the environment variable with key `key`
func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
