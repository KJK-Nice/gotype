package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrNameTaken     = errors.New("name taken")
	ErrAlreadyOwns   = errors.New("already owned this season")
	ErrBadState      = errors.New("bad order state")
	ErrAlreadyGranted = errors.New("already granted")
)

// Store persists Player progression entities (JSON file or Redis document).
type Store struct {
	mu   sync.Mutex
	st   storage
	data db
}

type storage interface {
	load() (db, error)
	save(db) error
}

type fileStorage struct {
	path string
}

func (f *fileStorage) load() (db, error) {
	b, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyDB(), nil
		}
		return db{}, err
	}
	var d db
	if err := json.Unmarshal(b, &d); err != nil {
		return db{}, err
	}
	normalizeDB(&d)
	return d, nil
}

func (f *fileStorage) save(d db) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

type db struct {
	Players    map[string]Player         `json:"players"`
	ByName     map[string]string         `json:"by_name"` // name_key → player id
	Inventory  map[string]InventoryItem  `json:"inventory"` // playerID|sku
	Equipment  map[string]Equipment      `json:"equipment"` // playerID|slot
	Seasons    map[int]Season            `json:"seasons"`
	Progress   map[string]SeasonProgress `json:"progress"` // playerID|seasonID
	Orders     map[string]Order          `json:"orders"`
	Daily      map[string]DailyXP        `json:"daily"` // playerID|day
	NextSeason int                       `json:"next_season"`
}

// Open loads or creates a JSON store at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, err
	}
	return openWith(&fileStorage{path: path})
}

func openWith(st storage) (*Store, error) {
	s := &Store{st: st}
	d, err := st.load()
	if err != nil {
		return nil, err
	}
	s.data = d
	before := len(s.data.Seasons)
	if err := s.ensureSeasonLocked(time.Now()); err != nil {
		return nil, err
	}
	if len(s.data.Seasons) > before {
		return s, s.saveLocked()
	}
	return s, nil
}

func emptyDB() db {
	d := db{NextSeason: 1}
	normalizeDB(&d)
	return d
}

// normalizeDB ensures every map field is non-nil so writes never panic.
func normalizeDB(d *db) {
	if d.Players == nil {
		d.Players = map[string]Player{}
	}
	if d.ByName == nil {
		d.ByName = map[string]string{}
	}
	if d.Inventory == nil {
		d.Inventory = map[string]InventoryItem{}
	}
	if d.Equipment == nil {
		d.Equipment = map[string]Equipment{}
	}
	if d.Seasons == nil {
		d.Seasons = map[int]Season{}
	}
	if d.Progress == nil {
		d.Progress = map[string]SeasonProgress{}
	}
	if d.Orders == nil {
		d.Orders = map[string]Order{}
	}
	if d.Daily == nil {
		d.Daily = map[string]DailyXP{}
	}
	if d.NextSeason < 1 {
		d.NextSeason = 1
	}
}

func (s *Store) saveLocked() error {
	return s.st.save(s.data)
}

func invKey(playerID, sku string) string   { return playerID + "|" + sku }
func eqKey(playerID, slot string) string   { return playerID + "|" + slot }
func progKey(playerID string, seasonID int) string {
	return fmt.Sprintf("%s|%d", playerID, seasonID)
}
func dailyKey(playerID, day string) string { return playerID + "|" + day }

func (s *Store) ensureSeasonLocked(now time.Time) error {
	now = now.UTC()
	for _, se := range s.data.Seasons {
		if !now.Before(se.StartsAt) && now.Before(se.EndsAt) {
			return nil
		}
	}
	id := s.data.NextSeason
	if id < 1 {
		id = 1
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	se := Season{
		ID:       id,
		StartsAt: start,
		EndsAt:   start.AddDate(0, 0, catalog.SeasonLengthDays),
		TrackRef: "v1",
	}
	s.data.Seasons[id] = se
	s.data.NextSeason = id + 1
	return nil
}

// CurrentSeason returns the active Season window.
func (s *Store) CurrentSeason(now time.Time) (Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureSeasonLocked(now); err != nil {
		return Season{}, err
	}
	now = now.UTC()
	for _, se := range s.data.Seasons {
		if !now.Before(se.StartsAt) && now.Before(se.EndsAt) {
			return se, nil
		}
	}
	return Season{}, ErrNotFound
}

// CreatePlayer inserts a new Player (name unique case-insensitive).
func (s *Store) CreatePlayer(p Player) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.ByName[p.NameKey]; ok {
		return ErrNameTaken
	}
	if _, ok := s.data.Players[p.ID]; ok {
		return fmt.Errorf("player id exists")
	}
	s.data.Players[p.ID] = p
	s.data.ByName[p.NameKey] = p.ID
	return s.saveLocked()
}

