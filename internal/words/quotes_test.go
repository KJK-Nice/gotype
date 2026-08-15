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

func TestFunnyQuotesInPool(t *testing.T) {
	want := []string{
		"No one can use you if you are useless.",
		"The longer you don't pee the longer you pee.",
		"If your enemy can predict your next move then don't move.",
	}
	short := quotesFor(QuoteShort)
	found := make(map[string]bool, len(want))
	for _, q := range short {
		found[q.Text] = true
	}
	for _, text := range want {
		if !found[text] {
			t.Fatalf("missing short funny quote: %q", text)
		}
	}
	if len(FunnyQuotes) == 0 {
		t.Fatal("FunnyQuotes empty")
	}
	funnyInShort, funnyInMedium, funnyInLong := 0, 0, 0
	for _, q := range FunnyQuotes {
		if q.Author == "" {
			t.Fatalf("funny quote missing author: %q", q.Text)
		}
		switch {
		case q.matches(QuoteShort):
			funnyInShort++
		case q.matches(QuoteMedium):
			funnyInMedium++
		case q.matches(QuoteLong):
			funnyInLong++
		}
	}
	if funnyInShort == 0 || funnyInMedium == 0 || funnyInLong == 0 {
		t.Fatalf("funny quotes missing a length bucket: short=%d medium=%d long=%d", funnyInShort, funnyInMedium, funnyInLong)
	}
}

func TestPickQuoteCanReturnFunny(t *testing.T) {
	funny := make(map[string]struct{}, len(FunnyQuotes))
	for _, q := range FunnyQuotes {
		if q.matches(QuoteShort) {
			funny[q.Text] = struct{}{}
		}
	}
	for seed := uint64(1); seed <= 500; seed++ {
		q := PickQuote(QuoteShort, seed)
		if _, ok := funny[q.Text]; ok {
			return
		}
	}
	t.Fatal("PickQuote never returned a short FunnyQuotes entry in 500 seeds")
}
