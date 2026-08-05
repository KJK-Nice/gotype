package game

import "testing"

func TestActivateRevealAndResetRace(t *testing.T) {
	s := NewSessionSeeded(Config{Mode: ModeWords, WordCount: 10}, 42)
	s.WordIdx = 2
	s.ActivateReveal(RevealPeekWords)
	if s.RevealThroughWord != 5 {
		t.Fatalf("reveal through=%d want 5", s.RevealThroughWord)
	}
	if !s.IsRevealPeek(s.CursorPos() + 3) {
		t.Fatalf("expected peek ahead")
	}
	s.Typed[0] = []rune("x")
	s.rebuildChars()
	words := s.Words
	s.ResetRace()
	if len(s.Words) != len(words) || s.WordIdx != 0 || s.Typed[0] != nil {
		t.Fatal("reset race")
	}
}