// GetPlayer returns a Player by id.
func (s *Store) GetPlayer(id string) (Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Players[id]
	if !ok {
		return Player{}, ErrNotFound
	}
	return p, nil
}

// GetPlayerByNameKey looks up by normalized name key.
func (s *Store) GetPlayerByNameKey(key string) (Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.data.ByName[key]
	if !ok {
		return Player{}, ErrNotFound
	}
	return s.data.Players[id], nil
}

// SetActiveSession claims the session (kicks older by overwrite).
func (s *Store) SetActiveSession(playerID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Players[playerID]
	if !ok {
		return ErrNotFound
	}
	p.ActiveSessionID = sessionID
	s.data.Players[playerID] = p
	return s.saveLocked()
}

// HasActiveSession reports whether sessionID is still the Player's active session.
func (s *Store) HasActiveSession(playerID, sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Players[playerID]
	if !ok {
		return false
	}
	return p.ActiveSessionID == sessionID
}

// RenamePlayer renames once per Season flag on Player.
func (s *Store) RenamePlayer(playerID, newName, newKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Players[playerID]
	if !ok {
		return ErrNotFound
	}
	if p.RenameUsed {
		return fmt.Errorf("rename already used this season")
	}
	if _, taken := s.data.ByName[newKey]; taken && newKey != p.NameKey {
		return ErrNameTaken
	}
	delete(s.data.ByName, p.NameKey)
	p.Name = newName
	p.NameKey = newKey
	p.RenameUsed = true
	s.data.Players[playerID] = p
	s.data.ByName[newKey] = playerID
	return s.saveLocked()
}

// GetOrCreateProgress returns SeasonProgress for player/season.
func (s *Store) GetOrCreateProgress(playerID string, seasonID int) (SeasonProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := progKey(playerID, seasonID)
	if p, ok := s.data.Progress[k]; ok {
		return p, nil
	}
	p := SeasonProgress{
		PlayerID:       playerID,
		SeasonID:       seasonID,
		ClaimedFree:    []int{},
		ClaimedPremium: []int{},
	}
	s.data.Progress[k] = p
	if err := s.saveLocked(); err != nil {
		return SeasonProgress{}, err
	}
	return p, nil
}

// SaveProgress persists SeasonProgress.
func (s *Store) SaveProgress(p SeasonProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Progress[progKey(p.PlayerID, p.SeasonID)] = p
	return s.saveLocked()
}

// GetDailyXP returns XP granted on a UTC day.
func (s *Store) GetDailyXP(playerID, day string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data.Daily[dailyKey(playerID, day)]
	if !ok {
		return 0, nil
	}
	return d.XP, nil
}

// SetDailyXP sets the day total.
func (s *Store) SetDailyXP(playerID, day string, xp int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Daily[dailyKey(playerID, day)] = DailyXP{PlayerID: playerID, Day: day, XP: xp}
	return s.saveLocked()
}

// AddInventory increments qty (creates row if needed).
func (s *Store) AddInventory(playerID, sku string, delta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := invKey(playerID, sku)
	it := s.data.Inventory[k]
	it.PlayerID = playerID
	it.SKU = sku
	it.Qty += delta
	if it.Qty < 0 {
		it.Qty = 0
	}
	s.data.Inventory[k] = it
	return s.saveLocked()
}

// ListInventory returns all items for a player.
func (s *Store) ListInventory(playerID string) []InventoryItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []InventoryItem
	for _, it := range s.data.Inventory {
		if it.PlayerID == playerID && it.Qty > 0 {
			out = append(out, it)
		}
	}
	return out
}

// InventoryQty returns qty for SKU.
func (s *Store) InventoryQty(playerID, sku string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Inventory[invKey(playerID, sku)].Qty
}

