package rssjson

import (
	"encoding/json"
	"testing"
)

func Test_Article_UnmarshalJSON_promotes_legacy_category_to_categories(t *testing.T) {
	// Given
	input := []byte(`{
		"id":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"url":"https://example.com/a",
		"title":"legacy",
		"published_at":null,
		"published_raw":"",
		"category":"threat-intelligence"
	}`)

	// When
	var got Article
	err := json.Unmarshal(input, &got)

	// Then
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if len(got.Categories) != 1 || got.Categories[0] != "threat-intelligence" {
		t.Fatalf("Categories = %#v", got.Categories)
	}
}

func Test_Article_MarshalJSON_writes_categories_without_legacy_category(t *testing.T) {
	// Given
	article := Article{
		ID:             "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		URL:            "https://example.com/a",
		Title:          "new",
		Categories:     []string{"Security", "Vulnerability"},
		Authors:        []Author{},
		Media:          []Media{},
		SourceMetadata: SourceMetadata{},
	}

	// When
	got, err := json.Marshal(article)

	// Then
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("decode marshaled article: %v", err)
	}
	if _, ok := decoded["category"]; ok {
		t.Fatalf("legacy category was marshaled: %s", got)
	}
	if _, ok := decoded["categories"]; !ok {
		t.Fatalf("categories missing from marshaled article: %s", got)
	}
}

func Test_Document_UnmarshalJSON_promotes_legacy_flat_source_metadata(t *testing.T) {
	// Given
	input := []byte(`{
		"schema_version": 1,
		"source": "cybersecuritynews",
		"updated_at": "2026-06-20T00:00:00Z",
		"retention_days": 10,
		"status": {
			"ok": true,
			"last_success_at": "2026-06-20T00:00:00Z",
			"last_error_at": null,
			"last_error": null
		},
		"articles": [{
			"id":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"url":"https://example.com/a",
			"title":"legacy",
			"published_at":null,
			"published_raw":"",
			"categories":[],
			"description":"",
			"content_encoded":"<p>legacy body</p>",
			"feed_id":"",
			"authors":[],
			"media":[],
			"source_metadata": {
				"guid_is_permalink": "false",
				"post_id": "123"
			}
		}]
	}`)

	// When
	var got Document
	err := json.Unmarshal(input, &got)

	// Then
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	metadata := got.Articles[0].SourceMetadata.CybersecurityNews
	if metadata == nil {
		t.Fatal("CybersecurityNews metadata missing")
	}
	if metadata.GUIDIsPermalink != "false" || metadata.PostID != "123" || metadata.ContentEncoded != "<p>legacy body</p>" {
		t.Fatalf("CybersecurityNews metadata = %#v", metadata)
	}
}
