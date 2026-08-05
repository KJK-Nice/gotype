package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
)

type tipPhase int

const (
	tipNone tipPhase = iota
	tipPick
	tipLoading
	tipShow
)

type tipMsg struct {
	bolt11 string
	qr     string
	sats   int
	err    string
}

func (m *Model) clearTip() {
	m.tipPhase = tipNone
	m.tipAmountIdx = 0
	m.tipBolt11 = ""
	m.tipQR = ""
	m.tipErr = ""
}

func (m Model) tipSats() int {
	if it, ok := m.tipList.SelectedItem().(menuItem); ok {
		if n, err := strconv.Atoi(it.action); err == nil {
			return n
		}
	}
	amts := ln.DefaultAmounts
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
	return func() tea.Msg {
		inv, err := ln.CreateInvoice(context.Background(), sats, comment)
		if err != nil {
			return tipMsg{sats: sats, err: err.Error()}
		}
		return tipMsg{
			sats:   inv.Sats,
			bolt11: inv.Bolt11,
			qr:     ln.QRString(inv.Bolt11),
		}
	}
}

func (m Model) viewTip() string {
	switch m.tipPhase {
	case tipPick:
		var body strings.Builder
		body.WriteString(m.tipList.View())
		return m.renderScreen(Screen{
			Title:    "tip",
			Subtitle: "lightning sats",
			Meta:     ln.Destination(),
			Body:     body.String(),
			Keys:     helpTipPick(),
		})
	case tipLoading:
		return m.renderPayment(PaymentView{
			Title:   "tip",
			Sats:    m.tipSats(),
			Spinner: m.spin.View(),
			Status:  fmt.Sprintf("fetching %d sat invoice…", m.tipSats()),
		}, helpTipLoading())
	case tipShow:
		if m.tipErr != "" {
			return m.renderPayment(PaymentView{
				Title: "tip",
				Err:   m.tipErr,
			}, helpTipShow())
		}
		return m.renderPayment(PaymentView{
			Title:    "tip",
			Subtitle: "scan with a lightning wallet",
			Sats:     m.tipSats(),
			QR:       m.tipQR,
			Bolt11:   m.tipBolt11,
		}, helpTipShow())
	}
	return ""
}
