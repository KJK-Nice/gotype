package ui

import (
	"testing"
	"time"
)

func TestPaceGhostInterp(t *testing.T) {
	g := PaceGhost{
		{At: 0, Chars: 0},
		{At: time.Second, Chars: 10},
		{At: 2 * time.Second, Chars: 20},
	}
	if g.CharsAt(500*time.Millisecond) != 5 {
		t.Fatalf("got %d", g.CharsAt(500*time.Millisecond))
	}
	if g.CharsAt(2*time.Second) != 20 {
		t.Fatalf("got %d", g.CharsAt(2*time.Second))
	}
}
