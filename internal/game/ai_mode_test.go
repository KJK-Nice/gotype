package game

import (
	"testing"

	"github.com/kjkusap/monkeytype-clone/internal/words"
)

func TestModeAIEndsOnPrompt(t *testing.T) {
	if !ModeAI.EndsOnPrompt() {
		t.Fatal("ModeAI should end on prompt")
	}
	if ModeAI.String() != "ai" {
		t.Fatalf("got %q", ModeAI.String())
	}
}

func TestNewSessionFromPassage(t *testing.T) {
	s := NewSessionFromPassage(Config{Mode: ModeAI, QuoteLen: words.QuoteShort}, "fortune favors the prepared mind always", "gpt-test")
	if len(s.Words) < 3 {
		t.Fatalf("words=%v", s.Words)
	}
	if s.QuoteAuthor != "gpt-test" {
		t.Fatalf("author=%q", s.QuoteAuthor)
	}
	if s.Finished {
		t.Fatal("should not be finished")
	}
}
