package ui

import (
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

// GhostSample is one point on a pace-ghost timeline.
type GhostSample struct {
	At    time.Duration
	Chars int
}

// PaceGhost replays prior race progress as a dim caret target.
type PaceGhost []GhostSample

func (g PaceGhost) CharsAt(elapsed time.Duration) int {
	if len(g) == 0 {
		return 0
	}
	if elapsed <= g[0].At {
		return g[0].Chars
	}
	for i := 1; i < len(g); i++ {
		if elapsed < g[i].At {
			a, b := g[i-1], g[i]
			span := b.At - a.At
			if span <= 0 {
				return b.Chars
			}
			t := float64(elapsed-a.At) / float64(span)
			return a.Chars + int(t*float64(b.Chars-a.Chars))
		}
	}
	return g[len(g)-1].Chars
}

// indexForProgressChars maps ProgressChars count onto flat Chars index.
func indexForProgressChars(chars []game.Char, n int) int {
	if n <= 0 || len(chars) == 0 {
		return 0
	}
	count := 0
	for i := range chars {
		count++
		if count >= n {
			return i
		}
	}
	return len(chars) - 1
}
