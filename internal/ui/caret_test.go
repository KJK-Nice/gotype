package ui

import (
	"testing"
	"time"

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
