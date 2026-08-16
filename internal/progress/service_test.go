package progress

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
)

func TestGrantFinishAndMatrixUnlock(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("Racer1", "10.0.0.1", "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)

	// Grant enough solo finishes to reach tier 10 (1000 XP) under daily cap — use multiple days.
	pid := reg.Player.ID
	total := 0
	day := now
	for total < 1000 {
		g, err := svc.GrantFinish(pid, FinishSolo, 0, day)
		if err != nil {
			t.Fatal(err)
		}
		total += g.Granted
		if g.Granted == 0 {
			day = day.Add(24 * time.Hour)
		}
	}
	if store.InventoryQty(pid, catalog.SKUMatrix) < 1 {
		t.Fatal("expected Matrix Cosmetic at free tier 10")
	}
	pass, err := svc.ViewPass(pid, day)
	if err != nil {
		t.Fatal(err)
	}
	if pass.Tier < 10 {
		t.Fatalf("tier = %d", pass.Tier)
	}
}

func TestClaimUnlockedRewardsIdempotent(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("Racer2", "10.0.0.2", "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	se, err := store.CurrentSeason(now)
	if err != nil {
		t.Fatal(err)
	}
	prog := persist.SeasonProgress{
		PlayerID:       reg.Player.ID,
		SeasonID:       se.ID,
		XP:             1000,
		ClaimedFree:    []int{},
		ClaimedPremium: []int{},
	}
	if err := store.SaveProgress(prog); err != nil {
		t.Fatal(err)
	}
	if err := svc.claimUnlockedRewards(reg.Player.ID, &prog); err != nil {
		t.Fatal(err)
	}
	if store.InventoryQty(reg.Player.ID, catalog.SKUMatrix) != 1 {
		t.Fatal("expected matrix after first claim")
	}
	// Simulate AwardXP / claim retry with stale in-memory claimed list.
	stale := persist.SeasonProgress{
		PlayerID:       reg.Player.ID,
		SeasonID:       se.ID,
		XP:             1000,
		ClaimedFree:    []int{},
		ClaimedPremium: []int{},
	}
	if err := svc.claimUnlockedRewards(reg.Player.ID, &stale); err != nil {
		t.Fatal(err)
	}
	if store.InventoryQty(reg.Player.ID, catalog.SKUMatrix) != 1 {
		t.Fatalf("qty=%d want 1 after retry", store.InventoryQty(reg.Player.ID, catalog.SKUMatrix))
	}
}

func TestGrantFinishComboBonusAndPB(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("Racer3", "10.0.0.3", "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	g, err := svc.GrantFinish(reg.Player.ID, FinishSolo, 47, now)
	if err != nil {
		t.Fatal(err)
	}
	if g.BonusXP != 4 {
		t.Fatalf("bonus=%d want 4", g.BonusXP)
	}
	if g.Granted != SoloFinishXP+4 {
		t.Fatalf("granted=%d", g.Granted)
	}
	if !g.ComboPBNew || g.ComboPB != 47 {
		t.Fatalf("pb new=%v pb=%d", g.ComboPBNew, g.ComboPB)
	}
	line := FormatGrantLine(g)
	if !strings.Contains(line, "combo +4") || !strings.Contains(line, "pb 47") {
		t.Fatalf("grant line %q", line)
	}
	g2, err := svc.GrantFinish(reg.Player.ID, FinishSolo, 20, now)
	if err != nil {
		t.Fatal(err)
	}
	if g2.ComboPBNew || g2.ComboPB != 47 {
		t.Fatalf("second race pb new=%v pb=%d", g2.ComboPBNew, g2.ComboPB)
	}
}

func TestNoteBestComboDNF(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	ps := player.NewService(store)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	reg, err := ps.Register("Racer4", "10.0.0.4", "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	g, err := svc.NoteBestCombo(reg.Player.ID, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !g.ComboPBNew || g.ComboPB != 30 {
		t.Fatalf("dnf pb %+v", g)
	}
	p, err := store.GetPlayer(reg.Player.ID)
	if err != nil || p.BestCombo != 30 {
		t.Fatalf("stored pb=%d err=%v", p.BestCombo, err)
	}
}
