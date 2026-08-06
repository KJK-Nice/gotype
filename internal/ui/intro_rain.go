package ui

import (
	"math/rand"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	introTitle     = "gotype"
	introFloor     = 400 * time.Millisecond
	introDurA      = 1600 * time.Millisecond // full-screen rain
	introDurB      = 2400 * time.Millisecond // rain + assemble title
	introTotal     = introDurA + introDurB // 4s, then instant home
	introTrailMin  = 4
	introTrailMax  = 12
	introColStride = 2
)

// ASCII soup for rain; title locks to real home sty.Title chars.
const introGlyphs = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz@#$%&*"

var (
	introHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c8ffc8")).Bold(true)
	introMidStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff41"))
	introDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#003b00"))
)

type introStage int

const (
	introStageRain introStage = iota
	introStageAssemble
	introStageDone
)

type rainCol struct {
	x      int
	head   float64
	speed  float64
	length int
	glyphs []rune
}

type introRain struct {
	w, h int
	cols []rainCol
	rng  *rand.Rand
}

func newIntroRain(w, h int, seed int64) *introRain {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	r := &introRain{
		w:   w,
		h:   h,
		rng: rand.New(rand.NewSource(seed)),
	}
	r.layout()
	return r
}

func (r *introRain) rebuild(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	r.w, r.h = w, h
	r.layout()
}

func (r *introRain) layout() {
	n := (r.w + introColStride - 1) / introColStride
	if n < 1 {
		n = 1
	}
	r.cols = make([]rainCol, n)
	for i := 0; i < n; i++ {
		r.cols[i] = r.spawnCol(i * introColStride)
	}
}

func (r *introRain) spawnCol(x int) rainCol {
	length := introTrailMin + r.rng.Intn(introTrailMax-introTrailMin+1)
	glyphs := make([]rune, length)
	for i := range glyphs {
		glyphs[i] = rune(introGlyphs[r.rng.Intn(len(introGlyphs))])
	}
	return rainCol{
		x:      x,
		head:   r.rng.Float64() * float64(r.h+length),
		speed:  8 + r.rng.Float64()*18,
		length: length,
		glyphs: glyphs,
	}
}

func (r *introRain) step(dt time.Duration) {
	if r == nil || len(r.cols) == 0 {
		return
	}
	sec := dt.Seconds()
	if sec <= 0 {
		return
	}
	for i := range r.cols {
		c := &r.cols[i]
		c.head += c.speed * sec
		if r.rng.Float64() < 0.15 {
			c.glyphs[r.rng.Intn(len(c.glyphs))] = rune(introGlyphs[r.rng.Intn(len(introGlyphs))])
		}
		if c.head-float64(c.length) > float64(r.h) {
			*c = r.spawnCol(c.x)
			c.head = -float64(c.length) * r.rng.Float64()
		}
	}
}

func (r *introRain) cell(x, y int) (rune, int) {
	if r == nil {
		return ' ', 0
	}
	bestCh := ' '
	bestB := 0
	for i := range r.cols {
		c := &r.cols[i]
		if c.x != x {
			continue
		}
		headY := int(c.head)
		dist := headY - y
		if dist < 0 || dist >= c.length {
			continue
		}
		b := 1
		switch {
		case dist == 0:
			b = 3
		case dist <= 2:
			b = 2
		}
		if b > bestB {
			bestB = b
			bestCh = c.glyphs[dist%len(c.glyphs)]
		}
	}
	return bestCh, bestB
}

func introProgress(elapsed time.Duration) (introStage, float64) {
	if elapsed < 0 {
		elapsed = 0
	}
	switch {
	case elapsed >= introTotal:
		return introStageDone, 1
	case elapsed >= introDurA:
		return introStageAssemble, float64(elapsed-introDurA) / float64(introDurB)
	default:
		return introStageRain, float64(elapsed) / float64(introDurA)
	}
}

func titleLockCount(stage introStage, stageProg float64) int {
	switch stage {
	case introStageRain:
		return 0
	case introStageAssemble:
		if stageProg < 0 {
			stageProg = 0
		}
		if stageProg > 1 {
			stageProg = 1
		}
		n := int(stageProg*float64(len(introTitle)) + 0.5)
		if n > len(introTitle) {
			n = len(introTitle)
		}
		return n
	default:
		return len(introTitle)
	}
}

func styleRainGlyph(ch rune, bright int) string {
	if bright == 0 {
		return " "
	}
	s := string(ch)
	switch bright {
	case 3:
		return introHeadStyle.Render(s)
	case 2:
		return introMidStyle.Render(s)
	default:
		return introDimStyle.Render(s)
	}
}

// renderScene paints rain on empty screen; locked title chars use home Title style.
func (r *introRain) renderScene(lockN, titleRow, titleCol int, titleStyle lipgloss.Style) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(r.w * r.h * 8)
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			if y == titleRow && x >= titleCol && x < titleCol+lockN {
				ch := introTitle[x-titleCol]
				b.WriteString(titleStyle.Render(string(ch)))
				continue
			}
			ch, bright := r.cell(x, y)
			if bright == 0 {
				b.WriteByte(' ')
				continue
			}
			b.WriteString(styleRainGlyph(ch, bright))
		}
		if y < r.h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// findTitleAnchor locates where "gotype" starts in a placed home frame.
func findTitleAnchor(placed string, w, h int) (row, col int) {
	grid := parseANSIGrid(placed, w, h)
	for y := 0; y < h; y++ {
		for x := 0; x <= w-len(introTitle); x++ {
			ok := true
			for i := 0; i < len(introTitle); i++ {
				if cellRune(grid[y][x+i]) != rune(introTitle[i]) {
					ok = false
					break
				}
			}
			if ok {
				return y, x
			}
		}
	}
	return h / 3, max(0, (w-len(introTitle))/2)
}

func cellRune(cell string) rune {
	plain := stripANSI(cell)
	if plain == "" {
		return ' '
	}
	return rune(plain[len(plain)-1])
}

// parseANSIGrid splits a screen string into w×h display cells (SGR + rune).
func parseANSIGrid(s string, w, h int) [][]string {
	grid := make([][]string, h)
	for y := 0; y < h; y++ {
		grid[y] = make([]string, w)
		for x := 0; x < w; x++ {
			grid[y][x] = " "
		}
	}
	x, y := 0, 0
	var sgr strings.Builder
	inEsc := false
	escBuf := strings.Builder{}
	for i := 0; i < len(s) && y < h; i++ {
		c := s[i]
		if inEsc {
			escBuf.WriteByte(c)
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				seq := escBuf.String()
				inEsc = false
				escBuf.Reset()
				if strings.HasSuffix(seq, "m") {
					if seq == "\x1b[0m" || seq == "\x1b[m" {
						sgr.Reset()
					} else {
						sgr.WriteString(seq)
					}
				}
			}
			continue
		}
		if c == 0x1b {
			inEsc = true
			escBuf.Reset()
			escBuf.WriteByte(c)
			continue
		}
		if c == '\n' {
			x = 0
			y++
			continue
		}
		if c == '\r' {
			x = 0
			continue
		}
		if x >= w {
			continue
		}
		ch := string(c)
		if sgr.Len() > 0 {
			grid[y][x] = sgr.String() + ch + "\x1b[0m"
		} else {
			grid[y][x] = ch
		}
		x++
	}
	return grid
}
