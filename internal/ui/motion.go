package ui

import (
	"math"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

const (
	raceBarWidth = 14

	shakeSpringFreq = 20.0
	shakeSpringDamp = 0.28
	shakeImpulse    = 2.4
	shakeSettlePos  = 0.05
	shakeSettleVel  = 0.08
	shakeMaxCells   = 3
)

var shakeSpring = harmonica.NewSpring(harmonica.FPS(30), shakeSpringFreq, shakeSpringDamp)

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
	m.raceBars = nil
}

func (m *Model) triggerShake() {
	if m.calmArmed {
		m.calmArmed = false
		return
	}
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

func newRaceProgressThemed(you bool, themeIdx int) progress.Model {
	th := themes[themeIdx%len(themes)]
	opts := []progress.Option{
		progress.WithWidth(raceBarWidth),
		progress.WithoutPercentage(),
		progress.WithSpringOptions(7.5, 0.85),
		progress.WithFillCharacters('█', '░'),
	}
	if you {
		opts = append(opts, progress.WithColors(lipgloss.Color(th.Main), lipgloss.Color(th.Text)))
	} else {
		opts = append(opts, progress.WithColors(lipgloss.Color(th.Sub)))
	}
	return progress.New(opts...)
}

// syncRaceBars drives bubbles progress.SetPercent toward each racer's share of the lead.
func (m *Model) syncRaceBars() {
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
	if m.raceBars == nil {
		m.raceBars = make(map[string]progress.Model)
	}
	live := make(map[string]struct{}, len(v.Players))
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		live[p.ID] = struct{}{}
		target := float64(p.Prog.Chars) / float64(maxChars)
		bar, ok := m.raceBars[p.ID]
		if !ok {
			bar = newRaceProgressThemed(p.You, m.themeIdx)
		}
		if ok && math.Abs(bar.Percent()-target) < 0.001 {
			m.raceBars[p.ID] = bar
			continue
		}
		cmd := bar.SetPercent(target)
		m.raceBars[p.ID] = bar
		m.queueCmd(cmd)
	}
	for id := range m.raceBars {
		if _, ok := live[id]; !ok {
			delete(m.raceBars, id)
		}
	}
}

func (m Model) barsAnimating() bool {
	if m.roomCode == "" || m.raceBars == nil {
		return false
	}
	for _, bar := range m.raceBars {
		if bar.IsAnimating() {
			return true
		}
	}
	return false
}

func (m Model) raceBarView(playerID string) string {
	if bar, ok := m.raceBars[playerID]; ok {
		return bar.View()
	}
	return progressBarFill(0, raceBarWidth)
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
