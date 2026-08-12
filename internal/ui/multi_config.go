package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/quoteai"
)

func (m Model) roomConfigEditable() bool {
	if !m.multiView.YouAreHost || m.multiView.YouAreSpectator {
		return false
	}
	switch m.multiView.Phase {
	case multi.PhaseLobby, multi.PhaseDone:
		return true
	default:
		return false
	}
}

func (m *Model) syncRoomConfigFromView() {
	m.cfg = m.multiView.Config
}

func (m *Model) pushRoomConfig() {
	v, err := m.hub.SetConfig(m.playerID, m.cfg)
	if err != nil {
		m.statusErr = err.Error()
		return
	}
	m.statusErr = ""
	m.multiView = v
	m.cfg = v.Config
}

func (m *Model) updateRoomConfig(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	if !m.roomConfigEditable() {
		return *m, nil, false
	}
	switch msg.String() {
	case "tab", "up", "k", "down", "j":
		m.toggleConfigFocus()
		return *m, nil, true
	case "left", "h":
		m.nudgeConfig(-1)
		m.pushRoomConfig()
		return *m, nil, true
	case "right", "l":
		m.nudgeConfig(1)
		m.pushRoomConfig()
		return *m, nil, true
	case "t":
		m.cfg.Mode = game.ModeTime
		m.focus = focusValue
		m.pushRoomConfig()
		return *m, nil, true
	case "w":
		m.cfg.Mode = game.ModeWords
		m.focus = focusValue
		m.pushRoomConfig()
		return *m, nil, true
	case "o":
		m.cfg.Mode = game.ModeQuotes
		m.focus = focusValue
		m.pushRoomConfig()
		return *m, nil, true
	case "a":
		if quoteai.Configured() {
			m.cfg.Mode = game.ModeAI
			m.focus = focusValue
			m.pushRoomConfig()
		}
		return *m, nil, true
	}
	return *m, nil, false
}

func (m Model) renderRoomConfigHost() string {
	var b strings.Builder
	modeHint := "t/w/o"
	if quoteai.Configured() {
		modeHint = "t/w/o/a"
	}
	if m.focus == focusMode {
		modeHint = "←/→"
	}
	b.WriteString(m.configFieldLabel("mode", modeHint, m.focus == focusMode))
	b.WriteString(m.renderModeOptions())
	b.WriteString("\n")
	valueHint := ""
	if m.focus == focusValue {
		valueHint = "←/→"
	}
	b.WriteString(m.configFieldLabel("value", valueHint, m.focus == focusValue))
	b.WriteString(m.renderValueOptions())
	return b.String()
}

func (m Model) renderRoomConfigGuest(v multi.View) string {
	line := fmt.Sprintf("next · %s %s", configDetail(v.Config, nil), v.Config.Mode)
	if v.Config.ThreeStrike {
		line += " · hardcore"
	}
	return m.sty.Sub.Render(line)
}

func (m Model) renderRoomConfigBlock(v multi.View) string {
	if v.Phase == multi.PhaseCountdown {
		return ""
	}
	var b strings.Builder
	if v.Phase == multi.PhaseGenerating {
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render(m.spin.View() + " generating ai quote…"))
		b.WriteString("\n")
	}
	if m.roomConfigEditable() {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.renderRoomConfigHost())
		b.WriteString("\n")
	} else if v.Phase == multi.PhaseLobby || v.Phase == multi.PhaseDone || v.Phase == multi.PhaseGenerating {
		b.WriteString(m.renderRoomConfigGuest(v))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) toggleRoomHardcore() {
	if !m.multiView.YouAreHost {
		m.statusErr = "only host toggles hardcore"
		return
	}
	v, err := m.hub.SetThreeStrike(m.playerID, !m.multiView.Config.ThreeStrike)
	if err != nil {
		m.statusErr = err.Error()
		return
	}
	m.statusErr = ""
	m.multiView = v
	m.cfg = v.Config
}
