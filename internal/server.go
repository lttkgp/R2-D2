package main

import (
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func scheduleJobs(logger *zap.Logger) {
	cronLogger := cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cronLogger)))
	_, err := c.AddFunc("@every 10h", func() {
		fetchLatestError := FetchLatestPosts(logger)
		if fetchLatestError != nil {
			logger.Error("Fetching latest posts failed", zap.Error(fetchLatestError))
		}
	})
	if err != nil {
		logger.Fatal("Unable to start FetchLatestPosts scheduler", zap.Error(err))
	}
	_, err = c.AddFunc("@every 10s", func() {
		dispatchError := DispatchFreshPosts(logger)
		if dispatchError != nil {
			logger.Error("Dispatching fresh posts failed", zap.Error(dispatchError))
		}
	})
	if err != nil {
		logger.Fatal("Unable to start DispatchFreshPosts scheduler", zap.Error(err))
	}
	c.Start()
}

func main() {
	// Logger setup
	logger := GetLogger()
	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Warn("Unable to gracefully flush buffered log entries", zap.Error(err))
		}
	}()

	scheduleJobs(logger)
	initializeAPIServer(logger)
}
