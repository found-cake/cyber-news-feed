package source

import (
	"strings"

	"github.com/found-cake/cyber-news-feed/internal/feed"
	"github.com/found-cake/cyber-news-feed/internal/urlnorm"
	"github.com/found-cake/cyber-news-feed/pkg/rssjson"
)

type Config struct {
	Name  string
	Feeds []Feed
	Kind  Kind
}

type Feed struct {
	URL      string
	Category string
}

type Kind int

const (
	Unfiltered Kind = iota
	TheHackerNews
	StepSecurity
	DarkReading
	BleepingComputer
)

var darkReadingCategories = []string{
	"threat-intelligence",
	"vulnerabilities-threats",
	"cyberattacks-data-breaches",
	"cloud-security",
	"application-security",
	"endpoint-security",
	"ics-ot-security",
}

func Default() []Config {
	return []Config{
		{
			Name: "boannews",
			Feeds: []Feed{
				{URL: "https://www.boannews.com/media/news_rss.xml?kind=1", Category: "사건ㆍ사고"},
				{URL: "http://www.boannews.com/media/news_rss.xml?skind=5", Category: "긴급경보"},
			},
		},
		{
			Name: "thehackernews",
			Kind: TheHackerNews,
			Feeds: []Feed{
				{URL: "https://thehackernews.com/feeds/posts/default/-/Threat%20Intelligence", Category: "threat-intelligence"},
				{URL: "https://thehackernews.com/feeds/posts/default/-/Vulnerability", Category: "vulnerability"},
				{URL: "https://thehackernews.com/feeds/posts/default/-/Cyber%20Attack", Category: "cyber-attack"},
			},
		},
		{Name: "cybersecuritynews", Feeds: []Feed{{URL: "https://cybersecuritynews.com/feed/"}}},
		{Name: "stepsecurity", Kind: StepSecurity, Feeds: []Feed{{URL: "https://www.stepsecurity.io/blog/rss.xml"}}},
		{Name: "darkreading", Kind: DarkReading, Feeds: []Feed{{URL: "https://www.darkreading.com/rss.xml"}}},
		{Name: "bleepingcomputer", Kind: BleepingComputer, Feeds: []Feed{{URL: "https://www.bleepingcomputer.com/feed/"}}},
		{Name: "securityweek", Feeds: []Feed{{URL: "https://www.securityweek.com/feed/"}}},
	}
}

func ArticleFromItem(source Config, sourceFeed Feed, item feed.Item) (rssjson.Article, bool) {
	canonicalURL := urlnorm.Normalize(item.URL)
	if canonicalURL == "" {
		return rssjson.Article{}, false
	}
	categories, include := categoriesForItem(source, sourceFeed, item, canonicalURL)
	if !include {
		return rssjson.Article{}, false
	}
	return rssjson.Article{
		ID:             urlnorm.StableArticleID(canonicalURL),
		URL:            canonicalURL,
		Title:          strings.TrimSpace(item.Title),
		PublishedAt:    item.PublishedAt,
		PublishedRaw:   strings.TrimSpace(item.PublishedRaw),
		Categories:     categories,
		Description:    item.Description,
		FeedID:         item.FeedID,
		Authors:        articleAuthors(item.Authors),
		Media:          articleMedia(item.Media),
		SourceMetadata: sourceMetadata(source.Name, item),
	}, true
}

func categoriesForItem(source Config, sourceFeed Feed, item feed.Item, canonicalURL string) ([]string, bool) {
	switch source.Kind {
	case TheHackerNews:
		return mergedCategories([]string{sourceFeed.Category}, item.Categories), true
	case StepSecurity:
		return mergedCategories(nil, item.Categories), includeStepSecurity(item)
	case DarkReading:
		category, include := darkReadingCategory(canonicalURL)
		return mergedCategories([]string{category}, item.Categories), include
	case BleepingComputer:
		if hasCategory(item.Categories, "Security") {
			return mergedCategories([]string{"security"}, item.Categories), true
		}
		return []string{}, false
	default:
		return mergedCategories(nil, item.Categories), true
	}
}

func darkReadingCategory(canonicalURL string) (string, bool) {
	for _, category := range darkReadingCategories {
		if strings.Contains(canonicalURL, category) {
			return category, true
		}
	}
	return "", false
}

func hasCategory(categories []string, want string) bool {
	for _, category := range categories {
		if strings.EqualFold(strings.TrimSpace(category), want) {
			return true
		}
	}
	return false
}

func includeStepSecurity(item feed.Item) bool {
	text := strings.ToLower(item.Title + " " + item.Description + " " + item.ContentEncoded)
	promoPhrases := []string{
		"artifact monitor",
		"demo",
		"harden-runner",
		"product update",
		"release notes",
		"secure-repo",
		"stepsecurity platform",
		"webinar",
	}
	for _, phrase := range promoPhrases {
		if strings.Contains(text, phrase) {
			return false
		}
	}
	return true
}

func articleAuthors(authors []feed.Author) []rssjson.Author {
	out := make([]rssjson.Author, 0, len(authors))
	for _, author := range authors {
		out = append(out, rssjson.Author{Name: author.Name, URI: author.URI, Email: author.Email})
	}
	return out
}

func articleMedia(media []feed.Media) []rssjson.Media {
	out := make([]rssjson.Media, 0, len(media))
	for _, item := range media {
		out = append(out, rssjson.Media{URL: item.URL, Kind: item.Kind, Medium: item.Medium})
	}
	return out
}

func sourceMetadata(sourceName string, item feed.Item) rssjson.SourceMetadata {
	metadata := item.SourceMetadata
	switch sourceName {
	case "cybersecuritynews":
		if metadata.GUIDIsPermalink == "" && metadata.PostID == "" && item.ContentEncoded == "" {
			return rssjson.SourceMetadata{}
		}
		return rssjson.SourceMetadata{
			CybersecurityNews: &rssjson.CybersecurityNewsMetadata{
				GUIDIsPermalink: metadata.GUIDIsPermalink,
				PostID:          metadata.PostID,
				ContentEncoded:  item.ContentEncoded,
			},
		}
	case "darkreading":
		if metadata.GUIDIsPermalink == "" {
			return rssjson.SourceMetadata{}
		}
		return rssjson.SourceMetadata{
			DarkReading: &rssjson.DarkReadingMetadata{
				GUIDIsPermalink: metadata.GUIDIsPermalink,
			},
		}
	case "bleepingcomputer":
		if metadata.GUIDIsPermalink == "" {
			return rssjson.SourceMetadata{}
		}
		return rssjson.SourceMetadata{
			BleepingComputer: &rssjson.BleepingComputerMetadata{
				GUIDIsPermalink: metadata.GUIDIsPermalink,
			},
		}
	case "securityweek":
		if metadata.Image == (feed.SourceImage{}) {
			return rssjson.SourceMetadata{}
		}
		return rssjson.SourceMetadata{
			SecurityWeek: &rssjson.SecurityWeekMetadata{
				Image: rssjson.SecurityWeekImage{
					URL:    metadata.Image.URL,
					Title:  metadata.Image.Title,
					Link:   metadata.Image.Link,
					Width:  metadata.Image.Width,
					Height: metadata.Image.Height,
				},
			},
		}
	default:
		return rssjson.SourceMetadata{}
	}
}

func mergedCategories(primary []string, rssCategories []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(rssCategories))
	out := make([]string, 0, len(primary)+len(rssCategories))
	for _, category := range append(primary, rssCategories...) {
		trimmed := strings.TrimSpace(category)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
