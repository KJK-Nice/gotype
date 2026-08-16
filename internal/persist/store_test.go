package persist

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func runStoreRoundTrip(t *testing.T, s *Store) {
	t.Helper()
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	se, err := s.CurrentSeason(now)
	if err != nil || se.ID < 1 {
		t.Fatalf("season %+v err=%v", se, err)
	}
	p := Player{ID: "p1", Name: "Neo", NameKey: "neo", ClaimHash: "x", CreatedAt: now}
	if err := s.CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	if err := s.AddInventory("p1", "heart", 2); err != nil {
		t.Fatal(err)
	}
}

func runGrantPaidOrderIdempotent(t *testing.T, s *Store) {
	t.Helper()
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.CreatePlayer(Player{ID: "p1", Name: "Neo", NameKey: "neo", ClaimHash: "x", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	o := Order{ID: "ord_1", PlayerID: "p1", SKU: "heart", State: OrderPaid, CreatedAt: now}
	if err := s.SaveOrder(o); err != nil {
		t.Fatal(err)
	}
	g1, err := s.GrantPaidOrder("ord_1", now, "heart", 1, 0)
	if err != nil || g1.State != OrderGranted {
		t.Fatalf("first grant: %+v %v", g1, err)
	}
	g2, err := s.GrantPaidOrder("ord_1", now, "heart", 1, 0)
	if err != nil || g2.State != OrderGranted {
		t.Fatalf("second grant: %+v %v", g2, err)
	}
	if s.InventoryQty("p1", "heart") != 1 {
		t.Fatalf("qty=%d want 1", s.InventoryQty("p1", "heart"))
	}
}

func runApplyRewardClaimsIdempotent(t *testing.T, s *Store) {
	t.Helper()
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	if err := s.CreatePlayer(Player{ID: "p1", Name: "Neo", NameKey: "neo", ClaimHash: "x", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	se, err := s.CurrentSeason(now)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProgress(SeasonProgress{PlayerID: "p1", SeasonID: se.ID, XP: 1000, ClaimedFree: []int{}, ClaimedPremium: []int{}}); err != nil {
		t.Fatal(err)
	}
	skuByFree := map[int]string{10: "matrix"}
	p1, err := s.ApplyRewardClaims("p1", se.ID, []int{10}, nil, skuByFree, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsInt(p1.ClaimedFree, 10) {
		t.Fatalf("claimed=%v", p1.ClaimedFree)
	}
	p2, err := s.ApplyRewardClaims("p1", se.ID, []int{10}, nil, skuByFree, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.InventoryQty("p1", "matrix") != 1 {
		t.Fatalf("qty=%d want 1 (no duplicate on retry)", s.InventoryQty("p1", "matrix"))
	}
	if len(p2.ClaimedFree) != 1 {
		t.Fatalf("claimed_free=%v", p2.ClaimedFree)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	runStoreRoundTrip(t, s)
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.GetPlayer("p1")
	if err != nil || got.Name != "Neo" {
		t.Fatalf("%+v %v", got, err)
	}
	if s2.InventoryQty("p1", "heart") != 2 {
		t.Fatal("inventory not persisted")
	}
}

func TestOpenNormalizesNilMaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.json")
	raw := []byte(`{
  "players": {"p1": {"id":"p1","name":"Neo","name_key":"neo","claim_hash":"x"}},
  "by_name": {"neo":"p1"},
  "inventory": null,
  "equipment": null,
  "progress": null,
  "orders": null,
  "daily": null
}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddInventory("p1", "heart", 1); err != nil {
		t.Fatalf("AddInventory: %v", err)
	}
	if err := s.CreatePlayer(Player{ID: "p2", Name: "Trinity", NameKey: "trinity", ClaimHash: "y", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	if err := s.Equip("p1", "theme", "matrix"); err != nil {
		t.Fatalf("Equip: %v", err)
	}
	if err := s.SaveOrder(Order{ID: "ord_1", PlayerID: "p1", SKU: "heart", State: OrderCreated, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("SaveOrder: %v", err)
	}
	if _, err := s.GetOrCreateProgress("p1", 1); err != nil {
		t.Fatalf("GetOrCreateProgress: %v", err)
	}
	if err := s.SetDailyXP("p1", "2026-07-31", 10); err != nil {
		t.Fatalf("SetDailyXP: %v", err)
	}
	if s.InventoryQty("p1", "heart") != 1 {
		t.Fatal("expected inventory after nil-map normalize")
	}
}

func TestGrantPaidOrderIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	runGrantPaidOrderIdempotent(t, s)
}

func TestApplyRewardClaimsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	runApplyRewardClaimsIdempotent(t, s)
}

func TestRecordBestCombo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	if err := s.CreatePlayer(Player{ID: "p1", Name: "Neo", NameKey: "neo", ClaimHash: "x", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	pb, improved, err := s.RecordBestCombo("p1", 40)
	if err != nil || !improved || pb != 40 {
		t.Fatalf("first pb=%d improved=%v err=%v", pb, improved, err)
	}
	pb, improved, err = s.RecordBestCombo("p1", 12)
	if err != nil || improved || pb != 40 {
		t.Fatalf("lower combo pb=%d improved=%v err=%v", pb, improved, err)
	}
	pb, improved, err = s.RecordBestCombo("p1", 41)
	if err != nil || !improved || pb != 41 {
		t.Fatalf("new pb=%d improved=%v err=%v", pb, improved, err)
	}
	got, err := s.GetPlayer("p1")
	if err != nil || got.BestCombo != 41 {
		t.Fatalf("stored %+v err=%v", got, err)
	}
}
