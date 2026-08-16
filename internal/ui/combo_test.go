package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestComboHUD(t *testing.T) {
	if comboHUD(9) != "" {
		t.Fatal("hide combo under 10")
	}
	if got := comboHUD(12); got != "×12" {
		t.Fatalf("comboHUD(12)=%q", got)
	}
	if !comboHUDHot(25) || comboHUDHot(24) {
		t.Fatal("hot at 25")
	}
	if !comboHUDFire(50) || comboHUDFire(49) {
		t.Fatal("fire at 50")
	}
}

func TestChainHUD(t *testing.T) {
	if chainHUD(1) != "" {
		t.Fatal("hide chain under 2")
	}
	if got := chainHUD(3); got != "3 chain" {
		t.Fatalf("chainHUD(3)=%q", got)
	}
}

func TestViewResultShowsComboAndChain(t *testing.T) {
	m := New()
	m.finishIntro()
	m.sess = game.NewSession(game.Config{Mode: game.ModeWords, WordCount: 3})
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	m.now = now
	for _, r := range []rune(m.sess.Words[0]) {
		m.sess.HandleRune(r, now)
	}
	m.sess.HandleSpace(now)
	_ = m.finishSolo()
	out := m.viewResult()
	if !strings.Contains(out, "combo") {
		t.Fatalf("missing combo:\n%s", out)
	}
	if !strings.Contains(out, "chain") {
		t.Fatalf("missing chain:\n%s", out)
	}
}

func TestViewTypingShowsCombo(t *testing.T) {
	m := New()
	m.finishIntro()
	m.sess = game.NewSession(game.Config{Mode: game.ModeWords, WordCount: 25})
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	m.now = now
	n := 0
	for wi := 0; n < 10 && wi < len(m.sess.Words); wi++ {
		for _, r := range []rune(m.sess.Words[wi]) {
			m.sess.HandleRune(r, now)
			n++
			if n >= 10 {
				break
			}
		}
		if n < 10 {
			m.sess.HandleSpace(now)
		}
	}
	out := m.viewTyping()
	if !strings.Contains(out, "×") {
		t.Fatalf("expected combo HUD, got:\n%s", out)
	}
}
