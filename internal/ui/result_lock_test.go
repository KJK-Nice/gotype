package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestResultKeysLockedFor3s(t *testing.T) {
	m := New()
	m.finishIntro()
	m.sess = game.NewSession(game.DefaultConfig)
	m.now = time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	_ = m.finishSolo()
	if m.phase != phaseResult {
		t.Fatalf("phase=%d", m.phase)
	}
	if !m.resultKeysLocked() {
		t.Fatal("expected lock right after finish")
	}

	next, _ := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := next.(Model)
	if nm.phase != phaseResult {
		t.Fatal("enter should be ignored during lock")
	}

	next, _ = nm.handleKey(tea.KeyPressMsg{Text: "i", Code: 'i'})
	nm = next.(Model)
	if nm.prog != progNone {
		t.Fatal("progress hotkey should be ignored during lock")
	}

	nm.now = nm.resultKeysUntil
	if nm.resultKeysLocked() {
		t.Fatal("lock should end at unlock time")
	}
	out := nm.viewResult()
	if strings.Contains(out, "keys in") {
		t.Fatal("countdown hint should be gone after unlock")
	}
}

func TestResultLockCountdownHint(t *testing.T) {
	m := New()
	m.finishIntro()
	m.sess = game.NewSession(game.DefaultConfig)
	m.now = time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	_ = m.finishSolo()
	out := m.viewResult()
	if !strings.Contains(out, "keys in 3s") {
		t.Fatalf("want countdown hint, got %q", out[len(out)-80:])
	}
}

func TestPodiumKeysLockedFor3s(t *testing.T) {
	m := New()
	m.finishIntro()
	m.now = time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	m.roomCode = "ABCD"
	m.phase = phaseTyping
	m.raceStarted = true
	m.sess = game.NewSession(game.DefaultConfig)
	m.multiView = multi.View{Code: "ABCD", Phase: multi.PhaseDone, RaceNumber: 1}
	m.applyMultiView(m.multiView)
	if m.phase != phasePodium {
		t.Fatalf("phase=%d want podium", m.phase)
	}
	if !m.resultKeysLocked() {
		t.Fatal("expected podium key lock")
	}

	next, _ := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := next.(Model)
	if nm.phase != phasePodium {
		t.Fatal("enter should be ignored during podium lock")
	}

	out := nm.viewPodium()
	if !strings.Contains(out, "keys in") {
		t.Fatalf("want countdown on podium, got tail %q", out[max(0, len(out)-60):])
	}

	// Re-applying Done while already on podium must not reset the lock clock.
	until := nm.resultKeysUntil
	nm.now = until.Add(-time.Second)
	nm.applyMultiView(nm.multiView)
	if !nm.resultKeysUntil.Equal(until) {
		t.Fatal("lock should not re-arm while staying on podium")
	}

	nm.now = until
	if nm.resultKeysLocked() {
		t.Fatal("lock should end at unlock time")
	}
}

