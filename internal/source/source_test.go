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
	if got.Category != "threat-intelligence" {
		t.Fatalf("Category = %q, want threat-intelligence", got.Category)
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
