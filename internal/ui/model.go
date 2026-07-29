package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

type phase int

const (
	phaseConfig phase = iota
	phaseTyping
	phaseResult
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Model is the Bubble Tea root model.
type Model struct {
	phase  phase
	cfg    game.Config
	focus  focusField
	sess   *game.Session
	width  int
	height int
	now    time.Time
}

type focusField int

const (
	focusMode focusField = iota
	focusValue
)

// New returns the initial menu model.
func New() Model {
	return Model{
		phase:  phaseConfig,
		cfg:    game.DefaultConfig,
		focus:  focusMode,
		width:  80,
		height: 24,
		now:    time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.phase == phaseTyping && m.sess != nil {
			if m.sess.Tick(m.now) {
				m.phase = phaseResult
			}
		}
		return m, tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	}

	switch m.phase {
	case phaseConfig:
		return m.updateConfig(msg)
	case phaseTyping:
		return m.updateTyping(msg)
	case phaseResult:
		return m.updateResult(msg)
	}
	return m, nil
}

func (m Model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab", "right", "l":
		if m.focus == focusMode {
			m.focus = focusValue
		} else {
			m.focus = focusMode
		}
	case "left", "h":
		if m.focus == focusValue {
			m.focus = focusMode
		} else {
			m.focus = focusValue
		}
	case "up", "k":
		m.nudgeConfig(-1)
	case "down", "j":
		m.nudgeConfig(1)
	case "t":
		m.cfg.Mode = game.ModeTime
		m.focus = focusValue
	case "w":
		m.cfg.Mode = game.ModeWords
		m.focus = focusValue
	case "enter", " ":
		m.startTest()
	}
	return m, nil
}

func (m *Model) nudgeConfig(dir int) {
	if m.focus == focusMode {
		if m.cfg.Mode == game.ModeTime {
			m.cfg.Mode = game.ModeWords
		} else {
			m.cfg.Mode = game.ModeTime
		}
		return
	}

	if m.cfg.Mode == game.ModeTime {
		idx := indexDuration(m.cfg.Duration)
		idx = (idx + dir + len(game.TimeOptions)) % len(game.TimeOptions)
		m.cfg.Duration = game.TimeOptions[idx]
		return
	}
	idx := indexInt(m.cfg.WordCount, game.WordOptions)
	idx = (idx + dir + len(game.WordOptions)) % len(game.WordOptions)
	m.cfg.WordCount = game.WordOptions[idx]
}

func indexDuration(d time.Duration) int {
	for i, v := range game.TimeOptions {
		if v == d {
			return i
		}
	}
	return 1
}

func indexInt(n int, opts []int) int {
	for i, v := range opts {
		if v == n {
			return i
		}
	}
	return 1
}

func (m *Model) startTest() {
	m.sess = game.NewSession(m.cfg)
	m.phase = phaseTyping
	m.now = time.Now()
}

func (m Model) updateTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sess == nil {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.phase = phaseConfig
		m.sess = nil
		return m, nil
	case "tab":
		m.startTest()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	}

	switch msg.Type {
	case tea.KeyBackspace:
		m.sess.HandleBackspace(m.now)
	case tea.KeySpace:
		m.sess.HandleSpace(m.now)
		if m.sess.Finished {
			m.phase = phaseResult
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.sess.HandleRune(r, m.now)
			if m.sess.Finished {
				m.phase = phaseResult
				break
			}
		}
	default:
		// Single printable from KeyMsg.String for some terminals.
		s := msg.String()
		if len(s) == 1 && s != " " {
			m.sess.HandleRune(rune(s[0]), m.now)
			if m.sess.Finished {
				m.phase = phaseResult
			}
		}
	}
	return m, nil
}

func (m Model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab", "enter", " ":
		m.startTest()
	case "esc":
		m.phase = phaseConfig
		m.sess = nil
	}
	return m, nil
}

