package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html/charset"
)

type envelope struct {
	XMLName xml.Name
	Channel rssChannel  `xml:"channel"`
	Entries []atomEntry `xml:"entry"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        string   `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	DCDate      string   `xml:"date"`
	Categories  []string `xml:"category"`
	Description string   `xml:"description"`
	Content     string   `xml:"encoded"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Categories []atomCategory `xml:"category"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
	Text string `xml:",chardata"`
}

func Parse(reader io.Reader) ([]Item, error) {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel

	var doc envelope
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode feed XML: %w", err)
	}
	if len(doc.Channel.Items) > 0 {
		return rssItems(doc.Channel.Items), nil
	}
	if len(doc.Entries) > 0 {
		return atomItems(doc.Entries), nil
	}
	return nil, fmt.Errorf("feed contains no rss items or atom entries")
}

func rssItems(items []rssItem) []Item {
	parsed := make([]Item, 0, len(items))
	for _, item := range items {
		rawDate := firstNonEmpty(item.PubDate, item.DCDate)
		itemURL := firstNonEmpty(item.Link, item.GUID)
		parsed = append(parsed, Item{
			Title:        strings.TrimSpace(item.Title),
			URL:          strings.TrimSpace(itemURL),
			PublishedAt:  parsePublishedAt(rawDate),
			PublishedRaw: strings.TrimSpace(rawDate),
			Categories:   trimStrings(item.Categories),
			Summary:      firstNonEmpty(item.Description, item.Content),
		})
	}
	return parsed
}

func atomItems(entries []atomEntry) []Item {
	parsed := make([]Item, 0, len(entries))
	for _, entry := range entries {
		rawDate := firstNonEmpty(entry.Published, entry.Updated)
		parsed = append(parsed, Item{
			Title:        strings.TrimSpace(entry.Title),
			URL:          strings.TrimSpace(atomURL(entry.Links)),
			PublishedAt:  parsePublishedAt(rawDate),
			PublishedRaw: strings.TrimSpace(rawDate),
			Categories:   atomCategories(entry.Categories),
			Summary:      firstNonEmpty(entry.Summary, entry.Content),
		})
	}
	return parsed
}

func atomURL(links []atomLink) string {
	for _, link := range links {
		if link.Rel == "" || link.Rel == "alternate" {
			return firstNonEmpty(link.Href, link.Text)
		}
	}
	if len(links) == 0 {
		return ""
	}
	return firstNonEmpty(links[0].Href, links[0].Text)
}

func atomCategories(categories []atomCategory) []string {
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		value := firstNonEmpty(category.Term, category.Text)
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
