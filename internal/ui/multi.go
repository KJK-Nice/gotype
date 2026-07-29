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
		m.multiView = v
		m.raceStarted = false
		m.sess = nil
		if v.Phase == multi.PhaseDone {
			m.phase = phasePodium
		} else {
			m.phase = phaseLobby
		}
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
	if m.chatMode {
		return m.updateChat(msg)
	}
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
	case "g":
		return m.sendChat("gg")
	case "/":
		m.chatMode = true
		m.chatInput = ""
		m.statusErr = ""
	default:
		if m.multiView.Phase == multi.PhaseCountdown {
			m.statusErr = "wait for countdown"
		}
	}
	return m, nil
}

func (m Model) updatePodium(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatMode {
		return m.updateChat(msg)
	}
	switch msg.String() {
	case "q":
		m.leaveMulti()
		return m, tea.Quit
	case "esc":
		m.leaveMulti()
		m.phase = phaseMultiMenu
		m.sess = nil
	case "tab", "enter", "r":
		v, err := m.hub.Rematch(m.playerID, m.now)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.statusErr = ""
		m.sess = nil
		m.raceStarted = false
		m.resetCaret()
		m.multiView = v
		m.phase = phaseLobby
	case "g":
		return m.sendChat("gg")
	case "/":
		m.chatMode = true
		m.chatInput = ""
		m.statusErr = ""
	}
	return m, nil
}

func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.chatMode = false
		m.chatInput = ""
	case "enter":
		text := m.chatInput
		m.chatMode = false
		m.chatInput = ""
		if text == "" {
			return m, nil
		}
		return m.sendChat(text)
	case "backspace":
		if len(m.chatInput) > 0 {
			r := []rune(m.chatInput)
			m.chatInput = string(r[:len(r)-1])
		}
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if len([]rune(m.chatInput)) >= 48 {
					break
				}
				m.chatInput += string(r)
			}
		} else if s := msg.String(); len(s) == 1 {
			if len([]rune(m.chatInput)) < 48 {
				m.chatInput += s
			}
		}
	}
	return m, nil
}

func (m Model) sendChat(text string) (tea.Model, tea.Cmd) {
	v, err := m.hub.Say(m.playerID, text)
	if err != nil {
		m.statusErr = err.Error()
		return m, nil
	}
	m.multiView = v
	m.statusErr = ""
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
	m.chatMode = false
	m.chatInput = ""
}

func (m *Model) syncMulti() {
	if m.hub == nil || m.roomCode == "" {
		return
	}
	var v multi.View
	if m.phase == phaseTyping && m.sess != nil {
		snap := m.sess.Snapshot(m.now)
		chars := m.sess.ProgressChars()
		v = m.hub.Report(m.playerID, multi.Progress{
			WPM:      snap.WPM,
			Accuracy: snap.Accuracy,
			Correct:  snap.Correct,
			Chars:    chars,
			Done:     m.sess.Finished,
		}, m.now)
		if m.sess.Started {
			elapsed := m.now.Sub(m.sess.StartedAt)
			m.ghostRec = append(m.ghostRec, GhostSample{At: elapsed, Chars: chars})
		}
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
		if m.phase == phasePodium || m.raceStarted {
			m.raceStarted = false
			m.sess = nil
			m.resetCaret()
			m.statusErr = ""
			m.phase = phaseLobby
			break
		}
		if m.phase == phaseTyping {
			break
		}
		m.phase = phaseLobby
	case multi.PhaseRacing:
		if !m.raceStarted {
			m.cfg = v.Config
			m.sess = game.NewSessionSeeded(v.Config, v.Seed)
			start := v.RaceStarted
			if start.IsZero() {
				start = m.now
			}
			m.sess.Started = true
			m.sess.StartedAt = start
			m.sess.Stats.Start(start)
			m.sess.NoAutoFinish = true
			m.raceStarted = true
			m.ghostRec = nil
			m.resetCaret()
			m.phase = phaseTyping
		}
	case multi.PhaseDone:
		if m.sess != nil && !m.sess.Finished {
			m.sess.ForceFinish(m.now)
		}
		if len(m.ghostRec) > 2 {
			m.paceGhost = m.ghostRec
		}
		m.ghostRec = nil
		m.phase = phasePodium
	}
}

