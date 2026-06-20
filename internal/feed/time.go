package feed

import (
	"strings"
	"time"
)

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RubyDate,
	time.UnixDate,
	time.ANSIC,
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05",
}

func parsePublishedAt(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	for _, layout := range timeLayouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			formatted := parsed.UTC().Format(time.RFC3339)
			return &formatted
		}
	}
	return nil
}
