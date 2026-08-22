package harvester

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func Test_parseBoanNewsClassification_returns_ordered_categories_when_hierarchy_present(t *testing.T) {
	// Given
	html := `<html><head><meta content=" 사건·사고 &gt; 인사이트 " name="cLaSsIfIcAtIoN"></head></html>`

	// When
	got, err := parseBoanNewsClassification(strings.NewReader(html))

	// Then
	if err != nil {
		t.Fatalf("parseBoanNewsClassification() error = %v", err)
	}
	want := []string{"사건·사고", "인사이트"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v, want %#v", got, want)
	}
}

func Test_parseBoanNewsClassification_returns_single_category_when_hierarchy_absent(t *testing.T) {
	// Given
	html := `<html><head><meta name="Classification" content="공공·정책"></head></html>`

	// When
	got, err := parseBoanNewsClassification(strings.NewReader(html))

	// Then
	if err != nil {
		t.Fatalf("parseBoanNewsClassification() error = %v", err)
	}
	want := []string{"공공·정책"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v, want %#v", got, want)
	}
}

func Test_parseBoanNewsClassification_deduplicates_categories_case_insensitively(t *testing.T) {
	// Given
	html := `<html><head><meta name="Classification" content="Insight &gt; insight &gt; Report"></head></html>`

	// When
	got, err := parseBoanNewsClassification(strings.NewReader(html))

	// Then
	if err != nil {
		t.Fatalf("parseBoanNewsClassification() error = %v", err)
	}
	want := []string{"Insight", "Report"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v, want %#v", got, want)
	}
}

func Test_parseBoanNewsClassification_returns_error_when_metadata_missing_or_invalid(t *testing.T) {
	tests := map[string]string{
		"missing meta":       `<html><head><meta name="description" content="x"></head></html>`,
		"missing content":    `<html><head><meta name="Classification"></head></html>`,
		"blank content":      `<html><head><meta name="Classification" content="  "></head></html>`,
		"leading delimiter":  `<html><head><meta name="Classification" content="&gt; 인사이트"></head></html>`,
		"adjacent delimiter": `<html><head><meta name="Classification" content="사건·사고 &gt; &gt; 인사이트"></head></html>`,
	}

	for name, document := range tests {
		t.Run(name, func(t *testing.T) {
			// Given
			reader := strings.NewReader(document)

			// When
			_, err := parseBoanNewsClassification(reader)

			// Then
			if err == nil {
				t.Fatal("parseBoanNewsClassification() error = nil, want error")
			}
		})
	}
}

func Test_fetchBoanNewsCategories_returns_page_classification_and_sets_request_headers(t *testing.T) {
	// Given
	var request *http.Request
	client := &http.Client{Transport: roundTripFunc(func(got *http.Request) (*http.Response, error) {
		request = got.Clone(got.Context())
		return htmlResponse(http.StatusOK, `<meta name="Classification" content="사건·사고 &gt; 인사이트">`), nil
	})}

	// When
	enricher := boanNewsEnricher{client: client, skipCache: true}
	got, err := enricher.fetchCategories(context.Background(), "https://www.boannews.com/news/articleView.html?idxno=1")

	// Then
	if err != nil {
		t.Fatalf("fetchBoanNewsCategories() error = %v", err)
	}
	want := []string{"사건·사고", "인사이트"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("categories = %#v, want %#v", got, want)
	}
	if request.Header.Get("User-Agent") != "cyber-news-feed/1.0" || request.Header.Get("Accept") == "" {
		t.Fatalf("request headers = %#v", request.Header)
	}
	if request.Header.Get("Cache-Control") != "no-cache" || request.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("cache bypass headers = %#v", request.Header)
	}
}

func Test_enrichBoanNewsArticles_replaces_categories_without_reordering_articles(t *testing.T) {
	// Given
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		classification := "비즈니스 &gt; 인사이트"
		if request.URL.Query().Get("idxno") == "2" {
			classification = "사건·사고 &gt; 국제"
		}
		return htmlResponse(http.StatusOK, `<meta name="Classification" content="`+classification+`">`), nil
	})}
	articles := []rssdoc.Article{
		{Title: "First", URL: "https://www.boannews.com/news/articleView.html?idxno=1", Categories: []string{"rss"}},
		{Title: "Second", URL: "https://www.boannews.com/news/articleView.html?idxno=2", Categories: []string{"rss"}},
	}

	// When
	err := (boanNewsEnricher{client: client}).enrich(context.Background(), articles)

	// Then
	if err != nil {
		t.Fatalf("enrichBoanNewsArticles() error = %v", err)
	}
	if articles[0].Title != "First" || !reflect.DeepEqual(articles[0].Categories, []string{"비즈니스", "인사이트"}) {
		t.Fatalf("first article = %#v", articles[0])
	}
	if articles[1].Title != "Second" || !reflect.DeepEqual(articles[1].Categories, []string{"사건·사고", "국제"}) {
		t.Fatalf("second article = %#v", articles[1])
	}
}

func Test_enrichBoanNewsArticles_leaves_articles_unchanged_when_page_fails(t *testing.T) {
	// Given
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("idxno") == "2" {
			return htmlResponse(http.StatusBadGateway, "upstream failed"), nil
		}
		return htmlResponse(http.StatusOK, `<meta name="Classification" content="비즈니스 &gt; 인사이트">`), nil
	})}
	articles := []rssdoc.Article{
		{Title: "First", URL: "https://www.boannews.com/news/articleView.html?idxno=1", Categories: []string{"rss-one"}},
		{Title: "Second", URL: "https://www.boannews.com/news/articleView.html?idxno=2", Categories: []string{"rss-two"}},
	}
	want := []rssdoc.Article{
		{Title: "First", URL: "https://www.boannews.com/news/articleView.html?idxno=1", Categories: []string{"rss-one"}},
		{Title: "Second", URL: "https://www.boannews.com/news/articleView.html?idxno=2", Categories: []string{"rss-two"}},
	}

	// When
	err := (boanNewsEnricher{client: client}).enrich(context.Background(), articles)

	// Then
	if err == nil {
		t.Fatal("enrichBoanNewsArticles() error = nil, want error")
	}
	if !reflect.DeepEqual(articles, want) {
		t.Fatalf("articles = %#v, want unchanged %#v", articles, want)
	}
}

func Test_enrichBoanNewsArticles_limits_page_requests_to_eight(t *testing.T) {
	// Given
	var active atomic.Int32
	var peak atomic.Int32
	entered := make(chan struct{}, 8)
	release := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		current := active.Add(1)
		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		active.Add(-1)
		return htmlResponse(http.StatusOK, `<meta name="Classification" content="비즈니스">`), nil
	})}
	articles := make([]rssdoc.Article, 9)
	for index := range articles {
		articles[index].URL = "https://www.boannews.com/news/articleView.html?idxno=" + string(rune('1'+index))
	}
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// When
	go func() {
		done <- (boanNewsEnricher{client: client}).enrich(ctx, articles)
	}()
	for range 8 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("eight page requests did not start before timeout")
		}
	}
	close(release)
	err := <-done

	// Then
	if err != nil {
		t.Fatalf("enrichBoanNewsArticles() error = %v", err)
	}
	if peak.Load() != 8 {
		t.Fatalf("peak requests = %d, want 8", peak.Load())
	}
}

func htmlResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
