package multi

import (
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestCreateJoinStartRace(t *testing.T) {
	h := NewHub()
	cfg := game.Config{Mode: game.ModeTime, Duration: 30 * time.Second, WordCount: 25}
	a := NewPlayerID()
	b := NewPlayerID()

	va, err := h.Create(a, "alice", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(va.Code) != 4 {
		t.Fatalf("code=%q", va.Code)
	}

	vb, err := h.Join(b, "bob", va.Code)
	if err != nil {
		t.Fatal(err)
	}
	if vb.Code != va.Code {
		t.Fatal("code mismatch")
	}

	now := time.Now()
	if _, err := h.Start(b, now); err != ErrNotHost {
		t.Fatalf("want ErrNotHost, got %v", err)
	}
	vs, err := h.Start(a, now)
	if err != nil {
		t.Fatal(err)
	}
	if vs.Phase != PhaseCountdown {
		t.Fatalf("phase=%v", vs.Phase)
	}

	racing := h.Snapshot(a, now.Add(4*time.Second))
	if racing.Phase != PhaseRacing {
		t.Fatalf("phase=%v want racing", racing.Phase)
	}
	if racing.Seed == 0 {
		t.Fatal("expected seed")
	}

	h.Report(a, Progress{WPM: 80, Correct: 40, Chars: 40, Done: true}, now.Add(5*time.Second))
	h.Report(b, Progress{WPM: 60, Correct: 30, Chars: 30, Done: true}, now.Add(5*time.Second))
	done := h.Snapshot(a, now.Add(5*time.Second))
	if done.Phase != PhaseDone {
		t.Fatalf("phase=%v want done", done.Phase)
	}
	if done.Players[0].Name != "alice" {
		t.Fatalf("winner=%s", done.Players[0].Name)
	}
	if done.Players[0].MatchWins != 1 || done.Players[0].Streak != 1 {
		t.Fatalf("alice wins/streak=%d/%d", done.Players[0].MatchWins, done.Players[0].Streak)
	}
	if done.Players[1].Crown {
		t.Fatal("bob should not have crown")
	}

	again, err := h.Rematch(a, now.Add(6*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if again.Phase != PhaseLobby {
		t.Fatalf("rematch phase=%v", again.Phase)
	}
	if again.Code != va.Code {
		t.Fatalf("code changed: %s vs %s", again.Code, va.Code)
	}
	if len(again.Players) != 2 {
		t.Fatalf("players=%d", len(again.Players))
	}

	c := NewPlayerID()
	vc, err := h.Join(c, "carol", va.Code)
	if err != nil {
		t.Fatal(err)
	}
	if len(vc.Players) != 3 {
		t.Fatalf("players after carol=%d", len(vc.Players))
	}
}

func TestBestOf3(t *testing.T) {
	h := NewHub()
	cfg := game.Config{Mode: game.ModeTime, Duration: 10 * time.Second}
	a, b := NewPlayerID(), NewPlayerID()
	va, _ := h.Create(a, "alice", cfg)
	_, _ = h.Join(b, "bob", va.Code)
	now := time.Now()

	play := func(awpm, bwpm float64, t0 time.Time) {
		_, _ = h.Start(a, t0)
		_ = h.Snapshot(a, t0.Add(4*time.Second))
		h.Report(a, Progress{WPM: awpm, Chars: 20, Correct: 20, Done: true}, t0.Add(5*time.Second))
		h.Report(b, Progress{WPM: bwpm, Chars: 10, Correct: 10, Done: true}, t0.Add(5*time.Second))
		_ = h.Snapshot(a, t0.Add(5*time.Second))
		_, _ = h.Rematch(a, t0.Add(6*time.Second))
	}

	play(90, 50, now)
	play(85, 40, now.Add(time.Minute))
	// alice should have won bo3 — rematch after second win resets?
	// After 2 wins MatchOver; Rematch clears series.
	v := h.Snapshot(a, now.Add(2*time.Minute))
	// After rematch of match-over, scores cleared — check mid-series instead.
	h2 := NewHub()
	va2, _ := h2.Create(a, "alice", cfg)
	_, _ = h2.Join(b, "bob", va2.Code)
	t0 := time.Now()
	_, _ = h2.Start(a, t0)
	_ = h2.Snapshot(a, t0.Add(4*time.Second))
	h2.Report(a, Progress{WPM: 90, Chars: 20, Correct: 20, Done: true}, t0.Add(5*time.Second))
	h2.Report(b, Progress{WPM: 40, Chars: 10, Correct: 10, Done: true}, t0.Add(5*time.Second))
	d1 := h2.Snapshot(a, t0.Add(5*time.Second))
	if d1.MatchOver {
		t.Fatal("match should not be over after 1 win")
	}
	_, _ = h2.Rematch(a, t0.Add(6*time.Second))
	_, _ = h2.Start(a, t0.Add(10*time.Second))
	_ = h2.Snapshot(a, t0.Add(14*time.Second))
	h2.Report(a, Progress{WPM: 88, Chars: 20, Correct: 20, Done: true}, t0.Add(15*time.Second))
	h2.Report(b, Progress{WPM: 50, Chars: 10, Correct: 10, Done: true}, t0.Add(15*time.Second))
	d2 := h2.Snapshot(a, t0.Add(15*time.Second))
	if !d2.MatchOver || d2.MatchWinnerName != "alice" {
		t.Fatalf("matchOver=%v winner=%q", d2.MatchOver, d2.MatchWinnerName)
	}
	_ = v
}

func TestJoinDuringPodium(t *testing.T) {
	h := NewHub()
	cfg := game.Config{Mode: game.ModeTime, Duration: 15 * time.Second, WordCount: 25}
	a, b := NewPlayerID(), NewPlayerID()
	va, _ := h.Create(a, "alice", cfg)
	_, _ = h.Join(b, "bob", va.Code)
	now := time.Now()
	_, _ = h.Start(a, now)
	_ = h.Snapshot(a, now.Add(4*time.Second))
	h.Report(a, Progress{WPM: 50, Correct: 20, Chars: 20, Done: true}, now.Add(20*time.Second))
	h.Report(b, Progress{WPM: 40, Correct: 15, Chars: 15, Done: true}, now.Add(20*time.Second))
	done := h.Snapshot(a, now.Add(20*time.Second))
	if done.Phase != PhaseDone {
		t.Fatalf("phase=%v", done.Phase)
	}
	c := NewPlayerID()
	vc, err := h.Join(c, "carol", va.Code)
	if err != nil {
		t.Fatal(err)
	}
	if vc.Phase != PhaseDone {
		t.Fatalf("join during podium phase=%v", vc.Phase)
	}
	if len(vc.Players) != 3 {
		t.Fatalf("players=%d", len(vc.Players))
	}
}

func TestChatGG(t *testing.T) {
	h := NewHub()
	a := NewPlayerID()
	va, _ := h.Create(a, "alice", game.DefaultConfig)
	v, err := h.Say(a, "/gg")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Chat) != 1 || v.Chat[0].Text != "gg" {
		t.Fatalf("chat=%v", v.Chat)
	}
	_ = va
}

func TestNormalizeChat(t *testing.T) {
	if normalizeChat("/glhf") != "glhf" {
		t.Fatal(normalizeChat("/glhf"))
	}
}
