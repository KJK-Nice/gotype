package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

func (m Model) multiEnabled() bool {
	return m.hub != nil
}

func (m Model) updateMultiMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.phase = phaseConfig
	case "c", "enter":
		v, err := m.hub.Create(m.playerID, m.playerName, m.cfg)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.roomCode = v.Code
		m.statusErr = ""
		m.phase = phaseLobby
		m.multiView = v
	case "j":
		m.joinInput = ""
		m.statusErr = ""
		m.phase = phaseJoin
	}
	return m, nil
}

func (m Model) updateJoin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = phaseMultiMenu
		m.joinInput = ""
		m.statusErr = ""
	case "enter":
		if len(m.joinInput) < 4 {
			m.statusErr = "need 4-letter code"
			return m, nil
		}
		v, err := m.hub.Join(m.playerID, m.playerName, m.joinInput)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.roomCode = v.Code
		m.statusErr = ""
		m.phase = phaseLobby
		m.multiView = v
	case "backspace":
		if len(m.joinInput) > 0 {
			m.joinInput = m.joinInput[:len(m.joinInput)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if len(m.joinInput) >= 4 {
					break
				}
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					if r >= 'a' {
						r -= 'a' - 'A'
					}
					m.joinInput += string(r)
				}
			}
		} else if s := msg.String(); len(s) == 1 {
			r := rune(s[0])
			if len(m.joinInput) < 4 && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				if r >= 'a' {
					r -= 'a' - 'A'
				}
				m.joinInput += string(r)
			}
		}
	}
	return m, nil
}

func (m Model) updateLobby(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If hub already racing, enter race on any key and apply this keystroke.
	if m.hub != nil && m.roomCode != "" {
		m.syncMulti()
		if m.phase == phaseTyping {
			return m.updateTyping(msg)
		}
		if m.phase == phasePodium {
			return m.updatePodium(msg)
		}
	}
	switch msg.String() {
	case "q":
		m.leaveMulti()
		return m, tea.Quit
	case "esc":
		m.leaveMulti()
		m.phase = phaseMultiMenu
	case "s", "enter":
		if !m.multiView.YouAreHost {
			m.statusErr = "wait for host to start"
			return m, nil
		}
		v, err := m.hub.Start(m.playerID, m.now)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.statusErr = ""
		m.multiView = v
		m.applyMultiView(v)
	default:
		if m.multiView.Phase == multi.PhaseCountdown {
			m.statusErr = "wait for countdown"
		}
	}
	return m, nil
}

func (m Model) updatePodium(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.leaveMulti()
		return m, tea.Quit
	case "esc", "enter", "tab":
		m.leaveMulti()
		m.phase = phaseMultiMenu
		m.sess = nil
	}
	return m, nil
}

func (m *Model) leaveMulti() {
	if m.hub != nil && m.playerID != "" {
		m.hub.Leave(m.playerID)
	}
	m.roomCode = ""
	m.multiView = multi.View{}
	m.raceStarted = false
	m.statusErr = ""
	m.joinInput = ""
}

func (m *Model) syncMulti() {
	if m.hub == nil || m.roomCode == "" {
		return
	}
	var v multi.View
	if m.phase == phaseTyping && m.sess != nil {
		snap := m.sess.Snapshot(m.now)
		v = m.hub.Report(m.playerID, multi.Progress{
			WPM:      snap.WPM,
			Accuracy: snap.Accuracy,
			Correct:  snap.Correct,
			Chars:    m.sess.ProgressChars(),
			Done:     m.sess.Finished,
		}, m.now)
	} else {
		v = m.hub.Snapshot(m.playerID, m.now)
	}
	if v.Err != "" && v.Code == "" {
		m.statusErr = v.Err
		m.roomCode = ""
		m.phase = phaseMultiMenu
		return
	}
	m.multiView = v
	m.applyMultiView(v)
}

func (m *Model) applyMultiView(v multi.View) {
	switch v.Phase {
	case multi.PhaseLobby, multi.PhaseCountdown:
		if m.phase == phaseTyping || m.phase == phasePodium || m.phase == phaseResult {
			break
		}
		m.phase = phaseLobby
	case multi.PhaseRacing:
		if !m.raceStarted {
			m.cfg = v.Config
			m.sess = game.NewSessionSeeded(v.Config, v.Seed)
			// Shared race clock — all players start timer together.
			start := v.RaceStarted
			if start.IsZero() {
				start = m.now
			}
			m.sess.Started = true
			m.sess.StartedAt = start
			m.sess.Stats.Start(start)
			m.sess.NoAutoFinish = true // hub PhaseDone ends race; keep input alive
			m.raceStarted = true
			m.resetCaret()
			m.phase = phaseTyping
		}
	case multi.PhaseDone:
		if m.sess != nil && !m.sess.Finished {
			m.sess.ForceFinish(m.now)
		}
		m.phase = phasePodium
	}
}

