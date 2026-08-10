package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
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
