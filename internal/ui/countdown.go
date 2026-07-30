package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/harmonica"

	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

// FIGlet-ish block digits for cinematic race countdown.
var (
	asciiDigit = map[int][]string{
		3: {
			` ██████╗ `,
			` ╚════██╗`,
			`  █████╔╝`,
			`  ╚═══██╗`,
			` ██████╔╝`,
			` ╚═════╝ `,
		},
		2: {
			` ██████╗ `,
			` ╚════██╗`,
			`  █████╔╝`,
			` ██╔═══╝ `,
			` ███████╗`,
			` ╚══════╝`,
		},
		1: {
			`  ██╗`,
			` ███║`,
			` ╚██║`,
			`  ██║`,
			`  ██║`,
			`  ╚═╝`,
		},
	}
	asciiGO = []string{
		`  ██████╗  ██████╗ ██╗`,
		` ██╔════╝ ██╔═══██╗██║`,
		` ██║  ███╗██║   ██║██║`,
		` ██║   ██║██║   ██║╚═╝`,
		` ╚██████╔╝╚██████╔╝██╗`,
		`  ╚═════╝  ╚═════╝ ╚═╝`,
	}
)

var cdPulseSpring = harmonica.NewSpring(harmonica.FPS(30), 14.0, 0.35)

func (m Model) countdownSecondsLeft() int {
	var left time.Duration
	if m.cdOn {
		left = m.cdTimer.Timeout
	} else {
		left = m.multiView.CountdownLeft
	}
	sec := int(left.Seconds() + 0.999)
	if sec < 0 {
		return 0
	}
	if sec > multi.CountdownSecs {
		return multi.CountdownSecs
	}
	return sec
}

func (m Model) countdownDigit() int {
	sec := m.countdownSecondsLeft()
	if sec <= 0 {
		return 0 // GO
	}
	return sec
}

func (m *Model) stepCountdownCinematic() {
	if m.multiView.Phase != multi.PhaseCountdown {
		m.cdDigit = -1
		m.cdPulse = spring1D{}
		return
	}
	dig := m.countdownDigit()
	if dig != m.cdDigit {
		m.cdDigit = dig
		// Pop hard on each beat (3→2→1→GO).
		m.cdPulse.x = 1.2
		m.cdPulse.v = 18
	}
	m.cdPulse.stepToward(cdPulseSpring, 0, 0.02, 0.05)
}

func (m Model) viewCinematicCountdown(banner string) string {
	dig := m.countdownDigit()
	var art []string
	label := ""
	switch {
	case dig <= 0:
		art = asciiGO
		label = "type!"
	default:
		art = asciiDigit[dig]
		if art == nil {
			art = asciiDigit[3]
		}
	}

	pulse := m.cdPulse.x
	if pulse < 0 {
		pulse = -pulse
	}
	// Side flare grows with pulse — cinematic impact lines.
	flareN := int(math.Round(pulse * 4))
	if flareN > 6 {
		flareN = 6
	}
	flare := strings.Repeat("═", flareN)

	var b strings.Builder
	if banner != "" {
		b.WriteString(m.sty.Sub.Render(banner))
		b.WriteString("\n\n")
	}

	topRule := "══════"
	if flareN > 0 {
		topRule = strings.Repeat("═", 6+flareN*2)
	}
	b.WriteString(m.sty.Sub.Render(topRule))
	b.WriteString("\n")

	digitStyle := m.sty.Main
	if pulse > 0.35 {
		digitStyle = m.sty.Main.Bold(true)
	}
	if dig <= 0 {
		digitStyle = m.sty.Correct.Bold(true)
	}

	for _, line := range art {
		row := line
		if flareN > 0 {
			row = flare + " " + line + " " + flare
		}
		b.WriteString(digitStyle.Render(row))
		b.WriteString("\n")
	}

	b.WriteString(m.sty.Sub.Render(topRule))
	b.WriteString("\n")
	if label != "" {
		b.WriteString(m.sty.Correct.Bold(true).Render(label))
		b.WriteString("\n")
	} else {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("race starts in %d", dig)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}
