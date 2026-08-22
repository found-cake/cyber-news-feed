package harvester

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func Test_fetchBoanNewsCategories_bounds_response_body_drain(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		maxRead      int64
		wantTooLarge bool
	}{
		{name: "oversized success response", status: http.StatusOK, maxRead: boanNewsPageMaxBytes + 1 + 128<<10, wantTooLarge: true},
		{name: "error response", status: http.StatusBadGateway, maxRead: 128 << 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			body := &countingReadCloser{reader: strings.NewReader(strings.Repeat("x", 3<<20))}
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.status,
					Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
					Body:       body,
				}, nil
			})}

			// When
			_, err := (boanNewsEnricher{client: client}).fetchCategories(
				context.Background(),
				"https://www.boannews.com/news/articleView.html?idxno=1",
			)

			// Then
			if err == nil {
				t.Fatal("fetchCategories() error = nil, want response error")
			}
			if test.wantTooLarge && !errors.Is(err, errBoanNewsPageTooLarge) {
				t.Fatalf("fetchCategories() error = %v, want %v", err, errBoanNewsPageTooLarge)
			}
			if body.bytesRead > test.maxRead {
				t.Fatalf("response bytes read = %d, want at most %d", body.bytesRead, test.maxRead)
			}
		})
	}
}

func Test_fetchBoanNewsCategories_enforces_redirect_limit_before_custom_policy(t *testing.T) {
	// Given
	var requests atomic.Int32
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requests.Add(1)
			hop, err := strconv.Atoi(request.URL.Query().Get("hop"))
			if err != nil {
				hop = 0
			}
			if hop >= 12 {
				return htmlResponse(http.StatusOK, `<meta name="Classification" content="인사이트">`), nil
			}
			nextURL := *request.URL
			query := nextURL.Query()
			query.Set("hop", strconv.Itoa(hop+1))
			nextURL.RawQuery = query.Encode()
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{nextURL.String()}},
				Body:       io.NopCloser(strings.NewReader("redirect")),
			}, nil
		}),
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return nil
		},
	}

	// When
	_, err := (boanNewsEnricher{client: client}).fetchCategories(
		context.Background(),
		"https://www.boannews.com/news/articleView.html?idxno=1&hop=0",
	)

	// Then
	if !errors.Is(err, errBoanNewsRedirectLimit) {
		t.Fatalf("fetchCategories() error = %v, want %v", err, errBoanNewsRedirectLimit)
	}
	if requests.Load() != 10 {
		t.Fatalf("page requests = %d, want 10", requests.Load())
	}
}

type countingReadCloser struct {
	reader    io.Reader
	bytesRead int64
}

func (body *countingReadCloser) Read(buffer []byte) (int, error) {
	read, err := body.reader.Read(buffer)
	body.bytesRead += int64(read)
	return read, err
}

func (body *countingReadCloser) Close() error {
	return nil
}
