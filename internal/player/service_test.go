package player

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

func TestRegisterAndClaim(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	reg, err := svc.Register("Neo_42", "1.2.3.4", "sess-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if reg.Display == "" || reg.ClaimCode == "" {
		t.Fatal("expected claim code")
	}
	if reg.Player.Name != "Neo_42" {
		t.Fatalf("name = %q", reg.Player.Name)
	}

	_, err = svc.Register("neo_42", "1.2.3.4", "sess-b", now)
	if err != persist.ErrNameTaken {
		t.Fatalf("want name taken, got %v", err)
	}

	p, err := svc.Claim("neo_42", reg.Display, "9.9.9.9", "sess-c", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != reg.Player.ID {
		t.Fatal("claim should return same player")
	}
	if p.ActiveSessionID != "sess-c" {
		t.Fatalf("session = %q", p.ActiveSessionID)
	}

	_, err = svc.Claim("Neo_42", "000000000000", "9.9.9.9", "sess-d", now)
	if err != ErrBadClaim {
		t.Fatalf("want bad claim, got %v", err)
	}
}
