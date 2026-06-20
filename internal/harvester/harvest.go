package harvester

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/found-cake/cyber-news-feed/internal/feed"
	"github.com/found-cake/cyber-news-feed/internal/jsonstore"
	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
	"github.com/found-cake/cyber-news-feed/internal/source"
)

func Run(ctx context.Context, cfg Config, logger *slog.Logger) (Summary, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 30 * time.Second}
	}
	now := time.Now().UTC()
	summary := Summary{}

	for _, src := range source.Default() {
		existing, err := jsonstore.Load(cfg.OutputDir, src.Name)
		if err != nil {
			return summary, fmt.Errorf("load %s: %w", src.Name, err)
		}
		articles, err := fetchSource(ctx, cfg.Client, src)
		var doc rssdoc.Document
		if err != nil {
			logger.Warn("source failed", "source", src.Name, "error", err)
			doc = rssdoc.MergeFailure(rssdoc.MergeRequest{
				Existing:      existing,
				Source:        src.Name,
				Now:           now,
				RetentionDays: cfg.RetentionDays,
				Err:           err,
			})
			summary.Failed++
		} else {
			doc = rssdoc.MergeSuccess(rssdoc.MergeRequest{
				Existing:      existing,
				Source:        src.Name,
				Now:           now,
				RetentionDays: cfg.RetentionDays,
				Fetched:       articles,
			})
		}
		if err := jsonstore.Write(cfg.OutputDir, doc); err != nil {
			return summary, fmt.Errorf("write %s: %w", src.Name, err)
		}
		summary.Processed++
	}
	return summary, nil
}

func fetchSource(ctx context.Context, client *http.Client, src source.Config) ([]rssdoc.Article, error) {
	articles := make([]rssdoc.Article, 0)
	for _, sourceFeed := range src.Feeds {
		items, err := fetchFeed(ctx, client, sourceFeed.URL)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", sourceFeed.URL, err)
		}
		for _, item := range items {
			article, include := source.ArticleFromItem(src, sourceFeed, item)
			if include {
				articles = append(articles, article)
			}
		}
	}
	return articles, nil
}

func fetchFeed(ctx context.Context, client *http.Client, feedURL string) (items []feed.Item, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "cyber-news-feed/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_, copyErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()
		if err == nil {
			err = errors.Join(copyErr, closeErr)
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	items, err = feed.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	return items, nil
}
