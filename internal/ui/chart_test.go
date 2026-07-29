package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/stats"
)

func TestRenderChart(t *testing.T) {
	points := []stats.Point{
		{At: time.Second, WPM: 40, Raw: 45},
		{At: 2 * time.Second, WPM: 55, Raw: 60},
		{At: 3 * time.Second, WPM: 70, Raw: 72},
		{At: 4 * time.Second, WPM: 65, Raw: 80},
	}
	errors := []time.Duration{1500 * time.Millisecond, 3 * time.Second}
	out := RenderChart(points, errors, 40, 6)
	if !strings.Contains(out, "│") {
		t.Fatalf("expected axis in chart, got:\n%s", out)
	}
	if !strings.Contains(out, "wpm") {
		t.Fatalf("expected legend, got:\n%s", out)
	}
	if !strings.Contains(out, "err") {
		t.Fatalf("expected err legend, got:\n%s", out)
	}
	if !strings.Contains(out, "●") {
		t.Fatalf("expected error dots, got:\n%s", out)
	}
}

func TestRenderChartEmpty(t *testing.T) {
	out := RenderChart(nil, nil, 40, 6)
	if !strings.Contains(out, "no chart") {
		t.Fatalf("expected empty message, got %q", out)
	}
}
