package ui

import (
	"strings"
	"testing"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestProgressBarFill(t *testing.T) {
	got := progressBarFill(4, 10)
	want := "████" + strings.Repeat("░", 6)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApplyShake(t *testing.T) {
	in := "ab\ncd"
	if got := applyShake(in, 2); got != "  ab\n  cd" {
		t.Fatalf("right=%q", got)
	}
	if got := applyShake("  hi", -2); got != "hi" {
		t.Fatalf("left=%q", got)
	}
	if got := applyShake(in, 0); got != in {
		t.Fatalf("zero=%q", got)
	}
}

func TestShakeSettles(t *testing.T) {
	m := New()
	m.triggerShake()
	if !m.shakeAnimating() {
		t.Fatal("expected animating after impulse")
	}
	for i := 0; i < 90; i++ {
		m.stepShake()
	}
	if m.shakeAnimating() || m.shake.x != 0 {
		t.Fatalf("did not settle: x=%v v=%v", m.shake.x, m.shake.v)
	}
}

func TestRaceBarsBubblesProgress(t *testing.T) {
	m := New()
	m.roomCode = "ABCD"
	m.multiView = multi.View{
		Phase: multi.PhaseRacing,
		Players: []multi.PlayerView{
			{ID: "a", Name: "alice", You: true, Prog: multi.Progress{Chars: 0}},
			{ID: "b", Name: "bob", Prog: multi.Progress{Chars: 0}},
		},
	}
	m.syncRaceBars()
	if m.raceBars["a"].Percent() != 0 {
		t.Fatalf("initial percent=%v", m.raceBars["a"].Percent())
	}
	m.multiView.Players[0].Prog.Chars = 50
	m.multiView.Players[1].Prog.Chars = 100
	m.syncRaceBars()
	if m.raceBars["a"].Percent() != 0.5 {
		t.Fatalf("alice target=%v want 0.5", m.raceBars["a"].Percent())
	}
	if m.raceBars["b"].Percent() != 1 {
		t.Fatalf("bob target=%v want 1", m.raceBars["b"].Percent())
	}
	if m.takePending() == nil {
		t.Fatal("expected SetPercent animation cmd")
	}
	view := m.raceBarView("a")
	if view == "" {
		t.Fatal("empty bar view")
	}
}
