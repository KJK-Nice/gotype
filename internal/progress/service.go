package progress

import (
	"fmt"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

// Kind of race finish for XP.
type FinishKind int

const (
	FinishNone FinishKind = iota
	FinishSolo
	FinishMulti
)

// Service grants XP and claims Season track rewards.
type Service struct {
	Store *persist.Store
}

func NewService(store *persist.Store) *Service {
	return &Service{Store: store}
}

// GrantResult is the outcome of a race XP grant.
type GrantResult struct {
	Granted    int
	DayXP      int
	SeasonXP   int
	Tier       int
	SeasonID   int
	Skipped    string // reason when 0 granted beyond cap/incomplete
	BonusXP    int
	ComboPB    int
	ComboPBNew bool
}

// GrantFinish awards Season XP for a completed race (idempotent caller-side).
func (s *Service) GrantFinish(playerID string, kind FinishKind, bestCombo int, now time.Time) (GrantResult, error) {
	pb, pbNew, err := s.Store.RecordBestCombo(playerID, bestCombo)
	if err != nil {
		return GrantResult{}, err
	}
	want := FinishXP(kind, bestCombo)
	bonus := ComboBonusXP(bestCombo)
	if want == 0 {
		return GrantResult{Skipped: "incomplete", BonusXP: bonus, ComboPB: pb, ComboPBNew: pbNew}, nil
	}
	se, err := s.Store.CurrentSeason(now)
	if err != nil {
		return GrantResult{}, err
	}
	prog, err := s.Store.GetOrCreateProgress(playerID, se.ID)
	if err != nil {
		return GrantResult{}, err
	}
	day := UTCDay(now)
	dayXP, err := s.Store.GetDailyXP(playerID, day)
	if err != nil {
		return GrantResult{}, err
	}
	granted, newDay, err := ApplyXPGrant(prog.XP, dayXP, want, now)
	if err != nil {
		return GrantResult{}, err
	}
	if granted == 0 {
		skip := "daily_cap"
		if want == 0 {
			skip = "incomplete"
		}
		return GrantResult{
			Granted:    0,
			DayXP:      newDay,
			SeasonXP:   prog.XP,
			Tier:       TierFromXP(prog.XP),
			SeasonID:   se.ID,
			Skipped:    skip,
			BonusXP:    bonus,
			ComboPB:    pb,
			ComboPBNew: pbNew,
		}, nil
	}
	prog.XP += granted
	if err := s.Store.SaveProgress(prog); err != nil {
		return GrantResult{}, err
	}
	if err := s.Store.SetDailyXP(playerID, day, newDay); err != nil {
		return GrantResult{}, err
	}
	if err := s.claimUnlockedRewards(playerID, &prog); err != nil {
		return GrantResult{}, err
	}
	return GrantResult{
		Granted:    granted,
		DayXP:      newDay,
		SeasonXP:   prog.XP,
		Tier:       TierFromXP(prog.XP),
		SeasonID:   se.ID,
		BonusXP:    bonus,
		ComboPB:    pb,
		ComboPBNew: pbNew,
	}, nil
}

// NoteBestCombo updates Combo PB without granting XP (DNF).
func (s *Service) NoteBestCombo(playerID string, bestCombo int) (GrantResult, error) {
	pb, pbNew, err := s.Store.RecordBestCombo(playerID, bestCombo)
	if err != nil {
		return GrantResult{}, err
	}
	return GrantResult{ComboPB: pb, ComboPBNew: pbNew, Skipped: "dnf"}, nil
}

func (s *Service) claimUnlockedRewards(playerID string, prog *persist.SeasonProgress) error {
	tier := TierFromXP(prog.XP)
	var freeTiers, premiumTiers []int
	skuByFree := map[int]string{}
	skuByPremium := map[int]string{}
	for t := 1; t <= tier; t++ {
		if sku := catalog.FreeTrackReward(t); sku != "" && !containsInt(prog.ClaimedFree, t) {
			freeTiers = append(freeTiers, t)
			skuByFree[t] = sku
		}
		if prog.PremiumUnlocked {
			if sku := catalog.PremiumTrackReward(t); sku != "" && !containsInt(prog.ClaimedPremium, t) {
				premiumTiers = append(premiumTiers, t)
				skuByPremium[t] = sku
			}
		}
	}
	if len(freeTiers) == 0 && len(premiumTiers) == 0 {
		return nil
	}
	updated, err := s.Store.ApplyRewardClaims(playerID, prog.SeasonID, freeTiers, premiumTiers, skuByFree, skuByPremium)
	if err != nil {
		return err
	}
	*prog = updated
	return nil
}

// ClaimPendingRewards claims any unlocked track rewards for the current season.
func (s *Service) ClaimPendingRewards(playerID string, now time.Time) error {
	se, err := s.Store.CurrentSeason(now)
	if err != nil {
		return err
	}
	prog, err := s.Store.GetOrCreateProgress(playerID, se.ID)
	if err != nil {
		return err
	}
	return s.claimUnlockedRewards(playerID, &prog)
}

// UnlockPremium marks Season premium for the current Season.
func (s *Service) UnlockPremium(playerID string, now time.Time) error {
	se, err := s.Store.CurrentSeason(now)
	if err != nil {
		return err
	}
	prog, err := s.Store.GetOrCreateProgress(playerID, se.ID)
	if err != nil {
		return err
	}
	if prog.PremiumUnlocked {
		return persist.ErrAlreadyOwns
	}
	prog.PremiumUnlocked = true
	if err := s.Store.SaveProgress(prog); err != nil {
		return err
	}
	return s.claimUnlockedRewards(playerID, &prog)
}

// PassView is a read model for the Season Pass UI.
type PassView struct {
	SeasonID        int
	DaysLeft        int
	XP              int
	Tier            int
	PremiumUnlocked bool
	NextTierXP      int
	MatrixOwned     bool
	RainOwned       bool
}

// ViewPass builds Season Pass summary.
func (s *Service) ViewPass(playerID string, now time.Time) (PassView, error) {
	se, err := s.Store.CurrentSeason(now)
	if err != nil {
		return PassView{}, err
	}
	prog, err := s.Store.GetOrCreateProgress(playerID, se.ID)
	if err != nil {
		return PassView{}, err
	}
	tier := TierFromXP(prog.XP)
	next := XPForTier(tier + 1)
	days := int(se.EndsAt.Sub(now.UTC()).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return PassView{
		SeasonID:        se.ID,
		DaysLeft:        days,
		XP:              prog.XP,
		Tier:            tier,
		PremiumUnlocked: prog.PremiumUnlocked,
		NextTierXP:      next,
		MatrixOwned:     s.Store.InventoryQty(playerID, catalog.SKUMatrix) > 0,
		RainOwned:       s.Store.InventoryQty(playerID, catalog.SKUMakeItRain) > 0,
	}, nil
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// Ensure claimed helper exported for tests.
func ClaimedContains(xs []int, v int) bool { return containsInt(xs, v) }

// FormatGrantLine is a short results banner.
func FormatGrantLine(g GrantResult) string {
	if g.Granted <= 0 {
		if g.Skipped == "daily_cap" {
			return fmt.Sprintf("+0 xp · day %d/%d (cap)", g.DayXP, DailyXPCap)
		}
		return ""
	}
	line := fmt.Sprintf("+%d xp", g.Granted)
	if g.BonusXP > 0 {
		line += fmt.Sprintf(" · combo +%d", g.BonusXP)
	}
	if g.ComboPBNew {
		line += fmt.Sprintf(" · pb %d", g.ComboPB)
	}
	line += fmt.Sprintf(" · day %d/%d · tier %d", g.DayXP, DailyXPCap, g.Tier)
	return line
}
