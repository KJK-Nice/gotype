package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/lnauth"
	"github.com/kjkusap/monkeytype-clone/internal/player"
)

type loginMode int

const (
	loginIdle loginMode = iota
	loginHelp           // LNURL-auth unavailable
	loginWait           // scan QR
	loginName           // new wallet picks display name
)

// Popular Lightning wallets that speak LNURL-auth (LUD-04).
const lightningWallets = "Phoenix · Zeus · Breez · Alby · Blink"

func (m *Model) initLoginInputs() {
	m.loginNameTI = textinput.New()
	m.loginNameTI.Placeholder = "display name"
	m.loginNameTI.CharLimit = player.NameMax
	m.loginNameTI.SetWidth(20)
}

func (m *Model) openLogin() tea.Cmd {
	if m.app == nil {
		m.statusErr = "progression unavailable"
		return nil
	}
	m.loginErr = ""
	m.loginWalletK1 = ""
	m.loginWalletQR = ""
	m.loginNameTI.SetValue("")
	if !m.walletLoginEnabled() {
		m.loginMode = loginHelp
		m.loginErr = "this server has no public login URL"
		return nil
	}
	m.loginMode = loginWait
	return m.startWalletAuth(lnauth.ActionLogin)
}

func (m *Model) clearLoginUI() {
	m.loginMode = loginIdle
	m.loginErr = ""
	m.loginWalletK1 = ""
	m.loginWalletQR = ""
	m.loginNameTI.Blur()
}

func (m Model) updateLogin(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.loginMode {
	case loginHelp:
		switch msg.String() {
		case "esc":
			m.clearLoginUI()
		case "q":
			return m, tea.Quit
		}
	case loginWait:
		switch msg.String() {
		case "esc":
			m.clearLoginUI()
		case "q":
			return m, tea.Quit
		}
	case loginName:
		switch msg.String() {
		case "esc":
			m.clearLoginUI()
		case "enter":
			return m, m.doWalletRegister()
		default:
			var cmd tea.Cmd
			m.loginNameTI, cmd = m.loginNameTI.Update(msg)
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
			return walletStartMsg{err: "login unavailable"}
		}
		start, err := m.app.LNAuth.Start(m.sessionID, action, "", time.Now())
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
		m.loginErr = msg.err
		m.loginMode = loginHelp
		return nil
	}
	m.loginWalletK1 = msg.k1
	m.loginWalletQR = msg.qr
	m.loginMode = loginWait
	m.loginErr = ""
	return tea.Batch(m.spin.Tick, m.walletPollCmd())
}

func (m Model) walletPollCmd() tea.Cmd {
	k1 := m.loginWalletK1
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
	if m.loginMode != loginWait && m.loginMode != loginName {
		return nil
	}
	if msg.err != "" {
		m.loginErr = msg.err
		return m.walletPollCmd()
	}
	switch msg.status.State {
	case lnauth.StatePending:
		return m.walletPollCmd()
	case lnauth.StateVerified:
		m.loginMode = loginName
		m.loginNameTI.SetValue("")
		return m.loginNameTI.Focus()
	case lnauth.StateOK:
		m.claimedID = msg.status.PlayerID
		m.playerName = msg.status.Name
		m.clearLoginUI()
		return nil
	case lnauth.StateError:
		m.loginErr = msg.status.Err
		m.loginMode = loginHelp
		return nil
	default:
		return m.walletPollCmd()
	}
}

func (m Model) doWalletRegister() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil || m.app.LNAuth == nil {
			return loginMsg{err: "no app"}
		}
		p, err := m.app.LNAuth.CompleteRegister(m.loginWalletK1, m.loginNameTI.Value(), m.remoteIP, time.Now())
		if err != nil {
			return loginMsg{err: err.Error()}
		}
		return loginMsg{playerID: p.ID, name: p.Name}
	}
}

type loginMsg struct {
	playerID string
	name     string
	err      string
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

func (m *Model) applyLoginMsg(msg loginMsg) {
	if msg.err != "" {
		m.loginErr = msg.err
		return
	}
	m.claimedID = msg.playerID
	m.playerName = msg.name
	m.clearLoginUI()
}

func (m Model) viewLogin() string {
	switch m.loginMode {
	case loginHelp:
		var body strings.Builder
		body.WriteString(m.sty.Main.Render("keep your name and progress with LNURL-auth"))
		body.WriteString("\n\n")
		body.WriteString(m.sty.Sub.Render("scan a QR in a Lightning wallet:"))
		body.WriteString("\n")
		body.WriteString(m.sty.Main.Render(lightningWallets))
		return m.renderScreen(Screen{
			Title:    "login",
			Subtitle: "Lightning wallet",
			Body:     body.String(),
			Status:   m.loginErr,
			Keys:     km(bind("esc", "esc", "back")),
		})
	case loginWait:
		return m.renderPayment(PaymentView{
			Title:    "login",
			Subtitle: "Lightning wallet",
			QR:       m.loginWalletQR,
			Spinner:  m.spin.View(),
			Status:   "waiting for wallet…",
			Hint:     lightningWallets,
		}, km(bind("esc", "esc", "cancel")))
	case loginName:
		var body strings.Builder
		body.WriteString(m.sty.Sub.Render("choose display name (3–16)"))
		body.WriteString("\n")
		body.WriteString(m.loginNameTI.View())
		return m.renderScreen(Screen{
			Title:    "login",
			Subtitle: "new Player",
			Body:     body.String(),
			Status:   m.loginErr,
			Keys:     km(bind("enter", "enter", "create"), bind("esc", "esc", "cancel")),
		})
	}
	return ""
}

func (m Model) isClaimed() bool {
	return m.claimedID != ""
}
