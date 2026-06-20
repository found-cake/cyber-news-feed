package rssdoc

import (
	"errors"
	"testing"
	"time"

	"github.com/found-cake/cyber-news-feed/internal/urlnorm"
)

func Test_MergeFailure_preserves_articles_and_updates_status_when_source_fails(t *testing.T) {
	// Given
	now := time.Date(2026, 6, 20, 4, 0, 8, 0, time.UTC)
	lastSuccess := "2026-06-20T03:00:00Z"
	publishedAt := "2026-06-20T02:00:00Z"
	existing := Document{
		SchemaVersion: 1,
		Source:        "darkreading",
		Status: Status{
			OK:            true,
			LastSuccessAt: &lastSuccess,
		},
		Articles: []Article{{
			ID:           urlnorm.StableArticleID("https://example.com/old"),
			URL:          "https://example.com/old",
			Title:        "old",
			PublishedAt:  &publishedAt,
			PublishedRaw: "Sat, 20 Jun 2026 02:00:00 GMT",
			Category:     "threat-intelligence",
		}},
	}

	// When
	got := MergeFailure(MergeRequest{
		Existing:      existing,
		Source:        "darkreading",
		Now:           now,
		RetentionDays: 10,
		Err:           errors.New("network down"),
	})

	// Then
	if got.Status.OK {
		t.Fatal("expected failed status")
	}
	if got.Status.LastSuccessAt == nil || *got.Status.LastSuccessAt != lastSuccess {
		t.Fatalf("LastSuccessAt = %#v, want %q", got.Status.LastSuccessAt, lastSuccess)
	}
	if got.Status.LastErrorAt == nil || *got.Status.LastErrorAt != now.Format(time.RFC3339) {
		t.Fatalf("LastErrorAt = %#v, want %q", got.Status.LastErrorAt, now.Format(time.RFC3339))
	}
	if len(got.Articles) != 1 || got.Articles[0].URL != "https://example.com/old" {
		t.Fatalf("articles changed on failure: %#v", got.Articles)
	}
}

func Test_MergeSuccess_orders_rss_items_before_existing_dedupes_and_prunes_old_items(t *testing.T) {
	// Given
	now := time.Date(2026, 6, 20, 4, 0, 8, 0, time.UTC)
	recent := "2026-06-19T22:00:00Z"
	old := "2026-06-01T00:00:00Z"
	existing := Document{
		SchemaVersion: 1,
		Source:        "darkreading",
		Articles: []Article{
			{
				ID:          urlnorm.StableArticleID("https://example.com/existing"),
				URL:         "https://example.com/existing",
				Title:       "existing",
				PublishedAt: &recent,
			},
			{
				ID:          urlnorm.StableArticleID("https://example.com/old"),
				URL:         "https://example.com/old",
				Title:       "old",
				PublishedAt: &old,
			},
		},
	}
	fetched := []Article{
		{
			URL:         "https://example.com/new/#fragment",
			Title:       "new",
			PublishedAt: &recent,
		},
		{
			URL:         "https://example.com/existing/",
			Title:       "existing refreshed",
			PublishedAt: &recent,
		},
	}

	// When
	got := MergeSuccess(MergeRequest{
		Existing:      existing,
		Source:        "darkreading",
		Now:           now,
		RetentionDays: 10,
		Fetched:       fetched,
	})

	// Then
	if !got.Status.OK {
		t.Fatal("expected successful status")
	}
	if len(got.Articles) != 2 {
		t.Fatalf("len(Articles) = %d, want 2: %#v", len(got.Articles), got.Articles)
	}
	if got.Articles[0].URL != "https://example.com/new" {
		t.Fatalf("first article URL = %q", got.Articles[0].URL)
	}
	if got.Articles[1].Title != "existing refreshed" {
		t.Fatalf("deduped article title = %q", got.Articles[1].Title)
	}
}
