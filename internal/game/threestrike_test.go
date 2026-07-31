package game

import (
	"testing"
	"time"
)

func TestThreeStrikeDNFOnTypos(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 10, ThreeStrike: true})
	if s.HP != ThreeStrikeStartHP {
		t.Fatalf("HP = %d", s.HP)
	}
	now := time.Now()
	// Force three incorrect commits against first letter.
	want := []rune(s.Words[0])[0]
	wrong := want + 1
	if wrong == want {
		wrong = 'x'
	}
	for i := 0; i < 3; i++ {
		s.HandleRune(wrong, now)
		if i < 2 && s.DNF {
			t.Fatalf("DNF too early at hit %d", i+1)
		}
	}
	if !s.DNF || s.HP != 0 || !s.Finished {
		t.Fatalf("DNF=%v HP=%d Finished=%v", s.DNF, s.HP, s.Finished)
	}
}

func TestThreeStrikeNoBackspaceRefund(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 10, ThreeStrike: true})
	now := time.Now()
	want := []rune(s.Words[0])[0]
	wrong := want + 1
	s.HandleRune(wrong, now)
	if s.HP != 2 {
		t.Fatalf("HP = %d after typo", s.HP)
	}
	s.HandleBackspace(now)
	if s.HP != 2 {
		t.Fatalf("backspace refunded HP: %d", s.HP)
	}
}

func TestAddHeart(t *testing.T) {
	s := NewSession(Config{Mode: ModeWords, WordCount: 5, ThreeStrike: true})
	if !s.AddHeart() || s.HP != 4 {
		t.Fatalf("HP = %d", s.HP)
	}
	s.HP = ThreeStrikeMaxHP
	if s.AddHeart() {
		t.Fatal("should not exceed max HP")
	}
}
