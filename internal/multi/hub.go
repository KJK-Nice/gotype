package multi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

const (
	MaxPlayers      = 4
	MaxSpectators   = 12
	MinPlayers      = 2
	CountdownSecs   = 3
	WinsToTakeMatch = 2 // best-of-3
	DemoCode        = "DEMO"
	codeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLen         = 4
	maxChatLines    = 6
	maxChatLen      = 48
)

type Phase int

const (
	PhaseLobby Phase = iota
	PhaseCountdown
	PhaseRacing
	PhaseDone
)

func (p Phase) String() string {
	switch p {
	case PhaseCountdown:
		return "countdown"
	case PhaseRacing:
		return "racing"
	case PhaseDone:
		return "done"
	default:
		return "lobby"
	}
}

type Progress struct {
	WPM      float64
	Accuracy float64
	Correct  int
	Chars    int
	Done     bool
	HP       int  // Three-Strike; 0 when classic
	DNF      bool // Three-Strike elimination
}

type ChatLine struct {
	Name string
	Text string
}

type Player struct {
	ID         string
	Name       string
	IsHost     bool
	Ready      bool
	Spectator  bool
	Bot        bool
	BotWPM     float64 // target pace for demo bots
	Prog       Progress
	MatchWins  int // series score (best-of-3)
	Streak     int // consecutive race wins
	JoinedAt   time.Time
}

type Room struct {
	Code          string
	Config        game.Config
	Seed          uint64
	HostID        string
	Phase         Phase
	Players       map[string]*Player
	CountdownEnds time.Time
	RaceStarted   time.Time
	RaceEnds      time.Time
	Scored        bool // awarded this race already
	MatchOver       bool
	MatchWinnerID   string
	LastRaceWinnerID string
	RacesPlayed     int // finished races in this series
	Chat            []ChatLine
}

type PlayerView struct {
	ID         string
	Name       string
	IsHost     bool
	Ready      bool
	Spectator  bool
	Bot        bool
	Prog       Progress
	You        bool
	Rank       int
	MatchWins  int
	Streak     int
	Crown      bool // streak > 0
}

type View struct {
	Code             string
	Phase            Phase
	Config           game.Config
	Seed             uint64
	HostID           string
	YouAreHost       bool
	YouAreSpectator  bool
	Players          []PlayerView
	CountdownLeft    time.Duration
	RaceRemaining    time.Duration
	RaceStarted      time.Time
	MatchOver        bool
	MatchWinnerName  string
	RaceNumber       int  // finished races this series (1 after first podium)
	RaceWinnerName   string
	MatchPoint       bool // someone is one win from taking the series
	Chat             []ChatLine
	Err              string
}

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomFull      = errors.New("room full")
	ErrAlreadyInRoom = errors.New("already in a room")
	ErrNotHost       = errors.New("only host can start")
	ErrNotEnough     = errors.New("need at least 2 players")
	ErrBadPhase      = errors.New("room not in lobby")
	ErrNotDone       = errors.New("race not finished")
	ErrEmptyChat     = errors.New("empty message")
	ErrNoLiveRace    = errors.New("no live race")
)

// Hub is an in-process multiplayer matchmaker (one Railway replica).
type Hub struct {
	mu       sync.Mutex
	rooms    map[string]*Room
	byPlayer map[string]string
}

func NewHub() *Hub {
	return &Hub{
		rooms:    make(map[string]*Room),
		byPlayer: make(map[string]string),
	}
}

func NewPlayerID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b[:])
}

func (h *Hub) Create(playerID, name string, cfg game.Config) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.byPlayer[playerID]; ok {
		return View{}, ErrAlreadyInRoom
	}
	if name == "" {
		name = "player"
	}
	code := h.uniqueCodeLocked()
	room := &Room{
		Code:   code,
		Config: cfg,
		HostID: playerID,
		Phase:  PhaseLobby,
		Players: map[string]*Player{
			playerID: {
				ID:       playerID,
				Name:     name,
				IsHost:   true,
				JoinedAt: time.Now(),
			},
		},
	}
	h.rooms[code] = room
	h.byPlayer[playerID] = code
	return h.viewLocked(room, playerID, time.Now()), nil
}

