package player

import "testing"

func TestNormalizeName(t *testing.T) {
	n, err := NormalizeName(" Neo_42 ")
	if err != nil {
		t.Fatal(err)
	}
	if n != "Neo_42" {
		t.Fatalf("got %q", n)
	}
	if NameKey(n) != "neo_42" {
		t.Fatalf("key = %q", NameKey(n))
	}
}

func TestNormalizeNameRejects(t *testing.T) {
	bad := []string{"", "ab", "this_name_is_way_too_long", "bad-name", "has space", "emoji😀"}
	for _, s := range bad {
		if _, err := NormalizeName(s); err == nil {
			t.Fatalf("expected reject %q", s)
		}
	}
}
