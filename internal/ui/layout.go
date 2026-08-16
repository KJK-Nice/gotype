package ui

import (
	"strings"
)

// Screen is the standard shell for menus, overlays, and wizards.
type Screen struct {
	Title    string
	Subtitle string
	Meta     string // breadcrumb · player name · room code
	Body     string
	Status   string // errors, hints
	Keys     phaseKeyMap
}

func (m Model) renderScreen(sc Screen) string {
	var b strings.Builder

	if sc.Title != "" {
		line := m.sty.Title.Render(sc.Title)
		if sc.Subtitle != "" {
			line += " " + m.sty.Sub.Render(sc.Subtitle)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if sc.Meta != "" {
		b.WriteString(m.sty.Sub.Render(sc.Meta))
		b.WriteString("\n")
	}
	if sc.Title != "" || sc.Meta != "" {
		b.WriteString(m.sty.Divider.Render(strings.Repeat("─", min(42, max(20, m.width-8)))))
		b.WriteString("\n")
	}

	if sc.Body != "" {
		b.WriteString(sc.Body)
		if !strings.HasSuffix(sc.Body, "\n") {
			b.WriteString("\n")
		}
	}

	if sc.Status != "" {
		if sc.Body != "" {
			b.WriteString("\n")
		}
		b.WriteString(m.sty.Incorrect.Render(sc.Status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp(sc.Keys))
	return b.String()
}

func (m Model) progMeta() string {
	if m.isClaimed() {
		return m.playerName
	}
	return "guest · login to save progress"
}

func (m Model) progTabBar() string {
	tab := func(key, label string, on bool) string {
		s := " " + key + " " + label + " "
		if on {
			return m.sty.TabActive.Render(s)
		}
		return m.sty.TabIdle.Render(s)
	}
	return strings.Join([]string{
		tab("i", "inv", m.prog == progInventory),
		tab("s", "shop", m.prog == progShop),
		tab("p", "pass", m.prog == progPass),
		tab("e", "equip", m.prog == progEquip),
	}, "")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
