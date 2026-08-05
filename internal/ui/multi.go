package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/invite"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

func (m Model) multiEnabled() bool {
	return m.hub != nil
}

func (m Model) updateMultiMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.phase = phaseConfig
		return m, nil
	case "c":
		m.multiMenuList.Select(0)
		return m.execMultiAction("create")
	case "j":
		m.multiMenuList.Select(1)
		return m.execMultiAction("join")
	case "d":
		m.multiMenuList.Select(2)
		return m.execMultiAction("demo")
	case "enter":
		return m.execMultiAction(selectedAction(m.multiMenuList))
	}
	var cmd tea.Cmd
	m.multiMenuList, cmd = m.multiMenuList.Update(msg)
	return m, cmd
}

func (m Model) execMultiAction(action string) (Model, tea.Cmd) {
	switch action {
	case "create":
		v, err := m.hub.Create(m.playerID, m.playerName, m.cfg)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.roomCode = v.Code
		m.statusErr = ""
		m.phase = phaseLobby
		m.multiView = v
	case "join":
		m.joinTI.SetValue("")
		m.statusErr = ""
		m.phase = phaseJoin
		return m, m.joinTI.Focus()
	case "demo":
		v, err := m.hub.SpectateLive(m.playerID, m.playerName, m.now)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.roomCode = v.Code
		m.statusErr = ""
		m.multiView = v
		m.raceStarted = false
		m.sess = nil
		m.applyMultiView(v)
	}
	return m, nil
}

func (m Model) updateJoin(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = phaseMultiMenu
		m.joinTI.Blur()
		m.joinTI.SetValue("")
		m.statusErr = ""
		return m, nil
	case "enter":
		code := strings.ToUpper(strings.TrimSpace(m.joinTI.Value()))
		if len(code) < 4 {
			m.statusErr = "need 4-letter code"
			return m, nil
		}
		v, err := m.hub.Join(m.playerID, m.playerName, code)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.roomCode = v.Code
		m.statusErr = ""
		m.multiView = v
		m.raceStarted = false
		m.sess = nil
		m.joinTI.Blur()
		m.applyMultiView(v)
		return m, nil
	}
	var cmd tea.Cmd
	m.joinTI, cmd = m.joinTI.Update(msg)
	// Force uppercase letters for room codes.
	v := strings.ToUpper(m.joinTI.Value())
	if v != m.joinTI.Value() {
		m.joinTI.SetValue(v)
	}
	return m, cmd
}

func (m Model) updateSpectate(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "g":
		return m.sendChat("gg")
	case "/":
		m.chatMode = true
		m.chatTI.SetValue("")
		m.statusErr = ""
		return m, m.chatTI.Focus()
	}
	return m, nil
}

func (m Model) updateLobby(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "h":
		if m.multiView.Phase != multi.PhaseLobby {
			m.statusErr = "hardcore locked"
			return m, nil
		}
		if !m.multiView.YouAreHost {
			m.statusErr = "only host toggles hardcore"
			return m, nil
		}
		v, err := m.hub.SetThreeStrike(m.playerID, !m.multiView.Config.ThreeStrike)
		if err != nil {
			m.statusErr = err.Error()
			return m, nil
		}
		m.statusErr = ""
		m.multiView = v
		m.cfg = v.Config
	case "g":
		return m.sendChat("gg")
	case "/":
		m.chatMode = true
		m.chatTI.SetValue("")
		m.statusErr = ""
		return m, m.chatTI.Focus()
	default:
		if m.multiView.Phase == multi.PhaseCountdown {
			m.statusErr = "wait for countdown"
		}
	}
	return m, nil
}

func (m Model) updatePodium(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		if m.multiView.YouAreSpectator {
			return m, nil
		}
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
		m.chatTI.SetValue("")
		m.statusErr = ""
		return m, m.chatTI.Focus()
	}
	return m, nil
}

