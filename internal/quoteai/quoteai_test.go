package quoteai

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kjkusap/monkeytype-clone/internal/words"
)

func TestSanitizeASCII(t *testing.T) {
	got, err := sanitize("  Hello,\nworld—café!  ", 200)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "\n") || strings.Contains(got, "é") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "Hello, world") {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeTruncates(t *testing.T) {
	long := strings.Repeat("word ", 80)
	got, err := sanitize(long, 50)
	if err != nil {
		t.Fatal(err)
	}
	if utf8.RuneCountInString(got) > 50 {
		t.Fatalf("len=%d", utf8.RuneCountInString(got))
	}
}

func TestRuneRange(t *testing.T) {
	lo, hi := runeRange(words.QuoteShort)
	if lo >= hi || hi > 120 {
		t.Fatalf("%d-%d", lo, hi)
	}
}

func TestConfiguredFollowsRoast(t *testing.T) {
	t.Setenv("ROAST_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	if Configured() {
		t.Fatal("expected unconfigured")
	}
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if !Configured() {
		t.Fatal("expected configured")
	}
}
