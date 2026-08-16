package persist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrNameTaken      = errors.New("name taken")
	ErrAlreadyOwns    = errors.New("already owned this season")
	ErrBadState       = errors.New("bad order state")
	ErrAlreadyGranted  = errors.New("already granted")
	ErrLinkingKeyTaken = errors.New("linking key taken")
)

// Store persists Player progression entities (JSON file or Redis document).
type Store struct {
	mu   sync.Mutex
	st   storage
	data db // file backend cache only
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
	ByName       map[string]string         `json:"by_name"`
	ByLinkingKey map[string]string         `json:"by_linking_key"`
	Inventory  map[string]InventoryItem  `json:"inventory"`
	Equipment  map[string]Equipment      `json:"equipment"`
	Seasons    map[int]Season            `json:"seasons"`
	Progress   map[string]SeasonProgress `json:"progress"`
	Orders     map[string]Order          `json:"orders"`
	Daily      map[string]DailyXP        `json:"daily"`
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
	if s.isStateless() {
		return s, s.mutate(func(d *db) error {
			before := len(d.Seasons)
			if err := ensureSeason(d, time.Now()); err != nil {
				return err
			}
			if len(d.Seasons) == before {
				return errNoSave
			}
			return nil
		})
	}
	d, err := st.load()
	if err != nil {
		return nil, err
	}
	s.data = d
	before := len(s.data.Seasons)
	if err := ensureSeason(&s.data, time.Now()); err != nil {
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

func normalizeDB(d *db) {
	if d.Players == nil {
		d.Players = map[string]Player{}
	}
	if d.ByName == nil {
		d.ByName = map[string]string{}
	}
	if d.ByLinkingKey == nil {
		d.ByLinkingKey = map[string]string{}
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

// CurrentSeason returns the active Season window.
func (s *Store) CurrentSeason(now time.Time) (Season, error) {
	var out Season
	err := s.mutate(func(d *db) error {
		before := len(d.Seasons)
		if err := ensureSeason(d, now); err != nil {
			return err
		}
		now = now.UTC()
		for _, se := range d.Seasons {
			if !now.Before(se.StartsAt) && now.Before(se.EndsAt) {
				out = se
				if len(d.Seasons) == before {
					return errNoSave
				}
				return nil
			}
		}
		return ErrNotFound
	})
	return out, err
}

func (s *Store) indexPlayer(d *db, p Player) error {
	if _, ok := d.ByName[p.NameKey]; ok {
		return ErrNameTaken
	}
	if p.LinkingKey != "" {
		if _, ok := d.ByLinkingKey[p.LinkingKey]; ok {
			return ErrLinkingKeyTaken
		}
	}
	if _, ok := d.Players[p.ID]; ok {
		return fmt.Errorf("player id exists")
	}
	d.Players[p.ID] = p
	d.ByName[p.NameKey] = p.ID
	if p.LinkingKey != "" {
		d.ByLinkingKey[p.LinkingKey] = p.ID
	}
	return nil
}

// CreatePlayer inserts a new Player (name unique case-insensitive).
func (s *Store) CreatePlayer(p Player) error {
	return s.mutate(func(d *db) error {
		return s.indexPlayer(d, p)
	})
}

// GetPlayer returns a Player by id.
func (s *Store) GetPlayer(id string) (Player, error) {
	return queryStore(s, func(d *db) (Player, error) {
		p, ok := d.Players[id]
		if !ok {
			return Player{}, ErrNotFound
		}
		return p, nil
	})
}

// GetPlayerByNameKey looks up by normalized name key.
func (s *Store) GetPlayerByNameKey(key string) (Player, error) {
	return queryStore(s, func(d *db) (Player, error) {
		id, ok := d.ByName[key]
		if !ok {
			return Player{}, ErrNotFound
		}
		return d.Players[id], nil
	})
}

// GetPlayerByLinkingKey looks up by wallet linking key.
func (s *Store) GetPlayerByLinkingKey(linkingKey string) (Player, error) {
	return queryStore(s, func(d *db) (Player, error) {
		id, ok := d.ByLinkingKey[linkingKey]
		if !ok {
			return Player{}, ErrNotFound
		}
		return d.Players[id], nil
	})
}

// SetLinkingKey attaches a wallet linking key to an existing Player.
func (s *Store) SetLinkingKey(playerID, linkingKey string) error {
	return s.mutate(func(d *db) error {
		if linkingKey == "" {
			return fmt.Errorf("linking key required")
		}
		if owner, ok := d.ByLinkingKey[linkingKey]; ok && owner != playerID {
			return ErrLinkingKeyTaken
		}
		p, ok := d.Players[playerID]
		if !ok {
			return ErrNotFound
		}
		if p.LinkingKey != "" && p.LinkingKey != linkingKey {
			return fmt.Errorf("wallet already linked")
		}
		if p.LinkingKey == linkingKey {
			return nil
		}
		p.LinkingKey = linkingKey
		d.Players[playerID] = p
		d.ByLinkingKey[linkingKey] = playerID
		return nil
	})
}

// SetActiveSession claims the session (kicks older by overwrite).
func (s *Store) SetActiveSession(playerID, sessionID string) error {
	return s.mutate(func(d *db) error {
		p, ok := d.Players[playerID]
		if !ok {
			return ErrNotFound
		}
		p.ActiveSessionID = sessionID
		d.Players[playerID] = p
		return nil
	})
}

// HasActiveSession reports whether sessionID is still the Player's active session.
func (s *Store) HasActiveSession(playerID, sessionID string) bool {
	ok, _ := queryStore(s, func(d *db) (bool, error) {
		p, found := d.Players[playerID]
		if !found {
			return false, nil
		}
		return p.ActiveSessionID == sessionID, nil
	})
	return ok
}

// RenamePlayer renames once per Season flag on Player.
func (s *Store) RenamePlayer(playerID, newName, newKey string) error {
	return s.mutate(func(d *db) error {
		p, ok := d.Players[playerID]
		if !ok {
			return ErrNotFound
		}
		if p.RenameUsed {
			return fmt.Errorf("rename already used this season")
		}
		if _, taken := d.ByName[newKey]; taken && newKey != p.NameKey {
			return ErrNameTaken
		}
		delete(d.ByName, p.NameKey)
		p.Name = newName
		p.NameKey = newKey
		p.RenameUsed = true
		d.Players[playerID] = p
		d.ByName[newKey] = playerID
		return nil
	})
}

// RecordBestCombo raises Combo PB when combo is a new personal best.
func (s *Store) RecordBestCombo(playerID string, combo int) (int, bool, error) {
	var pb int
	var improved bool
	err := s.mutate(func(d *db) error {
		p, ok := d.Players[playerID]
		if !ok {
			return ErrNotFound
		}
		pb = p.BestCombo
		if combo > p.BestCombo {
			p.BestCombo = combo
			d.Players[playerID] = p
			pb = combo
			improved = true
		}
		return nil
	})
	return pb, improved, err
}

// GetOrCreateProgress returns SeasonProgress for player/season.
func (s *Store) GetOrCreateProgress(playerID string, seasonID int) (SeasonProgress, error) {
	var out SeasonProgress
	err := s.mutate(func(d *db) error {
		k := progKey(playerID, seasonID)
		if p, ok := d.Progress[k]; ok {
			out = p
			return errNoSave
		}
		out = SeasonProgress{
			PlayerID:       playerID,
			SeasonID:       seasonID,
			ClaimedFree:    []int{},
			ClaimedPremium: []int{},
		}
		d.Progress[k] = out
		return nil
	})
	return out, err
}

// SaveProgress persists SeasonProgress.
func (s *Store) SaveProgress(p SeasonProgress) error {
	return s.mutate(func(d *db) error {
		d.Progress[progKey(p.PlayerID, p.SeasonID)] = p
		return nil
	})
}

// GetDailyXP returns XP granted on a UTC day.
func (s *Store) GetDailyXP(playerID, day string) (int, error) {
	return queryStore(s, func(d *db) (int, error) {
		daily, ok := d.Daily[dailyKey(playerID, day)]
		if !ok {
			return 0, nil
		}
		return daily.XP, nil
	})
}

// SetDailyXP sets the day total.
func (s *Store) SetDailyXP(playerID, day string, xp int) error {
	return s.mutate(func(d *db) error {
		d.Daily[dailyKey(playerID, day)] = DailyXP{PlayerID: playerID, Day: day, XP: xp}
		return nil
	})
}

// AddInventory increments qty (creates row if needed).
func (s *Store) AddInventory(playerID, sku string, delta int) error {
	return s.mutate(func(d *db) error {
		k := invKey(playerID, sku)
		it := d.Inventory[k]
		it.PlayerID = playerID
		it.SKU = sku
		it.Qty += delta
		if it.Qty < 0 {
			it.Qty = 0
		}
		d.Inventory[k] = it
		return nil
	})
}

// ListInventory returns all items for a player.
func (s *Store) ListInventory(playerID string) []InventoryItem {
	out, _ := queryStore(s, func(d *db) ([]InventoryItem, error) {
		var items []InventoryItem
		for _, it := range d.Inventory {
			if it.PlayerID == playerID && it.Qty > 0 {
				items = append(items, it)
			}
		}
		return items, nil
	})
	return out
}

// InventoryQty returns qty for SKU.
func (s *Store) InventoryQty(playerID, sku string) int {
	qty, _ := queryStore(s, func(d *db) (int, error) {
		return d.Inventory[invKey(playerID, sku)].Qty, nil
	})
	return qty
}

// Equip sets a slot → SKU (empty sku clears).
func (s *Store) Equip(playerID, slot, sku string) error {
	return s.mutate(func(d *db) error {
		k := eqKey(playerID, slot)
		if sku == "" {
			delete(d.Equipment, k)
			return nil
		}
		d.Equipment[k] = Equipment{PlayerID: playerID, Slot: slot, SKU: sku}
		return nil
	})
}

// ListEquipment returns equipped slots.
func (s *Store) ListEquipment(playerID string) []Equipment {
	out, _ := queryStore(s, func(d *db) ([]Equipment, error) {
		var items []Equipment
		for _, e := range d.Equipment {
			if e.PlayerID == playerID {
				items = append(items, e)
			}
		}
		return items, nil
	})
	return out
}

// SaveOrder upserts an Order.
func (s *Store) SaveOrder(o Order) error {
	return s.mutate(func(d *db) error {
		d.Orders[o.ID] = o
		return nil
	})
}

// GetOrder returns an Order by id.
func (s *Store) GetOrder(id string) (Order, error) {
	return queryStore(s, func(d *db) (Order, error) {
		o, ok := d.Orders[id]
		if !ok {
			return Order{}, ErrNotFound
		}
		return o, nil
	})
}

// GrantPaidOrder applies shop grant side-effects and marks the order granted in one save.
func (s *Store) GrantPaidOrder(id string, now time.Time, sku string, qty int, unlockPremiumSeason int) (Order, error) {
	var out Order
	err := s.mutate(func(d *db) error {
		o, ok := d.Orders[id]
		if !ok {
			return ErrNotFound
		}
		if o.State == OrderGranted {
			out = o
			return errNoSave
		}
		if o.State != OrderPaid {
			out = o
			return ErrBadState
		}
		if qty > 0 && sku != "" {
			k := invKey(o.PlayerID, sku)
			it := d.Inventory[k]
			it.PlayerID = o.PlayerID
			it.SKU = sku
			it.Qty += qty
			if it.Qty < 0 {
				it.Qty = 0
			}
			d.Inventory[k] = it
		}
		if unlockPremiumSeason > 0 {
			k := progKey(o.PlayerID, unlockPremiumSeason)
			p, ok := d.Progress[k]
			if !ok {
				p = SeasonProgress{
					PlayerID:       o.PlayerID,
					SeasonID:       unlockPremiumSeason,
					ClaimedFree:    []int{},
					ClaimedPremium: []int{},
				}
			}
			p.PremiumUnlocked = true
			d.Progress[k] = p
		}
		o.State = OrderGranted
		o.GrantedAt = now.UTC()
		d.Orders[id] = o
		out = o
		return nil
	})
	return out, err
}

// ApplyRewardClaims marks free/premium tiers claimed and adds inventory in one save.
func (s *Store) ApplyRewardClaims(playerID string, seasonID int, freeTiers, premiumTiers []int, skuByFree, skuByPremium map[int]string) (SeasonProgress, error) {
	var out SeasonProgress
	err := s.mutate(func(d *db) error {
		k := progKey(playerID, seasonID)
		prog, ok := d.Progress[k]
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
			it := d.Inventory[ik]
			it.PlayerID = playerID
			it.SKU = sku
			it.Qty++
			d.Inventory[ik] = it
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
			it := d.Inventory[ik]
			it.PlayerID = playerID
			it.SKU = sku
			it.Qty++
			d.Inventory[ik] = it
			prog.ClaimedPremium = append(prog.ClaimedPremium, t)
			changed = true
		}
		d.Progress[k] = prog
		out = prog
		if !changed {
			return errNoSave
		}
		return nil
	})
	return out, err
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
