package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/invite"
	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/roast"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

type phase int

const (
	phaseConfig phase = iota
	phaseTyping
	phaseResult
	phaseMultiMenu
	phaseJoin
	phaseLobby
	phaseSpectate
	phasePodium
)

type tickMsg time.Time

type roastMsg struct {
	text string
}

// Options configures optional multiplayer / SSH identity.
type Options struct {
	Hub          *multi.Hub
	PlayerName   string
	PlayerID     string
	AutoSpectate bool // ssh demo@… — jump into live/DEMO spectate
}

// Model is the Bubble Tea root model.
type Model struct {
	phase      phase
	cfg        game.Config
	focus      focusField
	sess       *game.Session
	width      int
	height     int
	now        time.Time
	caretOn    bool
	blinkTicks int
	caretX     float64
	caretReady bool
	trail      map[int]int // index → remaining trail life
	lastBlink  time.Time
	lastMulti  time.Time // throttle hub sync over SSH

	hub         *multi.Hub
	playerID    string
	playerName  string
	roomCode    string
	joinInput   string
	statusErr   string
	multiView   multi.View
	raceStarted bool

	themeIdx     int
	sty          Styles // per-session theme (not process-global)
	voice        roast.Voice
	ninjaCaret   bool // smooth caret + fading trail
	paceGhost    PaceGhost // last race to chase
	ghostRec     PaceGhost // recording current race
	ghostOn      bool      // show pace ghost caret
	chatMode     bool
	chatInput    string
	roastText    string
	roastLoading bool

	tipPhase     tipPhase
	tipAmountIdx int
	tipBolt11    string
	tipQR        string
	tipErr       string

	autoSpectate bool
}

type focusField int

const (
	focusMode focusField = iota
	focusValue
)

// New returns the initial menu model (solo).
func New() Model {
	return NewWithOptions(Options{})
}

// NewWithOptions returns a model with optional multiplayer hub.
func NewWithOptions(opts Options) Model {
	id := opts.PlayerID
	if id == "" {
		id = multi.NewPlayerID()
	}
	name := opts.PlayerName
	if name == "" {
		name = "player"
	}
	return Model{
		phase:        phaseConfig,
		cfg:          game.DefaultConfig,
		focus:        focusMode,
		width:        80,
		height:       24,
		now:          time.Now(),
		caretOn:      true,
		hub:          opts.Hub,
		playerID:     id,
		playerName:   name,
		sty:          NewStyles(0),
		ninjaCaret:   false,
		ghostOn:      true,
		autoSpectate: opts.AutoSpectate,
	}
}

func (m Model) Init() tea.Cmd {
	if m.autoSpectate && m.hub != nil {
		return tea.Batch(tickCmd(), m.startAutoSpectate())
	}
	return tickCmd()
}

func (m Model) startAutoSpectate() tea.Cmd {
	return func() tea.Msg {
		return autoSpectateMsg{}
	}
}

type autoSpectateMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if m.multiEnabled() && m.roomCode != "" {
			m.maybeSyncMulti(false)
		}
		var extra tea.Cmd
		if m.phase == phaseTyping && m.sess != nil {
			if m.sess.Started {
				elapsed := m.now.Sub(m.sess.StartedAt)
				m.ghostRec = append(m.ghostRec, GhostSample{At: elapsed, Chars: m.sess.ProgressChars()})
			}
			if m.sess.Tick(m.now) {
				if m.roomCode != "" {
					m.maybeSyncMulti(true)
				} else {
					extra = m.finishSolo()
				}
			} else {
				m.stepCaret()
			}
		}
		return m, tea.Batch(m.nextTickCmd(), extra)

	case roastMsg:
		m.roastLoading = false
		m.roastText = msg.text
		return m, nil

	case tipMsg:
		m.tipLoadingDone(msg)
		return m, nil

	case autoSpectateMsg:
		if m.hub == nil {
			return m, nil
		}
		v, err := m.hub.SpectateLive(m.playerID, m.playerName, m.now)
		if err != nil {
			m.statusErr = err.Error()
			m.phase = phaseMultiMenu
			return m, nil
		}
		m.roomCode = v.Code
		m.multiView = v
		m.statusErr = ""
		m.raceStarted = false
		m.sess = nil
		m.applyMultiView(v)
		return m, nil

	case tea.KeyPressMsg:
		next, cmd := m.handleKey(msg)
		nm, ok := next.(Model)
		if !ok {
			return next, cmd
		}
		if nm.phase == phaseTyping && nm.sess != nil {
			nm.caretOn = true
			nm.blinkTicks = 0
			nm.stepCaret()
		}
		return nm, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.phase {
	case phaseConfig:
		return m.updateConfig(msg)
	case phaseTyping:
		return m.updateTyping(msg)
	case phaseResult:
		return m.updateResult(msg)
	case phaseMultiMenu:
		return m.updateMultiMenu(msg)
	case phaseJoin:
		return m.updateJoin(msg)
	case phaseLobby:
		return m.updateLobby(msg)
	case phaseSpectate:
		return m.updateSpectate(msg)
	case phasePodium:
		return m.updatePodium(msg)
	}
	return m, nil
}

