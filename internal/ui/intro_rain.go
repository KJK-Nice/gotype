package ui

import (
	"math/rand"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	introFloor    = 400 * time.Millisecond
	introDurA     = 2500 * time.Millisecond // heavy rain over home
	introDurB     = 1500 * time.Millisecond // horizon wipe top→bottom
	introTotal    = introDurA + introDurB   // 4s
	introTrailMin = 4
	introTrailMax = 12
	introDenseW   = 100 // width ≤ this → stride 1
)

// ASCII soup for amber rain overlay.
const introGlyphs = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz@#$%&*"

// Fixed amber rain (default theme brand) — head / mid / dim.
var (
	introHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe066")).Bold(true)
	introMidStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2b714"))
	introDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5c4e12"))
)

type introStage int

const (
	introStageRain introStage = iota
	introStageClear
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
	w, h    int
	stride  int
	cols    []rainCol
	rng     *rand.Rand
}

func introStride(w int) int {
	if w <= introDenseW {
		return 1
	}
	return 2
}

func newIntroRain(w, h int, seed int64) *introRain {
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	r := &introRain{
		w:      w,
		h:      h,
		stride: introStride(w),
		rng:    rand.New(rand.NewSource(seed)),
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
	r.stride = introStride(w)
	r.layout()
}

func (r *introRain) layout() {
	stride := r.stride
	if stride < 1 {
		stride = 2
	}
	n := (r.w + stride - 1) / stride
	if n < 1 {
		n = 1
	}
	r.cols = make([]rainCol, n)
	for i := 0; i < n; i++ {
		r.cols[i] = r.spawnCol(i * stride)
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
		return introStageClear, float64(elapsed-introDurA) / float64(introDurB)
	default:
		return introStageRain, float64(elapsed) / float64(introDurA)
	}
}

// clearY is the horizon: rows y < clearY are home; y >= clearY keep rain.
func clearY(stage introStage, stageProg float64, h int) int {
	switch stage {
	case introStageRain:
		return 0
	case introStageClear:
		if stageProg < 0 {
			stageProg = 0
		}
		if stageProg > 1 {
			stageProg = 1
		}
		y := int(stageProg*float64(h) + 0.5)
		if y > h {
			return h
		}
		return y
	default:
		return h
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

// stampRainOver paints rain over home below the horizon (y >= clearY).
func stampRainOver(home string, rain *introRain, clear int, w, h int) string {
	if rain == nil || clear >= h {
		return home
	}
	grid := parseANSIGrid(home, w, h)
	for y := clear; y < h; y++ {
		for x := 0; x < w; x++ {
			ch, bright := rain.cell(x, y)
			if bright == 0 {
				grid[y][x] = " "
				continue
			}
			grid[y][x] = styleRainGlyph(ch, bright)
		}
	}
	var b strings.Builder
	b.Grow(w * h * 8)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.WriteString(grid[y][x])
		}
		if y < h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
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
