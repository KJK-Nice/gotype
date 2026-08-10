package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) beginIntro() {
	m.phase = phaseIntro
	m.introAt = m.now
	if m.introAt.IsZero() {
		m.introAt = time.Now()
		m.now = m.introAt
	}
	m.introSeed = m.introAt.UnixNano()
	m.introRain = newIntroRain(m.width, m.height, m.introSeed)
	m.introLast = m.introAt
}

func (m *Model) finishIntro() {
	m.phase = phaseConfig
	m.introRain = nil
	m.introAt = time.Time{}
	m.introLast = time.Time{}
}

func (m *Model) rebuildIntroRain() {
	if m.phase != phaseIntro {
		return
	}
	if m.introRain == nil {
		m.introRain = newIntroRain(m.width, m.height, m.introSeed)
		return
	}
	m.introRain.rebuild(m.width, m.height)
}

func (m *Model) stepIntro() {
	if m.phase != phaseIntro || m.introRain == nil {
		return
	}
	dt := m.now.Sub(m.introLast)
	if dt < 0 {
		dt = 0
	}
	if dt > 100*time.Millisecond {
		dt = 100 * time.Millisecond
	}
	m.introLast = m.now
	m.introRain.step(dt)

	stage, _ := introProgress(m.now.Sub(m.introAt))
	if stage == introStageDone {
		m.finishIntro()
	}
}

func (m Model) introElapsed() time.Duration {
	if m.introAt.IsZero() {
		return 0
	}
	return m.now.Sub(m.introAt)
}

func (m Model) introSkippable() bool {
	return m.introElapsed() >= introFloor
}

func (m Model) updateIntro(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+d":
		return m, tea.Quit
	default:
		if m.introSkippable() {
			m.finishIntro()
		}
	}
	return m, nil
}

func (m Model) viewIntro() string {
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}

	// Full home plate under rain — wipe reveals it top→bottom.
	home := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, m.sty.Box.Render(m.viewConfig()))
	if m.introRain == nil {
		return home
	}

	stage, prog := introProgress(m.introElapsed())
	cy := clearY(stage, prog, h)
	return stampRainOver(home, m.introRain, cy, w, h)
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		if c == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
