package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func TestRoomConfigEditable(t *testing.T) {
	m := New()
	m.multiView = multi.View{
		YouAreHost: true,
		Phase:      multi.PhaseDone,
		Config:     game.DefaultConfig,
	}
	if !m.roomConfigEditable() {
		t.Fatal("host on podium should edit config")
	}
	m.multiView.YouAreHost = false
	if m.roomConfigEditable() {
		t.Fatal("guest should not edit")
	}
	m.multiView.YouAreHost = true
	m.multiView.Phase = multi.PhaseGenerating
	if m.roomConfigEditable() {
		t.Fatal("generating should lock config")
	}
}

func TestUpdateRoomConfigNudge(t *testing.T) {
	h := multi.NewHub()
	m := New()
	m.hub = h
	m.playerID = "host"
	m.playerName = "host"
	v, err := h.Create("host", "host", game.Config{Mode: game.ModeTime, Duration: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	m.roomCode = v.Code
	m.multiView = v
	m.cfg = v.Config
	m.focus = focusMode

	nm, _, ok := m.updateRoomConfig(tea.KeyPressMsg{Code: tea.KeyRight})
	if !ok {
		t.Fatal("expected config key handled")
	}
	if nm.cfg.Mode != game.ModeWords {
		t.Fatalf("mode=%s", nm.cfg.Mode)
	}
	snap := h.Snapshot("host", time.Now())
	if snap.Config.Mode != game.ModeWords {
		t.Fatalf("hub mode=%s", snap.Config.Mode)
	}
}

func TestRenderRoomConfigGuest(t *testing.T) {
	m := New()
	v := multi.View{Config: game.Config{Mode: game.ModeTime, Duration: 60 * time.Second}}
	line := m.renderRoomConfigGuest(v)
	if line == "" {
		t.Fatal("expected guest line")
	}
}
