package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestCinematicCountdownArt(t *testing.T) {
	m := New()
	m.multiView.Phase = multi.PhaseCountdown
	m.multiView.CountdownLeft = 3 * time.Second
	m.cdOn = false
	m.cdDigit = -1

	m.stepCountdownCinematic()
	if m.cdDigit != 3 {
		t.Fatalf("digit=%d want 3", m.cdDigit)
	}
	out := m.viewCinematicCountdown("race 1")
	if !strings.Contains(out, "██") {
		t.Fatalf("expected big digit art, got %q", out)
	}
	if !strings.Contains(out, "race 1") {
		t.Fatalf("missing banner: %q", out)
	}

	m.multiView.CountdownLeft = 0
	m.stepCountdownCinematic()
	if m.cdDigit != 0 {
		t.Fatalf("digit=%d want 0 (GO)", m.cdDigit)
	}
	goView := m.viewCinematicCountdown("")
	if !strings.Contains(goView, "type!") {
		t.Fatalf("expected GO art, got %q", goView)
	}
}
