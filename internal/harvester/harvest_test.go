package harvester

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/source"
)

func Test_runWithSources_retries_failed_sources_after_first_pass_finishes(t *testing.T) {
	// Given
	order := []string{}
	firstRequests := 0
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstRequests++
		if r.Header.Get("Cache-Control") == "no-cache" {
			order = append(order, "first:retry")
		} else {
			order = append(order, "first:first-pass")
		}
		if firstRequests == 1 {
			http.NotFound(w, r)
			return
		}
		writeTestRSS(w, "Recovered first", "https://example.com/first")
	}))
	defer first.Close()

	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "second:first-pass")
		writeTestRSS(w, "Second", "https://example.com/second")
	}))
	defer second.Close()

	cfg := Config{
		OutputDir:     t.TempDir(),
		RetentionDays: 10,
		Client:        first.Client(),
	}
	sources := []source.Config{
		{Name: "first", Feeds: []source.Feed{{URL: first.URL}}},
		{Name: "second", Feeds: []source.Feed{{URL: second.URL}}},
	}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 2 || summary.Failed != 0 {
		t.Fatalf("Summary = %#v", summary)
	}
	wantOrder := []string{"first:first-pass", "second:first-pass", "first:retry"}
	if len(order) != len(wantOrder) {
		t.Fatalf("order = %#v, want %#v", order, wantOrder)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("order = %#v, want %#v", order, wantOrder)
		}
	}
	assertSourceOK(t, cfg.OutputDir, "first", 1)
	assertSourceOK(t, cfg.OutputDir, "second", 1)
}

func Test_runWithSources_writes_securityweek_image_metadata(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<image>
  <url>https://www.securityweek.com/wp-content/uploads/2023/01/cropped-SecurityWeek-Icon-32x32.jpeg</url>
  <title>SecurityWeek</title>
  <link>https://www.securityweek.com/</link>
  <width>32</width>
  <height>32</height>
</image>
<item>
  <title>SecurityWeek image</title>
  <link>https://www.securityweek.com/example</link>
</item></channel></rss>`))
	}))
	defer server.Close()

	cfg := Config{
		OutputDir:     t.TempDir(),
		RetentionDays: 10,
		Client:        server.Client(),
	}
	sources := []source.Config{
		{Name: "securityweek", Feeds: []source.Feed{{URL: server.URL}}},
	}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 1 || summary.Failed != 0 {
		t.Fatalf("Summary = %#v", summary)
	}
	assertSecurityWeekImage(t, cfg.OutputDir, "https://www.securityweek.com/wp-content/uploads/2023/01/cropped-SecurityWeek-Icon-32x32.jpeg")
}

func writeTestRSS(w http.ResponseWriter, title string, link string) {
	w.Header().Set("Content-Type", "application/rss+xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><item>
  <title>` + title + `</title>
  <link>` + link + `</link>
</item></channel></rss>`))
}

func assertSourceOK(t *testing.T, outputDir string, name string, articles int) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, name+".json"))
	if err != nil {
		t.Fatalf("read %s json: %v", name, err)
	}
	var doc struct {
		Status struct {
			OK bool `json:"ok"`
		} `json:"status"`
		Articles []struct{} `json:"articles"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode %s json: %v", name, err)
	}
	if !doc.Status.OK || len(doc.Articles) != articles {
		t.Fatalf("%s document status/articles = ok:%v articles:%d", name, doc.Status.OK, len(doc.Articles))
	}
}

func assertSecurityWeekImage(t *testing.T, outputDir string, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, "securityweek.json"))
	if err != nil {
		t.Fatalf("read securityweek json: %v", err)
	}
	var doc struct {
		Articles []struct {
			SourceMetadata struct {
				SecurityWeek struct {
					Image struct {
						URL    string `json:"url"`
						Title  string `json:"title"`
						Link   string `json:"link"`
						Width  string `json:"width"`
						Height string `json:"height"`
					} `json:"image"`
				} `json:"securityweek"`
			} `json:"source_metadata"`
		} `json:"articles"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode securityweek json: %v", err)
	}
	if len(doc.Articles) != 1 || doc.Articles[0].SourceMetadata.SecurityWeek.Image.URL != want {
		t.Fatalf("securityweek image metadata = %#v, want %q", doc.Articles, want)
	}
}
