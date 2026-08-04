package ui

import (
	"fmt"
	"strings"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
)

// PaymentView is shared QR / invoice UI for tip and shop Buy flows.
type PaymentView struct {
	Title    string
	Subtitle string
	Sats     int
	QR       string
	Bolt11   string
	Spinner  string
	Status   string // waiting, paid, error line
	Err      string
	Done     bool
}

func (m Model) renderPayment(pv PaymentView, keys phaseKeyMap) string {
	var body strings.Builder

	if pv.Done {
		body.WriteString(m.sty.Main.Render(pv.Status))
		body.WriteString("\n")
	} else if pv.Err != "" {
		body.WriteString(m.sty.Incorrect.Render(pv.Err))
		body.WriteString("\n")
	} else {
		if pv.Sats > 0 {
			body.WriteString(m.sty.Main.Render(fmt.Sprintf("%d sats", pv.Sats)))
			body.WriteString("\n")
		}
		if pv.Subtitle != "" {
			body.WriteString(m.sty.Sub.Render(pv.Subtitle))
			body.WriteString("\n")
		}
		body.WriteString("\n")
		if pv.QR != "" {
			body.WriteString(pv.QR)
			body.WriteString("\n\n")
		}
		if pv.Bolt11 != "" {
			body.WriteString(m.sty.Sub.Render(ln.ShortBolt11(pv.Bolt11)))
			body.WriteString("\n")
		}
		if pv.Spinner != "" || pv.Status != "" {
			line := strings.TrimSpace(pv.Spinner + " " + pv.Status)
			if line != "" {
				body.WriteString(m.sty.Sub.Render(line))
				body.WriteString("\n")
			}
		}
	}

	return m.renderScreen(Screen{
		Title:    pv.Title,
		Subtitle: pv.Subtitle,
		Body:     body.String(),
		Status:   pv.Err,
		Keys:     keys,
	})
}
