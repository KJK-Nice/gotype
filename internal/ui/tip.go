package ui

import (
	"context"
	"fmt"
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
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(helpTipShow()))
	}
	return b.String()
}
