package harvester

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var errUnexpectedFeedResponse = errors.New("unexpected feed response")

func ensureFeedBody(contentType string, body []byte) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return fmt.Errorf("%w: empty body", errUnexpectedFeedResponse)
	}
	lowerContentType := strings.ToLower(contentType)
	lowerPrefix := strings.ToLower(string(trimmed[:min(len(trimmed), 128)]))
	if strings.Contains(lowerContentType, "html") || strings.HasPrefix(lowerPrefix, "<!doctype html") ||
		strings.HasPrefix(lowerPrefix, "<html") || strings.HasPrefix(lowerPrefix, "<head") ||
		strings.HasPrefix(lowerPrefix, "<body") {
		return fmt.Errorf("%w: received HTML content-type %q", errUnexpectedFeedResponse, contentType)
	}
	return nil
}