func (m Model) updateConfig(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	case "o":
		m.cfg.Mode = game.ModeQuotes
		m.focus = focusValue
	case "enter", "space":
		m.startTest()
	case "m":
		if m.multiEnabled() {
			m.statusErr = ""
			m.phase = phaseMultiMenu
		}
	case "p":
		m.themeIdx = (m.themeIdx + 1) % ThemeCount()
		m.sty = NewStyles(m.themeIdx)
	case "v":
		if m.voice == roast.VoiceRoast {
			m.voice = roast.VoiceStoic
		} else {
			m.voice = roast.VoiceRoast
		}
	case "n":
		m.ninjaCaret = !m.ninjaCaret
		if !m.ninjaCaret {
			m.trail = nil
			if m.sess != nil {
				m.caretX = float64(m.sess.CursorPos())
			}
			m.caretOn = true
			m.blinkTicks = 0
			m.lastBlink = m.now
		}
	case "y":
		m.cfg.Daily = !m.cfg.Daily
	case "g":
		m.ghostOn = !m.ghostOn
	}
	return m, nil
}

func (m *Model) nudgeConfig(dir int) {
	if m.focus == focusMode {
		modes := []game.Mode{game.ModeTime, game.ModeWords, game.ModeQuotes}
		idx := 0
		for i, mode := range modes {
			if mode == m.cfg.Mode {
				idx = i
				break
			}
		}
		idx = (idx + dir + len(modes)) % len(modes)
		m.cfg.Mode = modes[idx]
		return
	}

	switch m.cfg.Mode {
	case game.ModeTime:
		idx := indexDuration(m.cfg.Duration)
		idx = (idx + dir + len(game.TimeOptions)) % len(game.TimeOptions)
		m.cfg.Duration = game.TimeOptions[idx]
	case game.ModeQuotes:
		idx := indexQuoteLen(m.cfg.QuoteLen)
		idx = (idx + dir + len(game.QuoteLenOptions)) % len(game.QuoteLenOptions)
		m.cfg.QuoteLen = game.QuoteLenOptions[idx]
	default:
		idx := indexInt(m.cfg.WordCount, game.WordOptions)
		idx = (idx + dir + len(game.WordOptions)) % len(game.WordOptions)
		m.cfg.WordCount = game.WordOptions[idx]
	}
}

