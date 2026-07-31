package progress

import (
	"testing"
	"time"
)

func TestGrantXPDailyCap(t *testing.T) {
	day := time.Date(2026, 7, 31, 15, 0, 0, 0, time.UTC)
	got, dayTotal, err := ApplyXPGrant(0, 180, SoloFinishXP, day)
	if err != nil {
		t.Fatal(err)
	}
	if got != SoloFinishXP {
		t.Fatalf("granted = %d, want %d", got, SoloFinishXP)
	}
	if dayTotal != 190 {
		t.Fatalf("dayTotal = %d, want 190", dayTotal)
	}

	// Cap remaining 10 of 200
	got, dayTotal, err = ApplyXPGrant(0, 190, MultiFinishXP, day)
	if err != nil {
		t.Fatal(err)
	}
	if got != 10 {
		t.Fatalf("granted = %d, want 10 (soft cap remainder)", got)
	}
	if dayTotal != DailyXPCap {
		t.Fatalf("dayTotal = %d, want %d", dayTotal, DailyXPCap)
	}

	got, dayTotal, err = ApplyXPGrant(0, DailyXPCap, SoloFinishXP, day)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("over cap should grant 0, got %d", got)
	}
	if dayTotal != DailyXPCap {
		t.Fatalf("dayTotal = %d", dayTotal)
	}
}

func TestTierFromXP(t *testing.T) {
	if TierFromXP(0) != 0 {
		t.Fatal("0 xp → tier 0")
	}
	if TierFromXP(99) != 0 {
		t.Fatal("99 xp → tier 0")
	}
	if TierFromXP(100) != 1 {
		t.Fatal("100 xp → tier 1")
	}
	if TierFromXP(2000) != 20 {
		t.Fatal("2000 xp → tier 20")
	}
	if TierFromXP(9999) != 20 {
		t.Fatal("cap at 20")
	}
}

func TestIncompleteGrantsZero(t *testing.T) {
	day := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	got, _, err := ApplyXPGrant(0, 0, 0, day)
	if err != nil || got != 0 {
		t.Fatalf("incomplete grant = %d err=%v", got, err)
	}
}
