package progress

import (
	"path/filepath"
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
		g, err := svc.GrantFinish(pid, FinishSolo, day)
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
