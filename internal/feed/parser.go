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
	Title       string      `xml:"title"`
	Links       []rssLink   `xml:"link"`
	GUID        rssGUID     `xml:"guid"`
	PubDate     string      `xml:"pubDate"`
	DCDate      string      `xml:"date"`
	Published   string      `xml:"published"`
	Updated     string      `xml:"updated"`
	Categories  []string    `xml:"category"`
	Description string      `xml:"description"`
	Content     string      `xml:"encoded"`
	Creators    []string    `xml:"creator"`
	Authors     []rssAuthor `xml:"author"`
	Thumbnails  []mediaNode `xml:"thumbnail"`
	Media       []mediaNode `xml:"content"`
	PostID      string      `xml:"post-id"`
}

type rssLink struct {
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Text        string `xml:",chardata"`
}

type rssAuthor struct {
	Text string `xml:",chardata"`
}

type atomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Categories []atomCategory `xml:"category"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Authors    []atomAuthor   `xml:"author"`
	Thumbnails []mediaNode    `xml:"thumbnail"`
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

type atomAuthor struct {
	Name  string `xml:"name"`
	URI   string `xml:"uri"`
	Email string `xml:"email"`
}

type mediaNode struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
}

func Parse(reader io.Reader) ([]Item, error) {
	xmlReader, err := readerWithEscapedAmpersands(reader)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(xmlReader)
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
		rawDate := firstNonEmpty(item.PubDate, item.DCDate, item.Published, item.Updated)
		itemURL := firstNonEmpty(rssURL(item.Links), item.GUID.Text)
		parsed = append(parsed, Item{
			Title:          strings.TrimSpace(item.Title),
			URL:            strings.TrimSpace(itemURL),
			PublishedAt:    parsePublishedAt(rawDate),
			PublishedRaw:   strings.TrimSpace(rawDate),
			Categories:     trimStrings(item.Categories),
			Description:    strings.TrimSpace(item.Description),
			ContentEncoded: strings.TrimSpace(item.Content),
			FeedID:         strings.TrimSpace(item.GUID.Text),
			Authors:        rssAuthors(item.Creators, item.Authors),
			Media:          mediaItems(item.Thumbnails, item.Media),
			SourceMetadata: rssMetadata(item),
		})
	}
	return parsed
}

func atomItems(entries []atomEntry) []Item {
	parsed := make([]Item, 0, len(entries))
	for _, entry := range entries {
		rawDate := firstNonEmpty(entry.Published, entry.Updated)
		parsed = append(parsed, Item{
			Title:          strings.TrimSpace(entry.Title),
			URL:            strings.TrimSpace(atomURL(entry.Links, entry.ID)),
			PublishedAt:    parsePublishedAt(rawDate),
			PublishedRaw:   strings.TrimSpace(rawDate),
			Categories:     atomCategories(entry.Categories),
			Description:    strings.TrimSpace(entry.Summary),
			ContentEncoded: strings.TrimSpace(entry.Content),
			FeedID:         strings.TrimSpace(entry.ID),
			Authors:        atomAuthors(entry.Authors),
			Media:          mediaItems(entry.Thumbnails, nil),
			SourceMetadata: SourceMetadata{},
		})
	}
	return parsed
}

func rssURL(links []rssLink) string {
	for _, link := range links {
		if value := firstNonEmpty(link.Href, link.Text); value != "" {
			return value
		}
	}
	return ""
}

func atomURL(links []atomLink, id string) string {
	for _, link := range links {
		if link.Rel == "" || link.Rel == "alternate" {
			return firstNonEmpty(link.Href, link.Text)
		}
	}
	if id != "" {
		return id
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

func rssAuthors(creators []string, authors []rssAuthor) []Author {
	out := make([]Author, 0, len(creators)+len(authors))
	for _, creator := range creators {
		if name := strings.TrimSpace(creator); name != "" {
			out = append(out, Author{Name: name})
		}
	}
	for _, author := range authors {
		if name := strings.TrimSpace(author.Text); name != "" {
			out = append(out, Author{Name: name})
		}
	}
	return out
}

func atomAuthors(authors []atomAuthor) []Author {
	out := make([]Author, 0, len(authors))
	for _, author := range authors {
		name := strings.TrimSpace(author.Name)
		uri := strings.TrimSpace(author.URI)
		email := strings.TrimSpace(author.Email)
		if name == "" && uri == "" && email == "" {
			continue
		}
		out = append(out, Author{Name: name, URI: uri, Email: email})
	}
	return out
}

func mediaItems(thumbnails []mediaNode, media []mediaNode) []Media {
	out := make([]Media, 0, len(thumbnails)+len(media))
	seen := make(map[string]struct{}, len(thumbnails)+len(media))
	for _, thumbnail := range thumbnails {
		out = appendMedia(out, seen, thumbnail, "thumbnail")
	}
	for _, item := range media {
		out = appendMedia(out, seen, item, "content")
	}
	return out
}

func appendMedia(out []Media, seen map[string]struct{}, item mediaNode, kind string) []Media {
	mediaURL := strings.TrimSpace(item.URL)
	if mediaURL == "" {
		return out
	}
	if _, ok := seen[kind+" "+mediaURL]; ok {
		return out
	}
	seen[kind+" "+mediaURL] = struct{}{}
	return append(out, Media{URL: mediaURL, Kind: kind, Medium: strings.TrimSpace(item.Medium)})
}

func rssMetadata(item rssItem) SourceMetadata {
	return SourceMetadata{
		GUIDIsPermalink: strings.TrimSpace(item.GUID.IsPermaLink),
		PostID:          strings.TrimSpace(item.PostID),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
