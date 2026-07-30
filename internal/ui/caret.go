package ui

import (
	"math"
	"time"

	"charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

const (
	tickFast   = 33 * time.Millisecond  // ninja lerp / trail
	tickActive = 100 * time.Millisecond // typing, countdown, racing spectate
	tickIdle   = 250 * time.Millisecond // menus / results / idle lobby
	multiPoll  = 100 * time.Millisecond // hub sync cadence over SSH
	caretLerp  = 0.42
	trailMaxLife = 6
	blinkEvery   = 530 * time.Millisecond
)

func tickCmd() tea.Cmd {
	return tickAfter(tickIdle)
}

func tickAfter(d time.Duration) tea.Cmd {
	if d < tickFast {
		d = tickFast
	}
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) nextTickCmd() tea.Cmd {
	return tickAfter(m.tickInterval())
}

// tickInterval picks a redraw cadence that stays light over SSH.
func (m Model) tickInterval() time.Duration {
	switch m.phase {
	case phaseTyping:
		if m.ninjaCaret && m.caretAnimating() {
			return tickFast
		}
		return tickActive
	case phaseSpectate:
		if m.multiView.Phase == multi.PhaseRacing || m.multiView.Phase == multi.PhaseCountdown {
			return tickActive
		}
		return tickIdle
	case phaseLobby:
		if m.multiView.Phase == multi.PhaseCountdown {
			return tickActive
		}
		return tickIdle
	case phasePodium:
		// DEMO loops; keep a light poll
		if m.roomCode == multi.DemoCode {
			return tickActive
		}
		return tickIdle
	default:
		return tickIdle
	}
}

func (m Model) caretAnimating() bool {
	if m.sess == nil {
		return false
	}
	target := float64(m.sess.CursorPos())
	if math.Abs(m.caretX-target) >= 0.05 {
		return true
	}
	return len(m.trail) > 0
}

func (m *Model) resetCaret() {
	m.caretX = 0
	m.caretReady = false
	m.caretOn = true
	m.blinkTicks = 0
	m.lastBlink = time.Time{}
	m.trail = nil
}

// stepCaret lerps the visual caret toward the logical cursor and leaves a fading trail.
func (m *Model) stepCaret() {
	if m.sess == nil {
		return
	}
	target := float64(m.sess.CursorPos())
	if !m.ninjaCaret {
		m.caretX = target
		m.caretReady = true
		m.trail = nil
		m.stepBlink(true)
		return
	}
	if !m.caretReady {
		m.caretX = target
		m.caretReady = true
		return
	}

	prev := m.caretX
	diff := target - m.caretX
	if math.Abs(diff) < 0.05 {
		m.caretX = target
	} else {
		m.caretX += diff * caretLerp
	}

	// Paint trail along the path the caret swept this frame (Neovide/ninja ghost).
	lo := int(math.Floor(math.Min(prev, m.caretX)))
	hi := int(math.Ceil(math.Max(prev, m.caretX)))
	visual := int(math.Round(m.caretX))
	if m.trail == nil {
		m.trail = make(map[int]int)
	}
	for i := lo; i <= hi; i++ {
		if i == visual {
			continue
		}
		if life := m.trail[i]; life < trailMaxLife {
			m.trail[i] = trailMaxLife
		}
	}

	for pos, life := range m.trail {
		if life <= 1 {
			delete(m.trail, pos)
			continue
		}
		m.trail[pos] = life - 1
	}

	// Blink only while idle (cursor settled).
	idle := math.Abs(m.caretX-target) < 0.05
	m.stepBlink(idle)
}

func (m *Model) stepBlink(idle bool) {
	if !idle {
		m.caretOn = true
		m.lastBlink = m.now
		return
	}
	if m.lastBlink.IsZero() {
		m.lastBlink = m.now
		return
	}
	if m.now.Sub(m.lastBlink) >= blinkEvery {
		m.caretOn = !m.caretOn
		m.lastBlink = m.now
	}
}

func (m Model) caretVisualIndex() int {
	return int(math.Round(m.caretX))
}
