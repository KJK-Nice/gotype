package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"

	"github.com/kjkusap/monkeytype-clone/internal/player"
)

type claimMode int

const (
	claimIdle claimMode = iota
	claimPick
	claimRegisterName
	claimRegisterShow // show one-time Claim Code
	claimReclaimName
	claimReclaimCode
)

func (m *Model) initClaimInputs() {
	m.claimNameTI = textinput.New()
	m.claimNameTI.Placeholder = "display name"
	m.claimNameTI.CharLimit = player.NameMax
	m.claimNameTI.SetWidth(20)
	m.claimCodeTI = textinput.New()
	m.claimCodeTI.Placeholder = "XXXX-XXXX-XXXX"
	m.claimCodeTI.CharLimit = 14
	m.claimCodeTI.SetWidth(16)
	m.claimCodeTI.EchoMode = textinput.EchoPassword
	m.claimCodeTI.EchoCharacter = '•'
}

func (m *Model) openClaim() {
	if m.app == nil {
		m.statusErr = "progression unavailable"
		return
	}
	m.claimMode = claimPick
	m.claimErr = ""
	m.claimShown = ""
	m.claimNameTI.SetValue("")
	m.claimCodeTI.SetValue("")
}

func (m *Model) clearClaimUI() {
	m.claimMode = claimIdle
	m.claimErr = ""
	m.claimShown = ""
	m.claimNameTI.Blur()
	m.claimCodeTI.Blur()
}

func (m Model) updateClaim(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.claimMode {
	case claimPick:
		switch msg.String() {
		case "esc":
			m.clearClaimUI()
		case "r":
			m.claimMode = claimRegisterName
			m.claimErr = ""
			return m, m.claimNameTI.Focus()
		case "c":
			m.claimMode = claimReclaimName
			m.claimErr = ""
			return m, m.claimNameTI.Focus()
		case "q":
			return m, tea.Quit
		}
	case claimRegisterName:
		switch msg.String() {
		case "esc":
			m.claimMode = claimPick
			m.claimNameTI.Blur()
		case "enter":
			return m, m.doRegister()
		default:
			var cmd tea.Cmd
			m.claimNameTI, cmd = m.claimNameTI.Update(msg)
			return m, cmd
		}
	case claimRegisterShow:
		switch msg.String() {
		case "esc", "enter", " ":
			m.clearClaimUI()
		case "q":
			return m, tea.Quit
		}
	case claimReclaimName:
		switch msg.String() {
		case "esc":
			m.claimMode = claimPick
			m.claimNameTI.Blur()
		case "enter":
			m.claimMode = claimReclaimCode
			m.claimNameTI.Blur()
			return m, m.claimCodeTI.Focus()
		default:
			var cmd tea.Cmd
			m.claimNameTI, cmd = m.claimNameTI.Update(msg)
			return m, cmd
		}
	case claimReclaimCode:
		switch msg.String() {
		case "esc":
			m.claimMode = claimReclaimName
			m.claimCodeTI.Blur()
			return m, m.claimNameTI.Focus()
		case "enter":
			return m, m.doClaim()
		default:
			var cmd tea.Cmd
			m.claimCodeTI, cmd = m.claimCodeTI.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) doRegister() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return claimMsg{err: "no app"}
		}
		reg, err := m.app.Players.Register(m.claimNameTI.Value(), m.remoteIP, m.sessionID, time.Now())
		if err != nil {
			return claimMsg{err: err.Error()}
		}
		return claimMsg{playerID: reg.Player.ID, name: reg.Player.Name, shown: reg.Display, registered: true}
	}
}

func (m Model) doClaim() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return claimMsg{err: "no app"}
		}
		p, err := m.app.Players.Claim(m.claimNameTI.Value(), m.claimCodeTI.Value(), m.remoteIP, m.sessionID, time.Now())
		if err != nil {
			return claimMsg{err: err.Error()}
		}
		return claimMsg{playerID: p.ID, name: p.Name}
	}
}

type claimMsg struct {
	playerID   string
	name       string
	shown      string
	registered bool
	err        string
}

func (m *Model) applyClaimMsg(msg claimMsg) {
	if msg.err != "" {
		m.claimErr = msg.err
		return
	}
	m.claimedID = msg.playerID
	m.playerName = msg.name
	m.claimErr = ""
	if msg.registered {
		m.claimShown = msg.shown
		m.claimMode = claimRegisterShow
		m.claimNameTI.Blur()
		return
	}
	m.clearClaimUI()
}

func (m Model) viewClaim() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("claim player"))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("persisted name + Claim Code · no recovery if lost"))
	b.WriteString("\n\n")
	switch m.claimMode {
	case claimPick:
		b.WriteString(m.sty.Text.Render("r  register new Player"))
		b.WriteString("\n")
		b.WriteString(m.sty.Text.Render("c  reclaim with Claim Code"))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(km(bind("esc", "esc", "back"))))
	case claimRegisterName:
		b.WriteString(m.sty.Sub.Render("choose display name (3–16)"))
		b.WriteString("\n")
		b.WriteString(m.claimNameTI.View())
		b.WriteString("\n\n")
		if m.claimErr != "" {
			b.WriteString(m.sty.Incorrect.Render(m.claimErr))
			b.WriteString("\n\n")
		}
		b.WriteString(m.renderHelp(km(bind("enter", "enter", "create"), bind("esc", "esc", "back"))))
	case claimRegisterShow:
		b.WriteString(m.sty.Main.Render("save your Claim Code — shown once"))
		b.WriteString("\n\n")
		b.WriteString(m.sty.Title.Render(m.claimShown))
		b.WriteString("\n\n")
		b.WriteString(m.sty.Incorrect.Render("lose this → new Player (no recovery v1)"))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(km(bind("enter", "enter", "done"))))
	case claimReclaimName:
		b.WriteString(m.sty.Sub.Render("name"))
		b.WriteString("\n")
		b.WriteString(m.claimNameTI.View())
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(km(bind("enter", "enter", "next"), bind("esc", "esc", "back"))))
	case claimReclaimCode:
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("Claim Code for %s", m.claimNameTI.Value())))
		b.WriteString("\n")
		b.WriteString(m.claimCodeTI.View())
		b.WriteString("\n\n")
		if m.claimErr != "" {
			b.WriteString(m.sty.Incorrect.Render(m.claimErr))
			b.WriteString("\n\n")
		}
		b.WriteString(m.renderHelp(km(bind("enter", "enter", "claim"), bind("esc", "esc", "back"))))
	}
	return b.String()
}

func (m Model) isClaimed() bool {
	return m.claimedID != ""
}
