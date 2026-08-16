package progress

import (
	"fmt"
	"time"
)

const (
	SoloFinishXP    = 10
	MultiFinishXP   = 25
	MaxComboBonusXP = 10
	DailyXPCap      = 200
	XPPerTier       = 100
	MaxTier         = 20
)

// ComboBonusXP is +1 XP per 10 Best Combo, capped at MaxComboBonusXP.
func ComboBonusXP(bestCombo int) int {
	if bestCombo < 10 {
		return 0
	}
	b := bestCombo / 10
	if b > MaxComboBonusXP {
		return MaxComboBonusXP
	}
	return b
}

// FinishXP is the uncapped race award: base plus Combo bonus.
func FinishXP(kind FinishKind, bestCombo int) int {
	var base int
	switch kind {
	case FinishSolo:
		base = SoloFinishXP
	case FinishMulti:
		base = MultiFinishXP
	default:
		return 0
	}
	return base + ComboBonusXP(bestCombo)
}

// ApplyXPGrant returns granted amount (may be soft-capped) and new day total.
// want is the race award (0 for incomplete/spectate/DNF).
func ApplyXPGrant(seasonXP, dayXP, want int, at time.Time) (granted, newDayXP int, err error) {
	_ = at // UTC day ownership is the caller's DailyXP key
	if want < 0 {
		return 0, dayXP, fmt.Errorf("negative xp")
	}
	if want == 0 {
		return 0, dayXP, nil
	}
	if dayXP >= DailyXPCap {
		return 0, DailyXPCap, nil
	}
	remain := DailyXPCap - dayXP
	granted = want
	if granted > remain {
		granted = remain
	}
	return granted, dayXP + granted, nil
}

// TierFromXP maps linear 100 XP/tier → tier 0..20 (2000 XP = tier 20).
func TierFromXP(xp int) int {
	if xp <= 0 {
		return 0
	}
	t := xp / XPPerTier
	if t > MaxTier {
		return MaxTier
	}
	return t
}

// XPForTier returns cumulative XP required to reach tier n (1..20).
func XPForTier(tier int) int {
	if tier <= 0 {
		return 0
	}
	if tier > MaxTier {
		tier = MaxTier
	}
	return tier * XPPerTier
}

// UTCDay is YYYY-MM-DD in UTC for DailyXP keys.
func UTCDay(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
