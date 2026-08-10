package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestAIModeHiddenWithoutKey(t *testing.T) {
	t.Setenv("ROAST_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	m := New()
	m.finishIntro()
	out := m.viewConfig()
	if containsAIChip(out) {
		t.Fatal("ai chip should be hidden without LLM key")
	}
	modes := m.availableModes()
	for _, mode := range modes {
		if mode == game.ModeAI {
			t.Fatal("ModeAI in availableModes without key")
		}
	}
}

func TestAIModeHotkeyRequiresKey(t *testing.T) {
	t.Setenv("ROAST_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	m := New()
	m.finishIntro()
	next, _ := m.updateConfig(tea.KeyPressMsg{Text: "a", Code: 'a'})
	nm := next.(Model)
	if nm.cfg.Mode == game.ModeAI {
		t.Fatal("a should not select ai without key")
	}
}

func TestAIModeVisibleWithKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("ROAST_PROVIDER", "openai")
	m := New()
	m.finishIntro()
	if !containsAIChip(m.viewConfig()) {
		t.Fatal("expected ai chip when configured")
	}
	next, _ := m.updateConfig(tea.KeyPressMsg{Text: "a", Code: 'a'})
	nm := next.(Model)
	if nm.cfg.Mode != game.ModeAI {
		t.Fatalf("mode=%v want ai", nm.cfg.Mode)
	}
}

func TestAIStartShowsGenerating(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("ROAST_PROVIDER", "openai")
	m := New()
	m.finishIntro()
	m.cfg.Mode = game.ModeAI
	_ = m.startTest()
	if !m.aiGenerating || m.phase != phaseTyping {
		t.Fatalf("generating=%v phase=%v", m.aiGenerating, m.phase)
	}
	out := m.viewTyping()
	if !containsPlain(out, "generating") {
		t.Fatalf("want generating UI: %q", out)
	}
}

func containsAIChip(s string) bool {
	// Selected or option-styled "ai" appears as substring after strip? keep simple.
	return containsPlain(stripANSI(s), " ai") || containsPlain(stripANSI(s), "ai ") || hasModeToken(stripANSI(s), "ai")
}

func hasModeToken(s, tok string) bool {
	for i := 0; i+len(tok) <= len(s); i++ {
		if s[i:i+len(tok)] != tok {
			continue
		}
		leftOK := i == 0 || s[i-1] == ' ' || s[i-1] == '\n'
		rightOK := i+len(tok) == len(s) || s[i+len(tok)] == ' ' || s[i+len(tok)] == '\n'
		if leftOK && rightOK {
			return true
		}
	}
	return false
}

func containsPlain(s, sub string) bool {
	return len(sub) == 0 || findSub(s, sub)
}

func findSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
