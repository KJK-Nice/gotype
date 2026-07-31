package multi

import (
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestSetThreeStrikeLobbyOnly(t *testing.T) {
	h := NewHub()
	now := time.Now()
	a, b := "a", "b"
	va, err := h.Create(a, "host", game.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(b, "guest", va.Code); err != nil {
		t.Fatal(err)
	}
	v, err := h.SetThreeStrike(a, true)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Config.ThreeStrike {
		t.Fatal("expected ThreeStrike on")
	}
	if _, err := h.SetThreeStrike(b, false); err != ErrNotHost {
		t.Fatalf("want not host, got %v", err)
	}
	if _, err := h.Start(a, now); err != nil {
		t.Fatal(err)
	}
	if _, err := h.SetThreeStrike(a, false); err != ErrBadPhase {
		t.Fatalf("want bad phase after start, got %v", err)
	}
}

func TestDNFExcludedFromWinner(t *testing.T) {
	h := NewHub()
	now := time.Now()
	cfg := game.Config{Mode: game.ModeWords, WordCount: 10, ThreeStrike: true}
	va, _ := h.Create("a", "alice", cfg)
	_, _ = h.Join("b", "bob", va.Code)
	_, _ = h.Start("a", now)
	// advance past countdown
	now = now.Add(4 * time.Second)
	_ = h.Snapshot("a", now)
	_ = h.Report("a", Progress{WPM: 10, Chars: 5, Done: true, DNF: true}, now)
	v := h.Report("b", Progress{WPM: 80, Chars: 40, Correct: 40, Done: true}, now)
	if v.Phase != PhaseDone {
		t.Fatalf("phase=%s", v.Phase)
	}
	if v.RaceWinnerName != "bob" {
		t.Fatalf("winner=%q want bob", v.RaceWinnerName)
	}
}
