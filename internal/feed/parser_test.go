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

func Test_Parse_preserves_rss_description_content_author_media_and_metadata(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
  xmlns:content="http://purl.org/rss/1.0/modules/content/"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:media="http://search.yahoo.com/mrss/"><channel><item>
  <title>Rich RSS</title>
  <link>https://example.com/rich</link>
  <guid isPermaLink="false">source-123</guid>
  <dc:creator>Alice</dc:creator>
  <description><![CDATA[Short <b>summary</b>]]></description>
  <content:encoded><![CDATA[<p>Long content</p>]]></content:encoded>
  <media:thumbnail url="https://example.com/thumb.jpg" />
  <media:content url="https://example.com/image.jpg" medium="image" />
  <post-id>987</post-id>
</item></channel></rss>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	item := got[0]
	if item.Description != "Short <b>summary</b>" {
		t.Fatalf("Description = %q", item.Description)
	}
	if item.ContentEncoded != "<p>Long content</p>" {
		t.Fatalf("ContentEncoded = %q", item.ContentEncoded)
	}
	if item.FeedID != "source-123" {
		t.Fatalf("FeedID = %q", item.FeedID)
	}
	if len(item.Authors) != 1 || item.Authors[0].Name != "Alice" {
		t.Fatalf("Authors = %#v", item.Authors)
	}
	if len(item.Media) != 2 {
		t.Fatalf("Media = %#v", item.Media)
	}
	if item.SourceMetadata.PostID != "987" || item.SourceMetadata.GUIDIsPermalink != "false" {
		t.Fatalf("SourceMetadata = %#v", item.SourceMetadata)
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

func Test_Parse_preserves_atom_summary_content_author_and_thumbnail(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/"><entry>
  <id>https://example.com/atom-rich</id>
  <title>Atom Rich</title>
  <link rel="alternate" href="https://example.com/atom-rich" />
  <updated>2026-06-19T22:31:04Z</updated>
  <summary>Atom summary</summary>
  <content type="html">&lt;p&gt;Atom content&lt;/p&gt;</content>
  <author><name>Bob</name><uri>https://example.com/bob</uri><email>bob@example.com</email></author>
  <media:thumbnail url="https://example.com/atom-thumb.jpg" />
</entry></feed>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	item := got[0]
	if item.Description != "Atom summary" {
		t.Fatalf("Description = %q", item.Description)
	}
	if item.ContentEncoded != "<p>Atom content</p>" {
		t.Fatalf("ContentEncoded = %q", item.ContentEncoded)
	}
	if len(item.Authors) != 1 || item.Authors[0].Email != "bob@example.com" {
		t.Fatalf("Authors = %#v", item.Authors)
	}
	if len(item.Media) != 1 || item.Media[0].URL != "https://example.com/atom-thumb.jpg" {
		t.Fatalf("Media = %#v", item.Media)
	}
}

func Test_Parse_reads_rss_link_href_and_dc_date_variants(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:atom="http://www.w3.org/2005/Atom"><channel><item>
  <title>Variant</title>
  <link href="https://example.com/variant/" />
  <dc:date>2026-06-19T22:31:04+09:00</dc:date>
  <category domain="topic">Vulnerability</category>
</item></channel></rss>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got[0].URL != "https://example.com/variant/" {
		t.Fatalf("URL = %q", got[0].URL)
	}
	if got[0].PublishedAt == nil || *got[0].PublishedAt != "2026-06-19T13:31:04Z" {
		t.Fatalf("PublishedAt = %#v", got[0].PublishedAt)
	}
	if got[0].Categories[0] != "Vulnerability" {
		t.Fatalf("Categories = %#v", got[0].Categories)
	}
}

func Test_Parse_uses_atom_id_when_alternate_link_is_missing(t *testing.T) {
	// Given
	input := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"><entry>
  <id>https://example.com/from-id</id>
  <title>Atom ID</title>
  <link rel="self" href="https://example.com/feed-entry" />
  <updated>2026-06-19T22:31:04Z</updated>
</entry></feed>`

	// When
	got, err := Parse(strings.NewReader(input))

	// Then
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got[0].URL != "https://example.com/from-id" {
		t.Fatalf("URL = %q", got[0].URL)
	}
}