func (m Model) viewMultiMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("multiplayer"))
	b.WriteString("\n")
	b.WriteString(styleSub.Render("race friends over SSH"))
	b.WriteString("\n\n")
	b.WriteString(styleSelected.Render("c"))
	b.WriteString(styleText.Render(" create room"))
	b.WriteString("\n")
	b.WriteString(styleSelected.Render("j"))
	b.WriteString(styleText.Render(" join room"))
	b.WriteString("\n\n")
	if m.statusErr != "" {
		b.WriteString(styleIncorrect.Render(m.statusErr))
		b.WriteString("\n\n")
	}
	b.WriteString(styleSub.Render("esc back  q quit"))
	return b.String()
}

func (m Model) viewJoin() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("join room"))
	b.WriteString("\n\n")
	b.WriteString(styleSub.Render("code  "))
	pad := m.joinInput + strings.Repeat("_", 4-len(m.joinInput))
	b.WriteString(styleMain.Render(pad))
	b.WriteString("\n\n")
	if m.statusErr != "" {
		b.WriteString(styleIncorrect.Render(m.statusErr))
		b.WriteString("\n\n")
	}
	b.WriteString(styleSub.Render("enter join  esc back"))
	return b.String()
}

func (m Model) viewLobby() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(styleTitle.Render("lobby"))
	b.WriteString("  ")
	b.WriteString(styleMain.Render(v.Code))
	b.WriteString("\n")
	detail := fmt.Sprintf("%s · %s", v.Config.Mode.String(), game.FormatSeconds(v.Config.Duration)+"s")
	if v.Config.Mode == game.ModeWords {
		detail = fmt.Sprintf("%s · %d words", v.Config.Mode.String(), v.Config.WordCount)
	}
	b.WriteString(styleSub.Render(detail))
	b.WriteString("\n\n")

	if v.Phase == multi.PhaseCountdown {
		sec := int(v.CountdownLeft.Seconds() + 0.999)
		if sec < 0 {
			sec = 0
		}
		b.WriteString(styleMain.Render(fmt.Sprintf("starting in %d…", sec)))
		b.WriteString("\n\n")
	}

	for _, p := range v.Players {
		line := fmt.Sprintf("%-12s", p.Name)
		if p.You {
			line = styleMain.Render(line)
		} else {
			line = styleText.Render(line)
		}
		b.WriteString(line)
		if p.IsHost {
			b.WriteString(styleSub.Render(" host"))
		}
		if p.You {
			b.WriteString(styleSub.Render(" (you)"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.statusErr != "" {
		b.WriteString(styleIncorrect.Render(m.statusErr))
		b.WriteString("\n\n")
	}
	if v.YouAreHost && v.Phase == multi.PhaseLobby {
		b.WriteString(styleSub.Render("s/enter start  esc leave"))
	} else if v.Phase == multi.PhaseCountdown {
		b.WriteString(styleSub.Render("get ready…"))
	} else {
		b.WriteString(styleSub.Render("waiting for host  esc leave"))
	}
	return b.String()
}

func (m Model) viewRaceOpponents() string {
	v := m.multiView
	if len(v.Players) == 0 {
		return ""
	}
	maxChars := 1
	for _, p := range v.Players {
		if p.Prog.Chars > maxChars {
			maxChars = p.Prog.Chars
		}
	}
	var b strings.Builder
	b.WriteString("\n")
	for _, p := range v.Players {
		name := fmt.Sprintf("%-10s", truncateName(p.Name, 10))
		if p.You {
			b.WriteString(styleMain.Render(name))
		} else {
			b.WriteString(styleSub.Render(name))
		}
		b.WriteString(" ")
		b.WriteString(styleStatValue.Render(fmt.Sprintf("%3.0f", p.Prog.WPM)))
		b.WriteString(styleSub.Render(" wpm "))
		bar := progressBar(p.Prog.Chars, maxChars, 10)
		if p.You {
			b.WriteString(styleMain.Render(bar))
		} else {
			b.WriteString(styleSub.Render(bar))
		}
		if p.Prog.Done {
			b.WriteString(styleSub.Render(" done"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewPodium() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(styleTitle.Render("podium"))
	b.WriteString("  ")
	b.WriteString(styleSub.Render(v.Code))
	b.WriteString("\n\n")
	for _, p := range v.Players {
		medal := fmt.Sprintf("%d.", p.Rank)
		acc := fmt.Sprintf("%3.0f%%", p.Prog.Accuracy)
		if p.Prog.Chars == 0 && p.Prog.Correct == 0 {
			acc = "  —"
		}
		line := fmt.Sprintf("%-3s %-10s  %5.0f wpm  %s", medal, truncateName(p.Name, 10), p.Prog.WPM, acc)
		if p.You {
			b.WriteString(styleMain.Render(line))
		} else {
			b.WriteString(styleText.Render(line))
		}
		b.WriteString("\n")
	}
	if m.sess != nil {
		snap := m.sess.Snapshot(m.now)
		b.WriteString("\n")
		if snap.Correct+snap.Incorrect+snap.Extra == 0 {
			b.WriteString(styleSub.Render("you · 0 wpm · — acc (no input)"))
		} else {
			b.WriteString(styleSub.Render(fmt.Sprintf("you · %.0f wpm · %.0f%% acc", snap.WPM, snap.Accuracy)))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(styleSub.Render("enter/esc lobby  q quit"))
	return b.String()
}

func truncateName(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func progressBar(cur, max, width int) string {
	if max < 1 {
		max = 1
	}
	filled := cur * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