func indexQuoteLen(n words.QuoteLen) int {
	for i, v := range game.QuoteLenOptions {
		if v == n {
			return i
		}
	}
	return 0
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

func (m *Model) tipLoadingDone(msg tipMsg) {
	if msg.err != "" {
		m.tipPhase = tipShow
		m.tipErr = msg.err
		m.tipBolt11 = ""
		m.tipQR = ""
		return
	}
	m.tipPhase = tipShow
	m.tipErr = ""
	m.tipBolt11 = msg.bolt11
	m.tipQR = msg.qr
	if msg.sats > 0 {
		for i, s := range ln.DefaultAmounts {
			if s == msg.sats {
				m.tipAmountIdx = i
				break
			}
		}
	}
}

func (m *Model) startTest() {
	seed := uint64(0)
	if m.cfg.Daily {
		seed = words.DailySeed(time.Now())
	}
	m.sess = game.NewSessionSeeded(m.cfg, seed)
	m.phase = phaseTyping
	m.now = time.Now()
	m.resetCaret()
	m.ghostRec = nil
	m.roastText = ""
	m.roastLoading = false
	m.clearTip()
}

// finishSolo moves to results and kicks off an async roast.
func (m *Model) finishSolo() tea.Cmd {
	if len(m.ghostRec) > 2 {
		m.paceGhost = m.ghostRec
	}
	m.ghostRec = nil
	m.phase = phaseResult
	m.roastText = ""
	m.roastLoading = true
	m.clearTip()
	return m.roastCmd()
}

func (m Model) roastCmd() tea.Cmd {
	if m.sess == nil {
		return nil
	}
	snap := m.sess.Snapshot(m.now)
	detail := configDetail(m.cfg, m.sess)
	in := roast.Input{
		WPM:      snap.WPM,
		RawWPM:   snap.RawWPM,
		Accuracy: snap.Accuracy,
		Correct:  snap.Correct,
		Wrong:    snap.Incorrect + snap.Extra,
		Elapsed:  snap.Elapsed,
		Mode:     m.cfg.Mode.String(),
		Detail:   detail,
		TypedAny: snap.Correct+snap.Incorrect+snap.Extra > 0,
		Voice:    m.voice,
	}
	return func() tea.Msg {
		text, _ := roast.Generate(context.Background(), in)
		return roastMsg{text: text}
	}
}

func (m Model) updateTyping(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.sess == nil {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		if m.roomCode != "" {
			m.leaveMulti()
			m.phase = phaseMultiMenu
			m.sess = nil
			return m, nil
		}
		m.phase = phaseConfig
		m.sess = nil
		return m, nil
	case "tab":
		if m.roomCode != "" {
			return m, nil // no solo restart mid-race
		}
		m.startTest()
		return m, nil
	case "ctrl+c":
		m.leaveMulti()
		return m, tea.Quit
	case "backspace":
		m.sess.HandleBackspace(m.now)
	case "space":
		m.sess.HandleSpace(m.now)
		if m.sess.Finished {
			if m.roomCode != "" {
				m.syncMulti()
			} else {
				return m, m.finishSolo()
			}
		}
	default:
		text := msg.Text
		if text == "" && len(msg.String()) == 1 {
			text = msg.String()
		}
		for _, r := range text {
			m.sess.HandleRune(r, m.now)
			if m.sess.Finished {
				if m.roomCode != "" {
					m.syncMulti()
				} else {
					return m, m.finishSolo()
				}
				break
			}
		}
	}
	if m.roomCode != "" {
		m.maybeSyncMulti(false)
	}
	return m, nil
}

func (m Model) updateResult(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.tipPhase != tipNone {
		return m.updateTip(msg)
	}
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab", "enter":
		m.startTest()
	case "t":
		if ln.Configured() {
			m.tipPhase = tipPick
			m.tipErr = ""
			m.tipBolt11 = ""
			m.tipQR = ""
		}
	case "esc":
		m.phase = phaseConfig
		m.sess = nil
		m.clearTip()
	}
	return m, nil
}

func (m Model) View() tea.View {
	var body string
	switch m.phase {
	case phaseConfig:
		body = m.viewConfig()
	case phaseTyping:
		body = m.viewTyping()
	case phaseResult:
		if m.tipPhase != tipNone {
			body = m.viewTip()
		} else {
			body = m.viewResult()
		}
	case phaseMultiMenu:
		body = m.viewMultiMenu()
	case phaseJoin:
		body = m.viewJoin()
	case phaseLobby:
		body = m.viewLobby()
	case phaseSpectate:
		body = m.viewSpectate()
	case phasePodium:
		body = m.viewPodium()
	}

	content := m.sty.Box.Render(body)
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	v := tea.NewView(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content))
	v.AltScreen = true
	return v
}