func (h *Hub) Join(playerID, name, code string) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.byPlayer[playerID]; ok {
		return View{}, ErrAlreadyInRoom
	}
	code = normalizeCode(code)
	room, ok := h.rooms[code]
	if !ok {
		return View{}, ErrRoomNotFound
	}
	now := time.Now()
	h.tickBotsLocked(room, now)
	h.advanceLocked(room, now)

	spectate := room.Phase == PhaseCountdown || room.Phase == PhaseRacing
	if spectate {
		if spectatorCountLocked(room) >= MaxSpectators {
			return View{}, ErrRoomFull
		}
	} else if room.Phase != PhaseLobby && room.Phase != PhaseDone {
		return View{}, ErrBadPhase
	} else if racerCountLocked(room) >= MaxPlayers {
		return View{}, ErrRoomFull
	}

	if name == "" {
		name = "player"
	}
	name = uniqueNameLocked(room, name, playerID)
	room.Players[playerID] = &Player{
		ID:        playerID,
		Name:      name,
		Spectator: spectate,
		JoinedAt:  now,
	}
	h.byPlayer[playerID] = code
	return h.viewLocked(room, playerID, now), nil
}

// SpectateLive joins the busiest live race as a spectator, or the looping DEMO room.
func (h *Hub) SpectateLive(playerID, name string, now time.Time) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.byPlayer[playerID]; ok {
		return View{}, ErrAlreadyInRoom
	}
	room := h.pickLiveRoomLocked(now)
	if room == nil {
		room = h.ensureDemoLocked(now)
	}
	if spectatorCountLocked(room) >= MaxSpectators {
		return View{}, ErrRoomFull
	}
	if name == "" {
		name = "viewer"
	}
	name = uniqueNameLocked(room, name, playerID)
	room.Players[playerID] = &Player{
		ID:        playerID,
		Name:      name,
		Spectator: true,
		JoinedAt:  now,
	}
	h.byPlayer[playerID] = room.Code
	return h.viewLocked(room, playerID, now), nil
}

func (h *Hub) Leave(playerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	code, ok := h.byPlayer[playerID]
	if !ok {
		return
	}
	delete(h.byPlayer, playerID)
	room := h.rooms[code]
	if room == nil {
		return
	}
	delete(room.Players, playerID)
	if len(room.Players) == 0 {
		delete(h.rooms, code)
		return
	}
	if room.Code != DemoCode && humanCountLocked(room) == 0 {
		delete(h.rooms, code)
		return
	}
	if room.HostID == playerID {
		var next *Player
		for _, p := range room.Players {
			if p.Spectator || p.Bot {
				continue
			}
			if next == nil || p.JoinedAt.Before(next.JoinedAt) {
				next = p
			}
		}
		if next == nil {
			for _, p := range room.Players {
				if next == nil || p.JoinedAt.Before(next.JoinedAt) {
					next = p
				}
			}
		}
		if next != nil {
			room.HostID = next.ID
			next.IsHost = true
		}
	}
}

// SetThreeStrike toggles hardcore (Three-Strike) in lobby only; host only.
func (h *Hub) SetThreeStrike(playerID string, on bool) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{}, err
	}
	if room.Phase != PhaseLobby {
		return View{}, ErrBadPhase
	}
	if room.HostID != playerID {
		return View{}, ErrNotHost
	}
	room.Config.ThreeStrike = on
	return h.viewLocked(room, playerID, time.Now()), nil
}

func (h *Hub) Start(playerID string, now time.Time) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{}, err
	}
	if room.HostID != playerID {
		return View{}, ErrNotHost
	}
	if room.Phase != PhaseLobby {
		return View{}, ErrBadPhase
	}
	if racerCountLocked(room) < MinPlayers {
		return View{}, ErrNotEnough
	}
	var seed uint64
	if room.Config.Daily {
		seed = words.DailySeed(now)
	} else {
		var err error
		seed, err = randomSeed()
		if err != nil {
			return View{}, err
		}
	}
	room.Seed = seed
	room.Phase = PhaseCountdown
	room.CountdownEnds = now.Add(CountdownSecs * time.Second)
	room.Scored = false
	return h.viewLocked(room, playerID, now), nil
}

