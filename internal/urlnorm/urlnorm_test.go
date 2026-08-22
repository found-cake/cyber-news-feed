package urlnorm

import "testing"

func Test_Normalize_removes_fragment_trims_space_and_trailing_slash(t *testing.T) {
	// Given
	raw := "  https://example.com/path/?a=1#section  "

	// When
	got := Normalize(raw)

	// Then
	want := "https://example.com/path?a=1"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func Test_Normalize_falls_back_for_unparseable_urls(t *testing.T) {
	// Given
	raw := "  https://example.com/a b/#frag  "

	// When
	got := Normalize(raw)

	// Then
	want := "https://example.com/a b"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func Test_Normalize_maps_legacy_and_current_BoanNews_urls_to_same_canonical_url(t *testing.T) {
	// Given
	urls := []string{
		"http://www.boannews.com/media/view.asp?idx=145185&kind=1&sub_kind=",
		"https://www.boannews.com/news/articleView.html?idxno=145185",
		"https://boannews.com/news/articleView.html?idxno=145185&utm_source=rss",
	}

	for _, raw := range urls {
		// When
		got := Normalize(raw)

		// Then
		want := "https://www.boannews.com/news/articleView.html?idxno=145185"
		if got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", raw, got, want)
		}
	}
}
