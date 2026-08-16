package stats

import (
	"testing"
	"time"
)

func TestSnapshotWPM(t *testing.T) {
	c := Calculator{}
	start := time.Unix(0, 0)
	c.Start(start)
	for i := 0; i < 50; i++ {
		c.Record(StrokeCorrect, start.Add(time.Duration(i)*time.Millisecond))
	}
	c.Finish(start.Add(60 * time.Second))

	snap := c.Snapshot(start.Add(60 * time.Second))
	if snap.WPM < 9.9 || snap.WPM > 10.1 {
		t.Fatalf("WPM = %v, want ~10", snap.WPM)
	}
}

func TestAccuracy(t *testing.T) {
	c := Calculator{}
	start := time.Unix(0, 0)
	c.Start(start)
	for i := 0; i < 80; i++ {
		c.Record(StrokeCorrect, start)
	}
	for i := 0; i < 20; i++ {
		c.Record(StrokeIncorrect, start)
	}
	snap := c.Snapshot(start.Add(time.Second))
	if snap.Accuracy < 79.9 || snap.Accuracy > 80.1 {
		t.Fatalf("Accuracy = %v, want ~80", snap.Accuracy)
	}
}

func TestBurstPointVariesWithPace(t *testing.T) {
	c := Calculator{}
	start := time.Unix(0, 0)
	c.Start(start)

	// Slow first second: 2 correct chars → 24 wpm over 1s window.
	c.Record(StrokeCorrect, start.Add(200*time.Millisecond))
	c.Record(StrokeCorrect, start.Add(800*time.Millisecond))
	slow := c.BurstPoint(start.Add(time.Second))

	// Fast burst: many chars in latest second.
	for i := 0; i < 20; i++ {
		c.Record(StrokeCorrect, start.Add(time.Second+time.Duration(i)*40*time.Millisecond))
	}
	fast := c.BurstPoint(start.Add(2 * time.Second))

	if fast.WPM <= slow.WPM {
		t.Fatalf("burst WPM should rise with pace: slow=%.1f fast=%.1f", slow.WPM, fast.WPM)
	}
}

func TestEmptyFinishedAccuracy(t *testing.T) {
	c := Calculator{}
	start := time.Unix(0, 0)
	c.Start(start)
	c.Finish(start.Add(time.Second))
	snap := c.Snapshot(start.Add(time.Second))
	if snap.Accuracy != 0 {
		t.Fatalf("empty finished Accuracy = %v, want 0", snap.Accuracy)
	}
}

func TestComboBuildsAndBreaks(t *testing.T) {
	c := Calculator{}
	now := time.Unix(0, 0)
	for i := 0; i < 12; i++ {
		c.Record(StrokeCorrect, now)
	}
	if c.Combo != 12 || c.BestCombo != 12 {
		t.Fatalf("combo=%d best=%d", c.Combo, c.BestCombo)
	}
	c.Record(StrokeIncorrect, now)
	if c.Combo != 0 {
		t.Fatalf("combo after miss = %d", c.Combo)
	}
	if c.BestCombo != 12 {
		t.Fatalf("best combo should stick, got %d", c.BestCombo)
	}
	c.Record(StrokeCorrect, now)
	c.Record(StrokeExtra, now)
	if c.Combo != 0 {
		t.Fatalf("combo after extra = %d", c.Combo)
	}
}

func TestComboUnrecordDecrements(t *testing.T) {
	c := Calculator{}
	now := time.Unix(0, 0)
	c.Record(StrokeCorrect, now)
	c.Record(StrokeCorrect, now)
	if !c.Unrecord(StrokeCorrect) {
		t.Fatal("unrecord")
	}
	if c.Combo != 1 {
		t.Fatalf("combo after backspace = %d", c.Combo)
	}
	c.Record(StrokeIncorrect, now)
	c.Unrecord(StrokeCorrect)
	if c.Combo != 0 {
		t.Fatalf("unrecord after miss should not revive combo, got %d", c.Combo)
	}
}

func TestRecordWordChain(t *testing.T) {
	c := Calculator{}
	c.RecordWord(true)
	c.RecordWord(true)
	if c.Chain != 2 || c.BestChain != 2 {
		t.Fatalf("chain=%d best=%d", c.Chain, c.BestChain)
	}
	c.RecordWord(false)
	if c.Chain != 0 {
		t.Fatalf("chain after miss = %d", c.Chain)
	}
	if c.BestChain != 2 {
		t.Fatalf("best chain should stick, got %d", c.BestChain)
	}
}

func TestSnapshotIncludesCombo(t *testing.T) {
	c := Calculator{}
	now := time.Unix(0, 0)
	c.Start(now)
	c.Record(StrokeCorrect, now)
	c.RecordWord(true)
	snap := c.Snapshot(now.Add(time.Second))
	if snap.Combo != 1 || snap.BestCombo != 1 || snap.Chain != 1 || snap.BestChain != 1 {
		t.Fatalf("snap combo/chain %+v", snap)
	}
}