// Rematch resets a finished race back to lobby. Clears series if match already won.
func (h *Hub) Rematch(playerID string, now time.Time) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{}, err
	}
	if room.Phase != PhaseDone {
		return View{}, ErrNotDone
	}
	if room.MatchOver {
		for _, p := range room.Players {
			p.MatchWins = 0
			p.Streak = 0
		}
		room.MatchOver = false
		room.MatchWinnerID = ""
		room.LastRaceWinnerID = ""
		room.RacesPlayed = 0
	}
	room.Phase = PhaseLobby
	room.Seed = 0
	room.CountdownEnds = time.Time{}
	room.RaceStarted = time.Time{}
	room.RaceEnds = time.Time{}
	room.Scored = false
	for _, p := range room.Players {
		p.Prog = Progress{}
		p.Ready = false
	}
	return h.viewLocked(room, playerID, now), nil
}

func (h *Hub) Say(playerID, text string) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{}, err
	}
	p := room.Players[playerID]
	if p == nil {
		return View{}, ErrRoomNotFound
	}
	text = normalizeChat(text)
	if text == "" {
		return View{}, ErrEmptyChat
	}
	room.Chat = append(room.Chat, ChatLine{Name: p.Name, Text: text})
	if len(room.Chat) > maxChatLines {
		room.Chat = room.Chat[len(room.Chat)-maxChatLines:]
	}
	return h.viewLocked(room, playerID, time.Now()), nil
}

func (h *Hub) Report(playerID string, prog Progress, now time.Time) View {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{Err: err.Error()}
	}
	if p := room.Players[playerID]; p != nil && !p.Spectator {
		p.Prog = prog
	}
	h.tickBotsLocked(room, now)
	h.advanceLocked(room, now)
	return h.viewLocked(room, playerID, now)
}

func (h *Hub) Snapshot(playerID string, now time.Time) View {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{Err: err.Error()}
	}
	h.tickBotsLocked(room, now)
	h.advanceLocked(room, now)
	return h.viewLocked(room, playerID, now)
}

func (h *Hub) advanceLocked(room *Room, now time.Time) {
	switch room.Phase {
	case PhaseCountdown:
		if !now.Before(room.CountdownEnds) {
			room.Phase = PhaseRacing
			room.RaceStarted = room.CountdownEnds
			if room.Config.Mode.EndsOnPrompt() {
				room.RaceEnds = room.RaceStarted.Add(24 * time.Hour)
			} else {
				room.RaceEnds = room.RaceStarted.Add(room.Config.Duration)
			}
		}
	case PhaseRacing:
		allDone := true
		anyRacer := false
		for _, p := range room.Players {
			if p.Spectator {
				continue
			}
			anyRacer = true
			if !p.Prog.Done {
				allDone = false
				break
			}
		}
		if !anyRacer {
			allDone = false
		}
		if allDone || !now.Before(room.RaceEnds) {
			room.Phase = PhaseDone
			for _, p := range room.Players {
				if !p.Spectator {
					p.Prog.Done = true
				}
			}
			h.awardRaceLocked(room)
		}
	case PhaseDone:
		if room.Code == DemoCode {
			// loop DEMO forever for creators filming
			if now.After(room.RaceEnds.Add(2 * time.Second)) {
				h.resetDemoLocked(room, now)
			}
		}
	}
}

func (h *Hub) awardRaceLocked(room *Room) {
	if room.Scored {
		return
	}
	room.Scored = true
	room.RacesPlayed++

	var winner *Player
	for _, p := range room.Players {
		if p.Spectator || p.Prog.DNF {
			continue
		}
		if p.Prog.Chars == 0 && p.Prog.Correct == 0 {
			continue
		}
		if winner == nil || p.Prog.WPM > winner.Prog.WPM {
			winner = p
		}
	}
	room.LastRaceWinnerID = ""
	for _, p := range room.Players {
		if p.Spectator {
			continue
		}
		if winner != nil && p.ID == winner.ID {
			p.Streak++
			p.MatchWins++
			room.LastRaceWinnerID = winner.ID
		} else {
			p.Streak = 0
		}
	}
	if winner != nil && winner.MatchWins >= WinsToTakeMatch {
		room.MatchOver = true
		room.MatchWinnerID = winner.ID
	}
}