func configDetail(cfg game.Config, sess *game.Session) string {
	switch cfg.Mode {
	case game.ModeWords:
		n := cfg.WordCount
		if sess != nil && sess.Config.WordCount > 0 {
			n = sess.Config.WordCount
		}
		return fmt.Sprintf("%d words", n)
	case game.ModeQuotes:
		return cfg.QuoteLen.String()
	default:
		return game.FormatSeconds(cfg.Duration) + "s"
	}
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("gotype"))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("typing races in your terminal"))
	b.WriteString("\n\n")

	modeTime := m.sty.Option.Render("time")
	modeWords := m.sty.Option.Render("words")
	modeQuote := m.sty.Option.Render("quote")
	switch m.cfg.Mode {
	case game.ModeTime:
		modeTime = m.sty.Selected.Render("time")
	case game.ModeWords:
		modeWords = m.sty.Selected.Render("words")
	case game.ModeQuotes:
		modeQuote = m.sty.Selected.Render("quote")
	}
	if m.focus == focusMode {
		b.WriteString(m.sty.Main.Render("mode  "))
	} else {
		b.WriteString(m.sty.Sub.Render("mode  "))
	}
	b.WriteString(modeTime)
	b.WriteString(" ")
	b.WriteString(modeWords)
	b.WriteString(" ")
	b.WriteString(modeQuote)
	b.WriteString("\n\n")

	if m.focus == focusValue {
		b.WriteString(m.sty.Main.Render("value "))
	} else {
		b.WriteString(m.sty.Sub.Render("value "))
	}

	switch m.cfg.Mode {
	case game.ModeTime:
		for i, d := range game.TimeOptions {
			label := fmt.Sprintf("%d", int(d.Seconds()))
			if d == m.cfg.Duration {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.TimeOptions)-1 {
				b.WriteString(" ")
			}
		}
	case game.ModeQuotes:
		for i, qlen := range game.QuoteLenOptions {
			label := qlen.String()
			if qlen == m.cfg.QuoteLen {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.QuoteLenOptions)-1 {
				b.WriteString(" ")
			}
		}
	default:
		for i, n := range game.WordOptions {
			label := fmt.Sprintf("%d", n)
			if n == m.cfg.WordCount {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.WordOptions)-1 {
				b.WriteString(" ")
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(m.sty.Sub.Render("theme "))
	b.WriteString(m.sty.Main.Render(ThemeName(m.themeIdx)))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("voice "))
	b.WriteString(m.sty.Main.Render(m.voice.String()))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("ninja "))
	if m.ninjaCaret {
		b.WriteString(m.sty.Main.Render("on"))
	} else {
		b.WriteString(m.sty.Main.Render("off"))
	}
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("daily "))
	if m.cfg.Daily {
		b.WriteString(m.sty.Main.Render(words.DailyLabel(m.now)))
	} else {
		b.WriteString(m.sty.Main.Render("off"))
	}
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("ghost "))
	if m.ghostOn {
		b.WriteString(m.sty.Main.Render("on"))
	} else {
		b.WriteString(m.sty.Main.Render("off"))
	}
	b.WriteString("\n\n")
	if m.multiEnabled() {
		b.WriteString(m.sty.Sub.Render("↑↓ change  tab focus  enter start  m multi  p theme  v voice  n ninja  y daily  g ghost  t/w/o mode  q quit"))
	} else {
		b.WriteString(m.sty.Sub.Render("↑↓ change  tab focus  enter start  p theme  v voice  n ninja  y daily  g ghost  t/w/o mode  q quit"))
	}
	return b.String()
}

func (m Model) viewTyping() string {
	if m.sess == nil {
		return ""
	}
	snap := m.sess.Snapshot(m.now)
	var b strings.Builder

	hud := m.sty.Main.Render(m.sess.ProgressLabel(m.now))
	if m.roomCode != "" && m.multiView.Phase == multi.PhaseRacing {
		hud = m.sty.Main.Render(game.FormatSeconds(m.multiView.RaceRemaining))
	}
	if m.sess.Started {
		hud += "  " + m.sty.StatValue.Render(fmt.Sprintf("%.0f", snap.WPM))
		hud += m.sty.Sub.Render(" wpm")
		hud += "  " + m.sty.StatValue.Render(fmt.Sprintf("%.0f%%", snap.Accuracy))
		hud += m.sty.Sub.Render(" acc")
		if m.ghostOn && len(m.paceGhost) > 0 {
			hud += "  " + m.sty.Sub.Render("ghost")
		}
	} else {
		hud += "  " + m.sty.Sub.Render("start typing…")
	}
	b.WriteString(hud)
	b.WriteString("\n\n")
	b.WriteString(m.renderPrompt())
	if m.sess.QuoteAuthor != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render("— " + m.sess.QuoteAuthor))
	}
	if m.roomCode != "" {
		b.WriteString(m.viewRaceOpponents())
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render("esc leave race"))
	} else {
		b.WriteString("\n\n")
		b.WriteString(m.sty.Sub.Render("tab restart  esc menu"))
	}
	return b.String()
}

