package persist

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
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