func (h *Hub) roomForLocked(playerID string) (*Room, error) {
	code, ok := h.byPlayer[playerID]
	if !ok {
		return nil, ErrRoomNotFound
	}
	room := h.rooms[code]
	if room == nil {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

func (h *Hub) viewLocked(room *Room, playerID string, now time.Time) View {
	you := room.Players[playerID]
	players := make([]PlayerView, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, PlayerView{
			ID:        p.ID,
			Name:      p.Name,
			IsHost:    p.ID == room.HostID,
			Ready:     p.Ready,
			Spectator: p.Spectator,
			Bot:       p.Bot,
			Prog:      p.Prog,
			You:       p.ID == playerID,
			MatchWins: p.MatchWins,
			Streak:    p.Streak,
			Crown:     p.Streak > 0,
		})
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].Spectator != players[j].Spectator {
			return !players[i].Spectator
		}
		if room.Phase == PhaseDone || room.Phase == PhaseRacing {
			if players[i].Prog.WPM != players[j].Prog.WPM {
				return players[i].Prog.WPM > players[j].Prog.WPM
			}
		}
		if players[i].MatchWins != players[j].MatchWins {
			return players[i].MatchWins > players[j].MatchWins
		}
		return players[i].Name < players[j].Name
	})
	rank := 0
	for i := range players {
		if players[i].Spectator {
			players[i].Rank = 0
			continue
		}
		rank++
		players[i].Rank = rank
	}

	v := View{
		Code:            room.Code,
		Phase:           room.Phase,
		Config:          room.Config,
		Seed:            room.Seed,
		HostID:          room.HostID,
		YouAreHost:      room.HostID == playerID,
		YouAreSpectator: you != nil && you.Spectator,
		Players:         players,
		MatchOver:       room.MatchOver,
		RaceNumber:      room.RacesPlayed,
		Chat:            append([]ChatLine(nil), room.Chat...),
	}
	if room.MatchWinnerID != "" {
		if p := room.Players[room.MatchWinnerID]; p != nil {
			v.MatchWinnerName = p.Name
		}
	}
	if room.LastRaceWinnerID != "" {
		if p := room.Players[room.LastRaceWinnerID]; p != nil {
			v.RaceWinnerName = p.Name
		}
	}
	if !room.MatchOver {
		for _, p := range room.Players {
			if p.Spectator {
				continue
			}
			if p.MatchWins == WinsToTakeMatch-1 {
				v.MatchPoint = true
				break
			}
		}
	}
	switch room.Phase {
	case PhaseCountdown:
		left := room.CountdownEnds.Sub(now)
		if left < 0 {
			left = 0
		}
		v.CountdownLeft = left
	case PhaseRacing:
		left := room.RaceEnds.Sub(now)
		if left < 0 {
			left = 0
		}
		v.RaceRemaining = left
		v.RaceStarted = room.RaceStarted
	case PhaseDone:
		v.RaceStarted = room.RaceStarted
	}
	return v
}

func (h *Hub) uniqueCodeLocked() string {
	for {
		code := randomCode()
		if _, ok := h.rooms[code]; !ok {
			return code
		}
	}
}

