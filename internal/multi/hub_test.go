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

	h.Report(a, Progress{WPM: 80, Correct: 40, Done: true}, now.Add(5*time.Second))
	h.Report(b, Progress{WPM: 60, Correct: 30, Done: true}, now.Add(5*time.Second))
	done := h.Snapshot(a, now.Add(5*time.Second))
	if done.Phase != PhaseDone {
		t.Fatalf("phase=%v want done", done.Phase)
	}
	if done.Players[0].Name != "alice" {
		t.Fatalf("winner=%s", done.Players[0].Name)
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
	if again.Players[0].Prog.WPM != 0 {
		t.Fatal("expected cleared progress")
	}

	// New company joins after rematch (lobby).
	c := NewPlayerID()
	vc, err := h.Join(c, "carol", va.Code)
	if err != nil {
		t.Fatal(err)
	}
	if len(vc.Players) != 3 {
		t.Fatalf("players after carol=%d", len(vc.Players))
	}
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
	h.Report(a, Progress{WPM: 50, Correct: 20, Done: true}, now.Add(20*time.Second))
	h.Report(b, Progress{WPM: 40, Correct: 15, Done: true}, now.Add(20*time.Second))
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

func TestGenerateSeedMatch(t *testing.T) {
	// smoke: two seeded sessions should share first word — tested via game package
}
