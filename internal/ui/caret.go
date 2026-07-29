package ui

import (
	"math"
	"time"

	"github.com/charmbracelet/bubbletea"
)

const (
	tickInterval = 33 * time.Millisecond
	caretLerp    = 0.42
	trailMaxLife = 6
)

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) resetCaret() {
	m.caretX = 0
	m.caretReady = false
	m.caretOn = true
	m.blinkTicks = 0
	m.trail = nil
}

// stepCaret lerps the visual caret toward the logical cursor and leaves a fading trail.
func (m *Model) stepCaret() {
	if m.sess == nil {
		return
	}
	target := float64(m.sess.CursorPos())
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
	if idle {
		m.blinkTicks++
		// ~530ms at 33ms ticks.
		if m.blinkTicks >= 16 {
			m.blinkTicks = 0
			m.caretOn = !m.caretOn
		}
	} else {
		m.caretOn = true
		m.blinkTicks = 0
	}
}

func (m Model) caretVisualIndex() int {
	return int(math.Round(m.caretX))
}
