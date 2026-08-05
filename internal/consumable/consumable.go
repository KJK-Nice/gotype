package consumable

import (
	"errors"
	"fmt"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

// Effect is applied by the UI after a successful spend.
type Effect int

const (
	EffectReveal Effect = iota
	EffectCalm
	EffectRetry
	EffectHeart
)

// Context describes the active race for eligibility checks.
type Context struct {
	Solo        bool
	ThreeStrike bool
	MatchPoint  bool
	Finished    bool
	DNF         bool
	Claimed     bool
	HP          int
	MaxHP       int
	UsedClass   map[string]bool // class → spent this race
}

var (
	ErrNotClaimed   = errors.New("claim a Player to use consumables")
	ErrMatchPoint   = errors.New("match point — consumables off")
	ErrUnavailable  = errors.New("consumable unavailable")
	ErrNoStock      = errors.New("none in inventory")
	ErrAlreadyUsed  = errors.New("already used this race")
	ErrSoloOnly     = errors.New("retry is solo only")
	ErrHardcoreOnly = errors.New("heart is hardcore only")
	ErrHeartFull    = errors.New("HP already full")
	ErrHeartDNF     = errors.New("too late — DNF")
	ErrFinished     = errors.New("race over")
)

// UseOrder is the slot order for keys 1–4 in the use strip.
var UseOrder = []string{
	catalog.SKUReveal,
	catalog.SKUCalm,
	catalog.SKURetry,
	catalog.SKUHeart,
}

// TrySpend validates, decrements inventory once, and returns the effect to apply.
func TrySpend(store *persist.Store, playerID, sku string, ctx Context) (Effect, error) {
	if store == nil {
		return 0, ErrUnavailable
	}
	if !ctx.Claimed {
		return 0, ErrNotClaimed
	}
	if ctx.Finished || ctx.DNF {
		if ctx.DNF {
			return 0, ErrHeartDNF
		}
		return 0, ErrFinished
	}
	if ctx.MatchPoint {
		return 0, ErrMatchPoint
	}
	item, ok := catalog.Lookup(sku)
	if !ok || item.Kind != catalog.KindConsumable {
		return 0, ErrUnavailable
	}
	if ctx.UsedClass != nil && ctx.UsedClass[item.Class] {
		return 0, ErrAlreadyUsed
	}
	switch item.Class {
	case catalog.SKURetry:
		if !ctx.Solo {
			return 0, ErrSoloOnly
		}
	case catalog.SKUHeart:
		if !ctx.ThreeStrike {
			return 0, ErrHardcoreOnly
		}
		if ctx.HP <= 0 {
			return 0, ErrHeartDNF
		}
		if ctx.HP >= ctx.MaxHP {
			return 0, ErrHeartFull
		}
	}
	if store.InventoryQty(playerID, sku) < 1 {
		return 0, ErrNoStock
	}
	if err := store.SpendInventory(playerID, sku, 1); err != nil {
		return 0, err
	}
	switch item.Class {
	case "reveal":
		return EffectReveal, nil
	case "calm":
		return EffectCalm, nil
	case "retry":
		return EffectRetry, nil
	case "heart":
		return EffectHeart, nil
	default:
		return 0, fmt.Errorf("unknown class %q", item.Class)
	}
}

// ErrMessage returns a short player-facing status line.
func ErrMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotClaimed):
		return "claim a Player to use consumables"
	case errors.Is(err, ErrMatchPoint):
		return "match point — consumables off"
	case errors.Is(err, ErrNoStock):
		return "none in inventory"
	case errors.Is(err, ErrAlreadyUsed):
		return "already used this race"
	case errors.Is(err, ErrSoloOnly):
		return "retry is solo only"
	case errors.Is(err, ErrHardcoreOnly):
		return "heart is hardcore only"
	case errors.Is(err, ErrHeartFull):
		return "HP already full"
	case errors.Is(err, ErrHeartDNF):
		return "too late — DNF"
	case errors.Is(err, ErrFinished):
		return "race over"
	default:
		if err != nil {
			return err.Error()
		}
		return ""
	}
}
