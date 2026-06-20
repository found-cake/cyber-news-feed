package harvester

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func ConfigFromEnv() (Config, error) {
	retentionDays := defaultRetentionDays
	if raw := os.Getenv("RETENTION_DAYS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse RETENTION_DAYS: %w", err)
		}
		if parsed < 1 {
			return Config{}, fmt.Errorf("RETENTION_DAYS must be positive")
		}
		retentionDays = parsed
	}

	return Config{
		OutputDir:     defaultOutputDir,
		RetentionDays: retentionDays,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}
