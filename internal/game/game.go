package game

import (
	"time"
	"unicode"

	"github.com/kjkusap/monkeytype-clone/internal/stats"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

// Mode selects how a test ends.
type Mode int

const (
	ModeTime Mode = iota
	ModeWords
)

func (m Mode) String() string {
	if m == ModeWords {
		return "words"
	}
	return "time"
}

// Config holds test settings chosen on the menu.
type Config struct {
	Mode      Mode
	Duration  time.Duration // time mode
	WordCount int           // words mode
}

var (
	TimeOptions   = []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second, 120 * time.Second}
	WordOptions   = []int{10, 25, 50, 100}
	DefaultConfig = Config{Mode: ModeTime, Duration: 30 * time.Second, WordCount: 25}
)

// CharState is the display state of one target character.
type CharState int

const (
	CharPending CharState = iota
	CharCorrect
	CharIncorrect
	CharExtra
)

// Char is one rendered glyph in the prompt.
type Char struct {
	R     rune
	State CharState
}

// Session is an in-progress typing test.
type Session struct {
	Config Config
	Words  []string
	Chars  []Char // flat prompt including spaces between words

	WordIdx int
	CharIdx int // index within current word (can exceed word length for extras)

	Typed        [][]rune // typed runes per word
	Started      bool
	Finished     bool
	Stats        stats.Calculator
	StartedAt    time.Time
	History      []stats.Point   // WPM samples for result chart
	Errors       []time.Duration // error timestamps for chart dots
	lastSampleAt time.Time
}

func NewSession(cfg Config) *Session {
	n := cfg.WordCount
	if cfg.Mode == ModeTime {
		// Extra buffer so time mode rarely runs out of words.
		n = 200
	}
	w := words.Generate(words.English, n)
	s := &Session{
		Config: cfg,
		Words:  w,
		Typed:  make([][]rune, len(w)),
	}
	s.rebuildChars()
	return s
}

func (s *Session) rebuildChars() {
	var out []Char
	for i, w := range s.Words {
		typed := s.Typed[i]
		for j, r := range w {
			st := CharPending
			if j < len(typed) {
				if typed[j] == r {
					st = CharCorrect
				} else {
					st = CharIncorrect
				}
			}
			out = append(out, Char{R: r, State: st})
		}
		// Extra typed letters beyond word length.
		for j := len(w); j < len(typed); j++ {
			out = append(out, Char{R: typed[j], State: CharExtra})
		}
		if i < len(s.Words)-1 {
			out = append(out, Char{R: ' ', State: CharPending})
		}
	}
	s.Chars = out
}

// CursorPos returns absolute index into Chars for the caret.
func (s *Session) CursorPos() int {
	pos := 0
	for i := 0; i < s.WordIdx; i++ {
		pos += len([]rune(s.Words[i]))
		extra := len(s.Typed[i]) - len([]rune(s.Words[i]))
		if extra > 0 {
			pos += extra
		}
		pos++ // space
	}
	wordRunes := []rune(s.Words[s.WordIdx])
	typed := s.Typed[s.WordIdx]
	if len(typed) <= len(wordRunes) {
		pos += len(typed)
	} else {
		pos += len(wordRunes) + (len(typed) - len(wordRunes))
	}
	return pos
}

func (s *Session) ensureStarted(now time.Time) {
	if !s.Started {
		s.Started = true
		s.StartedAt = now
		s.Stats.Start(now)
	}
}

func (s *Session) recordError(now time.Time) {
	if s.StartedAt.IsZero() {
		return
	}
	at := now.Sub(s.StartedAt)
	if at < 0 {
		at = 0
	}
	s.Errors = append(s.Errors, at)
}

// HandleRune processes a printable character (not space/backspace).
func (s *Session) HandleRune(r rune, now time.Time) {
	if s.Finished || !unicode.IsPrint(r) || r == ' ' {
		return
	}
	s.ensureStarted(now)

	word := []rune(s.Words[s.WordIdx])
	typed := s.Typed[s.WordIdx]

	if len(typed) < len(word) {
		if r == word[len(typed)] {
			s.Stats.Record(stats.StrokeCorrect, now)
		} else {
			s.Stats.Record(stats.StrokeIncorrect, now)
			s.recordError(now)
		}
	} else {
		// Cap extras to keep UI readable.
		if len(typed)-len(word) >= 8 {
			return
		}
		s.Stats.Record(stats.StrokeExtra, now)
		s.recordError(now)
	}
	s.Typed[s.WordIdx] = append(typed, r)
	s.CharIdx = len(s.Typed[s.WordIdx])
	s.rebuildChars()
	s.checkWordComplete(now)
}

