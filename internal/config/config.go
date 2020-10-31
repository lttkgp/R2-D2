package config

import (
	"log"
	"os"

	"go.uber.org/zap"
)

// GetEnv returns the environment variable with key `key`
func GetEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// GetLogger fetches the global logging config
func GetLogger() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to initialize Zap logger: %v", err)
	}
	return logger
}
