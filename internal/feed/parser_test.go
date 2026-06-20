package feed

import (
	"strings"
	"testing"
)

func Test_Parse_reads_rss_dates_and_categories(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><item>
  <title>Alert</title>
  <link>https://example.com/alert/</link>
  <pubDate>Fri, 19 Jun 2026 18:31:04 -0400</pubDate>
  <category>Security</category>
</item></channel></rss>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(got))
	}
	if got[0].PublishedAt == nil || *got[0].PublishedAt != "2026-06-19T22:31:04Z" {
		t.Fatalf("PublishedAt = %#v", got[0].PublishedAt)
	}
	if got[0].PublishedRaw != "Fri, 19 Jun 2026 18:31:04 -0400" {
		t.Fatalf("PublishedRaw = %q", got[0].PublishedRaw)
	}
	if got[0].Categories[0] != "Security" {
		t.Fatalf("Categories = %#v", got[0].Categories)
	}
}

func Test_Parse_reads_atom_links_and_dates(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"><entry>
  <title>Threat Intel</title>
  <link rel="alternate" href="https://example.com/threat" />
  <published>2026-06-19T22:31:04Z</published>
  <category term="Threat Intelligence" />
</entry></feed>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got[0].URL != "https://example.com/threat" {
		t.Fatalf("URL = %q", got[0].URL)
	}
	if got[0].PublishedAt == nil || *got[0].PublishedAt != "2026-06-19T22:31:04Z" {
		t.Fatalf("PublishedAt = %#v", got[0].PublishedAt)
	}
}
