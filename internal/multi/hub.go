package multi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

const (
	MaxPlayers    = 4
	MinPlayers    = 2
	CountdownSecs = 3
	codeAlphabet  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLen       = 4
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
}

type Player struct {
	ID       string
	Name     string
	IsHost   bool
	Ready    bool
	Prog     Progress
	JoinedAt time.Time
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
}

type PlayerView struct {
	ID     string
	Name   string
	IsHost bool
	Ready  bool
	Prog   Progress
	You    bool
	Rank   int
}

type View struct {
	Code          string
	Phase         Phase
	Config        game.Config
	Seed          uint64
	HostID        string
	YouAreHost    bool
	Players       []PlayerView
	CountdownLeft time.Duration
	RaceRemaining time.Duration
	RaceStarted   time.Time
	Err           string
}

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrRoomFull      = errors.New("room full")
	ErrAlreadyInRoom = errors.New("already in a room")
	ErrNotHost       = errors.New("only host can start")
	ErrNotEnough     = errors.New("need at least 2 players")
	ErrBadPhase      = errors.New("room not in lobby")
	ErrNotDone       = errors.New("race not finished")
	ErrNameTaken     = errors.New("name taken in room")
)

// Hub is an in-process multiplayer matchmaker (one Railway replica).
type Hub struct {
	mu       sync.Mutex
	rooms    map[string]*Room
	byPlayer map[string]string // playerID → room code
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
	if room.Phase != PhaseLobby {
		return View{}, ErrBadPhase
	}
	if len(room.Players) >= MaxPlayers {
		return View{}, ErrRoomFull
	}
	if name == "" {
		name = "player"
	}
	for _, p := range room.Players {
		if p.Name == name {
			name = fmt.Sprintf("%s_%s", name, playerID[:4])
			break
		}
	}
	room.Players[playerID] = &Player{
		ID:       playerID,
		Name:     name,
		JoinedAt: time.Now(),
	}
	h.byPlayer[playerID] = code
	return h.viewLocked(room, playerID, time.Now()), nil
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
	if room.HostID == playerID {
		// Promote oldest remaining player.
		var next *Player
		for _, p := range room.Players {
			if next == nil || p.JoinedAt.Before(next.JoinedAt) {
				next = p
			}
		}
		if next != nil {
			room.HostID = next.ID
			next.IsHost = true
		}
	}
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
	if len(room.Players) < MinPlayers {
		return View{}, ErrNotEnough
	}
	seed, err := randomSeed()
	if err != nil {
		return View{}, err
	}
	room.Seed = seed
	room.Phase = PhaseCountdown
	room.CountdownEnds = now.Add(CountdownSecs * time.Second)
	return h.viewLocked(room, playerID, now), nil
}

// Rematch resets a finished room back to lobby — same code, same players.
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
	room.Phase = PhaseLobby
	room.Seed = 0
	room.CountdownEnds = time.Time{}
	room.RaceStarted = time.Time{}
	room.RaceEnds = time.Time{}
	for _, p := range room.Players {
		p.Prog = Progress{}
		p.Ready = false
	}
	return h.viewLocked(room, playerID, now), nil
}

func (h *Hub) Report(playerID string, prog Progress, now time.Time) View {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, err := h.roomForLocked(playerID)
	if err != nil {
		return View{Err: err.Error()}
	}
	if p := room.Players[playerID]; p != nil {
		p.Prog = prog
	}
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
	h.advanceLocked(room, now)
	return h.viewLocked(room, playerID, now)
}

func (h *Hub) advanceLocked(room *Room, now time.Time) {
	switch room.Phase {
	case PhaseCountdown:
		if !now.Before(room.CountdownEnds) {
			room.Phase = PhaseRacing
			room.RaceStarted = room.CountdownEnds
			if room.Config.Mode == game.ModeWords {
				room.RaceEnds = room.RaceStarted.Add(24 * time.Hour)
			} else {
				room.RaceEnds = room.RaceStarted.Add(room.Config.Duration)
			}
		}
	case PhaseRacing:
		allDone := true
		for _, p := range room.Players {
			if !p.Prog.Done {
				allDone = false
				break
			}
		}
		if allDone || !now.Before(room.RaceEnds) {
			room.Phase = PhaseDone
			for _, p := range room.Players {
				p.Prog.Done = true
			}
		}
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
	players := make([]PlayerView, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, PlayerView{
			ID:     p.ID,
			Name:   p.Name,
			IsHost: p.ID == room.HostID,
			Ready:  p.Ready,
			Prog:   p.Prog,
			You:    p.ID == playerID,
		})
	}
	sort.Slice(players, func(i, j int) bool {
		if room.Phase == PhaseDone || room.Phase == PhaseRacing {
			if players[i].Prog.WPM != players[j].Prog.WPM {
				return players[i].Prog.WPM > players[j].Prog.WPM
			}
		}
		return players[i].Name < players[j].Name
	})
	for i := range players {
		players[i].Rank = i + 1
	}

	v := View{
		Code:       room.Code,
		Phase:      room.Phase,
		Config:     room.Config,
		Seed:       room.Seed,
		HostID:     room.HostID,
		YouAreHost: room.HostID == playerID,
		Players:    players,
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
