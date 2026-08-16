package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/timer"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/invite"
	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/quoteai"
	"github.com/kjkusap/monkeytype-clone/internal/roast"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

type phase int

const (
	phaseIntro phase = iota
	phaseConfig
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

type aiQuoteMsg struct {
	text   string
	author string
	err    string
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

	shake    spring1D                  // error screen shake (cells)
	raceBars map[string]progress.Model // playerID → animated race bar

	// Login intro (ASCII rain → assemble "gotype" → home).
	introRain *introRain
	introAt   time.Time
	introLast time.Time
	introSeed int64

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
	ninjaCaret   bool      // smooth caret + fading trail
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

	loginMode     loginMode
	loginNameTI   textinput.Model
	loginErr      string
	loginWalletK1 string
	loginWalletQR string

	prog          progSurface
	shopList      list.Model
	invList       list.Model
	equipList     list.Model
	multiMenuList list.Model
	buyOrder      persist.Order
	buyQR         string
	buyErr        string
	lastXPLine    string
	multiXPRace   int

	// consumables (in-race)
	usedConsumable map[string]bool
	useStripOpen   bool
	calmArmed      bool
	useStatus      string
	useStatusUntil time.Time
	raceSeed       uint64

	// Ignore result / progress hotkeys until this time (post-finish buffer).
	resultKeysUntil time.Time

	aiGenerating bool // ModeAI: waiting on LLM before typing
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
		phase:        phaseIntro,
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
		ghostOn:      false,
		autoSpectate: opts.AutoSpectate,
		app:          opts.App,
		sessionID:    sessID,
		remoteIP:     ip,
	}
	if opts.AutoSpectate {
		m.phase = phaseConfig // demo path skips brand intro
	} else {
		m.beginIntro()
	}
	m.initBubbles()
	m.initLoginInputs()
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
		m.rebuildIntroRain()
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
		m.stepIntro()
		m.stepShake()
		m.stepCountdownCinematic()
		m.syncRaceBars()
		cmds = append(cmds, m.nextTickCmd(), extra, m.takePending())
		return m, tea.Batch(cmds...)

	case roastMsg:
		m.roastLoading = false
		m.roastText = msg.text
		return m, tea.Batch(cmds...)

	case aiQuoteMsg:
		m.aiGenerating = false
		if msg.err != "" {
			m.phase = phaseConfig
			m.sess = nil
			m.statusErr = msg.err
			return m, tea.Batch(cmds...)
		}
		m.statusErr = ""
		m.sess = game.NewSessionFromPassage(m.cfg, msg.text, msg.author)
		m.resetCaret()
		m.shake = spring1D{}
		cmds = append(cmds, m.startRaceStopwatch())
		return m, tea.Batch(cmds...)

	case tipMsg:
		m.tipLoadingDone(msg)
		return m, tea.Batch(cmds...)

	case loginMsg:
		m.applyLoginMsg(msg)
		return m, tea.Batch(cmds...)

	case walletStartMsg:
		if c := m.applyWalletStart(msg); c != nil {
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)

	case walletPollMsg:
		if c := m.applyWalletPoll(msg); c != nil {
			cmds = append(cmds, c)
		}
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
	if m.phase == phaseIntro {
		return m.updateIntro(msg)
	}
	if m.loginMode != loginIdle {
		return m.updateLogin(msg)
	}
	if m.chatMode {
		return m.updateChat(msg)
	}
	// Post-finish buffer: eat keys (incl. i/s/p/e) so last keystrokes don't navigate away.
	if m.resultKeysLocked() {
		return m, nil
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
	case "tab":
		m.toggleConfigFocus()
	case "up", "k":
		m.toggleConfigFocus()
	case "down", "j":
		m.toggleConfigFocus()
	case "left", "h":
		m.nudgeConfig(-1)
	case "right":
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
	case "a":
		if quoteai.Configured() {
			m.cfg.Mode = game.ModeAI
			m.focus = focusValue
		}
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
	case "l":
		return m, m.openLogin()
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

func (m *Model) toggleConfigFocus() {
	if m.focus == focusMode {
		m.focus = focusValue
	} else {
		m.focus = focusMode
	}
}

func (m *Model) nudgeConfig(dir int) {
	if m.focus == focusMode {
		modes := m.availableModes()
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
	case game.ModeQuotes, game.ModeAI:
		idx := indexQuoteLen(m.cfg.QuoteLen)
		idx = (idx + dir + len(game.QuoteLenOptions)) % len(game.QuoteLenOptions)
		m.cfg.QuoteLen = game.QuoteLenOptions[idx]
	default:
		idx := indexInt(m.cfg.WordCount, game.WordOptions)
		idx = (idx + dir + len(game.WordOptions)) % len(game.WordOptions)
		m.cfg.WordCount = game.WordOptions[idx]
	}
}

func (m Model) availableModes() []game.Mode {
	modes := []game.Mode{game.ModeTime, game.ModeWords, game.ModeQuotes}
	if quoteai.Configured() {
		modes = append(modes, game.ModeAI)
	}
	return modes
}

func (m *Model) ensureModeAvailable() {
	if m.cfg.Mode == game.ModeAI && !quoteai.Configured() {
		m.cfg.Mode = game.ModeQuotes
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
	m.ensureModeAvailable()
	seed := uint64(0)
	if m.cfg.Daily {
		seed = words.DailySeed(time.Now())
	}
	m.raceSeed = seed
	m.resetConsumableRace()
	m.phase = phaseTyping
	m.now = time.Now()
	m.shake = spring1D{}
	m.ghostRec = nil
	m.roastText = ""
	m.roastLoading = false
	m.clearTip()
	m.statusErr = ""

	if m.cfg.Mode == game.ModeAI {
		m.aiGenerating = true
		m.sess = nil
		m.resetCaret()
		return tea.Batch(m.spin.Tick, m.aiQuoteCmd())
	}

	m.aiGenerating = false
	m.sess = game.NewSessionSeeded(m.cfg, seed)
	m.resetCaret()
	return m.startRaceStopwatch()
}

func (m Model) aiQuoteCmd() tea.Cmd {
	qlen := m.cfg.QuoteLen
	return func() tea.Msg {
		p, err := quoteai.Generate(context.Background(), qlen)
		if err != nil {
			return aiQuoteMsg{err: "ai quote failed: " + err.Error()}
		}
		return aiQuoteMsg{text: p.Text, author: p.Author}
	}
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
	m.armResultKeyLock()
	m.grantSoloXP()
	stop := m.stopRaceStopwatch()
	return tea.Batch(stop, m.roastCmd(), m.spin.Tick)
}

const resultKeyLock = 3 * time.Second

func (m Model) resultKeysLocked() bool {
	if m.resultKeysUntil.IsZero() || !m.now.Before(m.resultKeysUntil) {
		return false
	}
	return m.phase == phaseResult || m.phase == phasePodium
}

func (m *Model) armResultKeyLock() {
	m.resultKeysUntil = m.now.Add(resultKeyLock)
}

func (m Model) resultLockLeft() time.Duration {
	if !m.resultKeysLocked() {
		return 0
	}
	d := m.resultKeysUntil.Sub(m.now)
	if d <= 0 {
		return 0
	}
	sec := int((d + time.Second - 1) / time.Second) // ceil
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
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
	if m.aiGenerating {
		switch msg.String() {
		case "esc":
			m.aiGenerating = false
			m.phase = phaseConfig
			m.sess = nil
			m.statusErr = ""
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}
	}
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
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}

	// Full-bleed login rain — no box chrome.
	if m.phase == phaseIntro && m.loginMode == loginIdle && m.prog == progNone {
		v := tea.NewView(m.viewIntro())
		v.AltScreen = true
		return v
	}

	var body string
	switch {
	case m.loginMode != loginIdle:
		body = m.viewLogin()
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
	case game.ModeQuotes, game.ModeAI:
		return cfg.QuoteLen.String()
	default:
		return game.FormatSeconds(cfg.Duration) + "s"
	}
}

func (m Model) viewTyping() string {
	if m.aiGenerating {
		var b strings.Builder
		b.WriteString(m.sty.Title.Render("ai quote"))
		b.WriteString("\n\n")
		b.WriteString(m.sty.Sub.Render(m.spin.View() + " generating…"))
		b.WriteString("\n\n")
		b.WriteString(m.sty.Sub.Render("esc cancel"))
		return b.String()
	}
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
	if m.cfg.Mode == game.ModeQuotes || m.cfg.Mode == game.ModeAI {
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
	if left := m.resultLockLeft(); left > 0 {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("keys in %ds…", int(left.Seconds()))))
	} else {
		b.WriteString(m.renderHelp(helpResult(ln.Configured())))
	}
	return b.String()
}
