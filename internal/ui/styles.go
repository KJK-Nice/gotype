package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is a named color pack for the TUI.
type Theme struct {
	Name  string
	Bg    string
	Main  string
	Text  string
	Sub   string
	Error string
	Extra string
	Ghost string
}

var themes = []Theme{
	{Name: "amber", Bg: "#323437", Main: "#e2b714", Text: "#d1d0c5", Sub: "#646669", Error: "#ca4754", Extra: "#7e2a33", Ghost: "#5c4e12"},
	{Name: "amber light", Bg: "#e1e1e3", Main: "#e2b714", Text: "#323437", Sub: "#aaaeb3", Error: "#ca4754", Extra: "#7e2a33", Ghost: "#e8d48a"},
	{Name: "nord", Bg: "#242933", Main: "#88c0d0", Text: "#d8dee9", Sub: "#616e88", Error: "#bf616a", Extra: "#8f3d44", Ghost: "#3d5a66"},
	{Name: "olive", Bg: "#1e1e1e", Main: "#a7c080", Text: "#d3c6aa", Sub: "#7a8474", Error: "#e67e80", Extra: "#8b4b4f", Ghost: "#3f4f35"},
	{Name: "dracula", Bg: "#282a36", Main: "#bd93f9", Text: "#f8f8f2", Sub: "#6272a4", Error: "#ff5555", Extra: "#aa3333", Ghost: "#4a3a6a"},
}

// Styles holds per-user lipgloss styles (not process-global).
type Styles struct {
	Title     lipgloss.Style
	Sub       lipgloss.Style
	Text      lipgloss.Style
	Main      lipgloss.Style
	Correct   lipgloss.Style
	Incorrect lipgloss.Style
	ErrorDot  lipgloss.Style
	Extra     lipgloss.Style
	Pending   lipgloss.Style
	Caret     lipgloss.Style
	Ghost     lipgloss.Style
	Selected  lipgloss.Style
	Option    lipgloss.Style
	StatValue lipgloss.Style
	StatLabel lipgloss.Style
	Box       lipgloss.Style

	main  color.Color
	sub   color.Color
	ghost color.Color
}

func ThemeName(i int) string {
	if i < 0 || i >= len(themes) {
		return themes[0].Name
	}
	return themes[i].Name
}

func ThemeCount() int { return len(themes) }

// NewStyles builds an independent style set for one session/user.
func NewStyles(idx int) Styles {
	if idx < 0 {
		idx = 0
	}
	idx %= len(themes)
	t := themes[idx]

	bg := lipgloss.Color(t.Bg)
	main := lipgloss.Color(t.Main)
	text := lipgloss.Color(t.Text)
	sub := lipgloss.Color(t.Sub)
	errc := lipgloss.Color(t.Error)
	extra := lipgloss.Color(t.Extra)
	ghost := lipgloss.Color(t.Ghost)

	return Styles{
		Title:     lipgloss.NewStyle().Foreground(main).Bold(true),
		Sub:       lipgloss.NewStyle().Foreground(sub),
		Text:      lipgloss.NewStyle().Foreground(text),
		Main:      lipgloss.NewStyle().Foreground(main).Bold(true),
		Correct:   lipgloss.NewStyle().Foreground(text),
		Incorrect: lipgloss.NewStyle().Foreground(errc).Underline(true),
		ErrorDot:  lipgloss.NewStyle().Foreground(errc).Bold(true),
		Extra:     lipgloss.NewStyle().Foreground(extra).Underline(true),
		Pending:   lipgloss.NewStyle().Foreground(sub),
		Caret:     lipgloss.NewStyle().Foreground(bg).Background(main),
		Ghost:     lipgloss.NewStyle().Foreground(sub).Background(ghost),
		Selected:  lipgloss.NewStyle().Foreground(bg).Background(main).Padding(0, 1),
		Option:    lipgloss.NewStyle().Foreground(sub).Padding(0, 1),
		StatValue: lipgloss.NewStyle().Foreground(main).Bold(true),
		StatLabel: lipgloss.NewStyle().Foreground(sub),
		Box:       lipgloss.NewStyle().Padding(1, 2),
		main:      main,
		sub:       sub,
		ghost:     ghost,
	}
}

func (s Styles) trailBackground(life int) color.Color {
	switch {
	case life >= 5:
		return s.main
	case life >= 3:
		return s.ghost
	default:
		return s.sub
	}
}

func (s Styles) WithTrail(base lipgloss.Style, life int) lipgloss.Style {
	return base.Background(s.trailBackground(life))
}
