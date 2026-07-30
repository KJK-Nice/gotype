package ui

import (
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestNinjaOffByDefault(t *testing.T) {
	m := New()
	if m.ninjaCaret {
		t.Fatal("ninja should be off by default")
	}
}

func TestTickIntervalAdaptive(t *testing.T) {
	m := New()
	if d := m.tickInterval(); d != tickIdle {
		t.Fatalf("menu tick=%v want %v", d, tickIdle)
	}
	m.phase = phaseTyping
	m.ninjaCaret = false
	if d := m.tickInterval(); d != tickActive {
		t.Fatalf("typing tick=%v want %v", d, tickActive)
	}
	m.ninjaCaret = true
	m.caretX = 0
	m.sess = nil // caretAnimating false without sess
	if d := m.tickInterval(); d != tickActive {
		t.Fatalf("ninja idle tick=%v want %v", d, tickActive)
	}
	m.phase = phaseSpectate
	m.multiView.Phase = multi.PhaseRacing
	if d := m.tickInterval(); d != tickActive {
		t.Fatalf("spectate race tick=%v", d)
	}
	_ = time.Millisecond
}

func TestNinjaCaretSpringMovesTowardTarget(t *testing.T) {
	m := New()
	m.ninjaCaret = true
	m.caretReady = true
	m.caretX = 0
	m.caretVel = 0
	m.sess = game.NewSessionSeeded(game.Config{Mode: game.ModeWords, WordCount: 10}, 42)
	now := time.Now()
	for i := 0; i < 5; i++ {
		ch := m.sess.Chars[i]
		m.sess.HandleRune(ch.R, now)
	}
	target := float64(m.sess.CursorPos())
	if target < 4 {
		t.Fatalf("cursor=%v", target)
	}
	prev := m.caretX
	m.stepCaret()
	if m.caretX <= prev {
		t.Fatalf("spring should advance caret: %v -> %v (target %v)", prev, m.caretX, target)
	}
	for i := 0; i < 60; i++ {
		m.stepCaret()
		if !m.caretAnimating() && m.caretX == target && m.caretVel == 0 {
			return
		}
	}
	t.Fatalf("did not settle: x=%v vel=%v target=%v", m.caretX, m.caretVel, target)
}
