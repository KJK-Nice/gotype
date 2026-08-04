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
			return m, nil
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
		case "enter":
			switch selectedAction(m.claimList) {
			case "register":
				m.claimMode = claimRegisterName
				m.claimErr = ""
				return m, m.claimNameTI.Focus()
			case "reclaim":
				m.claimMode = claimReclaimName
				m.claimErr = ""
				return m, m.claimNameTI.Focus()
			}
		}
		var cmd tea.Cmd
		m.claimList, cmd = m.claimList.Update(msg)
		return m, cmd
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
	switch m.claimMode {
	case claimPick:
		var body strings.Builder
		body.WriteString(m.claimList.View())
		return m.renderScreen(Screen{
			Title:    "claim player",
			Subtitle: "persisted name + Claim Code",
			Meta:     "no recovery if code is lost",
			Body:     body.String(),
			Status:   m.claimErr,
			Keys:     km(bind("enter", "enter", "select"), bind("esc", "esc", "back")),
		})
	case claimRegisterName:
		var body strings.Builder
		body.WriteString(m.sty.Sub.Render("choose display name (3–16)"))
		body.WriteString("\n")
		body.WriteString(m.claimNameTI.View())
		return m.renderScreen(Screen{
			Title:  "register",
			Body:   body.String(),
			Status: m.claimErr,
			Keys:   km(bind("enter", "enter", "create"), bind("esc", "esc", "back")),
		})
	case claimRegisterShow:
		var body strings.Builder
		body.WriteString(m.sty.Main.Render("save your Claim Code — shown once"))
		body.WriteString("\n\n")
		body.WriteString(m.sty.Title.Render(m.claimShown))
		body.WriteString("\n\n")
		body.WriteString(m.sty.Incorrect.Render("lose this → new Player (no recovery v1)"))
		return m.renderScreen(Screen{
			Title: "registered",
			Body:  body.String(),
			Keys:  km(bind("enter", "enter", "done")),
		})
	case claimReclaimName:
		var body strings.Builder
		body.WriteString(m.sty.Sub.Render("name"))
		body.WriteString("\n")
		body.WriteString(m.claimNameTI.View())
		return m.renderScreen(Screen{
			Title: "reclaim",
			Body:  body.String(),
			Keys:  km(bind("enter", "enter", "next"), bind("esc", "esc", "back")),
		})
	case claimReclaimCode:
		var body strings.Builder
		body.WriteString(m.sty.Sub.Render(fmt.Sprintf("Claim Code for %s", m.claimNameTI.Value())))
		body.WriteString("\n")
		body.WriteString(m.claimCodeTI.View())
		return m.renderScreen(Screen{
			Title:  "reclaim",
			Body:   body.String(),
			Status: m.claimErr,
			Keys:   km(bind("enter", "enter", "claim"), bind("esc", "esc", "back")),
		})
	}
	return ""
}

func (m Model) isClaimed() bool {
	return m.claimedID != ""
}
