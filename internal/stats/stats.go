package stats

import "time"

// StrokeKind classifies one recorded keystroke for chart math.
type StrokeKind int

const (
	StrokeCorrect StrokeKind = iota
	StrokeIncorrect
	StrokeExtra
)

// Stroke is one counted keystroke with a timestamp.
type Stroke struct {
	At   time.Time
	Kind StrokeKind
}

// Point is one WPM sample along the test timeline.
type Point struct {
	At  time.Duration
	WPM float64
	Raw float64
}

// Snapshot holds live or final typing metrics.
type Snapshot struct {
	WPM       float64
	RawWPM    float64
	Accuracy  float64
	Correct   int
	Incorrect int
	Extra     int
	Missed    int
	Elapsed   time.Duration
}

// Calculator tracks keystroke outcomes for WPM/accuracy.
type Calculator struct {
	Correct   int
	Incorrect int
	Extra     int
	Missed    int
	Strokes   []Stroke
	StartedAt time.Time
	endedAt   time.Time
	finished  bool
}

func (c *Calculator) Start(now time.Time) {
	if c.StartedAt.IsZero() {
		c.StartedAt = now
	}
}

func (c *Calculator) Finish(now time.Time) {
	if !c.finished {
		c.endedAt = now
		c.finished = true
	}
}

func (c *Calculator) elapsed(now time.Time) time.Duration {
	if c.StartedAt.IsZero() {
		return 0
	}
	end := now
	if c.finished {
		end = c.endedAt
	}
	d := end.Sub(c.StartedAt)
	if d < time.Millisecond {
		d = time.Millisecond
	}
	return d
}

// Record appends a stroke and bumps the matching counter.
func (c *Calculator) Record(kind StrokeKind, now time.Time) {
	c.Strokes = append(c.Strokes, Stroke{At: now, Kind: kind})
	switch kind {
	case StrokeCorrect:
		c.Correct++
	case StrokeIncorrect:
		c.Incorrect++
	case StrokeExtra:
		c.Extra++
	}
}

// Unrecord pops the last stroke of kind (backspace). Returns false if none.
func (c *Calculator) Unrecord(kind StrokeKind) bool {
	for i := len(c.Strokes) - 1; i >= 0; i-- {
		if c.Strokes[i].Kind != kind {
			continue
		}
		c.Strokes = append(c.Strokes[:i], c.Strokes[i+1:]...)
		switch kind {
		case StrokeCorrect:
			c.Correct--
			if c.Correct < 0 {
				c.Correct = 0
			}
		case StrokeIncorrect:
			c.Incorrect--
			if c.Incorrect < 0 {
				c.Incorrect = 0
			}
		case StrokeExtra:
			c.Extra--
			if c.Extra < 0 {
				c.Extra = 0
			}
		}
		return true
	}
	return false
}

// Snapshot computes Monkeytype-style cumulative metrics (HUD / final stats).
func (c *Calculator) Snapshot(now time.Time) Snapshot {
	elapsed := c.elapsed(now)
	minutes := elapsed.Seconds() / 60.0
	if minutes <= 0 {
		minutes = 1.0 / 60.0
	}

	typed := c.Correct + c.Incorrect + c.Extra
	accDenom := typed
	acc := 0.0
	if accDenom > 0 {
		acc = float64(c.Correct) / float64(accDenom) * 100.0
	} else if c.finished {
		// No keystrokes in a finished test — not a perfect score.
		acc = 0.0
	} else {
		// Live, nothing typed yet: show 100% until first stroke (solo HUD nicety).
		acc = 100.0
	}

	return Snapshot{
		WPM:       float64(c.Correct) / 5.0 / minutes,
		RawWPM:    float64(typed) / 5.0 / minutes,
		Accuracy:  acc,
		Correct:   c.Correct,
		Incorrect: c.Incorrect,
		Extra:     c.Extra,
		Missed:    c.Missed,
		Elapsed:   elapsed,
	}
}

// BurstWindow is the rolling window used for chart WPM (Monkeytype-style).
const BurstWindow = time.Second

// BurstPoint computes instantaneous WPM/raw over a recent window.
// This is what the result chart should plot — cumulative average looks flat.
func (c *Calculator) BurstPoint(now time.Time) Point {
	elapsed := c.elapsed(now)
	window := BurstWindow
	if elapsed < window {
		window = elapsed
	}
	if window < time.Millisecond {
		window = time.Millisecond
	}

	from := now.Add(-window)
	correct, raw := 0, 0
	for i := len(c.Strokes) - 1; i >= 0; i-- {
		st := c.Strokes[i]
		if st.At.Before(from) {
			break
		}
		raw++
		if st.Kind == StrokeCorrect {
			correct++
		}
	}

	minutes := window.Seconds() / 60.0
	return Point{
		At:  elapsed,
		WPM: float64(correct) / 5.0 / minutes,
		Raw: float64(raw) / 5.0 / minutes,
	}
}
