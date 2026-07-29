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
}

func TestGenerateSeedMatch(t *testing.T) {
	// smoke: two seeded sessions should share first word — tested via game package
}
