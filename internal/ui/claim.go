package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/lnauth"
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
	claimWalletWait
	claimWalletName
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
	m.claimWalletK1 = ""
	m.claimWalletQR = ""
	m.claimNameTI.SetValue("")
	m.claimCodeTI.SetValue("")
	m.syncClaimList()
}

func (m *Model) clearClaimUI() {
	m.claimMode = claimIdle
	m.claimErr = ""
	m.claimShown = ""
	m.claimWalletK1 = ""
	m.claimWalletQR = ""
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
		case "w":
			if m.walletLoginEnabled() {
				return m, m.startWalletAuth(lnauth.ActionLogin)
			}
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
			case "wallet":
				return m, m.startWalletAuth(lnauth.ActionLogin)
			case "link":
				return m, m.startWalletAuth(lnauth.ActionLink)
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
	case claimWalletWait:
		switch msg.String() {
		case "esc":
			m.clearClaimUI()
		case "q":
			return m, tea.Quit
		}
	case claimWalletName:
		switch msg.String() {
		case "esc":
			m.clearClaimUI()
		case "enter":
			return m, m.doWalletRegister()
		default:
			var cmd tea.Cmd
			m.claimNameTI, cmd = m.claimNameTI.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) walletLoginEnabled() bool {
	return m.app != nil && m.app.LNAuth != nil && m.app.LNAuth.Enabled()
}

func (m Model) startWalletAuth(action lnauth.Action) tea.Cmd {
	return func() tea.Msg {
		if m.app == nil || m.app.LNAuth == nil {
			return walletStartMsg{err: "wallet login unavailable"}
		}
		playerID := ""
		if action == lnauth.ActionLink {
			playerID = m.claimedID
		}
		start, err := m.app.LNAuth.Start(m.sessionID, action, playerID, time.Now())
		if err != nil {
			return walletStartMsg{err: err.Error()}
		}
		return walletStartMsg{
			k1:    start.K1,
			lnurl: start.LNURL,
			qr:    ln.QRString(start.LNURL),
		}
	}
}

func (m *Model) applyWalletStart(msg walletStartMsg) tea.Cmd {
	if msg.err != "" {
		m.claimErr = msg.err
		m.claimMode = claimPick
		return nil
	}
	m.claimWalletK1 = msg.k1
	m.claimWalletQR = msg.qr
	m.claimMode = claimWalletWait
	m.claimErr = ""
	return tea.Batch(m.spin.Tick, m.walletPollCmd())
}

func (m Model) walletPollCmd() tea.Cmd {
	k1 := m.claimWalletK1
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		if m.app == nil || m.app.LNAuth == nil || k1 == "" {
			return walletPollMsg{err: "no wallet auth"}
		}
		st, err := m.app.LNAuth.Status(k1)
		if err != nil {
			return walletPollMsg{err: err.Error()}
		}
		return walletPollMsg{status: st}
	})
}

func (m *Model) applyWalletPoll(msg walletPollMsg) tea.Cmd {
	if m.claimMode != claimWalletWait && m.claimMode != claimWalletName {
		return nil
	}
	if msg.err != "" {
		m.claimErr = msg.err
		return m.walletPollCmd()
	}
	switch msg.status.State {
	case lnauth.StatePending:
		return m.walletPollCmd()
	case lnauth.StateVerified:
		m.claimMode = claimWalletName
		m.claimNameTI.SetValue("")
		return m.claimNameTI.Focus()
	case lnauth.StateOK:
		m.claimedID = msg.status.PlayerID
		m.playerName = msg.status.Name
		m.clearClaimUI()
		return nil
	case lnauth.StateError:
		m.claimErr = msg.status.Err
		m.claimMode = claimPick
		return nil
	default:
		return m.walletPollCmd()
	}
}

func (m Model) doWalletRegister() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil || m.app.LNAuth == nil {
			return claimMsg{err: "no app"}
		}
		p, err := m.app.LNAuth.CompleteRegister(m.claimWalletK1, m.claimNameTI.Value(), m.remoteIP, time.Now())
		if err != nil {
			return claimMsg{err: err.Error()}
		}
		return claimMsg{playerID: p.ID, name: p.Name}
	}
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

type walletStartMsg struct {
	k1    string
	lnurl string
	qr    string
	err   string
}

type walletPollMsg struct {
	status lnauth.Status
	err    string
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
		meta := "Claim Code · no recovery if code is lost"
		if m.walletLoginEnabled() {
			meta = "Claim Code or Lightning wallet"
		}
		return m.renderScreen(Screen{
			Title:    "claim player",
			Subtitle: "persisted name + secret",
			Meta:     meta,
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
	case claimWalletWait:
		return m.renderPayment(PaymentView{
			Title:    "lightning login",
			Subtitle: "scan with a lightning wallet",
			QR:       m.claimWalletQR,
			Spinner:  m.spin.View(),
			Status:   "waiting for wallet…",
		}, km(bind("esc", "esc", "cancel")))
	case claimWalletName:
		var body strings.Builder
		body.WriteString(m.sty.Sub.Render("choose display name (3–16)"))
		body.WriteString("\n")
		body.WriteString(m.claimNameTI.View())
		return m.renderScreen(Screen{
			Title:  "wallet registered",
			Body:   body.String(),
			Status: m.claimErr,
			Keys:   km(bind("enter", "enter", "create"), bind("esc", "esc", "cancel")),
		})
	}
	return ""
}

func (m Model) isClaimed() bool {
	return m.claimedID != ""
}
