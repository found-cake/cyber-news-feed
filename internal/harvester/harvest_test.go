package harvester

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/feed"
	"github.com/found-cake/cyber-news-feed/internal/jsonstore"
	"github.com/found-cake/cyber-news-feed/internal/source"
	"github.com/found-cake/cyber-news-feed/pkg/rssjson"
)

func Test_runWithSources_retries_failed_sources_after_first_pass_finishes(t *testing.T) {
	// Given
	order := []string{}
	firstRequests := 0
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstRequests++
		if r.Header.Get("Cache-Control") == "no-cache" {
			order = append(order, "first:retry")
		} else {
			order = append(order, "first:first-pass")
		}
		if firstRequests == 1 {
			http.NotFound(w, r)
			return
		}
		writeTestRSS(w, "Recovered first", "https://example.com/first")
	}))
	defer first.Close()

	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "second:first-pass")
		writeTestRSS(w, "Second", "https://example.com/second")
	}))
	defer second.Close()

	cfg := Config{
		OutputDir:     t.TempDir(),
		RetentionDays: 10,
		Client:        first.Client(),
	}
	sources := []source.Config{
		{Name: "first", Feeds: []source.Feed{{URL: first.URL}}},
		{Name: "second", Feeds: []source.Feed{{URL: second.URL}}},
	}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 2 || summary.Failed != 0 {
		t.Fatalf("Summary = %#v", summary)
	}
	wantOrder := []string{"first:first-pass", "second:first-pass", "first:retry"}
	if len(order) != len(wantOrder) {
		t.Fatalf("order = %#v, want %#v", order, wantOrder)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("order = %#v, want %#v", order, wantOrder)
		}
	}
	assertSourceOK(t, cfg.OutputDir, "first", 1)
	assertSourceOK(t, cfg.OutputDir, "second", 1)
}

func Test_runWithSources_writes_securityweek_image_metadata(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<image>
  <url>https://www.securityweek.com/wp-content/uploads/2023/01/cropped-SecurityWeek-Icon-32x32.jpeg</url>
  <title>SecurityWeek</title>
  <link>https://www.securityweek.com/</link>
  <width>32</width>
  <height>32</height>
</image>
<item>
  <title>SecurityWeek image</title>
  <link>https://www.securityweek.com/example</link>
</item></channel></rss>`))
	}))
	defer server.Close()

	cfg := Config{
		OutputDir:     t.TempDir(),
		RetentionDays: 10,
		Client:        server.Client(),
	}
	sources := []source.Config{
		{Name: "securityweek", Feeds: []source.Feed{{URL: server.URL}}, Metadata: testSecurityWeekMetadata},
	}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), sources)

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Processed != 1 || summary.Failed != 0 {
		t.Fatalf("Summary = %#v", summary)
	}
	assertSecurityWeekImage(t, cfg.OutputDir, "https://www.securityweek.com/wp-content/uploads/2023/01/cropped-SecurityWeek-Icon-32x32.jpeg")
}

func Test_runWithSources_enriches_BoanNews_categories_from_article_pages(t *testing.T) {
	// Given
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Hostname() {
		case "feed.test":
			return rssResponse(`
<item><title>First</title><link>https://www.boannews.com/news/articleView.html?idxno=1</link></item>
<item><title>Second</title><link>https://www.boannews.com/news/articleView.html?idxno=2</link></item>`), nil
		case "www.boannews.com":
			classification := "비즈니스 &gt; 인사이트"
			if request.URL.Query().Get("idxno") == "2" {
				classification = "사건·사고 &gt; 국제"
			}
			return htmlResponse(http.StatusOK, `<meta name="Classification" content="`+classification+`">`), nil
		default:
			return htmlResponse(http.StatusNotFound, "not found"), nil
		}
	})}
	src := source.Config{
		Name:  "boannews",
		Kind:  source.BoanNews,
		Feeds: []source.Feed{{URL: "https://feed.test/rss"}},
	}
	cfg := Config{OutputDir: t.TempDir(), RetentionDays: 10, Client: client}

	// When
	summary, err := runWithSources(context.Background(), cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)), []source.Config{src})

	// Then
	if err != nil {
		t.Fatalf("runWithSources() error = %v", err)
	}
	if summary.Failed != 0 {
		t.Fatalf("Summary = %#v", summary)
	}
	document, err := jsonstore.Load(cfg.OutputDir, "boannews")
	if err != nil {
		t.Fatalf("load harvested document: %v", err)
	}
	articles := document.Articles
	if len(articles) != 2 {
		t.Fatalf("articles = %#v, want 2", articles)
	}
	want := [][]string{{"비즈니스", "인사이트"}, {"사건·사고", "국제"}}
	for index := range articles {
		if len(articles[index].Categories) != len(want[index]) || articles[index].Categories[0] != want[index][0] || articles[index].Categories[1] != want[index][1] {
			t.Fatalf("article %d categories = %#v, want %#v", index, articles[index].Categories, want[index])
		}
	}
}