func (m Model) View() string {
	var body string
	switch m.phase {
	case phaseConfig:
		body = m.viewConfig()
	case phaseTyping:
		body = m.viewTyping()
	case phaseResult:
		body = m.viewResult()
	}

	content := styleBox.Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("mtype"))
	b.WriteString("\n")
	b.WriteString(styleSub.Render("monkeytype for your terminal"))
	b.WriteString("\n\n")

	modeTime := styleOption.Render("time")
	modeWords := styleOption.Render("words")
	if m.cfg.Mode == game.ModeTime {
		modeTime = styleSelected.Render("time")
	} else {
		modeWords = styleSelected.Render("words")
	}
	if m.focus == focusMode {
		b.WriteString(styleMain.Render("mode  "))
	} else {
		b.WriteString(styleSub.Render("mode  "))
	}
	b.WriteString(modeTime)
	b.WriteString(" ")
	b.WriteString(modeWords)
	b.WriteString("\n\n")

	if m.focus == focusValue {
		b.WriteString(styleMain.Render("value "))
	} else {
		b.WriteString(styleSub.Render("value "))
	}

	if m.cfg.Mode == game.ModeTime {
		for i, d := range game.TimeOptions {
			label := fmt.Sprintf("%d", int(d.Seconds()))
			if d == m.cfg.Duration {
				b.WriteString(styleSelected.Render(label))
			} else {
				b.WriteString(styleOption.Render(label))
			}
			if i < len(game.TimeOptions)-1 {
				b.WriteString(" ")
			}
		}
	} else {
		for i, n := range game.WordOptions {
			label := fmt.Sprintf("%d", n)
			if n == m.cfg.WordCount {
				b.WriteString(styleSelected.Render(label))
			} else {
				b.WriteString(styleOption.Render(label))
			}
			if i < len(game.WordOptions)-1 {
				b.WriteString(" ")
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(styleSub.Render("↑↓ change  tab focus  enter start  t/w mode  q quit"))
	return b.String()
}

func (m Model) viewTyping() string {
	if m.sess == nil {
		return ""
	}
	snap := m.sess.Snapshot(m.now)
	var b strings.Builder

	hud := styleMain.Render(m.sess.ProgressLabel(m.now))
	if m.sess.Started {
		hud += "  " + styleStatValue.Render(fmt.Sprintf("%.0f", snap.WPM))
		hud += styleSub.Render(" wpm")
		hud += "  " + styleStatValue.Render(fmt.Sprintf("%.0f%%", snap.Accuracy))
		hud += styleSub.Render(" acc")
	} else {
		hud += "  " + styleSub.Render("start typing…")
	}
	b.WriteString(hud)
	b.WriteString("\n\n")
	b.WriteString(m.renderPrompt())
	b.WriteString("\n\n")
	b.WriteString(styleSub.Render("tab restart  esc menu"))
	return b.String()
}

func (m Model) renderPrompt() string {
	s := m.sess
	cursor := s.CursorPos()
	wrapWidth := m.width - 8
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	if wrapWidth > 70 {
		wrapWidth = 70
	}

	var line strings.Builder
	var out strings.Builder
	col := 0

	for i, ch := range s.Chars {
		glyph := string(ch.R)
		var styled string
		if i == cursor {
			styled = styleCaret.Render(glyph)
		} else {
			switch ch.State {
			case game.CharCorrect:
				styled = styleCorrect.Render(glyph)
			case game.CharIncorrect:
				styled = styleIncorrect.Render(glyph)
			case game.CharExtra:
				styled = styleExtra.Render(glyph)
			default:
				styled = stylePending.Render(glyph)
			}
		}

		// Soft-wrap on spaces when line gets long.
		if ch.R == ' ' && col >= wrapWidth {
			out.WriteString(line.String())
			out.WriteByte('\n')
			line.Reset()
			col = 0
			continue
		}
		line.WriteString(styled)
		col++
	}
	out.WriteString(line.String())

	// Limit visible lines around cursor for readability.
	lines := strings.Split(out.String(), "\n")
	if len(lines) <= 3 {
		return out.String()
	}

	// Find which visual line has caret (approx by scanning raw chars).
	caretLine := 0
	rawCol := 0
	lineIdx := 0
	for i, ch := range s.Chars {
		if ch.R == ' ' && rawCol >= wrapWidth {
			lineIdx++
			rawCol = 0
			if i < cursor {
				caretLine = lineIdx
			}
			continue
		}
		if i == cursor {
			caretLine = lineIdx
			break
		}
		rawCol++
	}

	start := caretLine - 1
	if start < 0 {
		start = 0
	}
	end := start + 3
	if end > len(lines) {
		end = len(lines)
		start = end - 3
		if start < 0 {
			start = 0
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func (m Model) viewResult() string {
	if m.sess == nil {
		return ""
	}
	snap := m.sess.Snapshot(m.now)
	var b strings.Builder
	b.WriteString(styleTitle.Render("result"))
	b.WriteString("\n\n")

	chartW := m.width - 10
	if chartW > 56 {
		chartW = 56
	}
	if chartW < 24 {
		chartW = 24
	}
	b.WriteString(RenderChart(m.sess.History, m.sess.Errors, chartW, 8))
	b.WriteString("\n\n")

	row := func(label, value string) {
		b.WriteString(styleStatLabel.Render(fmt.Sprintf("%-10s", label)))
		b.WriteString(styleStatValue.Render(value))
		b.WriteString("\n")
	}

	row("wpm", fmt.Sprintf("%.0f", snap.WPM))
	row("raw", fmt.Sprintf("%.0f", snap.RawWPM))
	row("acc", fmt.Sprintf("%.0f%%", snap.Accuracy))
	row("time", fmt.Sprintf("%.1fs", snap.Elapsed.Seconds()))
	row("correct", fmt.Sprintf("%d", snap.Correct))
	row("wrong", fmt.Sprintf("%d", snap.Incorrect+snap.Extra))

	mode := m.cfg.Mode.String()
	detail := game.FormatSeconds(m.cfg.Duration) + "s"
	if m.cfg.Mode == game.ModeWords {
		detail = fmt.Sprintf("%d words", m.cfg.WordCount)
	}
	b.WriteString("\n")
	b.WriteString(styleSub.Render(mode + " · " + detail))
	b.WriteString("\n\n")
	b.WriteString(styleSub.Render("tab/enter again  esc menu  q quit"))
	return b.String()
}
