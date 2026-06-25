package jsonstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/found-cake/cyber-news-feed/pkg/rssjson"
)

func Test_Write_preserves_literal_html_characters(t *testing.T) {
	// Given
	outputDir := t.TempDir()
	doc := rssjson.Document{
		SchemaVersion: rssjson.SchemaVersion,
		Source:        "cybersecuritynews",
		Articles: []rssjson.Article{{
			ID:          "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			URL:         "https://example.com/article",
			Title:       "HTML",
			Categories:  []string{},
			Description: "<p>short & useful</p>",
			Authors:     []rssjson.Author{},
			Media:       []rssjson.Media{},
			SourceMetadata: rssjson.NewSourceMetadata("cybersecuritynews", rssjson.MetadataObject{
				rssjson.MetadataText("content_encoded", "<div><a href=\"https://example.com?a=1&b=2\">body</a></div>"),
			}),
		}},
	}

	// When
	if err := Write(outputDir, doc); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Then
	written, err := os.ReadFile(filepath.Join(outputDir, "cybersecuritynews.json"))
	if err != nil {
		t.Fatalf("read written JSON: %v", err)
	}
	text := string(written)
	for _, escaped := range []string{`\u003c`, `\u003e`, `\u0026`} {
		if strings.Contains(text, escaped) {
			t.Fatalf("written JSON contains escaped HTML sequence %s: %s", escaped, text)
		}
	}
	for _, literal := range []string{"<p>short & useful</p>", `<a href=\"https://example.com?a=1&b=2\">`} {
		if !strings.Contains(text, literal) {
			t.Fatalf("written JSON missing literal HTML fragment %q: %s", literal, text)
		}
	}
}
