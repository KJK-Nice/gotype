package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestIntroProgressStages(t *testing.T) {
	cases := []struct {
		at    time.Duration
		stage introStage
	}{
		{0, introStageRain},
		{introDurA - time.Millisecond, introStageRain},
		{introDurA, introStageAssemble},
		{introTotal - time.Millisecond, introStageAssemble},
		{introTotal, introStageDone},
	}
	for _, tc := range cases {
		stage, _ := introProgress(tc.at)
		if stage != tc.stage {
			t.Fatalf("at %v: stage=%d want %d", tc.at, stage, tc.stage)
		}
	}
}

func TestTitleLockCount(t *testing.T) {
	if n := titleLockCount(introStageRain, 0.9); n != 0 {
		t.Fatalf("rain lock=%d", n)
	}
	if n := titleLockCount(introStageAssemble, 0); n != 0 {
		t.Fatalf("assemble start lock=%d", n)
	}
	if n := titleLockCount(introStageAssemble, 1); n != len(introTitle) {
		t.Fatalf("assemble end lock=%d want %d", n, len(introTitle))
	}
}

func TestNewStartsIntro(t *testing.T) {
	m := New()
	if m.phase != phaseIntro {
		t.Fatalf("phase=%d want intro", m.phase)
	}
	if m.introRain == nil {
		t.Fatal("expected rain grid")
	}
}

func TestAutoSpectateSkipsIntro(t *testing.T) {
	m := NewWithOptions(Options{AutoSpectate: true})
	if m.phase != phaseConfig {
		t.Fatalf("phase=%d want config", m.phase)
	}
	if m.introRain != nil {
		t.Fatal("demo must not allocate rain")
	}
}

func TestIntroSkipFloor(t *testing.T) {
	m := New()
	m.now = m.introAt.Add(introFloor - time.Millisecond)
	next, _ := m.updateIntro(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := next.(Model)
	if nm.phase != phaseIntro {
		t.Fatal("skip before floor")
	}

	nm.now = nm.introAt.Add(introFloor)
	next, _ = nm.updateIntro(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm = next.(Model)
	if nm.phase != phaseConfig {
		t.Fatalf("phase=%d want config after soft skip", nm.phase)
	}
	if nm.introRain != nil {
		t.Fatal("rain should clear")
	}
}

func TestIntroQSkipsNotQuits(t *testing.T) {
	m := New()
	m.now = m.introAt.Add(introFloor)
	next, cmd := m.updateIntro(tea.KeyPressMsg{Text: "q", Code: 'q'})
	nm := next.(Model)
	if nm.phase != phaseConfig {
		t.Fatal("q should soft-skip")
	}
	if cmd != nil {
		t.Fatalf("unexpected cmd on q-skip")
	}
}

func TestIntroCtrlDQuits(t *testing.T) {
	m := New()
	_, cmd := m.updateIntro(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

func TestIntroFinishesOnTotal(t *testing.T) {
	m := New()
	m.now = m.introAt.Add(introTotal)
	m.introLast = m.now
	m.stepIntro()
	if m.phase != phaseConfig {
		t.Fatalf("phase=%d want config", m.phase)
	}
}

func TestIntroResizeRebuilds(t *testing.T) {
	m := New()
	m.width = 40
	m.height = 12
	m.rebuildIntroRain()
	if m.introRain.w != 40 || m.introRain.h != 12 {
		t.Fatalf("size=%dx%d", m.introRain.w, m.introRain.h)
	}
	wantCols := (40 + introColStride - 1) / introColStride
	if len(m.introRain.cols) != wantCols {
		t.Fatalf("cols=%d want %d", len(m.introRain.cols), wantCols)
	}
}

func TestIntroAssemblesTitle(t *testing.T) {
	m := New()
	m.now = m.introAt.Add(introDurA + introDurB - time.Millisecond)
	out := m.viewIntro()
	plain := stripANSI(out)
	if !strings.Contains(plain, introTitle) {
		t.Fatalf("expected assembled title, got %q", plain[:min(200, len(plain))])
	}
	// Rain phase — no home menu yet.
	m.now = m.introAt.Add(introDurA / 2)
	rainOnly := stripANSI(m.viewIntro())
	if strings.Contains(rainOnly, "typing races") {
		t.Fatal("home menu should not show during rain")
	}
}

func TestFindTitleAnchor(t *testing.T) {
	sty := NewStyles(0)
	body := sty.Box.Render("gotype\nsubtitle")
	placed := lipgloss.Place(40, 12, lipgloss.Center, lipgloss.Center, body)
	row, col := findTitleAnchor(placed, 40, 12)
	grid := parseANSIGrid(placed, 40, 12)
	if cellRune(grid[row][col]) != 'g' {
		t.Fatalf("anchor (%d,%d)=%q", row, col, cellRune(grid[row][col]))
	}
}

func TestIntroRainSteps(t *testing.T) {
	r := newIntroRain(20, 10, 42)
	before := r.cols[0].head
	r.step(50 * time.Millisecond)
	if r.cols[0].head == before {
		t.Fatal("head should advance")
	}
}
