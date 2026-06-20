package source

import (
	"strings"

	"github.com/found-cake/cyber-news-feed/internal/feed"
	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
	"github.com/found-cake/cyber-news-feed/internal/urlnorm"
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
				{URL: "https://www.boannews.com/media/news_rss.xml?kind=1"},
				{URL: "http://www.boannews.com/media/news_rss.xml?skind=5"},
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
	}
}

func ArticleFromItem(source Config, sourceFeed Feed, item feed.Item) (rssdoc.Article, bool) {
	canonicalURL := urlnorm.Normalize(item.URL)
	if canonicalURL == "" {
		return rssdoc.Article{}, false
	}
	category, include := categoryForItem(source, sourceFeed, item, canonicalURL)
	if !include {
		return rssdoc.Article{}, false
	}
	return rssdoc.Article{
		ID:           urlnorm.StableArticleID(canonicalURL),
		URL:          canonicalURL,
		Title:        strings.TrimSpace(item.Title),
		PublishedAt:  item.PublishedAt,
		PublishedRaw: strings.TrimSpace(item.PublishedRaw),
		Category:     category,
	}, true
}

func categoryForItem(source Config, sourceFeed Feed, item feed.Item, canonicalURL string) (string, bool) {
	switch source.Kind {
	case TheHackerNews:
		return sourceFeed.Category, true
	case StepSecurity:
		return "", includeStepSecurity(item)
	case DarkReading:
		return darkReadingCategory(canonicalURL)
	case BleepingComputer:
		if hasCategory(item.Categories, "Security") {
			return "security", true
		}
		return "", false
	default:
		return "", true
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
	text := strings.ToLower(item.Title + " " + item.Summary)
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