func (m Model) updateChat(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.chatMode = false
		m.chatTI.Blur()
		m.chatTI.SetValue("")
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.chatTI.Value())
		m.chatMode = false
		m.chatTI.Blur()
		m.chatTI.SetValue("")
		if text == "" {
			return m, nil
		}
		return m.sendChat(text)
	}
	var cmd tea.Cmd
	m.chatTI, cmd = m.chatTI.Update(msg)
	return m, cmd
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
	m.chatMode = false
	m.chatTI.Blur()
	m.chatTI.SetValue("")
	m.joinTI.Blur()
	m.joinTI.SetValue("")
	m.stopCountdownTimer()
	m.resetMotion()
}

func (m *Model) syncMulti() {
	m.maybeSyncMulti(true)
}

func (m *Model) maybeSyncMulti(force bool) {
	if m.hub == nil || m.roomCode == "" {
		return
	}
	if !force && !m.lastMulti.IsZero() && m.now.Sub(m.lastMulti) < multiPoll {
		return
	}
	m.lastMulti = m.now
	var v multi.View
	if m.phase == phaseTyping && m.sess != nil && !m.multiView.YouAreSpectator {
		snap := m.sess.Snapshot(m.now)
		chars := m.sess.ProgressChars()
		v = m.hub.Report(m.playerID, multi.Progress{
			WPM:      snap.WPM,
			Accuracy: snap.Accuracy,
			Correct:  snap.Correct,
			Chars:    chars,
			Done:     m.sess.Finished || m.sess.DNF,
			HP:       m.sess.HP,
			DNF:      m.sess.DNF,
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
	m.syncChatViewport()

	if v.Phase == multi.PhaseCountdown {
		if !m.cdOn {
			m.queueCmd(m.startCountdownTimer(v.CountdownLeft))
		}
	} else {
		m.stopCountdownTimer()
	}

	if v.YouAreSpectator {
		switch v.Phase {
		case multi.PhaseDone:
			m.sess = nil
			m.raceStarted = false
			m.phase = phasePodium
			m.syncPodiumTable()
			m.queueCmd(m.stopRaceStopwatch())
		default:
			m.sess = nil
			m.raceStarted = false
			m.phase = phaseSpectate
		}
		return
	}
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
			m.raceSeed = v.Seed
			m.resetConsumableRace()
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
			m.raceBars = nil
			m.shake = spring1D{}
			m.resetCaret()
			m.phase = phaseTyping
			m.queueCmd(m.startRaceStopwatch())
		}
	case multi.PhaseDone:
		if m.sess != nil && !m.sess.Finished {
			m.sess.ForceFinish(m.now)
		}
		if len(m.ghostRec) > 2 {
			m.paceGhost = m.ghostRec
		}
		m.ghostRec = nil
		dnf := m.sess != nil && m.sess.DNF
		m.grantMultiXP(v.RaceNumber, dnf)
		m.phase = phasePodium
		m.syncPodiumTable()
		m.queueCmd(m.stopRaceStopwatch())
	}
}

func (m Model) viewMultiMenu() string {
	var body strings.Builder
	body.WriteString(m.multiMenuList.View())
	return m.renderScreen(Screen{
		Title:    "multiplayer",
		Subtitle: "race friends · best of 3",
		Body:     body.String(),
		Status:   m.statusErr,
		Keys:     helpMultiMenu(),
	})
}

func (m Model) viewJoin() string {
	var body strings.Builder
	body.WriteString(m.sty.Sub.Render("code  "))
	body.WriteString(m.joinTI.View())
	return m.renderScreen(Screen{
		Title:  "join room",
		Body:   body.String(),
		Status: m.statusErr,
		Keys:   helpJoin(),
	})
}

func (m Model) viewChat() string {
	var b strings.Builder
	if m.chatVP.TotalLineCount() > 0 {
		b.WriteString(m.chatVP.View())
		b.WriteString("\n")
	}
	if m.chatMode {
		b.WriteString(m.sty.Main.Render(m.chatTI.View()))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewLobby() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("lobby"))
	b.WriteString("  ")
	b.WriteString(m.sty.Main.Render(v.Code))
	b.WriteString("  ")
	b.WriteString(m.sty.Sub.Render("bo3"))
	b.WriteString("\n")
	detail := fmt.Sprintf("%s · %s", v.Config.Mode.String(), configDetail(v.Config, nil))
	if v.Config.Daily {
		detail += " · " + words.DailyHeadline(m.now)
	}
	if v.Config.ThreeStrike {
		detail += " · hardcore · 3 HP · typo −1"
	} else {
		detail += " · classic"
	}
	b.WriteString(m.sty.Sub.Render(detail))
	b.WriteString("\n")
	if v.MatchPoint && v.Phase == multi.PhaseLobby {
		b.WriteString(m.sty.Main.Render("⚔ MATCH POINT"))
		b.WriteString("\n")
	}
	if v.RaceNumber > 0 && v.Phase == multi.PhaseLobby && !v.MatchOver {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("series %s · next race %d",
			seriesScore(v), v.RaceNumber+1)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if v.Phase == multi.PhaseCountdown {
		next := v.RaceNumber + 1
		if next < 1 {
			next = 1
		}
		banner := fmt.Sprintf("race %d", next)
		if v.MatchPoint {
			banner += " · MATCH POINT"
		}
		b.WriteString(m.viewCinematicCountdown(banner))
	} else {
		b.WriteString("\n")
	}

	for _, p := range v.Players {
		crown := "  "
		if p.Crown {
			crown = "👑"
		}
		if p.Spectator {
			line := fmt.Sprintf("   · %-9s watching", truncateName(p.Name, 9))
			if p.You {
				b.WriteString(m.sty.Sub.Render(line))
			} else {
				b.WriteString(m.sty.Sub.Render(line))
			}
			b.WriteString("\n")
			continue
		}
		score := fmt.Sprintf(" %d", p.MatchWins)
		name := fmt.Sprintf("%s %-10s", crown, truncateName(p.Name, 10))
		if p.You {
			b.WriteString(m.sty.Main.Render(name))
		} else {
			b.WriteString(m.sty.Text.Render(name))
		}
		b.WriteString(m.sty.StatValue.Render(score))
		if p.IsHost {
			b.WriteString(m.sty.Sub.Render(" host"))
		}
		if p.You {
			b.WriteString(m.sty.Sub.Render(" (you)"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.viewChat())
	if v.Phase == multi.PhaseLobby {
		b.WriteString(m.sty.Main.Render(invite.BeatMe(v.Code)))
		b.WriteString("\n")
	}
	if m.statusErr != "" {
		b.WriteString(m.sty.Incorrect.Render(m.statusErr))
		b.WriteString("\n")
	}
	if m.chatMode {
		b.WriteString(m.renderHelp(helpChat()))
	} else {
		b.WriteString(m.renderHelp(helpLobby(v.YouAreHost && v.Phase == multi.PhaseLobby, v.Phase == multi.PhaseCountdown)))
	}
	return b.String()
}

func (m Model) viewRaceOpponents() string {
	v := m.multiView
	racers := 0
	for _, p := range v.Players {
		if !p.Spectator {
			racers++
		}
	}
	if racers == 0 {
		return ""
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
	var b strings.Builder
	b.WriteString("\n")
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		crown := ""
		if p.Crown {
			crown = "👑"
		}
		name := fmt.Sprintf("%s%-8s", crown, truncateName(p.Name, 8))
		if p.You {
			b.WriteString(m.sty.Main.Render(fmt.Sprintf("%-10s", name)))
		} else {
			b.WriteString(m.sty.Sub.Render(fmt.Sprintf("%-10s", name)))
		}
		b.WriteString(" ")
		b.WriteString(m.sty.StatValue.Render(fmt.Sprintf("%3.0f", p.Prog.WPM)))
		b.WriteString(m.sty.Sub.Render(" wpm "))
		b.WriteString(m.raceBarView(p.ID))
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf(" %d", p.MatchWins)))
		if p.Prog.Done {
			b.WriteString(m.sty.Sub.Render(" done"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewPodium() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("podium"))
	b.WriteString("  ")
	b.WriteString(m.sty.Sub.Render(v.Code))
	b.WriteString("\n")
	if v.MatchOver {
		b.WriteString(m.sty.Main.Render("★★ MATCH ★★"))
		b.WriteString("\n")
		b.WriteString(m.sty.Main.Render(v.MatchWinnerName + " takes the series"))
		b.WriteString("\n")
		if v.RaceNumber > 0 {
			b.WriteString(m.sty.Sub.Render(fmt.Sprintf("bo3 · race %d", v.RaceNumber)))
			b.WriteString("\n")
		}
	} else {
		race := v.RaceNumber
		if race < 1 {
			race = 1
		}
		b.WriteString(m.sty.Main.Render(fmt.Sprintf("race %d", race)))
		b.WriteString(m.sty.Sub.Render(" · best of 3"))
		b.WriteString("\n")
		if v.RaceWinnerName != "" {
			b.WriteString(m.sty.Text.Render("race winner · " + v.RaceWinnerName))
			b.WriteString("\n")
		}
		if v.MatchPoint {
			b.WriteString(m.sty.Main.Render("⚔ MATCH POINT NEXT"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	m.syncPodiumTable()
	b.WriteString(m.podiumTable.View())
	b.WriteString("\n")
	if m.sess != nil {
		snap := m.sess.Snapshot(m.now)
		b.WriteString("\n")
		if snap.Correct+snap.Incorrect+snap.Extra == 0 {
			b.WriteString(m.sty.Sub.Render("you · 0 wpm · — acc (no input)"))
		} else {
			b.WriteString(m.sty.Sub.Render(fmt.Sprintf("you · %.0f wpm · %.0f%% acc", snap.WPM, snap.Accuracy)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.viewChat())
	b.WriteString(m.sty.Main.Render(invite.BeatMe(v.Code)))
	b.WriteString("\n")
	if m.chatMode {
		b.WriteString(m.renderHelp(helpChat()))
	} else {
		b.WriteString(m.renderHelp(helpPodium(v.YouAreSpectator, v.MatchOver, v.MatchPoint)))
	}
	return b.String()
}

func (m Model) viewSpectate() string {
	v := m.multiView
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("spectate"))
	b.WriteString("  ")
	b.WriteString(m.sty.Sub.Render(v.Code))
	b.WriteString("\n")
	switch v.Phase {
	case multi.PhaseCountdown:
		b.WriteString(m.viewCinematicCountdown(v.Code))
	case multi.PhaseRacing:
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("%.0fs left", v.RaceRemaining.Seconds())))
	case multi.PhaseLobby:
		b.WriteString(m.sty.Sub.Render("waiting for next race…"))
	default:
		b.WriteString(m.sty.Sub.Render(v.Phase.String()))
	}
	b.WriteString("\n")
	b.WriteString(m.viewRaceOpponents())
	b.WriteString("\n")
	b.WriteString(m.viewChat())
	b.WriteString(m.sty.Main.Render(invite.BeatMe(v.Code)))
	b.WriteString("\n")
	if m.chatMode {
		b.WriteString(m.renderHelp(helpChat()))
	} else {
		b.WriteString(m.renderHelp(helpSpectate()))
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

func seriesScore(v multi.View) string {
	// show top two match wins as a–b
	a, b := 0, 0
	for _, p := range v.Players {
		if p.Spectator {
			continue
		}
		if p.MatchWins > a {
			b = a
			a = p.MatchWins
		} else if p.MatchWins > b {
			b = p.MatchWins
		}
	}
	return fmt.Sprintf("%d–%d", a, b)
}