func randomCode() string {
	var b [codeLen]byte
	_, _ = rand.Read(b[:])
	out := make([]byte, codeLen)
	for i := range out {
		out[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(out)
}

func normalizeCode(code string) string {
	out := make([]byte, 0, codeLen)
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c >= 'A' && c <= 'Z' {
			out = append(out, c)
		}
	}
	return string(out)
}

func randomSeed() (uint64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	var seed uint64
	for _, x := range b {
		seed = seed<<8 | uint64(x)
	}
	if seed == 0 {
		seed = 1
	}
	return seed, nil
}

func uniqueNameLocked(room *Room, name, playerID string) string {
	for _, p := range room.Players {
		if p.Name == name {
			suffix := playerID
			if len(suffix) > 4 {
				suffix = suffix[:4]
			}
			return fmt.Sprintf("%s_%s", name, suffix)
		}
	}
	return name
}

func racerCountLocked(room *Room) int {
	n := 0
	for _, p := range room.Players {
		if !p.Spectator {
			n++
		}
	}
	return n
}

func spectatorCountLocked(room *Room) int {
	n := 0
	for _, p := range room.Players {
		if p.Spectator {
			n++
		}
	}
	return n
}

func botCountLocked(room *Room) int {
	n := 0
	for _, p := range room.Players {
		if p.Bot {
			n++
		}
	}
	return n
}

func humanCountLocked(room *Room) int {
	n := 0
	for _, p := range room.Players {
		if !p.Bot {
			n++
		}
	}
	return n
}

func (h *Hub) pickLiveRoomLocked(now time.Time) *Room {
	var best *Room
	bestN := -1
	for _, room := range h.rooms {
		if room.Code == DemoCode {
			continue
		}
		h.tickBotsLocked(room, now)
		h.advanceLocked(room, now)
		if room.Phase != PhaseRacing && room.Phase != PhaseCountdown {
			continue
		}
		n := racerCountLocked(room)
		if n > bestN {
			best = room
			bestN = n
		}
	}
	return best
}

func (h *Hub) ensureDemoLocked(now time.Time) *Room {
	if room, ok := h.rooms[DemoCode]; ok {
		h.tickBotsLocked(room, now)
		h.advanceLocked(room, now)
		return room
	}
	cfg := game.Config{Mode: game.ModeTime, Duration: 30 * time.Second, WordCount: 25}
	seed, err := randomSeed()
	if err != nil {
		seed = 1
	}
	botA := &Player{
		ID: "bot-neon", Name: "neon", IsHost: true, Bot: true, BotWPM: 92,
		JoinedAt: now,
	}
	botB := &Player{
		ID: "bot-pixel", Name: "pixel", Bot: true, BotWPM: 78,
		JoinedAt: now.Add(time.Millisecond),
	}
	room := &Room{
		Code:          DemoCode,
		Config:        cfg,
		Seed:          seed,
		HostID:        botA.ID,
		Phase:         PhaseCountdown,
		CountdownEnds: now.Add(CountdownSecs * time.Second),
		Players: map[string]*Player{
			botA.ID: botA,
			botB.ID: botB,
		},
	}
	h.rooms[DemoCode] = room
	return room
}

func (h *Hub) resetDemoLocked(room *Room, now time.Time) {
	seed, err := randomSeed()
	if err != nil {
		seed = room.Seed + 1
	}
	room.Seed = seed
	room.Phase = PhaseCountdown
	room.CountdownEnds = now.Add(CountdownSecs * time.Second)
	room.RaceStarted = time.Time{}
	room.RaceEnds = time.Time{}
	room.Scored = false
	room.MatchOver = false
	room.MatchWinnerID = ""
	for _, p := range room.Players {
		p.Prog = Progress{}
		p.Ready = false
		if p.Bot {
			p.MatchWins = 0
			p.Streak = 0
		}
	}
}

func (h *Hub) tickBotsLocked(room *Room, now time.Time) {
	if room.Phase != PhaseRacing {
		return
	}
	elapsed := now.Sub(room.RaceStarted).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	dur := room.Config.Duration.Seconds()
	if dur < 1 {
		dur = 30
	}
	// ~5 chars per word
	for _, p := range room.Players {
		if !p.Bot || p.Spectator {
			continue
		}
		wpm := p.BotWPM
		if wpm < 1 {
			wpm = 70
		}
		chars := int(wpm * 5 * elapsed / 60)
		target := int(wpm * 5 * dur / 60)
		if chars > target {
			chars = target
		}
		done := elapsed >= dur || chars >= target
		acc := 97.0
		if p.Name == "pixel" {
			acc = 94.0
		}
		p.Prog = Progress{
			WPM:      wpm * (0.85 + 0.15*(elapsed/dur)),
			Accuracy: acc,
			Correct:  chars,
			Chars:    chars,
			Done:     done,
		}
		if done && p.Prog.WPM < wpm*0.9 {
			p.Prog.WPM = wpm
		}
	}
}

func normalizeChat(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	switch lower {
	case "/gg", "gg":
		return "gg"
	case "/glhf", "glhf":
		return "glhf"
	case "/wp", "wp":
		return "wp"
	case "/nt", "nt":
		return "nt"
	}
	if strings.HasPrefix(lower, "/") {
		text = strings.TrimSpace(text[1:])
	}
	if utf8.RuneCountInString(text) > maxChatLen {
		r := []rune(text)
		text = string(r[:maxChatLen])
	}
	return text
}
