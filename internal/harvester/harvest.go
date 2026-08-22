package harvester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/found-cake/cyber-news-feed/internal/feed"
	"github.com/found-cake/cyber-news-feed/internal/jsonstore"
	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
	"github.com/found-cake/cyber-news-feed/internal/source"
	"github.com/found-cake/cyber-news-feed/internal/urlnorm"
)

func Run(ctx context.Context, cfg Config, logger *slog.Logger) (Summary, error) {
	return runWithSources(ctx, cfg, logger, source.Default())
}

func runWithSources(ctx context.Context, cfg Config, logger *slog.Logger, sources []source.Config) (Summary, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 30 * time.Second}
	}
	now := time.Now().UTC()
	summary := Summary{}
	pendingRetry := make([]source.Config, 0)

	for _, src := range sources {
		result, err := processSource(ctx, processSourceRequest{
			cfg:    cfg,
			logger: logger,
			now:    now,
			src:    src,
		})
		if err != nil {
			return summary, err
		}
		if result.retry {
			pendingRetry = append(pendingRetry, src)
		} else if !result.succeeded {
			summary.Failed++
		}
		summary.Processed++
	}
	for _, src := range pendingRetry {
		result, err := processSource(ctx, processSourceRequest{
			cfg:       cfg,
			logger:    logger,
			now:       now,
			src:       src,
			skipCache: true,
		})
		if err != nil {
			return summary, err
		}
		if !result.succeeded {
			summary.Failed++
		}
	}
	return summary, nil
}

type processSourceRequest struct {
	cfg       Config
	logger    *slog.Logger
	now       time.Time
	src       source.Config
	skipCache bool
}

type processSourceResult struct {
	succeeded bool
	retry     bool
}

func processSource(ctx context.Context, req processSourceRequest) (processSourceResult, error) {
	existing, err := jsonstore.Load(req.cfg.OutputDir, req.src.Name)
	if err != nil {
		return processSourceResult{}, fmt.Errorf("load %s: %w", req.src.Name, err)
	}
	articles, err := fetchSource(ctx, fetchSourceRequest{
		client:    req.cfg.Client,
		src:       req.src,
		skipCache: req.skipCache,
	})
	var doc rssdoc.Document
	retry := false
	if err != nil {
		retry = true
		var statusErr *unexpectedHTTPStatusError
		if req.src.Kind == source.BoanNews && errors.As(err, &statusErr) && statusErr.statusCode == http.StatusForbidden {
			retry = false
		}
		switch {
		case req.skipCache:
			req.logger.Warn("source retry failed", "source", req.src.Name, "error", err)
		case retry:
			req.logger.Warn("source queued for retry", "source", req.src.Name, "error", err)
		default:
			req.logger.Warn("source retry suppressed", "source", req.src.Name, "error", err)
		}
		doc = rssdoc.MergeFailure(rssdoc.MergeRequest{
			Existing:      existing,
			Source:        req.src.Name,
			Now:           req.now,
			RetentionDays: req.cfg.RetentionDays,
			Err:           err,
		})
	} else {
		if req.src.Kind == source.BoanNews {
			knownCategories := make(map[string][]string, len(existing.Articles))
			for _, article := range existing.Articles {
				if len(article.Categories) > 0 {
					knownCategories[urlnorm.Normalize(article.URL)] = slices.Clone(article.Categories)
				}
			}
			for index := range articles {
				if categories, exists := knownCategories[articles[index].URL]; exists {
					articles[index].Categories = categories
				}
			}
		}
		doc = rssdoc.MergeSuccess(rssdoc.MergeRequest{
			Existing:      existing,
			Source:        req.src.Name,
			Now:           req.now,
			RetentionDays: req.cfg.RetentionDays,
			Fetched:       articles,
		})
		if req.src.Kind == source.BoanNews {
			enricher := boanNewsEnricher{client: req.cfg.Client, skipCache: req.skipCache}
			if enrichErr := enricher.enrich(ctx, doc.Articles); enrichErr != nil {
				req.logger.Warn("boannews category enrichment incomplete", "source", req.src.Name, "error", enrichErr)
			}
		}
	}
	if err := jsonstore.Write(req.cfg.OutputDir, doc); err != nil {
		return processSourceResult{}, fmt.Errorf("write %s: %w", req.src.Name, err)
	}
	return processSourceResult{succeeded: err == nil, retry: retry}, nil
}

type fetchSourceRequest struct {
	client    *http.Client
	src       source.Config
	skipCache bool
}

func fetchSource(ctx context.Context, req fetchSourceRequest) ([]rssdoc.Article, error) {
	articles := make([]rssdoc.Article, 0)
	for _, sourceFeed := range req.src.Feeds {
		items, err := fetchFeed(ctx, req.client, sourceFeed.URL, req.skipCache)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", sourceFeed.URL, err)
		}
		for _, item := range items {
			article, include := source.ArticleFromItem(req.src, sourceFeed, item)
			if include {
				articles = append(articles, article)
			}
		}
	}
	return articles, nil
}

func fetchFeed(ctx context.Context, client *http.Client, feedURL string, skipCache bool) (items []feed.Item, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "cyber-news-feed/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.1")
	if skipCache {
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
	}

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
		return nil, fmt.Errorf("%w: %w", errUnexpectedFeedResponse, &unexpectedHTTPStatusError{statusCode: resp.StatusCode})
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read feed body: %w", err)
	}
	if err := ensureFeedBody(resp.Header.Get("Content-Type"), body); err != nil {
		return nil, err
	}
	items, err = feed.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: parse feed: %w", errUnexpectedFeedResponse, err)
	}
	return items, nil
}
