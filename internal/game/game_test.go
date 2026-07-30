package game

import (
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/words"
)

func TestHandleRuneAndSpace(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 3})
	now := time.Now()

	word := []rune(s.Words[0])
	for _, r := range word {
		s.HandleRune(r, now)
	}
	s.HandleSpace(now)

	if s.WordIdx != 1 {
		t.Fatalf("WordIdx = %d, want 1", s.WordIdx)
	}
	if s.Stats.Correct != len(word) {
		t.Fatalf("Correct = %d, want %d", s.Stats.Correct, len(word))
	}
}

func TestTimeModeFinish(t *testing.T) {
	s := NewSession(Config{Mode: ModeTime, Duration: time.Second})
	now := time.Now()
	s.HandleRune('a', now)
	finished := s.Tick(now.Add(2 * time.Second))
	if !finished || !s.Finished {
		t.Fatal("expected time mode to finish")
	}
	if len(s.History) < 2 {
		t.Fatalf("History len = %d, want >= 2", len(s.History))
	}
}

func TestSampleHistory(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 10})
	now := time.Now()
	s.HandleRune('a', now)
	s.Tick(now.Add(100 * time.Millisecond))
	s.Tick(now.Add(200 * time.Millisecond))
	s.Tick(now.Add(350 * time.Millisecond))
	s.finish(now.Add(500 * time.Millisecond))
	if len(s.History) < 3 {
		t.Fatalf("History len = %d, want >= 3 (subsecond samples)", len(s.History))
	}
}

func TestQuoteModeSession(t *testing.T) {
	s := NewSessionSeeded(Config{Mode: ModeQuotes, QuoteLen: words.QuoteShort}, 7)
	if s.QuoteAuthor == "" {
		t.Fatal("expected quote author")
	}
	if len(s.Words) < 3 {
		t.Fatalf("words = %d", len(s.Words))
	}
	if s.Config.WordCount != len(s.Words) {
		t.Fatalf("WordCount = %d, want %d", s.Config.WordCount, len(s.Words))
	}
	now := time.Now()
	for wi, w := range s.Words {
		for _, r := range w {
			s.HandleRune(r, now)
		}
		if wi < len(s.Words)-1 {
			s.HandleSpace(now)
		}
	}
	if !s.Finished {
		t.Fatal("expected quote race to finish")
	}
}

func TestMistypeSurvivesCorrection(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 5})
	now := time.Now()
	word := []rune(s.Words[0])
	wrong := rune('x')
	if word[0] == wrong {
		wrong = 'z'
	}

	s.HandleRune(wrong, now)
	if s.Stats.Incorrect != 1 {
		t.Fatalf("Incorrect after mistype = %d, want 1", s.Stats.Incorrect)
	}

	s.HandleBackspace(now.Add(10 * time.Millisecond))
	if s.Stats.Incorrect != 1 {
		t.Fatalf("Incorrect after backspace = %d, want 1 (errors stick)", s.Stats.Incorrect)
	}
	if len(s.Errors) != 1 {
		t.Fatalf("Errors after backspace = %d, want 1", len(s.Errors))
	}
	if len(s.Typed[0]) != 0 {
		t.Fatalf("typed should be empty after backspace, got %q", string(s.Typed[0]))
	}

	s.HandleRune(word[0], now.Add(20*time.Millisecond))
	if s.Stats.Correct != 1 {
		t.Fatalf("Correct after fix = %d, want 1", s.Stats.Correct)
	}
	if s.Stats.Incorrect != 1 {
		t.Fatalf("Incorrect after fix = %d, want 1", s.Stats.Incorrect)
	}

	snap := s.Snapshot(now.Add(time.Second))
	// 1 correct + 1 incorrect → 50%
	if snap.Accuracy < 49 || snap.Accuracy > 51 {
		t.Fatalf("Accuracy = %.1f, want ~50", snap.Accuracy)
	}
}
