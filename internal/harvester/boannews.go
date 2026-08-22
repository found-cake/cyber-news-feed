package harvester

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	boanNewsPageDrainMaxBytes = 64 << 10
	boanNewsPageMaxBytes      = 2 << 20
)

var (
	errBoanNewsClassificationMissing = errors.New("boannews classification metadata missing")
	errBoanNewsClassificationInvalid = errors.New("boannews classification metadata invalid")
	errBoanNewsPageResponse          = errors.New("unexpected boannews article response")
	errBoanNewsPageTooLarge          = errors.New("boannews article response too large")
	errBoanNewsPageURL               = errors.New("invalid boannews article URL")
	errBoanNewsRedirectLimit         = errors.New("boannews article redirect limit exceeded")
)

type boanNewsEnricher struct {
	client    *http.Client
	skipCache bool
}

func (enricher boanNewsEnricher) enrich(ctx context.Context, articles []rssdoc.Article) error {
	var enrichErrors []error
	for index := range articles {
		if len(articles[index].Categories) > 0 {
			continue
		}
		categories, err := enricher.fetchCategories(ctx, articles[index].URL)
		if err != nil {
			articleErr := fmt.Errorf("enrich boannews article %d %s: %w", index, articles[index].URL, err)
			var statusErr *unexpectedHTTPStatusError
			forbidden := errors.As(err, &statusErr) && statusErr.statusCode == http.StatusForbidden
			if forbidden || ctx.Err() != nil {
				return errors.Join(append(enrichErrors, articleErr)...)
			}
			enrichErrors = append(enrichErrors, articleErr)
			continue
		}
		articles[index].Categories = categories
	}
	return errors.Join(enrichErrors...)
}

func (enricher boanNewsEnricher) fetchCategories(ctx context.Context, articleURL string) (categories []string, err error) {
	if err := validateBoanNewsArticleURL(articleURL); err != nil {
		return nil, err
	}

	pageClient := *enricher.client
	checkRedirect := pageClient.CheckRedirect
	pageClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if err := validateBoanNewsArticleURL(request.URL.String()); err != nil {
			return err
		}
		if len(via) >= 10 {
			return errBoanNewsRedirectLimit
		}
		if checkRedirect != nil {
			return checkRedirect(request, via)
		}
		return nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create boannews article request: %w", err)
	}
	request.Header.Set("User-Agent", "cyber-news-feed/1.0")
	request.Header.Set("Accept", "text/html, application/xhtml+xml;q=0.9, */*;q=0.1")
	if enricher.skipCache {
		request.Header.Set("Cache-Control", "no-cache")
		request.Header.Set("Pragma", "no-cache")
	}

	response, err := pageClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch boannews article: %w", err)
	}
	defer func() {
		_, drainErr := io.Copy(io.Discard, io.LimitReader(response.Body, boanNewsPageDrainMaxBytes))
		closeErr := response.Body.Close()
		if err == nil {
			err = errors.Join(drainErr, closeErr)
		}
	}()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: %w", errBoanNewsPageResponse, &unexpectedHTTPStatusError{statusCode: response.StatusCode})
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, boanNewsPageMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read boannews article: %w", err)
	}
	if len(body) > boanNewsPageMaxBytes {
		return nil, errBoanNewsPageTooLarge
	}
	decoded, err := charset.NewReader(bytes.NewReader(body), response.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("decode boannews article: %w", err)
	}
	categories, err = parseBoanNewsClassification(decoded)
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func validateBoanNewsArticleURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", errBoanNewsPageURL, err)
	}
	host := parsed.Hostname()
	allowedHost := strings.EqualFold(host, "boannews.com") || strings.EqualFold(host, "www.boannews.com")
	if parsed.Scheme != "https" || !allowedHost || parsed.User != nil || (parsed.Port() != "" && parsed.Port() != "443") {
		return fmt.Errorf("%w: %s", errBoanNewsPageURL, rawURL)
	}
	return nil
}

func parseBoanNewsClassification(reader io.Reader) ([]string, error) {
	document, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("parse boannews article HTML: %w", err)
	}

	meta := findBoanNewsClassificationMeta(document)
	if meta == nil {
		return nil, errBoanNewsClassificationMissing
	}
	content, ok := htmlAttribute(meta, "content")
	if !ok || strings.TrimSpace(content) == "" {
		return nil, errBoanNewsClassificationInvalid
	}

	parts := strings.Split(content, ">")
	categories := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		category := strings.TrimSpace(part)
		if category == "" {
			return nil, errBoanNewsClassificationInvalid
		}
		key := strings.ToLower(category)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, category)
	}
	return categories, nil
}

func findBoanNewsClassificationMeta(node *html.Node) *html.Node {
	if node.Type == html.ElementNode && strings.EqualFold(node.Data, "meta") {
		name, ok := htmlAttribute(node, "name")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Classification") {
			return node
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if meta := findBoanNewsClassificationMeta(child); meta != nil {
			return meta
		}
	}
	return nil
}

func htmlAttribute(node *html.Node, name string) (string, bool) {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val, true
		}
	}
	return "", false
}
