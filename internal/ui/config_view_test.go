package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestConfigNavFocusSwap(t *testing.T) {
	m := New()
	m.finishIntro()
	if m.focus != focusMode {
		t.Fatalf("focus=%d want mode", m.focus)
	}

	next, _ := m.updateConfig(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(Model)
	if m.focus != focusValue {
		t.Fatalf("down: focus=%d want value", m.focus)
	}

	next, _ = m.updateConfig(tea.KeyPressMsg{Code: tea.KeyUp})
	m = next.(Model)
	if m.focus != focusMode {
		t.Fatalf("up: focus=%d want mode", m.focus)
	}
}

func TestConfigNavLeftRightNudges(t *testing.T) {
	m := New()
	m.finishIntro()
	m.cfg.Mode = game.ModeTime
	m.cfg.Duration = game.TimeOptions[1]
	m.focus = focusValue

	next, _ := m.updateConfig(tea.KeyPressMsg{Code: tea.KeyRight})
	m = next.(Model)
	if m.cfg.Duration == game.TimeOptions[1] {
		t.Fatal("right should change duration")
	}

	m.focus = focusMode
	m.cfg.Mode = game.ModeTime
	next, _ = m.updateConfig(tea.KeyPressMsg{Code: tea.KeyRight})
	m = next.(Model)
	if m.cfg.Mode == game.ModeTime {
		t.Fatal("right should change mode")
	}
}

func TestViewConfigNoFooter(t *testing.T) {
	m := New()
	m.finishIntro()
	out := m.viewConfig()
	// Old footer was a dense bubbles help bar; inline rows are one hint per line.
	for _, line := range strings.Split(out, "\n") {
		if strings.Count(line, "  ") >= 6 && strings.Contains(line, "theme") && strings.Contains(line, "voice") {
			t.Fatalf("footer-like help bar still present: %q", line)
		}
	}
	if !strings.Contains(out, "theme") || !strings.Contains(out, "enter") {
		t.Fatalf("missing inline hints: %q", out[:min(300, len(out))])
	}
}

func TestViewConfigShowsColoredValue(t *testing.T) {
	m := New()
	m.finishIntro()
	m.now = time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	out := m.viewConfig()
	plain := stripANSI(out)
	if !strings.Contains(plain, ThemeName(m.themeIdx)) {
		t.Fatalf("expected theme value %q in %q", ThemeName(m.themeIdx), plain)
	}
}
