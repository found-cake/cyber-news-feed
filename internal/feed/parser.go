package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html/charset"
)

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
		return rssItems(doc.Channel.Items, sourceImage(doc.Channel.Image)), nil
	}
	if len(doc.Entries) > 0 {
		return atomItems(doc.Entries), nil
	}
	return nil, fmt.Errorf("feed contains no rss items or atom entries")
}

func rssItems(items []rssItem, channelImage SourceImage) []Item {
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
			SourceMetadata: rssMetadata(item, channelImage),
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

func rssMetadata(item rssItem, channelImage SourceImage) SourceMetadata {
	image := sourceImage(item.Image)
	if image == (SourceImage{}) {
		image = channelImage
	}
	return SourceMetadata{
		GUIDIsPermalink: strings.TrimSpace(item.GUID.IsPermaLink),
		Image:           image,
		PostID:          strings.TrimSpace(item.PostID),
	}
}

func sourceImage(image rssImage) SourceImage {
	return SourceImage{
		URL:    strings.TrimSpace(firstNonEmpty(image.Href, image.URL, image.Text)),
		Title:  strings.TrimSpace(image.Title),
		Link:   strings.TrimSpace(image.Link),
		Width:  strings.TrimSpace(image.Width),
		Height: strings.TrimSpace(image.Height),
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