// HandleSpace advances to the next word.
func (s *Session) HandleSpace(now time.Time) {
	if s.Finished {
		return
	}
	s.ensureStarted(now)

	word := []rune(s.Words[s.WordIdx])
	typed := s.Typed[s.WordIdx]

	// Missed letters count as incorrect (Monkeytype-style).
	if len(typed) < len(word) {
		missed := len(word) - len(typed)
		s.Stats.Missed += missed
		for i := 0; i < missed; i++ {
			s.Stats.Record(stats.StrokeIncorrect, now)
			s.recordError(now)
		}
	}

	if s.WordIdx >= len(s.Words)-1 {
		if s.Config.Mode == ModeWords {
			s.finish(now)
		}
		return
	}

	s.WordIdx++
	s.CharIdx = 0
	s.rebuildChars()

	if s.Config.Mode == ModeWords && s.WordIdx >= s.Config.WordCount {
		s.finish(now)
	}
}

// HandleBackspace deletes the last typed character or returns to previous word.
// Mistypes stay in accuracy forever — only correct keystrokes can be undone.
func (s *Session) HandleBackspace(now time.Time) {
	if s.Finished || !s.Started {
		return
	}

	typed := s.Typed[s.WordIdx]
	if len(typed) > 0 {
		last := typed[len(typed)-1]
		word := []rune(s.Words[s.WordIdx])
		idx := len(typed) - 1
		// Extra / incorrect: remove from prompt only. Error keystroke stays counted.
		if idx < len(word) && last == word[idx] {
			s.Stats.Unrecord(stats.StrokeCorrect)
		}
		s.Typed[s.WordIdx] = typed[:len(typed)-1]
		s.CharIdx = len(s.Typed[s.WordIdx])
		s.rebuildChars()
		return
	}

	// Jump back to previous word if current is empty.
	if s.WordIdx == 0 {
		return
	}
	s.WordIdx--
	s.CharIdx = len(s.Typed[s.WordIdx])
	s.rebuildChars()
}

func (s *Session) checkWordComplete(now time.Time) {
	if s.Config.Mode != ModeWords {
		return
	}
	// Auto-finish when last word fully correct (optional nicety).
	if s.WordIdx == s.Config.WordCount-1 {
		word := s.Words[s.WordIdx]
		typed := string(s.Typed[s.WordIdx])
		if typed == word {
			s.finish(now)
		}
	}
}

// Tick samples chart data and advances time mode.
// Returns true when the test just finished.
func (s *Session) Tick(now time.Time) bool {
	if s.Finished || !s.Started {
		return false
	}
	s.Sample(now)
	if s.Config.Mode != ModeTime {
		return false
	}
	if now.Sub(s.StartedAt) >= s.Config.Duration {
		s.finish(now)
		return true
	}
	return false
}

const sampleInterval = 100 * time.Millisecond

// Sample records a burst-WPM point on a subsecond cadence (~10 Hz).
// Chart uses rolling 1s window (not cumulative avg — that looks flat).
func (s *Session) Sample(now time.Time) {
	if !s.Started {
		return
	}
	elapsed := now.Sub(s.StartedAt)
	if elapsed < sampleInterval && !s.Finished {
		return
	}
	pt := s.Stats.BurstPoint(now)
	if len(s.History) > 0 && !s.lastSampleAt.IsZero() && now.Sub(s.lastSampleAt) < sampleInterval {
		s.History[len(s.History)-1] = pt
		return
	}
	s.lastSampleAt = now
	s.History = append(s.History, pt)
}

func (s *Session) finish(now time.Time) {
	s.Finished = true
	s.Stats.Finish(now)
	s.Sample(now)
	if len(s.History) == 0 {
		s.History = []stats.Point{
			{At: 0, WPM: 0, Raw: 0},
			s.Stats.BurstPoint(now),
		}
		return
	}
	if len(s.History) == 1 {
		end := s.History[0]
		s.History = []stats.Point{{At: 0, WPM: 0, Raw: 0}, end}
	}
}

func (s *Session) Snapshot(now time.Time) stats.Snapshot {
	return s.Stats.Snapshot(now)
}

// Remaining returns time left in time mode.
func (s *Session) Remaining(now time.Time) time.Duration {
	if s.Config.Mode != ModeTime {
		return 0
	}
	if !s.Started {
		return s.Config.Duration
	}
	left := s.Config.Duration - now.Sub(s.StartedAt)
	if left < 0 {
		return 0
	}
	return left
}

// ProgressLabel for the HUD (timer or word count).
func (s *Session) ProgressLabel(now time.Time) string {
	if s.Config.Mode == ModeTime {
		return FormatSeconds(s.Remaining(now))
	}
	return s.WordsProgress()
}

// FormatSeconds renders duration as whole seconds.
func FormatSeconds(d time.Duration) string {
	sec := int(d.Round(time.Second).Seconds())
	if sec < 0 {
		sec = 0
	}
	return itoa(sec)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// WordsProgress returns "current/total" for words mode.
func (s *Session) WordsProgress() string {
	cur := s.WordIdx
	if s.Finished {
		cur = s.Config.WordCount
	}
	return itoa(cur) + "/" + itoa(s.Config.WordCount)
}
