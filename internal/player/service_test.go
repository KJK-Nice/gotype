package player

import (
	"fmt"
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

func TestClaimWrongCodesDoNotBlockCorrectCode(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := svc.Register("Owner1", "2.2.2.2", "sess-a", now)
	if err != nil {
		t.Fatal(err)
	}
	// Attacker burns failed attempts under the limit (Register already used 1 name: slot).
	// Owner correct code must still work — unlike the old bug that rate-limited before verify.
	for i := 0; i < 8; i++ {
		_, err := svc.Claim("Owner1", "000000000000", "8.8.8.8", "atk", now.Add(time.Duration(i)*time.Second))
		if err != ErrBadClaim {
			t.Fatalf("attempt %d: want bad claim, got %v", i, err)
		}
	}
	p, err := svc.Claim("Owner1", reg.Display, "8.8.8.8", "sess-owner", now.Add(20*time.Second))
	if err != nil {
		t.Fatalf("correct code after failed attempts: %v", err)
	}
	if p.ID != reg.Player.ID || p.ActiveSessionID != "sess-owner" {
		t.Fatalf("unexpected claim result %+v", p)
	}
}

func TestClaimFailedAttemptsEventuallyRateLimit(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	reg, err := svc.Register("Owner2", "3.3.3.3", "sess-a", now)
	if err != nil {
		t.Fatal(err)
	}
	// Register already counted 1 against name:; 9 more failures fill the window (limit 10).
	// Use distinct IPs so the per-IP flood limit does not mask the name lockout.
	for i := 0; i < 9; i++ {
		ip := fmt.Sprintf("7.7.7.%d", i+1)
		_, err := svc.Claim("Owner2", "000000000000", ip, "atk", now.Add(time.Duration(i)*time.Second))
		if err != ErrBadClaim {
			t.Fatalf("attempt %d: want bad claim, got %v", i, err)
		}
	}
	_, err = svc.Claim("Owner2", "000000000000", "7.7.7.99", "atk", now.Add(30*time.Second))
	if err != ErrRateLimited {
		t.Fatalf("want rate limited after failures, got %v", err)
	}
	// Correct code is also blocked during failed-attempt lockout (by design).
	_, err = svc.Claim("Owner2", reg.Display, "7.7.7.100", "owner", now.Add(31*time.Second))
	if err != ErrRateLimited {
		t.Fatalf("want lockout for correct code too, got %v", err)
	}
}
