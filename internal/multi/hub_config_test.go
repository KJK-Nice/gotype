package multi

import (
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/quoteai"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

func TestSetConfigLobbyAndPodium(t *testing.T) {
	h := NewHub()
	a, b := "a", "b"
	va, err := h.Create(a, "host", game.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(b, "guest", va.Code); err != nil {
		t.Fatal(err)
	}
	cfg := game.Config{
		Mode:      game.ModeWords,
		WordCount: 50,
		Duration:  60 * time.Second,
		QuoteLen:  words.QuoteMedium,
	}
	v, err := h.SetConfig(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if v.Config.Mode != game.ModeWords || v.Config.WordCount != 50 {
		t.Fatalf("config=%+v", v.Config)
	}
	if _, err := h.SetConfig(b, cfg); err != ErrNotHost {
		t.Fatalf("guest set config: %v", err)
	}

	now := time.Now()
	if _, err := h.Start(a, now); err != nil {
		t.Fatal(err)
	}
	now = now.Add(4 * time.Second)
	_ = h.Snapshot(a, now)
	_ = h.Report(a, Progress{WPM: 80, Chars: 40, Correct: 40, Done: true}, now)
	done := h.Report(b, Progress{WPM: 70, Chars: 35, Correct: 35, Done: true}, now)
	if done.Phase != PhaseDone {
		t.Fatalf("phase=%s", done.Phase)
	}

	cfg2 := game.Config{
		Mode:     game.ModeTime,
		Duration: 15 * time.Second,
	}
	v2, err := h.SetConfig(a, cfg2)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Config.Mode != game.ModeTime || v2.Config.Duration != 15*time.Second {
		t.Fatalf("podium config=%+v", v2.Config)
	}
}

func TestSetThreeStrikeOnPodium(t *testing.T) {
	h := NewHub()
	now := time.Now()
	a, b := "a", "b"
	va, _ := h.Create(a, "host", game.DefaultConfig)
	_, _ = h.Join(b, "guest", va.Code)
	_, _ = h.Start(a, now)
	now = now.Add(4 * time.Second)
	_ = h.Snapshot(a, now)
	_ = h.Report(a, Progress{WPM: 80, Chars: 40, Correct: 40, Done: true}, now)
	done := h.Report(b, Progress{WPM: 70, Chars: 35, Correct: 35, Done: true}, now)
	if done.Phase != PhaseDone {
		t.Fatal(done.Phase)
	}
	v, err := h.SetThreeStrike(a, true)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Config.ThreeStrike {
		t.Fatal("expected hardcore on podium")
	}
}

func TestSetConfigBlockedDuringRace(t *testing.T) {
	h := NewHub()
	now := time.Now()
	a, b := "a", "b"
	va, _ := h.Create(a, "host", game.DefaultConfig)
	_, _ = h.Join(b, "guest", va.Code)
	_, _ = h.Start(a, now)
	if _, err := h.SetConfig(a, game.DefaultConfig); err != ErrBadPhase {
		t.Fatalf("want bad phase during countdown/race, got %v", err)
	}
}

func TestStartAIEntersGenerating(t *testing.T) {
	if !quoteai.Configured() {
		t.Skip("no llm key")
	}
	h := NewHub()
	now := time.Now()
	a, b := "a", "b"
	cfg := game.Config{Mode: game.ModeAI, QuoteLen: words.QuoteShort}
	va, err := h.Create(a, "host", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(b, "guest", va.Code); err != nil {
		t.Fatal(err)
	}
	v, err := h.Start(a, now)
	if err != nil {
		t.Fatal(err)
	}
	if v.Phase != PhaseGenerating {
		t.Fatalf("phase=%s want generating", v.Phase)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		snap := h.Snapshot(a, time.Now())
		if snap.Phase == PhaseCountdown || snap.Phase == PhaseLobby {
			if snap.Phase == PhaseLobby && snap.GenError == "" {
				t.Fatal("unexpected lobby without error")
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for ai generation")
}
