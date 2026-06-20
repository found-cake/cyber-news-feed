package rssdoc

import (
	"time"

	"github.com/found-cake/cyber-news-feed/internal/urlnorm"
	"github.com/found-cake/cyber-news-feed/pkg/rssjson"
)

type MergeRequest struct {
	Existing      Document
	Source        string
	Now           time.Time
	RetentionDays int
	Fetched       []Article
	Err           error
}

func MergeSuccess(req MergeRequest) Document {
	now := req.Now.UTC().Format(time.RFC3339)
	return Document{
		SchemaVersion: SchemaVersion,
		Source:        req.Source,
		UpdatedAt:     now,
		RetentionDays: req.RetentionDays,
		Status: Status{
			OK:            true,
			LastSuccessAt: &now,
			LastErrorAt:   nil,
			LastError:     nil,
		},
		Articles: mergedArticles(req.Existing.Articles, req.Fetched, req.Now, req.RetentionDays),
	}
}

func MergeFailure(req MergeRequest) Document {
	now := req.Now.UTC().Format(time.RFC3339)
	message := "unknown error"
	if req.Err != nil {
		message = req.Err.Error()
	}
	return Document{
		SchemaVersion: SchemaVersion,
		Source:        req.Source,
		UpdatedAt:     now,
		RetentionDays: req.RetentionDays,
		Status: Status{
			OK:            false,
			LastSuccessAt: req.Existing.Status.LastSuccessAt,
			LastErrorAt:   &now,
			LastError:     &message,
		},
		Articles: req.Existing.Articles,
	}
}

func mergedArticles(existing []Article, fetched []Article, now time.Time, retentionDays int) []Article {
	seen := make(map[string]struct{}, len(existing)+len(fetched))
	out := make([]Article, 0, len(existing)+len(fetched))
	for _, article := range fetched {
		normalized := normalizedArticle(article)
		if _, ok := seen[normalized.URL]; ok || expired(normalized, now, retentionDays) {
			continue
		}
		seen[normalized.URL] = struct{}{}
		out = append(out, normalized)
	}
	for _, article := range existing {
		normalized := normalizedArticle(article)
		if _, ok := seen[normalized.URL]; ok || expired(normalized, now, retentionDays) {
			continue
		}
		seen[normalized.URL] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizedArticle(article Article) Article {
	canonicalURL := urlnorm.Normalize(article.URL)
	article.URL = canonicalURL
	article.ID = urlnorm.StableArticleID(canonicalURL)
	if article.Categories == nil {
		article.Categories = []string{}
	}
	if article.Authors == nil {
		article.Authors = []rssjson.Author{}
	}
	if article.Media == nil {
		article.Media = []rssjson.Media{}
	}
	return article
}

func expired(article Article, now time.Time, retentionDays int) bool {
	if article.PublishedAt == nil {
		return false
	}
	publishedAt, err := time.Parse(time.RFC3339, *article.PublishedAt)
	if err != nil {
		return false
	}
	cutoff := now.UTC().AddDate(0, 0, -retentionDays)
	return publishedAt.UTC().Before(cutoff)
}
