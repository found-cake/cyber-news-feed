package harvester

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/source"
)

func Test_runWithSources_retries_non_BoanNews_source_after_forbidden_response(t *testing.T) {
	// Given
	var feedRequests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if feedRequests.Add(1) == 1 {
			return htmlResponse(http.StatusForbidden, "blocked"), nil
		}
		return rssResponse(`<item><title>Recovered</title><link>https://example.com/recovered</link></item>`), nil
	})}
	cfg := Config{OutputDir: t.TempDir(), RetentionDays: 10, Client: client}
	sources := []source.Config{{
		Name:  "other",
		Feeds: []source.Feed{{URL: "https://feed.test/rss"}},
	}}

	// When
	summary, err := runWithSources(
		context.Background(),
		cfg,
		slog.New(slog.NewTextHandler(os.Stderr, nil)),
		sources,
	)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %#v, want one recovered source", summary)
	}
	if feedRequests.Load() != 2 {
		t.Fatalf("feed requests = %d, want 2", feedRequests.Load())
	}
}
