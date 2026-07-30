package words

import (
	"testing"
	"unicode/utf8"
)

func TestPickQuoteSeedStable(t *testing.T) {
	a := PickQuote(QuoteShort, 42)
	b := PickQuote(QuoteShort, 42)
	if a.Text != b.Text || a.Author != b.Author {
		t.Fatalf("seed mismatch: %#v vs %#v", a, b)
	}
	c := PickQuote(QuoteShort, 43)
	if c.Text == a.Text {
		t.Fatal("expected different quote for different seed")
	}
}

func TestQuoteBucketsPopulated(t *testing.T) {
	for _, qlen := range []QuoteLen{QuoteShort, QuoteMedium, QuoteLong} {
		pool := quotesFor(qlen)
		if len(pool) == 0 {
			t.Fatalf("empty pool for %s", qlen)
		}
		for _, q := range pool {
			if !q.matches(qlen) {
				t.Fatalf("%q (%d runes) not in %s", q.Text, utf8.RuneCountInString(q.Text), qlen)
			}
			if len(q.Words()) < 3 {
				t.Fatalf("quote too short after split: %q", q.Text)
			}
		}
	}
}
