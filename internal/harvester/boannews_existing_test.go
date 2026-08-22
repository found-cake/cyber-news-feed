package harvester

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/jsonstore"
	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
	"github.com/found-cake/cyber-news-feed/internal/source"
)

func Test_runWithSources_fetches_BoanNews_categories_only_for_uncategorized_articles(t *testing.T) {
	// Given
	outputDir := t.TempDir()
	existing := rssdoc.Document{
		SchemaVersion: rssdoc.SchemaVersion,
		Source:        "boannews",
		Articles: []rssdoc.Article{
			{URL: "http://www.boannews.com/media/view.asp?idx=1&kind=1", Categories: []string{"cached"}},
			{URL: "https://www.boannews.com/news/articleView.html?idxno=2", Categories: []string{}},
		},
	}
	if err := jsonstore.Write(outputDir, existing); err != nil {
		t.Fatalf("write existing document: %v", err)
	}

	var pageRequests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Hostname() {
		case "feed.test":
			return rssResponse(`
<item><title>Cached</title><link>https://www.boannews.com/news/articleView.html?idxno=1</link></item>
<item><title>Known empty</title><link>https://www.boannews.com/news/articleView.html?idxno=2</link></item>
<item><title>New</title><link>https://www.boannews.com/news/articleView.html?idxno=3</link></item>`), nil
		case "www.boannews.com":
			pageRequests.Add(1)
			if request.URL.Query().Get("idxno") == "1" {
				return htmlResponse(http.StatusBadGateway, "categorized article was requested"), nil
			}
			classification := "공공·정책"
			if request.URL.Query().Get("idxno") == "3" {
				classification = "사건·사고 &gt; 인사이트"
			}
			return htmlResponse(http.StatusOK, `<meta name="Classification" content="`+classification+`">`), nil
		default:
			return htmlResponse(http.StatusNotFound, "not found"), nil
		}
	})}
	cfg := Config{OutputDir: outputDir, RetentionDays: 10, Client: client}
	sources := []source.Config{{
		Name:  "boannews",
		Kind:  source.BoanNews,
		Feeds: []source.Feed{{URL: "https://feed.test/rss"}},
	}}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Failed != 0 || pageRequests.Load() != 2 {
		t.Fatalf("summary/page requests = %#v/%d, want success/2", summary, pageRequests.Load())
	}
	document, err := jsonstore.Load(outputDir, "boannews")
	if err != nil {
		t.Fatalf("load harvested document: %v", err)
	}
	want := map[string][]string{
		"https://www.boannews.com/news/articleView.html?idxno=1": {"cached"},
		"https://www.boannews.com/news/articleView.html?idxno=2": {"공공·정책"},
		"https://www.boannews.com/news/articleView.html?idxno=3": {"사건·사고", "인사이트"},
	}
	if len(document.Articles) != len(want) {
		t.Fatalf("articles = %#v, want %d", document.Articles, len(want))
	}
	for _, article := range document.Articles {
		if !reflect.DeepEqual(article.Categories, want[article.URL]) {
			t.Fatalf("categories for %s = %#v, want %#v", article.URL, article.Categories, want[article.URL])
		}
	}
}

func Test_runWithSources_preserves_new_BoanNews_article_when_category_lookup_fails(t *testing.T) {
	// Given
	outputDir := t.TempDir()
	var pageRequests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Hostname() {
		case "feed.test":
			return rssResponse(`
<item><title>Blocked</title><link>https://www.boannews.com/news/articleView.html?idxno=4</link></item>
<item><title>Must not be requested</title><link>https://www.boannews.com/news/articleView.html?idxno=5</link></item>`), nil
		case "www.boannews.com":
			pageRequests.Add(1)
			if request.URL.Query().Get("idxno") == "5" {
				t.Fatalf("article request continued after a 403: %s", request.URL)
			}
			return htmlResponse(http.StatusForbidden, "blocked"), nil
		default:
			return htmlResponse(http.StatusNotFound, "not found"), nil
		}
	})}
	cfg := Config{OutputDir: outputDir, RetentionDays: 10, Client: client}
	sources := []source.Config{{
		Name:  "boannews",
		Kind:  source.BoanNews,
		Feeds: []source.Feed{{URL: "https://feed.test/rss"}},
	}}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Failed != 0 || pageRequests.Load() != 1 {
		t.Fatalf("summary/page requests = %#v/%d, want success/1", summary, pageRequests.Load())
	}
	document, err := jsonstore.Load(outputDir, "boannews")
	if err != nil {
		t.Fatalf("load harvested document: %v", err)
	}
	if !document.Status.OK || len(document.Articles) != 2 {
		t.Fatalf("document = %#v, want successful RSS with two uncategorized articles", document)
	}
	for _, article := range document.Articles {
		if len(article.Categories) != 0 {
			t.Fatalf("categories for %s = %#v, want empty", article.URL, article.Categories)
		}
	}
}

func Test_runWithSources_does_not_retry_BoanNews_feed_after_forbidden_response(t *testing.T) {
	// Given
	outputDir := t.TempDir()
	var feedRequests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		feedRequests.Add(1)
		return htmlResponse(http.StatusForbidden, "blocked"), nil
	})}
	cfg := Config{OutputDir: outputDir, RetentionDays: 10, Client: client}
	sources := []source.Config{{
		Name:  "boannews",
		Kind:  source.BoanNews,
		Feeds: []source.Feed{{URL: "https://feed.test/rss"}},
	}}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 1 || summary.Failed != 1 || feedRequests.Load() != 1 {
		t.Fatalf("summary/feed requests = %#v/%d, want one failed source/1", summary, feedRequests.Load())
	}
	document, err := jsonstore.Load(outputDir, "boannews")
	if err != nil {
		t.Fatalf("load harvested document: %v", err)
	}
	if document.Status.OK {
		t.Fatalf("document status = %#v, want failed RSS status", document.Status)
	}
}
