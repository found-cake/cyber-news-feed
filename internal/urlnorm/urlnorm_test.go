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
