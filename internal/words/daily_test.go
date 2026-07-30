package words

import (
	"testing"
	"time"
)

func TestDailySeedStableSameUTCDay(t *testing.T) {
	a := time.Date(2026, 7, 29, 1, 0, 0, 0, time.UTC)
	b := time.Date(2026, 7, 29, 23, 59, 0, 0, time.UTC)
	if DailySeed(a) != DailySeed(b) {
		t.Fatal("same UTC day must match")
	}
	c := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	if DailySeed(a) == DailySeed(c) {
		t.Fatal("next day must differ")
	}
}

func TestDailySeedTimezoneNormalized(t *testing.T) {
	loc := time.FixedZone("UTC+7", 7*3600)
	// 2026-07-30 01:00 +7 == 2026-07-29 18:00 UTC
	local := time.Date(2026, 7, 30, 1, 0, 0, 0, loc)
	utc := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	if DailySeed(local) != DailySeed(utc) {
		t.Fatalf("got %d want %d", DailySeed(local), DailySeed(utc))
	}
}
