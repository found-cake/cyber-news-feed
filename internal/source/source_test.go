package source

import (
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/feed"
)

func Test_ArticleFromItem_sets_darkreading_category_when_url_matches_allowed_slug(t *testing.T) {
	// Given
	source := Config{Name: "darkreading", Kind: DarkReading}
	item := feed.Item{
		Title: "Threat report",
		URL:   "https://www.darkreading.com/threat-intelligence/report/",
	}

	// When
	got, include := ArticleFromItem(source, Feed{}, item)

	// Then
	if !include {
		t.Fatal("expected darkreading item to be included")
	}
	if len(got.Categories) != 1 || got.Categories[0] != "threat-intelligence" {
		t.Fatalf("Categories = %#v, want threat-intelligence", got.Categories)
	}
}

func Test_ArticleFromItem_preserves_rss_categories_and_adds_filter_category(t *testing.T) {
	// Given
	source := Config{Name: "thehackernews", Kind: TheHackerNews}
	sourceFeed := Feed{Category: "vulnerability"}
	item := feed.Item{
		Title:      "Vulnerability report",
		URL:        "https://example.com/vuln",
		Categories: []string{"Threat Intelligence", "vulnerability"},
	}

	// When
	got, include := ArticleFromItem(source, sourceFeed, item)

	// Then
	if !include {
		t.Fatal("expected thehackernews item to be included")
	}
	want := []string{"vulnerability", "Threat Intelligence"}
	if len(got.Categories) != len(want) {
		t.Fatalf("Categories = %#v, want %#v", got.Categories, want)
	}
	for i := range want {
		if got.Categories[i] != want[i] {
			t.Fatalf("Categories = %#v, want %#v", got.Categories, want)
		}
	}
}

func Test_ArticleFromItem_preserves_feed_metadata_fields(t *testing.T) {
	// Given
	source := Config{Name: "cybersecuritynews"}
	item := feed.Item{
		Title:          "Rich article",
		URL:            "https://example.com/rich",
		Description:    "short summary",
		ContentEncoded: "<p>body</p>",
		FeedID:         "source-123",
		Authors:        []feed.Author{{Name: "Alice"}},
		Media:          []feed.Media{{URL: "https://example.com/image.jpg", Kind: "content", Medium: "image"}},
		SourceMetadata: feed.SourceMetadata{GUIDIsPermalink: "false", PostID: "987"},
	}

	// When
	got, include := ArticleFromItem(source, Feed{}, item)

	// Then
	if !include {
		t.Fatal("expected article to be included")
	}
	if got.Description != "short summary" {
		t.Fatalf("Description = %q", got.Description)
	}
	if got.FeedID != "source-123" || got.SourceMetadata.CybersecurityNews == nil || got.SourceMetadata.CybersecurityNews.PostID != "987" || got.SourceMetadata.CybersecurityNews.ContentEncoded != "<p>body</p>" {
		t.Fatalf("source metadata not preserved: %#v", got)
	}
	if len(got.Authors) != 1 || got.Authors[0].Name != "Alice" {
		t.Fatalf("Authors = %#v", got.Authors)
	}
	if len(got.Media) != 1 || got.Media[0].URL != "https://example.com/image.jpg" {
		t.Fatalf("Media = %#v", got.Media)
	}
}

func Test_ArticleFromItem_excludes_bleepingcomputer_without_security_category(t *testing.T) {
	// Given
	source := Config{Name: "bleepingcomputer", Kind: BleepingComputer}
	item := feed.Item{
		Title:      "Software update",
		URL:        "https://www.bleepingcomputer.com/news/software/update/",
		Categories: []string{"Software"},
	}

	// When
	_, include := ArticleFromItem(source, Feed{}, item)

	// Then
	if include {
		t.Fatal("expected non-security bleepingcomputer item to be excluded")
	}
}

func Test_ArticleFromItem_excludes_stepsecurity_product_promotions(t *testing.T) {
	// Given
	source := Config{Name: "stepsecurity", Kind: StepSecurity}
	item := feed.Item{
		Title: "Harden-Runner product update",
		URL:   "https://www.stepsecurity.io/blog/harden-runner-product-update",
	}

	// When
	_, include := ArticleFromItem(source, Feed{}, item)

	// Then
	if include {
		t.Fatal("expected stepsecurity promotional item to be excluded")
	}
}
