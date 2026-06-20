package urlnorm

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
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
