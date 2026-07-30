package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	wlc "github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"

	"github.com/kjkusap/monkeytype-clone/internal/stats"
)

const markError = '●'

// RenderChart draws WPM + raw wave lines via ntcharts, with error dots overlaid.
func RenderChart(sty Styles, points []stats.Point, errors []time.Duration, width, height int) string {
	if len(points) == 0 || width < 16 || height < 5 {
		return sty.Sub.Render("(no chart data)")
	}
	if height > 14 {
		height = 14
	}

	maxY := 0.0
	maxX := 0.0
	for _, p := range points {
		if p.WPM > maxY {
			maxY = p.WPM
		}
		if p.Raw > maxY {
			maxY = p.Raw
		}
		sec := p.At.Seconds()
		if sec > maxX {
			maxX = sec
		}
	}
	if maxY < 10 {
		maxY = 10
	}
	maxY = math.Ceil(maxY/10) * 10
	if maxX <= 0 {
		maxX = 1
	}

	chart := wlc.New(width, height,
		wlc.WithXYRange(0, maxX, 0, maxY),
	)
	chart.AxisStyle = sty.Sub
	chart.LabelStyle = sty.Sub
	chart.XLabelFormatter = secondsLabelFormatter()
	chart.YLabelFormatter = linechart.DefaultLabelFormatter()
	chart.SetXStep(max(1, width/12))
	chart.SetYStep(2)

	chart.SetStyles(runes.ArcLineStyle, sty.Main.UnsetBold())
	chart.SetDataSetStyles("raw", runes.ArcLineStyle, sty.Sub)

	for _, p := range points {
		x := p.At.Seconds()
		chart.Plot(canvas.Float64Point{X: x, Y: p.WPM})
		chart.PlotDataSet("raw", canvas.Float64Point{X: x, Y: p.Raw})
	}
	chart.DrawAll()

	// Error markers on top of wave (Y = WPM at that instant).
	for _, at := range errors {
		x := at.Seconds()
		if x < 0 {
			x = 0
		}
		if x > maxX {
			x = maxX
		}
		y := wpmAt(points, at)
		if y < 0 {
			y = 0
		}
		if y > maxY {
			y = maxY
		}
		chart.DrawRuneWithStyle(canvas.Float64Point{X: x, Y: y}, markError, sty.ErrorDot)
	}

	var b strings.Builder
	b.WriteString(sty.Sub.Render("wpm "))
	b.WriteString(sty.Main.Render("━"))
	b.WriteString(sty.Sub.Render("  raw "))
	b.WriteString(sty.Sub.Render("·"))
	b.WriteString(sty.Sub.Render("  err "))
	b.WriteString(sty.ErrorDot.Render(string(markError)))
	b.WriteString("\n")
	b.WriteString(chart.View())
	return b.String()
}

func wpmAt(points []stats.Point, at time.Duration) float64 {
	if len(points) == 0 {
		return 0
	}
	if at <= points[0].At {
		return points[0].WPM
	}
	last := points[len(points)-1]
	if at >= last.At {
		return last.WPM
	}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if at <= b.At {
			span := float64(b.At - a.At)
			if span <= 0 {
				return b.WPM
			}
			t := float64(at-a.At) / span
			return a.WPM*(1-t) + b.WPM*t
		}
	}
	return last.WPM
}

func secondsLabelFormatter() linechart.LabelFormatter {
	return func(_ int, v float64) string {
		if v < 10 {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.0f", v)
	}
}
