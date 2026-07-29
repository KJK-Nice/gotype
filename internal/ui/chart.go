package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/stats"
)

const (
	markWPM   = '━'
	markRaw   = '·'
	markConn  = '│'
	markError = '●'
)

// RenderChart draws a Monkeytype-style WPM/raw line chart with error dots.
// height is plot rows (excluding axis labels).
func RenderChart(points []stats.Point, errors []time.Duration, width, height int) string {
	if len(points) == 0 || width < 12 || height < 3 {
		return styleSub.Render("(no chart data)")
	}

	plotW := width - 5 // room for y-axis labels
	if plotW < 8 {
		plotW = 8
	}
	if height > 12 {
		height = 12
	}

	maxY := 0.0
	for _, p := range points {
		if p.WPM > maxY {
			maxY = p.WPM
		}
		if p.Raw > maxY {
			maxY = p.Raw
		}
	}
	if maxY < 10 {
		maxY = 10
	}
	maxY = math.Ceil(maxY/10) * 10

	startAt := points[0].At
	endAt := points[len(points)-1].At
	span := endAt - startAt
	if span <= 0 {
		span = time.Millisecond
	}

	wpmCol := downsample(points, plotW, func(p stats.Point) float64 { return p.WPM })
	rawCol := downsample(points, plotW, func(p stats.Point) float64 { return p.Raw })

	grid := make([][]rune, height)
	for y := range grid {
		grid[y] = make([]rune, plotW)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	plotLine(grid, rawCol, maxY, markRaw)
	plotLine(grid, wpmCol, maxY, markWPM)
	plotErrors(grid, errors, startAt, span, wpmCol, maxY)

	var b strings.Builder
	b.WriteString(styleSub.Render("wpm"))
	b.WriteString(" ")
	b.WriteString(styleMain.Render(string(markWPM)))
	b.WriteString("  ")
	b.WriteString(styleSub.Render("raw"))
	b.WriteString(" ")
	b.WriteString(styleSub.Render(string(markRaw)))
	b.WriteString("  ")
	b.WriteString(styleSub.Render("err"))
	b.WriteString(" ")
	b.WriteString(styleErrorDot.Render(string(markError)))
	b.WriteString("\n")

	for row := 0; row < height; row++ {
		yVal := maxY * (1 - float64(row)/float64(height-1))
		label := fmt.Sprintf("%3.0f", yVal)
		if row != 0 && row != height-1 {
			label = "   "
		}
		b.WriteString(styleSub.Render(label))
		b.WriteString(styleSub.Render("│"))

		var colored strings.Builder
		for _, r := range grid[row] {
			switch r {
			case markWPM, markConn:
				colored.WriteString(styleMain.Render(string(r)))
			case markRaw:
				colored.WriteString(styleSub.Render(string(r)))
			case markError:
				colored.WriteString(styleErrorDot.Render(string(r)))
			default:
				colored.WriteRune(r)
			}
		}
		b.WriteString(colored.String())
		b.WriteByte('\n')
	}

	b.WriteString(styleSub.Render("   └"))
	b.WriteString(styleSub.Render(strings.Repeat("─", plotW)))
	b.WriteByte('\n')

	first := startAt.Seconds()
	last := endAt.Seconds()
	mid := (first + last) / 2
	var axis string
	if last < 15 {
		axis = fmt.Sprintf("    %-6.1f%-6.1f%6.1f", first, mid, last)
	} else {
		axis = fmt.Sprintf("    %-6.0f%-6.0f%6.0f", first, mid, last)
	}
	b.WriteString(styleSub.Render(axis))
	b.WriteString(styleSub.Render("s"))

	return b.String()
}

func downsample(points []stats.Point, n int, get func(stats.Point) float64) []float64 {
	out := make([]float64, n)
	if len(points) == 1 {
		for i := range out {
			out[i] = get(points[0])
		}
		return out
	}
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1)
		idx := t * float64(len(points)-1)
		lo := int(math.Floor(idx))
		hi := int(math.Ceil(idx))
		if hi >= len(points) {
			hi = len(points) - 1
		}
		if lo == hi {
			out[i] = get(points[lo])
			continue
		}
		frac := idx - float64(lo)
		out[i] = get(points[lo])*(1-frac) + get(points[hi])*frac
	}
	return out
}

func plotLine(grid [][]rune, values []float64, maxY float64, mark rune) {
	h := len(grid)
	if h == 0 || len(values) == 0 || maxY <= 0 {
		return
	}
	prevRow := -1
	for x, v := range values {
		if v < 0 {
			v = 0
		}
		row := int(math.Round((1 - v/maxY) * float64(h-1)))
		if row < 0 {
			row = 0
		}
		if row >= h {
			row = h - 1
		}
		grid[row][x] = mark
		if prevRow >= 0 && prevRow != row {
			step := 1
			if row < prevRow {
				step = -1
			}
			for r := prevRow + step; r != row; r += step {
				if grid[r][x] == ' ' {
					grid[r][x] = markConn
				}
			}
		}
		prevRow = row
	}
}

// plotErrors places red dots at the WPM height for each error time.
func plotErrors(grid [][]rune, errors []time.Duration, startAt, span time.Duration, wpmCol []float64, maxY float64) {
	h := len(grid)
	plotW := len(grid[0])
	if h == 0 || plotW == 0 || len(errors) == 0 || span <= 0 {
		return
	}
	for _, at := range errors {
		t := float64(at-startAt) / float64(span)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		x := int(math.Round(t * float64(plotW-1)))
		if x < 0 {
			x = 0
		}
		if x >= plotW {
			x = plotW - 1
		}

		v := 0.0
		if x < len(wpmCol) {
			v = wpmCol[x]
		}
		row := int(math.Round((1 - v/maxY) * float64(h-1)))
		if row < 0 {
			row = 0
		}
		if row >= h {
			row = h - 1
		}
		grid[row][x] = markError
	}
}
