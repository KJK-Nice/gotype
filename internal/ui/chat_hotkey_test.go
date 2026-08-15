package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestPodiumChatSDoesNotOpenShop(t *testing.T) {
	m := claimedPodium(t)
	m.enterChat()

	next, _ := m.handleKey(tea.KeyPressMsg{Text: "s", Code: 's'})
	nm := next.(Model)
	if nm.prog == progShop {
		t.Fatal("s in chat should not open shop")
	}
	if nm.claimMode != claimIdle {
		t.Fatal("s in chat should not open claim")
	}
	if !nm.chatMode {
		t.Fatal("should stay in chat")
	}
	if nm.chatTI.Value() != "s" {
		t.Fatalf("chat value=%q want s", nm.chatTI.Value())
	}
}

func TestPodiumChatProgHotkeysStayInChat(t *testing.T) {
	for _, key := range []string{"i", "s", "p", "e"} {
		m := claimedPodium(t)
		m.enterChat()
		next, _ := m.handleKey(tea.KeyPressMsg{Text: key, Code: []rune(key)[0]})
		nm := next.(Model)
		if nm.prog != progNone {
			t.Fatalf("%s in chat opened prog=%d", key, nm.prog)
		}
		if nm.chatTI.Value() != key {
			t.Fatalf("%s in chat: value=%q", key, nm.chatTI.Value())
		}
	}
}

func TestPodiumSOpensShopWhenNotChatting(t *testing.T) {
	m := claimedPodium(t)
	next, _ := m.handleKey(tea.KeyPressMsg{Text: "s", Code: 's'})
	nm := next.(Model)
	if nm.prog != progShop {
		t.Fatalf("prog=%d want shop", nm.prog)
	}
}

func TestPodiumSOpensShopAfterLeavingChat(t *testing.T) {
	m := claimedPodium(t)
	m.enterChat()
	next, _ := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	nm := next.(Model)
	if nm.chatMode {
		t.Fatal("esc should leave chat")
	}
	next, _ = nm.handleKey(tea.KeyPressMsg{Text: "s", Code: 's'})
	nm = next.(Model)
	if nm.prog != progShop {
		t.Fatalf("prog=%d want shop after leaving chat", nm.prog)
	}
}

func TestRaceStartClosesChat(t *testing.T) {
	m := New()
	m.finishIntro()
	m.enterChat()
	m.applyMultiView(multi.View{
		Phase:  multi.PhaseRacing,
		Config: game.DefaultConfig,
		Seed:   1,
	})
	if m.chatMode {
		t.Fatal("starting a race should close chat")
	}
	if m.phase != phaseTyping {
		t.Fatalf("phase=%d want typing", m.phase)
	}
}

func TestLobbyChatSDoesNotStartRace(t *testing.T) {
	h := multi.NewHub()
	v, err := h.Create("host", "host", game.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	m := NewWithOptions(Options{Hub: h, PlayerID: "host", PlayerName: "host"})
	m.finishIntro()
	m.phase = phaseLobby
	m.roomCode = v.Code
	m.multiView = v
	m.enterChat()

	next, _ := m.handleKey(tea.KeyPressMsg{Text: "s", Code: 's'})
	nm := next.(Model)
	if nm.multiView.Phase != multi.PhaseLobby {
		t.Fatalf("phase=%s want lobby", nm.multiView.Phase)
	}
	if nm.chatTI.Value() != "s" {
		t.Fatalf("chat value=%q want s", nm.chatTI.Value())
	}
}

func claimedPodium(t *testing.T) Model {
	t.Helper()
	app, err := OpenApp(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	reg, err := app.Players.Register("Typist", "1.1.1.1", "sess", now)
	if err != nil {
		t.Fatal(err)
	}
	m := NewWithOptions(Options{App: app, SessionID: "sess", PlayerID: reg.Player.ID, PlayerName: reg.Player.Name})
	m.finishIntro()
	m.claimedID = reg.Player.ID
	m.now = now
	m.phase = phasePodium
	m.multiView = multi.View{Code: "ABCD", Phase: multi.PhaseDone, RaceNumber: 1}
	return m
}

func (m *Model) enterChat() {
	m.chatMode = true
	m.chatTI.SetValue("")
	m.chatTI.Focus()
}
