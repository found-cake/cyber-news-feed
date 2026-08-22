package urlnorm

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
)

func Normalize(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.Contains(trimmed, " ") {
		return fallbackNormalize(trimmed)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fallbackNormalize(trimmed)
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "boannews.com") || strings.EqualFold(host, "www.boannews.com") {
		articleID := parsed.Query().Get("idxno")
		if articleID == "" {
			articleID = parsed.Query().Get("idx")
		}
		if _, err := strconv.ParseUint(articleID, 10, 64); err == nil {
			return "https://www.boannews.com/news/articleView.html?idxno=" + articleID
		}
	}
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}

func StableArticleID(canonicalURL string) string {
	sum := sha256.Sum256([]byte(canonicalURL))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fallbackNormalize(raw string) string {
	withoutFragment, _, _ := strings.Cut(raw, "#")
	return strings.TrimRight(withoutFragment, "/")
}
