package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/found-cake/cyber-news-feed/internal/harvester"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := harvester.ConfigFromEnv()
	if err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(1)
	}

	summary, err := harvester.Run(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("harvest failed", "error", err)
		os.Exit(1)
	}
	logger.Info("harvest finished", "processed", summary.Processed, "failed", summary.Failed)
}
