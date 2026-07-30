package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/harmonica"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

const (
	raceBarWidth = 10

	barSpringFreq = 7.5
	barSpringDamp = 0.88
	barSettlePos  = 0.03
	barSettleVel  = 0.05

	shakeSpringFreq = 20.0
	shakeSpringDamp = 0.28
	shakeImpulse    = 2.4
	shakeSettlePos  = 0.05
	shakeSettleVel  = 0.08
	shakeMaxCells   = 3
)

var (
	barSpring   = harmonica.NewSpring(harmonica.FPS(30), barSpringFreq, barSpringDamp)
	shakeSpring = harmonica.NewSpring(harmonica.FPS(30), shakeSpringFreq, shakeSpringDamp)
)

type spring1D struct {
	x, v float64
}

func (s spring1D) settled(target, posEps, velEps float64) bool {
	return math.Abs(s.x-target) < posEps && math.Abs(s.v) < velEps
}

func (s *spring1D) stepToward(spring harmonica.Spring, target, posEps, velEps float64) {
	s.x, s.v = spring.Update(s.x, s.v, target)
	if s.settled(target, posEps, velEps) {
		s.x = target
		s.v = 0
	}
}

func (m *Model) resetMotion() {
	m.shake = spring1D{}
	m.barFill = nil
}

func (m *Model) triggerShake() {
	// Fresh impulse each mistype — under-damped spring wobbles back to 0.
	m.shake.x = shakeImpulse
	if m.shake.v < 0 {
		m.shake.v = 0
	}
	m.shake.v += 12
}

func (m *Model) stepShake() {
	m.shake.stepToward(shakeSpring, 0, shakeSettlePos, shakeSettleVel)
}

func (m Model) shakeAnimating() bool {
	return !m.shake.settled(0, shakeSettlePos, shakeSettleVel)
}

func (m *Model) stepRaceBars() {
	if m.roomCode == "" {
		return
	}
	v := m.multiView
	if v.Phase != multi.PhaseRacing && v.Phase != multi.PhaseCountdown {
		return
	}
	maxChars := 1
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		if p.Prog.Chars > maxChars {
			maxChars = p.Prog.Chars
		}
	}
	if m.barFill == nil {
		m.barFill = make(map[string]spring1D)
	}
	live := make(map[string]struct{}, len(v.Players))
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		live[p.ID] = struct{}{}
		target := float64(p.Prog.Chars) / float64(maxChars) * float64(raceBarWidth)
		st, ok := m.barFill[p.ID]
		if !ok {
			// First sight: snap so mid-race join doesn't crawl from empty.
			st = spring1D{x: target}
		} else {
			st.stepToward(barSpring, target, barSettlePos, barSettleVel)
		}
		m.barFill[p.ID] = st
	}
	for id := range m.barFill {
		if _, ok := live[id]; !ok {
			delete(m.barFill, id)
		}
	}
}

func (m Model) barsAnimating() bool {
	if m.roomCode == "" || m.barFill == nil {
		return false
	}
	v := m.multiView
	if v.Phase != multi.PhaseRacing && v.Phase != multi.PhaseCountdown {
		return false
	}
	maxChars := 1
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		if p.Prog.Chars > maxChars {
			maxChars = p.Prog.Chars
		}
	}
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		st, ok := m.barFill[p.ID]
		if !ok {
			continue
		}
		target := float64(p.Prog.Chars) / float64(maxChars) * float64(raceBarWidth)
		if !st.settled(target, barSettlePos, barSettleVel) {
			return true
		}
	}
	return false
}

func (m Model) raceBarFor(playerID string, chars, maxChars int) string {
	if maxChars < 1 {
		maxChars = 1
	}
	target := float64(chars) / float64(maxChars) * float64(raceBarWidth)
	fill := target
	if st, ok := m.barFill[playerID]; ok {
		fill = st.x
	}
	return progressBarFill(fill, raceBarWidth)
}

func progressBarFill(fill float64, width int) string {
	filled := int(math.Round(fill))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func progressBar(cur, max, width int) string {
	if max < 1 {
		max = 1
	}
	return progressBarFill(float64(cur)/float64(max)*float64(width), width)
}

// applyShake shifts each line by dx cells (positive = right). Used after Place.
func applyShake(s string, dx int) string {
	if dx == 0 {
		return s
	}
	if dx > shakeMaxCells {
		dx = shakeMaxCells
	} else if dx < -shakeMaxCells {
		dx = -shakeMaxCells
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if dx > 0 {
			lines[i] = strings.Repeat(" ", dx) + line
			continue
		}
		r := []rune(line)
		skip := -dx
		if skip > len(r) {
			skip = len(r)
		}
		lines[i] = string(r[skip:])
	}
	return strings.Join(lines, "\n")
}
