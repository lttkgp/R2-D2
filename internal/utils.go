package main

import (
	"os"

	"github.com/joho/godotenv"
)

// fileExists returns true if  filename is a valid file
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// updateEnvFile writes the key=value pair to a .env file if it exists
func updateEnvFile(key string, value string) {
	envFile := "./.env"
	if fileExists(envFile) {
		envMap, err := godotenv.Read(envFile)
		if err != nil {
			return
		}
		envMap[key] = value

		err = godotenv.Write(envMap, envFile)
		if err != nil {
			return
		}
	}
}
