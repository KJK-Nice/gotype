package quoteai

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kjkusap/monkeytype-clone/internal/roast"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

const (
	systemPrompt = "You write short passages for a terminal typing game called gotype. " +
		"Voice: calm Stoic / Naval Ravikant-ish — practical wisdom, concise, unsentimental. " +
		"Output ONLY the passage text. No title, no author, no quotes wrapping the whole reply, no markdown, no bullet lists, no emoji. " +
		"Use plain ASCII letters, spaces, and basic punctuation (. , ; : ! ? — - '). " +
		"One or two short paragraphs max. Make it pleasant and fair to type."
)

// Passage is a generated typing prompt.
type Passage struct {
	Text   string
	Author string // LLM model id
}

// Configured reports whether AI quote generation can run (same keys as roast).
func Configured() bool {
	return roast.Configured()
}

// Generate asks the LLM for a stoic/Naval-ish passage in the given length bucket.
func Generate(ctx context.Context, qlen words.QuoteLen) (Passage, error) {
	if !Configured() {
		return Passage{}, fmt.Errorf("ai quotes unavailable")
	}
	lo, hi := runeRange(qlen)
	user := fmt.Sprintf(
		"Write one original passage for a typing race.\nTarget length: about %d–%d characters (count letters and spaces).\nTheme: stoic clarity or startup/ Naval-style wisdom.\nDo not attribute the text to a real person.",
		lo, hi,
	)
	maxTok := 120
	switch qlen {
	case words.QuoteMedium:
		maxTok = 220
	case words.QuoteLong:
		maxTok = 360
	}
	raw, err := roast.Complete(ctx, systemPrompt, user, maxTok)
	if err != nil {
		return Passage{}, err
	}
	text, err := sanitize(raw, hi+40)
	if err != nil {
		return Passage{}, err
	}
	author := roast.ModelName()
	if author == "" {
		author = "ai"
	}
	return Passage{Text: text, Author: author}, nil
}

func runeRange(qlen words.QuoteLen) (lo, hi int) {
	switch qlen {
	case words.QuoteMedium:
		return 120, 220
	case words.QuoteLong:
		return 250, 400
	default:
		return 60, 110
	}
}

func sanitize(raw string, maxRunes int) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "\"'`")
	var b strings.Builder
	b.Grow(len(raw))
	prevSpace := false
	for _, r := range raw {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
		case r > 127:
			// skip non-ASCII
			continue
		case unicode.IsPrint(r):
			if r == ' ' {
				if prevSpace || b.Len() == 0 {
					continue
				}
				prevSpace = true
				b.WriteByte(' ')
				continue
			}
			prevSpace = false
			b.WriteRune(r)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("blank ai quote")
	}
	if maxRunes > 0 && utf8.RuneCountInString(text) > maxRunes {
		runes := []rune(text)
		text = strings.TrimSpace(string(runes[:maxRunes]))
		// trim to last space so we don't cut a word mid-way if possible
		if i := strings.LastIndexByte(text, ' '); i > maxRunes/2 {
			text = text[:i]
		}
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("blank ai quote")
	}
	return text, nil
}
