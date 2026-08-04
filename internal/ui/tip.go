package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

type tipPhase int

const (
	tipNone tipPhase = iota
	tipPick
	tipLoading
	tipShow
	tipThanks
)

type tipMsg struct {
	id      string
	bolt11  string
	qr      string
	sats    int
	tracked bool
	err     string
}

type tipPollMsg struct {
	tip persist.TipIntent
	err string
}

func (m *Model) clearTip() {
	m.tipPhase = tipNone
	m.tipAmountIdx = 0
	m.tipID = ""
	m.tipBolt11 = ""
	m.tipQR = ""
	m.tipErr = ""
	m.tipTracked = false
}

func (m Model) tipSats() int {
	amts := ln.DefaultAmounts
	if item, ok := m.tipList.SelectedItem().(tipItem); ok {
		return item.sats
	}
	if m.tipAmountIdx < 0 || m.tipAmountIdx >= len(amts) {
		return amts[0]
	}
	return amts[m.tipAmountIdx]
}

func (m Model) updateTip(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.tipPhase {
	case tipPick:
		switch msg.String() {
		case "esc":
			m.clearTip()
			return m, nil
		case "enter", " ":
			m.tipAmountIdx = m.tipList.Index()
			m.tipPhase = tipLoading
			m.tipErr = ""
			return m, tea.Batch(m.tipInvoiceCmd(), m.spin.Tick)
		case "q":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.tipList, cmd = m.tipList.Update(msg)
		return m, cmd
	case tipLoading:
		switch msg.String() {
		case "esc":
			m.clearTip()
		case "q":
			return m, tea.Quit
		}
	case tipShow:
		switch msg.String() {
		case "esc", "enter", " ":
			m.clearTip()
		case "q":
			return m, tea.Quit
		}
	case tipThanks:
		switch msg.String() {
		case "esc", "enter", " ", "q":
			if msg.String() == "q" {
				return m, tea.Quit
			}
			m.clearTip()
		}
	}
	return m, nil
}

func (m Model) tipInvoiceCmd() tea.Cmd {
	sats := m.tipSats()
	comment := "gotype tip"
	if m.sess != nil {
		snap := m.sess.Snapshot(m.now)
		comment = fmt.Sprintf("gotype · %.0fwpm · %.0f%%", snap.WPM, snap.Accuracy)
	}
	var store *persist.Store
	if m.app != nil {
		store = m.app.Store
	}
	return func() tea.Msg {
		inv, err := ln.CreateInvoice(context.Background(), store, sats, comment, time.Now())
		if err != nil {
			return tipMsg{sats: sats, err: err.Error()}
		}
		return tipMsg{
			id:      inv.ID,
			sats:    inv.Sats,
			bolt11:  inv.Bolt11,
			qr:      ln.QRString(inv.Bolt11),
			tracked: inv.Tracked,
		}
	}
}

func (m Model) tipPollCmd() tea.Cmd {
	id := m.tipID
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		if m.app == nil || id == "" {
			return tipPollMsg{}
		}
		tip, err := ln.PollTip(context.Background(), m.app.Store, id, t)
		if err != nil {
			return tipPollMsg{err: err.Error()}
		}
		return tipPollMsg{tip: tip}
	})
}

func (m *Model) tipLoadingDone(msg tipMsg) tea.Cmd {
	if msg.err != "" {
		m.tipPhase = tipShow
		m.tipErr = msg.err
		m.tipID = ""
		m.tipBolt11 = ""
		m.tipQR = ""
		m.tipTracked = false
		return nil
	}
	m.tipPhase = tipShow
	m.tipErr = ""
	m.tipID = msg.id
	m.tipBolt11 = msg.bolt11
	m.tipQR = msg.qr
	m.tipTracked = msg.tracked
	if msg.sats > 0 {
		for i, s := range ln.DefaultAmounts {
			if s == msg.sats {
				m.tipAmountIdx = i
				break
			}
		}
	}
	if msg.tracked && msg.id != "" {
		return m.tipPollCmd()
	}
	return nil
}

func (m *Model) applyTipPoll(msg tipPollMsg) tea.Cmd {
	if m.tipPhase != tipShow || !m.tipTracked {
		return nil
	}
	if msg.err != "" {
		m.tipErr = msg.err
		return m.tipPollCmd()
	}
	if msg.tip.ID == "" {
		return nil
	}
	if msg.tip.State == persist.TipPaid {
		m.tipPhase = tipThanks
		m.tipErr = ""
		return nil
	}
	return m.tipPollCmd()
}

func (m Model) viewTip() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("tip"))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("lightning sats · " + ln.Destination()))
	b.WriteString("\n\n")

	switch m.tipPhase {
	case tipPick:
		b.WriteString(m.tipList.View())
		b.WriteString("\n")
		b.WriteString(m.renderHelp(helpTipPick()))
	case tipLoading:
		b.WriteString(m.sty.Sub.Render(m.spin.View() + fmt.Sprintf(" fetching %d sat invoice…", m.tipSats())))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(helpTipLoading()))
	case tipShow:
		if m.tipErr != "" {
			b.WriteString(m.sty.Incorrect.Render(m.tipErr))
			b.WriteString("\n\n")
			b.WriteString(m.renderHelp(helpTipShow()))
			return b.String()
		}
		b.WriteString(m.sty.Main.Render(fmt.Sprintf("%d sats", m.tipSats())))
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render("scan with a lightning wallet"))
		b.WriteString("\n\n")
		if m.tipQR != "" {
			b.WriteString(m.tipQR)
			b.WriteString("\n\n")
		}
		b.WriteString(m.sty.Sub.Render(ln.ShortBolt11(m.tipBolt11)))
		b.WriteString("\n")
		if m.tipTracked {
			b.WriteString(m.sty.Sub.Render(m.spin.View() + " waiting for payment…"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(m.renderHelp(helpTipShow()))
	case tipThanks:
		b.WriteString(m.sty.Main.Render(fmt.Sprintf("thanks · %d sats received", m.tipSats())))
		b.WriteString("\n\n")
		b.WriteString(m.sty.Sub.Render("operator tip settled"))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(helpTipThanks()))
	}
	return b.String()
}