func Test_fetchSource_does_not_fetch_article_pages_for_other_sources(t *testing.T) {
	// Given
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		return rssResponse(`<item><title>Article</title><link>https://www.boannews.com/news/articleView.html?idxno=1</link><category>rss</category></item>`), nil
	})}
	src := source.Config{Name: "other", Feeds: []source.Feed{{URL: "https://feed.test/rss"}}}

	// When
	articles, err := fetchSource(context.Background(), fetchSourceRequest{client: client, src: src})

	// Then
	if err != nil {
		t.Fatalf("fetchSource() error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only the feed request", requests)
	}
	if len(articles) != 1 || len(articles[0].Categories) != 1 || articles[0].Categories[0] != "rss" {
		t.Fatalf("articles = %#v", articles)
	}
}

func writeTestRSS(w http.ResponseWriter, title string, link string) {
	w.Header().Set("Content-Type", "application/rss+xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><item>
  <title>` + title + `</title>
  <link>` + link + `</link>
</item></channel></rss>`))
}

func rssResponse(items string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/rss+xml"}},
		Body: io.NopCloser(strings.NewReader(
			`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel>` + items + `</channel></rss>`,
		)),
	}
}

func assertSourceOK(t *testing.T, outputDir string, name string, articles int) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, name+".json"))
	if err != nil {
		t.Fatalf("read %s json: %v", name, err)
	}
	var doc struct {
		Status struct {
			OK bool `json:"ok"`
		} `json:"status"`
		Articles []struct{} `json:"articles"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode %s json: %v", name, err)
	}
	if !doc.Status.OK || len(doc.Articles) != articles {
		t.Fatalf("%s document status/articles = ok:%v articles:%d", name, doc.Status.OK, len(doc.Articles))
	}
}

func assertSecurityWeekImage(t *testing.T, outputDir string, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, "securityweek.json"))
	if err != nil {
		t.Fatalf("read securityweek json: %v", err)
	}
	var doc struct {
		Articles []struct {
			SourceMetadata struct {
				SecurityWeek struct {
					Image struct {
						URL    string `json:"url"`
						Title  string `json:"title"`
						Link   string `json:"link"`
						Width  string `json:"width"`
						Height string `json:"height"`
					} `json:"image"`
				} `json:"securityweek"`
			} `json:"source_metadata"`
		} `json:"articles"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode securityweek json: %v", err)
	}
	if len(doc.Articles) != 1 || doc.Articles[0].SourceMetadata.SecurityWeek.Image.URL != want {
		t.Fatalf("securityweek image metadata = %#v, want %q", doc.Articles, want)
	}
}

func testSecurityWeekMetadata(item feed.Item) rssjson.SourceMetadata {
	metadata := item.SourceMetadata
	if metadata.Image == (feed.SourceImage{}) {
		return rssjson.SourceMetadata{}
	}
	return rssjson.NewSourceMetadata("securityweek", rssjson.MetadataObject{
		rssjson.MetadataNested("image", rssjson.MetadataObject{
			rssjson.MetadataText("url", metadata.Image.URL),
			rssjson.MetadataText("title", metadata.Image.Title),
			rssjson.MetadataText("link", metadata.Image.Link),
			rssjson.MetadataText("width", metadata.Image.Width),
			rssjson.MetadataText("height", metadata.Image.Height),
		}),
	})
}
