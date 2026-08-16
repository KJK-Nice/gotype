package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestConfigLOpensLoginNotClaimCode(t *testing.T) {
	m := guestWithApp(t)
	next, _ := m.handleKey(tea.KeyPressMsg{Text: "l", Code: 'l'})
	nm := next.(Model)
	if nm.loginMode == loginIdle {
		t.Fatal("l should open login")
	}
	out := stripANSI(nm.viewLogin())
	if !strings.Contains(strings.ToLower(out), "login") {
		t.Fatalf("want login copy, got %q", out)
	}
	for _, banned := range []string{"Claim Code", "Register new Player", "Reclaim"} {
		if strings.Contains(out, banned) {
			t.Fatalf("claim-code path still offered (%q): %q", banned, out)
		}
	}
	assertLoginWallets(t, out)
}

func TestConfigCDoesNotOpenLogin(t *testing.T) {
	m := guestWithApp(t)
	next, _ := m.handleKey(tea.KeyPressMsg{Text: "c", Code: 'c'})
	nm := next.(Model)
	if nm.loginMode != loginIdle {
		t.Fatal("c should not open login")
	}
}

func TestConfigLDoesNotNudgeValue(t *testing.T) {
	m := guestWithApp(t)
	m.cfg.Mode = game.ModeTime
	m.cfg.Duration = game.TimeOptions[1]
	m.focus = focusValue
	next, _ := m.handleKey(tea.KeyPressMsg{Text: "l", Code: 'l'})
	nm := next.(Model)
	if nm.cfg.Duration != game.TimeOptions[1] {
		t.Fatal("l should login, not change duration")
	}
}

func TestGuestMetaSaysLogin(t *testing.T) {
	m := New()
	got := m.progMeta()
	if !strings.Contains(got, "login") {
		t.Fatalf("want login in guest meta, got %q", got)
	}
	if strings.Contains(got, "claim") {
		t.Fatalf("claim still in guest meta: %q", got)
	}
}

func TestViewConfigLoginKeyHint(t *testing.T) {
	m := New()
	m.finishIntro()
	plain := stripANSI(m.viewConfig())
	if !strings.Contains(plain, "guest") {
		t.Fatalf("want guest player row, got %q", plain)
	}
	// Hint column is 10-wide after the "player" label.
	if !strings.Contains(plain, "player") {
		t.Fatalf("want player row, got %q", plain)
	}
	if !strings.Contains(plain, "l") {
		t.Fatalf("want l shortcut, got %q", plain)
	}
}

func TestWalletLoginStartsLNURL(t *testing.T) {
	t.Setenv("GOTYPE_PUBLIC_URL", "https://gotype.fun")
	t.Setenv("REDIS_URL", "")
	app, err := OpenApp(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewWithOptions(Options{App: app, SessionID: "sess"})
	m.finishIntro()
	next, cmd := m.handleKey(tea.KeyPressMsg{Text: "l", Code: 'l'})
	nm := next.(Model)
	if cmd == nil {
		t.Fatal("expected LNURL start cmd")
	}
	msg := cmd()
	ws, ok := msg.(walletStartMsg)
	if !ok {
		t.Fatalf("msg type %T", msg)
	}
	if ws.err != "" || ws.lnurl == "" || ws.qr == "" {
		t.Fatalf("start %+v", ws)
	}
	_ = nm.applyWalletStart(ws)
	out := stripANSI(nm.viewLogin())
	if !strings.Contains(strings.ToLower(out), "login") {
		t.Fatalf("want login title, got %q", out)
	}
	if strings.Contains(out, "Claim Code") {
		t.Fatalf("claim code still on wallet screen: %q", out)
	}
	assertLoginWallets(t, out)
}

func TestProgHotkeyOpensLoginWhenGuest(t *testing.T) {
	m := guestWithApp(t)
	next, _ := m.handleKey(tea.KeyPressMsg{Text: "i", Code: 'i'})
	nm := next.(Model)
	if nm.loginMode == loginIdle {
		t.Fatal("inventory as guest should open login")
	}
	if nm.prog != progNone {
		t.Fatal("should not open inventory before login")
	}
	out := stripANSI(nm.viewLogin())
	if strings.Contains(out, "Claim Code") {
		t.Fatalf("claim-code path from progress hotkey: %q", out)
	}
}

func assertLoginWallets(t *testing.T, out string) {
	t.Helper()
	want := []string{"Wallet of Satoshi", "Phoenix", "Zeus", "Breez", "Alby", "Blink"}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("missing wallet %s in %q", w, out)
		}
	}
	if i, j := strings.Index(out, "Wallet of Satoshi"), strings.Index(out, "Phoenix"); i < 0 || i > j {
		t.Fatalf("Wallet of Satoshi should be listed first, got %q", out)
	}
}

func guestWithApp(t *testing.T) Model {
	t.Helper()
	t.Setenv("GOTYPE_PUBLIC_URL", "")
	t.Setenv("REDIS_URL", "")
	app, err := OpenApp(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewWithOptions(Options{App: app, SessionID: "sess"})
	m.finishIntro()
	return m
}
