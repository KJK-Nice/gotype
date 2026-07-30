package ui

import (
	"math"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

const (
	tickFast     = 33 * time.Millisecond // ninja spring / trail (~30fps)
	tickActive   = 100 * time.Millisecond
	tickIdle     = 250 * time.Millisecond
	multiPoll    = 100 * time.Millisecond
	trailMaxLife = 6
	blinkEvery   = 530 * time.Millisecond

	// Spring tuned for char-index motion: snappy with a soft overshoot.
	caretSpringFreq = 9.0
	caretSpringDamp = 0.62
	caretSettlePos  = 0.04
	caretSettleVel  = 0.08
)

// caretSpring matches tickFast (~30fps). Reused; harmonica.Spring is a value type.
var caretSpring = harmonica.NewSpring(harmonica.FPS(30), caretSpringFreq, caretSpringDamp)

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
	if m.shakeAnimating() || m.barsAnimating() {
		return tickFast
	}
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
	if math.Abs(m.caretX-target) >= caretSettlePos || math.Abs(m.caretVel) >= caretSettleVel {
		return true
	}
	return len(m.trail) > 0
}

func (m *Model) resetCaret() {
	m.caretX = 0
	m.caretVel = 0
	m.caretReady = false
	m.caretOn = true
	m.blinkTicks = 0
	m.lastBlink = time.Time{}
	m.trail = nil
}

// stepCaret springs the visual caret toward the logical cursor and leaves a fading trail.
func (m *Model) stepCaret() {
	if m.sess == nil {
		return
	}
	target := float64(m.sess.CursorPos())
	if !m.ninjaCaret {
		m.caretX = target
		m.caretVel = 0
		m.caretReady = true
		m.trail = nil
		m.stepBlink(true)
		return
	}
	if !m.caretReady {
		m.caretX = target
		m.caretVel = 0
		m.caretReady = true
		return
	}

	prev := m.caretX
	m.caretX, m.caretVel = caretSpring.Update(m.caretX, m.caretVel, target)
	if math.Abs(m.caretX-target) < caretSettlePos && math.Abs(m.caretVel) < caretSettleVel {
		m.caretX = target
		m.caretVel = 0
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

	idle := math.Abs(m.caretX-target) < caretSettlePos && math.Abs(m.caretVel) < caretSettleVel
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
