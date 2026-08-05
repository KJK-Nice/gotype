package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/timer"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/invite"
	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
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
	App          *App
	SessionID    string
	RemoteIP     string
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
	caretVel   float64 // harmonica spring velocity (ninja caret)
	caretReady bool
	trail      map[int]int // index → remaining trail life
	lastBlink  time.Time
	lastMulti  time.Time // throttle hub sync over SSH

	shake    spring1D                   // error screen shake (cells)
	raceBars map[string]progress.Model // playerID → animated race bar


	hub         *multi.Hub
	playerID    string
	playerName  string
	roomCode    string
	statusErr   string
	multiView   multi.View
	raceStarted bool

	// bubbles widgets
	joinTI      textinput.Model
	chatTI      textinput.Model
	spin        spinner.Model
	help        help.Model
	cdTimer     timer.Model
	cdOn        bool
	cdDigit     int      // last cinematic digit (3/2/1/0=GO); -1 idle
	cdPulse     spring1D // pop intensity on digit change
	stopwatch   stopwatch.Model
	tipList     list.Model
	podiumTable table.Model
	chatVP      viewport.Model
	pendingCmd  tea.Cmd

	themeIdx     int
	sty          Styles // per-session theme (not process-global)
	voice        roast.Voice
	ninjaCaret   bool // smooth caret + fading trail
	paceGhost    PaceGhost // last race to chase
	ghostRec     PaceGhost // recording current race
	ghostOn      bool      // show pace ghost caret
	chatMode     bool
	roastText    string
	roastLoading bool

	tipPhase     tipPhase
	tipAmountIdx int
	tipBolt11    string
	tipQR        string
	tipErr       string

	autoSpectate bool

	app       *App
	sessionID string
	remoteIP  string
	claimedID string

	claimMode   claimMode
	claimNameTI textinput.Model
	claimCodeTI textinput.Model
	claimErr    string
	claimShown  string

	prog       progSurface
	shopList      list.Model
	invList       list.Model
	equipList     list.Model
	multiMenuList list.Model
	claimList     list.Model
	buyOrder      persist.Order
	buyQR      string
	buyErr     string
	lastXPLine string
	multiXPRace int

	// consumables (in-race)
	usedConsumable map[string]bool
	useStripOpen   bool
	calmArmed      bool
	useStatus      string
	useStatusUntil time.Time
	raceSeed       uint64
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
	sessID := opts.SessionID
	if sessID == "" {
		sessID = id
	}
	ip := opts.RemoteIP
	if ip == "" {
		ip = "local"
	}
	m := Model{
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
		app:          opts.App,
		sessionID:    sessID,
		remoteIP:     ip,
	}
	m.initBubbles()
	m.initClaimInputs()
	m.applyBubblesTheme()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.autoSpectate && m.hub != nil {
		return tea.Batch(tickCmd(), m.startAutoSpectate())
	}
	return tickCmd()
}

func (m *Model) queueCmd(c tea.Cmd) {
	if c == nil {
		return
	}
	if m.pendingCmd == nil {
		m.pendingCmd = c
		return
	}
	m.pendingCmd = tea.Batch(m.pendingCmd, c)
}

func (m *Model) takePending() tea.Cmd {
	c := m.pendingCmd
	m.pendingCmd = nil
	return c
}

func (m Model) startAutoSpectate() tea.Cmd {
	return func() tea.Msg {
		return autoSpectateMsg{}
	}
}

type autoSpectateMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var c tea.Cmd

	// Keep running widgets alive regardless of phase.
	m.spin, c = m.spin.Update(msg)
	if c != nil {
		cmds = append(cmds, c)
	}
	if m.cdOn {
		m.cdTimer, c = m.cdTimer.Update(msg)
		if c != nil {
			cmds = append(cmds, c)
		}
	}
	m.stopwatch, c = m.stopwatch.Update(msg)
	if c != nil {
		cmds = append(cmds, c)
	}
	if m.raceBars != nil {
		for id, bar := range m.raceBars {
			var bc tea.Cmd
			bar, bc = bar.Update(msg)
			m.raceBars[id] = bar
			if bc != nil {
				cmds = append(cmds, bc)
			}
		}
	}

	switch msg := msg.(type) {
	case progress.FrameMsg:
		// Already applied above; just flush cmds.
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		m.help.SetWidth(max(20, m.width-4))
		m.chatVP.SetWidth(min(48, max(20, m.width-8)))
		m.podiumTable.SetWidth(min(48, max(24, m.width-8)))
		m.syncListSizes()
		return m, tea.Batch(cmds...)

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
		m.stepShake()
		m.stepCountdownCinematic()
		m.syncRaceBars()
		cmds = append(cmds, m.nextTickCmd(), extra, m.takePending())
		return m, tea.Batch(cmds...)

	case roastMsg:
		m.roastLoading = false
		m.roastText = msg.text
		return m, tea.Batch(cmds...)

	case tipMsg:
		m.tipLoadingDone(msg)
		return m, tea.Batch(cmds...)

	case claimMsg:
		m.applyClaimMsg(msg)
		return m, tea.Batch(cmds...)

	case buyMsg:
		if c := m.applyBuyMsg(msg); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)

	case buyPollMsg:
		if c := m.applyBuyPoll(msg); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)

	case autoSpectateMsg:
		if m.hub == nil {
			return m, tea.Batch(cmds...)
		}
		v, err := m.hub.SpectateLive(m.playerID, m.playerName, m.now)
		if err != nil {
			m.statusErr = err.Error()
			m.phase = phaseMultiMenu
			return m, tea.Batch(cmds...)
		}
		m.roomCode = v.Code
		m.multiView = v
		m.statusErr = ""
		m.raceStarted = false
		m.sess = nil
		m.applyMultiView(v)
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		next, cmd := m.handleKey(msg)
		nm, ok := next.(Model)
		if !ok {
			return next, tea.Batch(append(cmds, cmd)...)
		}
		if nm.phase == phaseTyping && nm.sess != nil {
			nm.caretOn = true
			nm.blinkTicks = 0
			nm.stepCaret()
		}
		return nm, tea.Batch(append(cmds, cmd, nm.takePending())...)
	}
	return m, tea.Batch(append(cmds, m.takePending())...)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.claimMode != claimIdle {
		return m.updateClaim(msg)
	}
	if m.prog != progNone {
		if nm, cmd, ok := m.tryProgHotkey(msg); ok {
			return nm, cmd
		}
		return m.updateProg(msg)
	}
	if nm, cmd, ok := m.tryProgHotkey(msg); ok {
		return nm, cmd
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
		return m, m.startTest()
	case "m":
		if m.multiEnabled() {
			m.statusErr = ""
			m.phase = phaseMultiMenu
		}
	case "u":
		m.themeIdx = (m.themeIdx + 1) % ThemeCount()
		m.sty = NewStyles(m.themeIdx)
		m.applyBubblesTheme()
		m.raceBars = nil // rebuild with new theme colors
	case "c":
		m.openClaim()
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
			m.caretVel = 0
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

// startTest begins a solo session.
func (m *Model) startTest() tea.Cmd {
	seed := uint64(0)
	if m.cfg.Daily {
		seed = words.DailySeed(time.Now())
	}
	m.raceSeed = seed
	m.resetConsumableRace()
	m.sess = game.NewSessionSeeded(m.cfg, seed)
	m.phase = phaseTyping
	m.now = time.Now()
	m.resetCaret()
	m.shake = spring1D{}
	m.ghostRec = nil
	m.roastText = ""
	m.roastLoading = false
	m.clearTip()
	return m.startRaceStopwatch()
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
	m.grantSoloXP()
	stop := m.stopRaceStopwatch()
	return tea.Batch(stop, m.roastCmd(), m.spin.Tick)
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
	if handled, cmd := m.handleConsumableKey(msg); handled {
		return m, cmd
	}

	errN := len(m.sess.Errors)
	finished := false
	var finishCmd tea.Cmd
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
		return m, m.startTest()
	case "ctrl+c":
		m.leaveMulti()
		return m, tea.Quit
	case "backspace":
		m.sess.HandleBackspace(m.now)
	case "space":
		m.sess.HandleSpace(m.now)
		if m.sess.Finished {
			finished = true
		}
	default:
		text := msg.Text
		if text == "" && len(msg.String()) == 1 {
			text = msg.String()
		}
		for _, r := range text {
			m.sess.HandleRune(r, m.now)
			if m.sess.Finished {
				finished = true
				break
			}
		}
	}
	if len(m.sess.Errors) > errN {
		m.triggerShake()
	}
	if finished {
		if m.roomCode != "" {
			m.syncMulti()
		} else {
			finishCmd = m.finishSolo()
		}
	} else if m.roomCode != "" {
		m.maybeSyncMulti(false)
	}
	return m, finishCmd
}

