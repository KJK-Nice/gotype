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

func TestRaceBarsSpring(t *testing.T) {
	m := New()
	m.roomCode = "ABCD"
	m.multiView = multi.View{
		Phase: multi.PhaseRacing,
		Players: []multi.PlayerView{
			{ID: "a", Name: "alice", Prog: multi.Progress{Chars: 0}},
			{ID: "b", Name: "bob", Prog: multi.Progress{Chars: 0}},
		},
	}
	m.stepRaceBars()
	m.multiView.Players[0].Prog.Chars = 50
	m.multiView.Players[1].Prog.Chars = 100
	for i := 0; i < 80; i++ {
		m.stepRaceBars()
	}
	if !m.barFill["a"].settled(5, barSettlePos, barSettleVel) {
		t.Fatalf("alice fill=%v want ~5", m.barFill["a"].x)
	}
	if !m.barFill["b"].settled(10, barSettlePos, barSettleVel) {
		t.Fatalf("bob fill=%v want ~10", m.barFill["b"].x)
	}
	bar := m.raceBarFor("a", 50, 100)
	if len([]rune(bar)) != raceBarWidth {
		t.Fatalf("bar=%q", bar)
	}
}
