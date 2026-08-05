package ui

import (
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/timer"
	"charm.land/bubbles/v2/viewport"

	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/multi"
)

// phaseKeyMap is a help.KeyMap for one screen's short footer.
type phaseKeyMap struct {
	short []key.Binding
}

func (k phaseKeyMap) ShortHelp() []key.Binding { return k.short }
func (k phaseKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.short}
}

func km(bindings ...key.Binding) phaseKeyMap {
	return phaseKeyMap{short: bindings}
}

func bind(keys, helpKey, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys), key.WithHelp(helpKey, desc))
}

func (m Model) helpConfig() phaseKeyMap {
	binds := []key.Binding{
		bind("up/down", "↑↓", "change"),
		bind("tab", "tab", "focus"),
		bind("enter", "enter", "start"),
		bind("u", "u", "theme"),
		bind("c", "c", "claim"),
		bind("i/s/p/e", "i/s/p/e", "progress"),
		bind("v", "v", "voice"),
		bind("n", "n", "ninja"),
		bind("y", "y", "daily"),
		bind("g", "g", "ghost"),
		bind("t/w/o", "t/w/o", "mode"),
		bind("q", "q", "quit"),
	}
	if m.multiEnabled() {
		out := make([]key.Binding, 0, len(binds)+1)
		out = append(out, binds[:3]...)
		out = append(out, bind("m", "m", "multi"))
		out = append(out, binds[3:]...)
		return km(out...)
	}
	return km(binds...)
}

func helpMultiMenu() phaseKeyMap {
	return km(
		bind("enter", "enter", "select"),
		bind("c/j/d", "c/j/d", "shortcut"),
		bind("esc", "esc", "back"),
		bind("q", "q", "quit"),
	)
}

func helpJoin() phaseKeyMap {
	return km(
		bind("enter", "enter", "join"),
		bind("esc", "esc", "back"),
	)
}

func helpLobby(host, countdown bool) phaseKeyMap {
	if countdown {
		return km(bind("esc", "esc", "leave"))
	}
	b := []key.Binding{
		bind("g", "g", "gg"),
		bind("/", "/", "chat"),
		bind("esc", "esc", "leave"),
	}
	if host {
		b = append([]key.Binding{
			bind("s/enter", "s", "start"),
			bind("h", "h", "hardcore"),
		}, b...)
	}
	return km(b...)
}

func helpSpectate() phaseKeyMap {
	return km(
		bind("g", "g", "gg"),
		bind("/", "/", "chat"),
		bind("esc", "esc", "leave"),
	)
}

func helpPodium(spectator, matchOver, matchPoint bool) phaseKeyMap {
	b := []key.Binding{
		bind("g", "g", "gg"),
		bind("/", "/", "chat"),
		bind("esc", "esc", "leave"),
	}
	if !spectator {
		label := "next"
		if matchOver {
			label = "new series"
		} else if matchPoint {
			label = "match point"
		}
		b = append([]key.Binding{bind("enter/r", "enter", label)}, b...)
	}
	return km(b...)
}

func helpResult(tipOK bool) phaseKeyMap {
	b := []key.Binding{
		bind("tab/enter", "tab", "again"),
		bind("i/s/p/e", "i/s/p/e", "progress"),
		bind("esc", "esc", "menu"),
		bind("q", "q", "quit"),
	}
	if tipOK {
		b = append([]key.Binding{bind("t", "t", "tip")}, b...)
	}
	return km(b...)
}

func helpTipPick() phaseKeyMap {
	return km(
		bind("up/down", "↑↓", "amount"),
		bind("enter", "enter", "invoice"),
		bind("esc", "esc", "back"),
	)
}

func helpTipLoading() phaseKeyMap {
	return km(bind("esc", "esc", "cancel"))
}

func helpTipShow() phaseKeyMap {
	return km(bind("esc/enter", "esc", "back"))
}

func helpChat() phaseKeyMap {
	return km(
		bind("enter", "enter", "send"),
		bind("esc", "esc", "cancel"),
	)
}

func helpTyping(multi bool) phaseKeyMap {
	if multi {
		return km(
			bind("ctrl+u", "ctrl+u", "use"),
			bind("esc", "esc", "leave race"),
		)
	}
	return km(
		bind("ctrl+u", "ctrl+u", "use"),
		bind("tab", "tab", "restart"),
		bind("esc", "esc", "menu"),
	)
}

func (m Model) renderHelp(k phaseKeyMap) string {
	h := m.help
	h.SetWidth(max(20, m.width-4))
	return h.View(k)
}

func newJoinInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "ABCD"
	ti.CharLimit = 4
	ti.Prompt = ""
	ti.SetWidth(6)
	ti.Validate = func(s string) error {
		for _, r := range s {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return fmt.Errorf("letters only")
			}
		}
		return nil
	}
	return ti
}

func newChatInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "say something"
	ti.CharLimit = 48
	ti.Prompt = "> "
	ti.SetWidth(40)
	return ti
}

func newSpinner() spinner.Model {
	return spinner.New(spinner.WithSpinner(spinner.MiniDot))
}

func newTipList() list.Model {
	items := make([]list.Item, 0, len(ln.DefaultAmounts))
	for _, n := range ln.DefaultAmounts {
		items = append(items, menuItem{
			title:  fmt.Sprintf("%d sats", n),
			desc:   "lightning tip",
			action: strconv.Itoa(n),
		})
	}
	l := newMenuList(28, 8, "amount", nil)
	l.SetItems(items)
	return l
}

func newPodiumTable() table.Model {
	cols := []table.Column{
		{Title: "#", Width: 3},
		{Title: "name", Width: 10},
		{Title: "wpm", Width: 5},
		{Title: "acc", Width: 5},
		{Title: "bo3", Width: 5},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithHeight(6),
		table.WithFocused(false),
	)
	return t
}

func (m *Model) syncPodiumTable() {
	v := m.multiView
	rows := make([]table.Row, 0, len(v.Players))
	for _, p := range v.Players {
		if p.Spectator {
			rows = append(rows, table.Row{"·", truncateName(p.Name, 9), "—", "watch", ""})
			continue
		}
		acc := fmt.Sprintf("%.0f%%", p.Prog.Accuracy)
		if p.Prog.Chars == 0 && p.Prog.Correct == 0 {
			acc = "—"
		}
		name := truncateName(p.Name, 9)
		if p.Crown {
			name = "👑" + name
		}
		if p.You {
			name += "*"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", p.Rank),
			name,
			fmt.Sprintf("%.0f", p.Prog.WPM),
			acc,
			fmt.Sprintf("%d/%d", p.MatchWins, multi.WinsToTakeMatch),
		})
	}
	m.podiumTable.SetRows(rows)
}

func newChatViewport() viewport.Model {
	return viewport.New(viewport.WithWidth(48), viewport.WithHeight(5))
}

func (m *Model) syncChatViewport() {
	v := m.multiView
	if len(v.Chat) == 0 {
		m.chatVP.SetContent("")
		return
	}
	var b string
	for _, line := range v.Chat {
		b += line.Name + ": " + line.Text + "\n"
	}
	m.chatVP.SetContent(b)
	m.chatVP.GotoBottom()
}

func (m *Model) initBubbles() {
	m.joinTI = newJoinInput()
	m.chatTI = newChatInput()
	m.spin = newSpinner()
	m.help = help.New()
	m.cdTimer = timer.New(multi.CountdownSecs*time.Second, timer.WithInterval(100*time.Millisecond))
	m.cdDigit = -1
	m.stopwatch = stopwatch.New(stopwatch.WithInterval(100 * time.Millisecond))
	m.tipList = newTipList()
	m.podiumTable = newPodiumTable()
	m.chatVP = newChatViewport()
	w, h := m.listSize()
	m.shopList = newMenuList(w, h, "", nil)
	m.invList = newMenuList(w, h, "", nil)
	m.equipList = newMenuList(w, min(8, h), "", nil)
	m.multiMenuList = newMultiMenuList()
	m.claimList = newClaimList(false, false)
}

func (m *Model) applyBubblesTheme() {
	m.sty.ApplyHelp(&m.help)
	m.sty.ApplyTextInput(&m.joinTI)
	m.sty.ApplyTextInput(&m.chatTI)
	m.sty.ApplyTextInput(&m.claimNameTI)
	m.sty.ApplyTextInput(&m.claimCodeTI)
	m.sty.ApplyList(&m.tipList)
	m.sty.ApplyList(&m.shopList)
	m.sty.ApplyList(&m.invList)
	m.sty.ApplyList(&m.equipList)
	m.sty.ApplyList(&m.multiMenuList)
	m.sty.ApplyList(&m.claimList)
	m.sty.ApplyTable(&m.podiumTable)
}

func (m *Model) startCountdownTimer(left time.Duration) tea.Cmd {
	if left < time.Second {
		left = time.Second
	}
	m.cdTimer = timer.New(left, timer.WithInterval(100*time.Millisecond))
	m.cdOn = true
	m.cdDigit = -1
	m.cdPulse = spring1D{}
	return m.cdTimer.Init()
}

func (m *Model) stopCountdownTimer() {
	m.cdOn = false
	m.cdDigit = -1
	m.cdPulse = spring1D{}
}

func (m *Model) startRaceStopwatch() tea.Cmd {
	m.stopwatch = stopwatch.New(stopwatch.WithInterval(100 * time.Millisecond))
	return m.stopwatch.Init()
}

func (m *Model) stopRaceStopwatch() tea.Cmd {
	if !m.stopwatch.Running() {
		return nil
	}
	return m.stopwatch.Stop()
}
