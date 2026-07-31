// PROTOTYPE — throwaway. Answers: what SSH TUI IA for Inventory / shop / Season Pass?
// Run: go run ./cmd/proto-shopia
// Keys: 1/2/3 variant · tab/n next screen · q quit
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type screen int

const (
	scHub screen = iota
	scInventory
	scShop
	scBuyWait
	scPass
	scEquip
)

var screenNames = []string{"hub", "inventory", "shop", "buy-wait", "pass", "equip"}

type model struct {
	variant int // 0..2
	screen  screen
	width   int
	height  int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.variant = 0
		case "2":
			m.variant = 1
		case "3":
			m.variant = 2
		case "tab", "n", " ":
			m.screen = (m.screen + 1) % screen(len(screenNames))
		case "p", "left":
			m.screen = (m.screen + screen(len(screenNames)) - 1) % screen(len(screenNames))
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var body string
	switch m.variant {
	case 0:
		body = variantA(m.screen)
	case 1:
		body = variantB(m.screen)
	default:
		body = variantC(m.screen)
	}
	return tea.NewView(fmt.Sprintf(
		"PROTOTYPE shop IA  ·  variant %d/3  ·  screen %s\n"+
			"[1][2][3] variant  ·  [n]/tab] next screen  ·  [p] prev  ·  [q] quit\n\n%s\n"+
			"── state: variant=%d screen=%s ──\n",
		m.variant+1, screenNames[m.screen], body, m.variant+1, screenNames[m.screen],
	))
}

// A — top-level hub with explicit tabs
func variantA(s screen) string {
	tabs := "[inv] [shop] [pass] [equip]   player:neo  xp:450  t:5/20"
	switch s {
	case scHub:
		return tabs + `

  progression hub
  ───────────────
  i  inventory     cosmetics + consumables
  s  shop          sats Buy (LN invoice)
  p  season pass   free + premium tracks
  e  equip         theme/caret/title/fx

  esc  back to main menu`
	case scInventory:
		return tabs + `

  inventory
  ─────────
  Cosmetics
    Theme   Matrix ★equipped
    FX      Make it Rain
    Caret   (default)
    Title   (none)
  Consumables
    Reveal x2   Calm x1   Heart x3   Retry x0

  enter equip focused · esc hub`
	case scShop:
		return tabs + `

  shop  ·  sats (no credits)
  ──────────────────────────
  > Reveal .............. 21 sats
    Calm ................ 21 sats
    Retry ............... 50 sats
    Heart .............. 100 sats
    Season premium .... 2100 sats

  enter Buy · esc hub`
	case scBuyWait:
		return tabs + `

  Buy  Heart  ·  100 sats
  ─────────────────────
  ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
  ▄  QR PLACEHOLDER ▄
  ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
  lnbc1…short…bolt11
  waiting for payment…
  esc cancel wait`
	case scPass:
		return tabs + `

  season 1  ·  42d left  ·  premium ON
  free ●●●●●○○…  5/20   premium ●●●○○… 3/20
  next free @ 100 xp (have 450)

  t10 Matrix (Theme)     claimed
  t15 Make it Rain (FX)  locked · premium

  esc hub`
	case scEquip:
		return tabs + `

  equip
  ─────
  Theme  > Matrix
  Caret    default
  Title    —
  FX       Make it Rain

  j/k slot · enter cycle owned · esc hub`
	}
	return ""
}

// B — under profile / claim; shop nested in inventory
func variantB(s screen) string {
	head := "profile:neo  ·  claimed  ·  [bag] shop is inside bag"
	switch s {
	case scHub:
		return head + `

  profile
  ───────
  name neo · rename 0 left this season
  season pass  5/20  ·  open pass →
  bag          open inventory →
  claim code   (hidden)

  this variant: shop lives inside bag, not top-level`
	case scInventory:
		return head + `

  bag
  ───
  [cosmetics] [consumables] [> shop]

  Matrix (theme)  Make it Rain (fx)
  Reveal x2  Calm x1  Heart x3

  s focus shop tab · e equip · esc profile`
	case scShop:
		return head + `

  bag > shop
  ──────────
  Reveal 21 · Calm 21 · Retry 50 · Heart 100
  Season premium 2100

  same Buy flow · fewer top-level verbs`
	case scBuyWait:
		return head + `

  bag > shop > pay Heart 100 sats
  [QR] waiting… esc`
	case scPass:
		return head + `

  profile > season pass
  (same track UI as A, reached from profile not hub)`
	case scEquip:
		return head + `

  bag > equip
  Theme/Caret/Title/FX pickers`
	}
	return ""
}

// C — results-contextual + hotkeys from anywhere; minimal hub
func variantC(s screen) string {
	hot := "hotkeys anywhere: i inv · s shop · p pass · e equip"
	switch s {
	case scHub:
		return hot + `

  (no progression hub)

  after race results:
    +25 xp  ·  day 75/200
    [i] loot  [s] spend sats  [p] pass

  main menu stays typing-first;
  progression is pull, not a place`
	case scInventory:
		return hot + `

  overlay inventory (modal over menu/results)
  Matrix★  Rain  |  Revealx2 Calmx1 Heartx3
  esc dismiss`
	case scShop:
		return hot + `

  overlay shop (modal)
  list + enter Buy`
	case scBuyWait:
		return hot + `

  full-screen Buy wait (only blocking screen)
  QR · poll · esc`
	case scPass:
		return hot + `

  overlay season pass
  tracks compact 2 columns`
	case scEquip:
		return hot + `

  overlay equip
  four lines Theme/Caret/Title/FX`
	}
	return ""
}

func main() {
	m := model{variant: 0, screen: scHub}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
