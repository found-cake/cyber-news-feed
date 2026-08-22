package harvester

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/found-cake/cyber-news-feed/internal/rssdoc"
)

func Test_enrichBoanNewsArticles_continues_after_non_forbidden_page_error(t *testing.T) {
	// Given
	var pageRequests atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pageRequests.Add(1)
		if request.URL.Query().Get("idxno") == "1" {
			return htmlResponse(http.StatusBadGateway, "upstream failed"), nil
		}
		return htmlResponse(http.StatusOK, `<meta name="Classification" content="사건·사고 &gt; 인사이트">`), nil
	})}
	articles := []rssdoc.Article{
		{URL: "https://www.boannews.com/news/articleView.html?idxno=1"},
		{URL: "https://www.boannews.com/news/articleView.html?idxno=2"},
	}

	// When
	err := (boanNewsEnricher{client: client}).enrich(context.Background(), articles)

	// Then
	if err == nil {
		t.Fatal("enrich() error = nil, want first page error")
	}
	if pageRequests.Load() != 2 {
		t.Fatalf("page requests = %d, want 2", pageRequests.Load())
	}
	if len(articles[0].Categories) != 0 {
		t.Fatalf("first categories = %#v, want empty", articles[0].Categories)
	}
	want := []string{"사건·사고", "인사이트"}
	if len(articles[1].Categories) != len(want) || articles[1].Categories[0] != want[0] || articles[1].Categories[1] != want[1] {
		t.Fatalf("second categories = %#v, want %#v", articles[1].Categories, want)
	}
}
