package words

import (
	"fmt"
	"time"
)

// DailySeed is a UTC-date seed so everyone shares today's prompt.
func DailySeed(t time.Time) uint64 {
	d := t.UTC()
	// Stable mix of Y-M-D (nonzero).
	y, m, day := uint64(d.Year()), uint64(d.Month()), uint64(d.Day())
	seed := y*10000 + m*100 + day
	seed ^= 0x670747450444149 // "gotypeDI" nibbles
	if seed == 0 {
		seed = 1
	}
	return seed
}

// DailyLabel is a short UTC date for UI (e.g. 2026-07-29).
func DailyLabel(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// DailyHeadline for menus / results.
func DailyHeadline(t time.Time) string {
	return fmt.Sprintf("daily · %s", DailyLabel(t))
}
