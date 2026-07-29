package ui

import "github.com/charmbracelet/lipgloss"

// Monkeytype-inspired palette (serika dark-ish).
var (
	colBg    = lipgloss.Color("#323437")
	colMain  = lipgloss.Color("#e2b714")
	colText  = lipgloss.Color("#d1d0c5")
	colSub   = lipgloss.Color("#646669")
	colError = lipgloss.Color("#ca4754")
	colExtra = lipgloss.Color("#7e2a33")
	colCaret = lipgloss.Color("#e2b714")
)

var (
	styleTitle = lipgloss.NewStyle().
			Foreground(colMain).
			Bold(true)

	styleSub = lipgloss.NewStyle().
			Foreground(colSub)

	styleText = lipgloss.NewStyle().
			Foreground(colText)

	styleMain = lipgloss.NewStyle().
			Foreground(colMain).
			Bold(true)

	styleCorrect = lipgloss.NewStyle().
			Foreground(colText)

	styleIncorrect = lipgloss.NewStyle().
			Foreground(colError).
			Underline(true)

	styleErrorDot = lipgloss.NewStyle().
			Foreground(colError).
			Bold(true)

	styleExtra = lipgloss.NewStyle().
			Foreground(colExtra).
			Underline(true)

	stylePending = lipgloss.NewStyle().
			Foreground(colSub)

	styleCaret = lipgloss.NewStyle().
			Foreground(colBg).
			Background(colCaret)

	styleSelected = lipgloss.NewStyle().
			Foreground(colBg).
			Background(colMain).
			Padding(0, 1)

	styleOption = lipgloss.NewStyle().
			Foreground(colSub).
			Padding(0, 1)

	styleStatValue = lipgloss.NewStyle().
			Foreground(colMain).
			Bold(true)

	styleStatLabel = lipgloss.NewStyle().
			Foreground(colSub)

	styleBox = lipgloss.NewStyle().
			Padding(1, 2)
)