func (m Model) renderPrompt() string {
	s := m.sess
	cursor := s.CursorPos()
	visual := m.caretVisualIndex()
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
		base := m.sty.Pending
		switch ch.State {
		case game.CharCorrect:
			base = m.sty.Correct
		case game.CharIncorrect:
			base = m.sty.Incorrect
		case game.CharExtra:
			base = m.sty.Extra
		}

		var styled string
		showBlock := i == visual && m.caretOn
		ghostIdx := -1
		if m.ghostOn && len(m.paceGhost) > 0 && m.sess.Started {
			elapsed := m.now.Sub(m.sess.StartedAt)
			ghostIdx = indexForProgressChars(s.Chars, m.paceGhost.CharsAt(elapsed))
		}
		if showBlock {
			styled = m.sty.Caret.Render(glyph)
		} else if i == ghostIdx {
			styled = m.sty.Ghost.Render(glyph)
		} else if m.ninjaCaret {
			if life, ok := m.trail[i]; ok && life > 0 {
				styled = m.sty.WithTrail(base, life).Render(glyph)
			} else {
				styled = base.Render(glyph)
			}
		} else {
			styled = base.Render(glyph)
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
	// Block caret past last character.
	if visual >= len(s.Chars) && m.caretOn {
		line.WriteString(m.sty.Caret.Render(" "))
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
	focus := visual
	if focus < 0 {
		focus = cursor
	}
	for i, ch := range s.Chars {
		if ch.R == ' ' && rawCol >= wrapWidth {
			lineIdx++
			rawCol = 0
			if i < focus {
				caretLine = lineIdx
			}
			continue
		}
		if i == focus {
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
	b.WriteString(m.sty.Title.Render("result"))
	b.WriteString("\n\n")

	chartW := m.width - 10
	if chartW > 64 {
		chartW = 64
	}
	if chartW < 28 {
		chartW = 28
	}

	passW := chartW
	maxPassLines := 6
	if m.cfg.Mode == game.ModeQuotes {
		maxPassLines = 10
	}
	passage := m.renderResultPassage(passW, maxPassLines)
	if passage != "" {
		b.WriteString(passage)
		if m.sess.QuoteAuthor != "" {
			b.WriteString("\n")
			b.WriteString(m.sty.Sub.Render("— " + m.sess.QuoteAuthor))
		}
		b.WriteString("\n\n")
	}

	b.WriteString(RenderChart(m.sty, m.sess.History, m.sess.Errors, chartW, 10))
	b.WriteString("\n\n")

	row := func(label, value string) {
		b.WriteString(m.sty.StatLabel.Render(fmt.Sprintf("%-10s", label)))
		b.WriteString(m.sty.StatValue.Render(value))
		b.WriteString("\n")
	}

	row("wpm", fmt.Sprintf("%.0f", snap.WPM))
	row("raw", fmt.Sprintf("%.0f", snap.RawWPM))
	if snap.Correct+snap.Incorrect+snap.Extra == 0 {
		row("acc", "—")
	} else {
		row("acc", fmt.Sprintf("%.0f%%", snap.Accuracy))
	}
	row("time", fmt.Sprintf("%.1fs", snap.Elapsed.Seconds()))
	row("correct", fmt.Sprintf("%d", snap.Correct))
	row("wrong", fmt.Sprintf("%d", snap.Incorrect+snap.Extra))

	mode := m.cfg.Mode.String()
	detail := configDetail(m.cfg, m.sess)
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render(mode + " · " + detail))
	if m.cfg.Daily {
		b.WriteString("\n")
		b.WriteString(m.sty.Main.Render(words.DailyHeadline(m.now)))
	}
	b.WriteString("\n\n")
	b.WriteString(m.sty.Main.Render(m.voice.String()))
	b.WriteString("\n")
	switch {
	case m.roastLoading && m.roastText == "":
		if m.voice == roast.VoiceStoic {
			b.WriteString(m.sty.Sub.Render("consulting the porch…"))
		} else {
			b.WriteString(m.sty.Sub.Render("sharpening insults…"))
		}
	case m.roastText != "":
		b.WriteString(m.sty.Text.Render(lipgloss.NewStyle().Width(chartW).Render(m.roastText)))
	}
	b.WriteString("\n\n")
	b.WriteString(m.sty.Sub.Render(invite.RaceMe(snap.WPM)))
	b.WriteString("\n\n")
	if ln.Configured() {
		b.WriteString(m.sty.Sub.Render("tab/enter again  t tip  esc menu  q quit"))
	} else {
		b.WriteString(m.sty.Sub.Render("tab/enter again  esc menu  q quit"))
	}
	return b.String()
}
