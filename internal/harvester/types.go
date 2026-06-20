package harvester

import "net/http"

const (
	defaultOutputDir     = "data/rss"
	defaultRetentionDays = 10
)

type Config struct {
	OutputDir     string
	RetentionDays int
	Client        *http.Client
}

type Summary struct {
	Processed int
	Failed    int
}