func (m Model) viewMultiMenu() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("multiplayer"))
	b.WriteString("\n")
	b.WriteString(styleSub.Render("race friends · best of 3"))
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

func (m Model) viewChat() string {
	v := m.multiView
	var b strings.Builder
	if len(v.Chat) > 0 {
		for _, line := range v.Chat {
			b.WriteString(styleSub.Render(line.Name + ": "))
			b.WriteString(styleText.Render(line.Text))
			b.WriteString("\n")
		}
	}
	if m.chatMode {
		b.WriteString(styleMain.Render("> " + m.chatInput + "█"))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewLobby() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(styleTitle.Render("lobby"))
	b.WriteString("  ")
	b.WriteString(styleMain.Render(v.Code))
	b.WriteString("  ")
	b.WriteString(styleSub.Render("bo3"))
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
		crown := "  "
		if p.Crown {
			crown = "👑"
		}
		score := fmt.Sprintf(" %d", p.MatchWins)
		name := fmt.Sprintf("%s %-10s", crown, truncateName(p.Name, 10))
		if p.You {
			b.WriteString(styleMain.Render(name))
		} else {
			b.WriteString(styleText.Render(name))
		}
		b.WriteString(styleStatValue.Render(score))
		if p.IsHost {
			b.WriteString(styleSub.Render(" host"))
		}
		if p.You {
			b.WriteString(styleSub.Render(" (you)"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.viewChat())
	if m.statusErr != "" {
		b.WriteString(styleIncorrect.Render(m.statusErr))
		b.WriteString("\n")
	}
	if m.chatMode {
		b.WriteString(styleSub.Render("enter send  esc cancel"))
	} else if v.YouAreHost && v.Phase == multi.PhaseLobby {
		b.WriteString(styleSub.Render("s start  g gg  / chat  esc leave"))
	} else if v.Phase == multi.PhaseCountdown {
		b.WriteString(styleSub.Render("get ready…"))
	} else {
		b.WriteString(styleSub.Render("waiting  g gg  / chat  esc leave"))
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
		crown := ""
		if p.Crown {
			crown = "👑"
		}
		name := fmt.Sprintf("%s%-8s", crown, truncateName(p.Name, 8))
		if p.You {
			b.WriteString(styleMain.Render(fmt.Sprintf("%-10s", name)))
		} else {
			b.WriteString(styleSub.Render(fmt.Sprintf("%-10s", name)))
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
		b.WriteString(styleSub.Render(fmt.Sprintf(" %d", p.MatchWins)))
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
	b.WriteString("\n")
	if v.MatchOver {
		b.WriteString(styleMain.Render("match · " + v.MatchWinnerName + " wins bo3"))
		b.WriteString("\n")
	} else {
		b.WriteString(styleSub.Render("best of 3"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	for _, p := range v.Players {
		medal := fmt.Sprintf("%d.", p.Rank)
		acc := fmt.Sprintf("%3.0f%%", p.Prog.Accuracy)
		if p.Prog.Chars == 0 && p.Prog.Correct == 0 {
			acc = "  —"
		}
		crown := " "
		if p.Crown {
			crown = "👑"
		}
		line := fmt.Sprintf("%-3s%s %-9s %5.0f wpm %s  %d/%d",
			medal, crown, truncateName(p.Name, 9), p.Prog.WPM, acc, p.MatchWins, multi.WinsToTakeMatch)
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
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.viewChat())
	b.WriteString(styleSub.Render("share "))
	b.WriteString(styleMain.Render(v.Code))
	b.WriteString(styleSub.Render(" — friends join anytime"))
	b.WriteString("\n")
	if m.chatMode {
		b.WriteString(styleSub.Render("enter send  esc cancel"))
	} else if v.MatchOver {
		b.WriteString(styleSub.Render("enter new series  g gg  / chat  esc leave"))
	} else {
		b.WriteString(styleSub.Render("enter next race  g gg  / chat  esc leave"))
	}
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