// Equip sets a slot → SKU (empty sku clears).
func (s *Store) Equip(playerID, slot, sku string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := eqKey(playerID, slot)
	if sku == "" {
		delete(s.data.Equipment, k)
		return s.saveLocked()
	}
	s.data.Equipment[k] = Equipment{PlayerID: playerID, Slot: slot, SKU: sku}
	return s.saveLocked()
}

// ListEquipment returns equipped slots.
func (s *Store) ListEquipment(playerID string) []Equipment {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Equipment
	for _, e := range s.data.Equipment {
		if e.PlayerID == playerID {
			out = append(out, e)
		}
	}
	return out
}

// SaveOrder upserts an Order.
func (s *Store) SaveOrder(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Orders[o.ID] = o
	return s.saveLocked()
}

// GetOrder returns an Order by id.
func (s *Store) GetOrder(id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data.Orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// GrantPaidOrder applies shop grant side-effects and marks the order granted in one save.
// qty>0 adds inventory for sku; unlockPremiumSeason>0 sets PremiumUnlocked for that season.
// Idempotent: if the order is already granted, returns it unchanged.
func (s *Store) GrantPaidOrder(id string, now time.Time, sku string, qty int, unlockPremiumSeason int) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.data.Orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	if o.State == OrderGranted {
		return o, nil
	}
	if o.State != OrderPaid {
		return o, ErrBadState
	}
	if qty > 0 && sku != "" {
		k := invKey(o.PlayerID, sku)
		it := s.data.Inventory[k]
		it.PlayerID = o.PlayerID
		it.SKU = sku
		it.Qty += qty
		if it.Qty < 0 {
			it.Qty = 0
		}
		s.data.Inventory[k] = it
	}
	if unlockPremiumSeason > 0 {
		k := progKey(o.PlayerID, unlockPremiumSeason)
		p, ok := s.data.Progress[k]
		if !ok {
			p = SeasonProgress{
				PlayerID:       o.PlayerID,
				SeasonID:       unlockPremiumSeason,
				ClaimedFree:    []int{},
				ClaimedPremium: []int{},
			}
		}
		p.PremiumUnlocked = true
		s.data.Progress[k] = p
	}
	o.State = OrderGranted
	o.GrantedAt = now.UTC()
	s.data.Orders[id] = o
	if err := s.saveLocked(); err != nil {
		return Order{}, err
	}
	return o, nil
}

// ApplyRewardClaims marks free/premium tiers claimed and adds inventory in one save.
// Tiers already present in persisted Claimed* are skipped (idempotent retries).
func (s *Store) ApplyRewardClaims(playerID string, seasonID int, freeTiers, premiumTiers []int, skuByFree, skuByPremium map[int]string) (SeasonProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := progKey(playerID, seasonID)
	prog, ok := s.data.Progress[k]
	if !ok {
		prog = SeasonProgress{
			PlayerID:       playerID,
			SeasonID:       seasonID,
			ClaimedFree:    []int{},
			ClaimedPremium: []int{},
		}
	}
	changed := false
	for _, t := range freeTiers {
		if containsInt(prog.ClaimedFree, t) {
			continue
		}
		sku := skuByFree[t]
		if sku == "" {
			continue
		}
		ik := invKey(playerID, sku)
		it := s.data.Inventory[ik]
		it.PlayerID = playerID
		it.SKU = sku
		it.Qty++
		s.data.Inventory[ik] = it
		prog.ClaimedFree = append(prog.ClaimedFree, t)
		changed = true
	}
	for _, t := range premiumTiers {
		if containsInt(prog.ClaimedPremium, t) {
			continue
		}
		sku := skuByPremium[t]
		if sku == "" {
			continue
		}
		ik := invKey(playerID, sku)
		it := s.data.Inventory[ik]
		it.PlayerID = playerID
		it.SKU = sku
		it.Qty++
		s.data.Inventory[ik] = it
		prog.ClaimedPremium = append(prog.ClaimedPremium, t)
		changed = true
	}
	s.data.Progress[k] = prog
	if !changed {
		return prog, nil
	}
	if err := s.saveLocked(); err != nil {
		return SeasonProgress{}, err
	}
	return prog, nil
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
