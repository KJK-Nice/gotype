package ui

import "github.com/charmbracelet/lipgloss"

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

var (
	colBg    lipgloss.Color
	colMain  lipgloss.Color
	colText  lipgloss.Color
	colSub   lipgloss.Color
	colError lipgloss.Color
	colExtra lipgloss.Color
	colCaret lipgloss.Color
	colGhost lipgloss.Color

	styleTitle     lipgloss.Style
	styleSub       lipgloss.Style
	styleText      lipgloss.Style
	styleMain      lipgloss.Style
	styleCorrect   lipgloss.Style
	styleIncorrect lipgloss.Style
	styleErrorDot  lipgloss.Style
	styleExtra     lipgloss.Style
	stylePending   lipgloss.Style
	styleCaret     lipgloss.Style
	styleGhost     lipgloss.Style
	styleSelected  lipgloss.Style
	styleOption    lipgloss.Style
	styleStatValue lipgloss.Style
	styleStatLabel lipgloss.Style
	styleBox       lipgloss.Style
)

func init() {
	ApplyTheme(0)
}

func ThemeName(i int) string {
	if i < 0 || i >= len(themes) {
		return themes[0].Name
	}
	return themes[i].Name
}

// ApplyTheme rebuilds lipgloss styles for theme index.
func ApplyTheme(idx int) {
	if idx < 0 {
		idx = 0
	}
	idx %= len(themes)
	t := themes[idx]

	colBg = lipgloss.Color(t.Bg)
	colMain = lipgloss.Color(t.Main)
	colText = lipgloss.Color(t.Text)
	colSub = lipgloss.Color(t.Sub)
	colError = lipgloss.Color(t.Error)
	colExtra = lipgloss.Color(t.Extra)
	colCaret = lipgloss.Color(t.Main)
	colGhost = lipgloss.Color(t.Ghost)

	styleTitle = lipgloss.NewStyle().Foreground(colMain).Bold(true)
	styleSub = lipgloss.NewStyle().Foreground(colSub)
	styleText = lipgloss.NewStyle().Foreground(colText)
	styleMain = lipgloss.NewStyle().Foreground(colMain).Bold(true)
	styleCorrect = lipgloss.NewStyle().Foreground(colText)
	styleIncorrect = lipgloss.NewStyle().Foreground(colError).Underline(true)
	styleErrorDot = lipgloss.NewStyle().Foreground(colError).Bold(true)
	styleExtra = lipgloss.NewStyle().Foreground(colExtra).Underline(true)
	stylePending = lipgloss.NewStyle().Foreground(colSub)
	styleCaret = lipgloss.NewStyle().Foreground(colBg).Background(colCaret)
	styleGhost = lipgloss.NewStyle().Foreground(colSub).Background(colGhost)
	styleSelected = lipgloss.NewStyle().Foreground(colBg).Background(colMain).Padding(0, 1)
	styleOption = lipgloss.NewStyle().Foreground(colSub).Padding(0, 1)
	styleStatValue = lipgloss.NewStyle().Foreground(colMain).Bold(true)
	styleStatLabel = lipgloss.NewStyle().Foreground(colSub)
	styleBox = lipgloss.NewStyle().Padding(1, 2)
}

func trailBackground(life int) lipgloss.Color {
	switch {
	case life >= 5:
		return colMain
	case life >= 3:
		return colGhost
	default:
		return colSub
	}
}

func styleWithTrail(base lipgloss.Style, life int) lipgloss.Style {
	return base.Background(trailBackground(life))
}
