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
