package ui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
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
	Divider   lipgloss.Style
	TabActive lipgloss.Style
	TabIdle   lipgloss.Style

	main  color.Color
	sub   color.Color
	text  color.Color
	ghost color.Color
	dark  bool
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

	isDark := idx != 1 // amber light is the only light theme

	s := Styles{
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
		Divider:   lipgloss.NewStyle().Foreground(sub),
		TabActive: lipgloss.NewStyle().Foreground(bg).Background(main).Padding(0, 1),
		TabIdle:   lipgloss.NewStyle().Foreground(sub).Padding(0, 1),
		main:      main,
		sub:       sub,
		text:      text,
		ghost:     ghost,
		dark:      isDark,
	}
	return s
}

// ApplyHelp themes the bubbles help bar.
func (s Styles) ApplyHelp(h *help.Model) {
	st := help.DefaultStyles(s.dark)
	st.ShortKey = lipgloss.NewStyle().Foreground(s.main)
	st.ShortDesc = lipgloss.NewStyle().Foreground(s.sub)
	st.ShortSeparator = lipgloss.NewStyle().Foreground(s.sub)
	h.Styles = st
}

// ApplyTextInput themes a text input.
func (s Styles) ApplyTextInput(ti *textinput.Model) {
	tis := textinput.DefaultStyles(s.dark)
	tis.Focused.Text = lipgloss.NewStyle().Foreground(s.text)
	tis.Focused.Prompt = lipgloss.NewStyle().Foreground(s.main)
	tis.Blurred.Text = lipgloss.NewStyle().Foreground(s.sub)
	tis.Blurred.Prompt = lipgloss.NewStyle().Foreground(s.sub)
	ti.SetStyles(tis)
}

// ListDelegate returns a themed list item delegate.
func (s Styles) ListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	st := list.NewDefaultItemStyles(s.dark)
	ld := lipgloss.LightDark(s.dark)
	st.NormalTitle = lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#323437"), lipgloss.Color("#d1d0c5"))).PaddingLeft(1)
	st.NormalDesc = lipgloss.NewStyle().Foreground(s.sub).PaddingLeft(1)
	st.SelectedTitle = lipgloss.NewStyle().Foreground(s.main).Bold(true).BorderLeft(true).BorderForeground(s.main).PaddingLeft(0)
	st.SelectedDesc = lipgloss.NewStyle().Foreground(s.sub).PaddingLeft(1)
	st.DimmedTitle = st.NormalTitle
	st.DimmedDesc = st.NormalDesc
	d.Styles = st
	d.SetHeight(2)
	d.SetSpacing(0)
	d.ShowDescription = true
	return d
}

// ApplyList themes a list component.
func (s Styles) ApplyList(l *list.Model) {
	ls := list.DefaultStyles(s.dark)
	ls.Title = s.Title
	ls.TitleBar = lipgloss.NewStyle().Padding(0, 1)
	ls.DividerDot = lipgloss.NewStyle().Foreground(s.sub)
	ls.StatusBar = lipgloss.NewStyle().Foreground(s.sub)
	ls.NoItems = lipgloss.NewStyle().Foreground(s.sub)
	l.Styles = ls
	l.SetDelegate(s.ListDelegate())
}

// ApplyTable themes a table component.
func (s Styles) ApplyTable(t *table.Model) {
	ts := table.DefaultStyles()
	ld := lipgloss.LightDark(s.dark)
	ts.Header = ts.Header.Foreground(s.main).Bold(true)
	ts.Cell = ts.Cell.Foreground(ld(lipgloss.Color("#323437"), lipgloss.Color("#d1d0c5")))
	ts.Selected = ts.Selected.Foreground(s.main)
	t.SetStyles(ts)
}

// TierProgress returns a themed progress bar for season pass tiers.
func (s Styles) TierProgress(width int) progress.Model {
	p := progress.New(progress.WithColors(s.main, s.main), progress.WithWidth(width))
	p.ShowPercentage = false
	return p
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