func (m Model) updateResult(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.tipPhase != tipNone {
		return m.updateTip(msg)
	}
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "tab", "enter":
		return m, m.startTest()
	case "t":
		if ln.Configured() {
			m.tipPhase = tipPick
			m.tipErr = ""
			m.tipBolt11 = ""
			m.tipQR = ""
			m.tipList = newTipList()
			m.tipList.Select(m.tipAmountIdx)
		}
		return m, nil
	case "esc":
		m.phase = phaseConfig
		m.sess = nil
		m.clearTip()
	}
	return m, nil
}

func (m Model) View() tea.View {
	var body string
	switch {
	case m.claimMode != claimIdle:
		body = m.viewClaim()
	case m.prog == progBuyWait:
		body = m.viewBuyWait()
	case m.prog != progNone:
		body = m.viewProg()
	case m.phase == phaseConfig:
		body = m.viewConfig()
	case m.phase == phaseTyping:
		body = m.viewTyping()
	case m.phase == phaseResult:
		if m.tipPhase != tipNone {
			body = m.viewTip()
		} else {
			body = m.viewResult()
		}
	case m.phase == phaseMultiMenu:
		body = m.viewMultiMenu()
	case m.phase == phaseJoin:
		body = m.viewJoin()
	case m.phase == phaseLobby:
		body = m.viewLobby()
	case m.phase == phaseSpectate:
		body = m.viewSpectate()
	case m.phase == phasePodium:
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
	placed := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
	dx := int(math.Round(m.shake.x))
	if dx != 0 {
		placed = applyShake(placed, dx)
	}
	v := tea.NewView(placed)
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
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("player "))
	if m.isClaimed() {
		b.WriteString(m.sty.Main.Render(m.playerName))
	} else {
		b.WriteString(m.sty.Sub.Render("guest · c claim"))
	}
	if m.statusErr != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Incorrect.Render(m.statusErr))
	}
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp(m.helpConfig()))
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
		if m.stopwatch.Running() {
			hud += "  " + m.sty.Sub.Render(m.stopwatch.View())
		}
		if m.ghostOn && len(m.paceGhost) > 0 {
			hud += "  " + m.sty.Sub.Render("ghost")
		}
		if m.sess.Config.ThreeStrike {
			hud += "  " + m.sty.Main.Render(heartHUD(m.sess.HP, m.sess.MaxHP))
			if m.sess.DNF {
				hud += "  " + m.sty.Incorrect.Render("DNF")
			}
		}
	} else {
		hud += "  " + m.sty.Sub.Render("start typing…")
		if m.sess.Config.ThreeStrike {
			hud += "  " + m.sty.Sub.Render(heartHUD(m.sess.HP, m.sess.MaxHP))
		}
	}
	b.WriteString(hud)
	b.WriteString("\n\n")
	b.WriteString(m.renderPrompt())
	if m.sess.QuoteAuthor != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render("— " + m.sess.QuoteAuthor))
	}
	if strip := m.viewUseStrip(); strip != "" {
		b.WriteString("\n")
		b.WriteString(strip)
	}
	if m.roomCode != "" {
		b.WriteString(m.viewRaceOpponents())
		b.WriteString("\n")
		b.WriteString(m.renderHelp(m.typingHelp()))
	} else {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(m.typingHelp()))
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

		if ch.State == game.CharPending && m.sess.IsRevealPeek(i) {
			base = m.sty.Sub
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

func heartHUD(hp, max int) string {
	if max <= 0 {
		max = game.ThreeStrikeStartHP
	}
	var b strings.Builder
	for i := 0; i < max; i++ {
		if i < hp {
			b.WriteString("❤")
		} else {
			b.WriteString("♡")
		}
	}
	return b.String()
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
	if m.lastXPLine != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Main.Render(m.lastXPLine))
	}
	if m.sess != nil && m.sess.DNF {
		b.WriteString("\n")
		b.WriteString(m.sty.Incorrect.Render("DNF · hardcore"))
	}
	b.WriteString("\n\n")
	b.WriteString(m.sty.Main.Render(m.voice.String()))
	b.WriteString("\n")
	switch {
	case m.roastLoading && m.roastText == "":
		msg := "sharpening insults…"
		if m.voice == roast.VoiceStoic {
			msg = "consulting the porch…"
		}
		b.WriteString(m.sty.Sub.Render(m.spin.View() + " " + msg))
	case m.roastText != "":
		b.WriteString(m.sty.Text.Render(lipgloss.NewStyle().Width(chartW).Render(m.roastText)))
	}
	b.WriteString("\n\n")
	b.WriteString(m.sty.Sub.Render(invite.RaceMe(snap.WPM)))
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp(helpResult(ln.Configured())))
	return b.String()
}
