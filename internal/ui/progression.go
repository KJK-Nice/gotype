package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/ln"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/progress"
	"github.com/kjkusap/monkeytype-clone/internal/shop"
)

type progSurface int

const (
	progNone progSurface = iota
	progHub
	progInventory
	progShop
	progPass
	progEquip
	progBuyWait
)

type buyMsg struct {
	order persist.Order
	qr    string
	err   string
}

type buyPollMsg struct {
	order persist.Order
	err   string
}

func (m *Model) openProg(s progSurface) {
	if m.app == nil {
		m.statusErr = "progression unavailable"
		return
	}
	if !m.isClaimed() {
		m.openClaim()
		return
	}
	if !m.sessionActive() {
		m.claimedID = ""
		m.statusErr = "session claimed elsewhere — reclaim"
		m.openClaim()
		return
	}
	m.prog = s
	m.shopIdx = 0
	m.equipIdx = 0
	m.buyErr = ""
	m.statusErr = ""
}

func (m Model) sessionActive() bool {
	if m.app == nil || m.claimedID == "" {
		return false
	}
	return m.app.Store.HasActiveSession(m.claimedID, m.sessionID)
}

func (m *Model) clearProg() {
	m.prog = progNone
	m.buyOrder = persist.Order{}
	m.buyQR = ""
	m.buyErr = ""
}

func (m Model) progHotkeysActive() bool {
	if m.claimMode != claimIdle || m.tipPhase != tipNone {
		return false
	}
	if m.prog == progBuyWait {
		return false
	}
	switch m.phase {
	case phaseConfig, phaseResult, phaseMultiMenu, phasePodium:
		return true
	default:
		return false
	}
}

func (m Model) tryProgHotkey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if !m.progHotkeysActive() {
		return m, nil, false
	}
	// When already on a surface (except buy wait), hotkeys still switch tabs.
	if m.prog != progNone && m.prog != progHub {
		switch msg.String() {
		case "i":
			m.openProg(progInventory)
			return m, nil, true
		case "s":
			m.openProg(progShop)
			return m, nil, true
		case "p":
			m.openProg(progPass)
			return m, nil, true
		case "e":
			m.openProg(progEquip)
			return m, nil, true
		case "h":
			m.openProg(progHub)
			return m, nil, true
		}
	}
	switch msg.String() {
	case "i":
		m.openProg(progInventory)
		return m, nil, true
	case "s":
		// Config used s? no. Lobby host uses s to start — not in progHotkeysActive.
		m.openProg(progShop)
		return m, nil, true
	case "p":
		m.openProg(progPass)
		return m, nil, true
	case "e":
		m.openProg(progEquip)
		return m, nil, true
	case "h":
		m.openProg(progHub)
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) updateProg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.prog {
	case progHub:
		switch msg.String() {
		case "esc":
			m.clearProg()
		case "i":
			m.openProg(progInventory)
		case "s":
			m.openProg(progShop)
		case "p":
			m.openProg(progPass)
		case "e":
			m.openProg(progEquip)
		case "q":
			return m, tea.Quit
		}
	case progInventory:
		switch msg.String() {
		case "esc":
			m.openProg(progHub)
		case "e":
			m.openProg(progEquip)
		case "q":
			return m, tea.Quit
		}
	case progShop:
		items := catalog.ShopItems()
		switch msg.String() {
		case "esc":
			m.openProg(progHub)
		case "up", "k":
			if m.shopIdx > 0 {
				m.shopIdx--
			}
		case "down", "j":
			if m.shopIdx < len(items)-1 {
				m.shopIdx++
			}
		case "enter", " ":
			if len(items) == 0 {
				return m, nil
			}
			sku := items[m.shopIdx].SKU
			m.prog = progBuyWait
			m.buyErr = ""
			return m, tea.Batch(m.buyCreateCmd(sku), m.spin.Tick)
		case "q":
			return m, tea.Quit
		}
	case progPass:
		switch msg.String() {
		case "esc":
			m.openProg(progHub)
		case "q":
			return m, tea.Quit
		}
	case progEquip:
		slots := []catalog.Slot{catalog.SlotTheme, catalog.SlotCaret, catalog.SlotTitle, catalog.SlotFX}
		switch msg.String() {
		case "esc":
			m.openProg(progHub)
		case "up", "k":
			if m.equipIdx > 0 {
				m.equipIdx--
			}
		case "down", "j":
			if m.equipIdx < len(slots)-1 {
				m.equipIdx++
			}
		case "enter", " ":
			m.cycleEquip(slots[m.equipIdx])
		case "q":
			return m, tea.Quit
		}
	case progBuyWait:
		switch msg.String() {
		case "esc":
			// Leave wait; invoice may still settle later.
			m.clearProg()
			m.openProg(progShop)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) cycleEquip(slot catalog.Slot) {
	if m.app == nil || !m.isClaimed() {
		return
	}
	owned := m.app.Store.ListInventory(m.claimedID)
	var skus []string
	skus = append(skus, "") // unequip / default
	for _, it := range owned {
		item, ok := catalog.Lookup(it.SKU)
		if ok && item.Kind == catalog.KindCosmetic && item.Slot == slot {
			skus = append(skus, it.SKU)
		}
	}
	cur := ""
	for _, e := range m.app.Store.ListEquipment(m.claimedID) {
		if e.Slot == string(slot) {
			cur = e.SKU
			break
		}
	}
	idx := 0
	for i, s := range skus {
		if s == cur {
			idx = i
			break
		}
	}
	next := skus[(idx+1)%len(skus)]
	_ = m.app.Store.Equip(m.claimedID, string(slot), next)
}

func (m Model) buyCreateCmd(sku string) tea.Cmd {
	return func() tea.Msg {
		if m.app == nil || !m.isClaimed() {
			return buyMsg{err: "claim a Player first"}
		}
		if !m.app.Store.HasActiveSession(m.claimedID, m.sessionID) {
			return buyMsg{err: "session claimed elsewhere — reclaim"}
		}
		o, err := m.app.Shop.CreateBuy(context.Background(), m.claimedID, sku, time.Now())
		if err != nil {
			return buyMsg{err: err.Error()}
		}
		return buyMsg{order: o, qr: ln.QRString(o.Bolt11)}
	}
}

func (m Model) buyPollCmd() tea.Cmd {
	id := m.buyOrder.ID
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		if m.app == nil || id == "" {
			return buyPollMsg{}
		}
		o, err := m.app.Shop.PollAndGrant(context.Background(), id, t)
		if err != nil {
			return buyPollMsg{err: err.Error()}
		}
		return buyPollMsg{order: o}
	})
}

func (m *Model) applyBuyMsg(msg buyMsg) tea.Cmd {
	if msg.err != "" {
		m.buyErr = msg.err
		m.prog = progShop
		return nil
	}
	m.buyOrder = msg.order
	m.buyQR = msg.qr
	m.buyErr = ""
	m.prog = progBuyWait
	return m.buyPollCmd()
}

func (m *Model) applyBuyPoll(msg buyPollMsg) tea.Cmd {
	if m.prog != progBuyWait {
		return nil
	}
	if msg.err != "" {
		m.buyErr = msg.err
		return m.buyPollCmd()
	}
	if msg.order.ID == "" {
		return nil
	}
	m.buyOrder = msg.order
	if msg.order.State == persist.OrderGranted {
		m.buyErr = ""
		return nil
	}
	return m.buyPollCmd()
}

func (m Model) viewProg() string {
	switch m.prog {
	case progHub:
		return m.viewProgHub()
	case progInventory:
		return m.viewInventory()
	case progShop:
		return m.viewShop()
	case progPass:
		return m.viewPass()
	case progEquip:
		return m.viewEquip()
	case progBuyWait:
		return m.viewBuyWait()
	default:
		return ""
	}
}

func (m Model) progTabs() string {
	tab := func(key, label string, on bool) string {
		s := fmt.Sprintf("[%s %s]", key, label)
		if on {
			return m.sty.Main.Render(s)
		}
		return m.sty.Sub.Render(s)
	}
	return strings.Join([]string{
		tab("i", "inv", m.prog == progInventory),
		tab("s", "shop", m.prog == progShop),
		tab("p", "pass", m.prog == progPass),
		tab("e", "equip", m.prog == progEquip),
	}, "  ")
}

func (m Model) viewProgHub() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("progression hub"))
	b.WriteString("  ")
	b.WriteString(m.sty.Main.Render(m.playerName))
	b.WriteString("\n")
	b.WriteString(m.progTabs())
	b.WriteString("\n\n")
	b.WriteString(m.sty.Text.Render("i  inventory     cosmetics + consumables"))
	b.WriteString("\n")
	b.WriteString(m.sty.Text.Render("s  shop          sats Buy (LN invoice)"))
	b.WriteString("\n")
	b.WriteString(m.sty.Text.Render("p  season pass   free + premium tracks"))
	b.WriteString("\n")
	b.WriteString(m.sty.Text.Render("e  equip         theme/caret/title/fx"))
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp(km(bind("esc", "esc", "back"))))
	return b.String()
}

func (m Model) viewInventory() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("inventory"))
	b.WriteString("\n")
	b.WriteString(m.progTabs())
	b.WriteString("\n\n")
	if m.app == nil {
		return b.String()
	}
	eq := map[string]string{}
	for _, e := range m.app.Store.ListEquipment(m.claimedID) {
		eq[e.Slot] = e.SKU
	}
	b.WriteString(m.sty.Sub.Render("Cosmetics"))
	b.WriteString("\n")
	for _, it := range catalog.NamedCosmetics() {
		qty := m.app.Store.InventoryQty(m.claimedID, it.SKU)
		mark := "  "
		if eq[string(it.Slot)] == it.SKU {
			mark = "★ "
		}
		line := fmt.Sprintf("  %s%-14s %s", mark, it.Name, it.Slot)
		if qty < 1 {
			line += "  (locked)"
			b.WriteString(m.sty.Sub.Render(line))
		} else {
			b.WriteString(m.sty.Text.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("Consumables"))
	b.WriteString("\n")
	for _, it := range catalog.ShopItems() {
		if it.Kind != catalog.KindConsumable {
			continue
		}
		qty := m.app.Store.InventoryQty(m.claimedID, it.SKU)
		b.WriteString(m.sty.Text.Render(fmt.Sprintf("  %-8s x%d", it.Name, qty)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.renderHelp(km(bind("e", "e", "equip"), bind("esc", "esc", "hub"))))
	return b.String()
}

func (m Model) viewShop() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("shop"))
	b.WriteString(m.sty.Sub.Render("  ·  sats (no credits)"))
	b.WriteString("\n")
	b.WriteString(m.progTabs())
	b.WriteString("\n\n")
	items := catalog.ShopItems()
	for i, it := range items {
		prefix := "  "
		if i == m.shopIdx {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-16s %4d sats", prefix, it.Name, it.Sats)
		if i == m.shopIdx {
			b.WriteString(m.sty.Main.Render(line))
		} else {
			b.WriteString(m.sty.Text.Render(line))
		}
		b.WriteString("\n")
	}
	if m.buyErr != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Incorrect.Render(m.buyErr))
		b.WriteString("\n")
	}
	if m.app != nil && !shop.ShopConfigured() {
		b.WriteString("\n")
		b.WriteString(m.sty.Sub.Render("set PHOENIXD_URL + PHOENIXD_PASSWORD to Buy"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.renderHelp(km(bind("enter", "enter", "Buy"), bind("esc", "esc", "hub"))))
	return b.String()
}

func (m Model) viewPass() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("season pass"))
	b.WriteString("\n")
	b.WriteString(m.progTabs())
	b.WriteString("\n\n")
	if m.app == nil {
		return b.String()
	}
	pv, err := m.app.Progress.ViewPass(m.claimedID, time.Now())
	if err != nil {
		b.WriteString(m.sty.Incorrect.Render(err.Error()))
		return b.String()
	}
	prem := "OFF"
	if pv.PremiumUnlocked {
		prem = "ON"
	}
	b.WriteString(m.sty.Main.Render(fmt.Sprintf("season %d  ·  %dd left  ·  premium %s", pv.SeasonID, pv.DaysLeft, prem)))
	b.WriteString("\n")
	b.WriteString(m.sty.Text.Render(fmt.Sprintf("xp %d  ·  tier %d/%d  ·  next @ %d xp", pv.XP, pv.Tier, progress.MaxTier, pv.NextTierXP)))
	b.WriteString("\n\n")
	matrix := "locked"
	if pv.MatrixOwned {
		matrix = "owned"
	}
	rain := "locked · premium"
	if pv.RainOwned {
		rain = "owned"
	} else if !pv.PremiumUnlocked {
		rain = "locked · needs premium"
	}
	b.WriteString(m.sty.Text.Render(fmt.Sprintf("t10 Matrix (Theme)     %s", matrix)))
	b.WriteString("\n")
	b.WriteString(m.sty.Text.Render(fmt.Sprintf("t15 Make it Rain (FX)  %s", rain)))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("(other tier Cosmetics: placeholder — deferred)"))
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp(km(bind("esc", "esc", "hub"))))
	return b.String()
}

func (m Model) viewEquip() string {
	var b strings.Builder
	b.WriteString(m.sty.Title.Render("equip"))
	b.WriteString("\n")
	b.WriteString(m.progTabs())
	b.WriteString("\n\n")
	slots := []catalog.Slot{catalog.SlotTheme, catalog.SlotCaret, catalog.SlotTitle, catalog.SlotFX}
	eq := map[string]string{}
	if m.app != nil {
		for _, e := range m.app.Store.ListEquipment(m.claimedID) {
			eq[e.Slot] = e.SKU
		}
	}
	for i, slot := range slots {
		prefix := "  "
		if i == m.equipIdx {
			prefix = "> "
		}
		sku := eq[string(slot)]
		name := "default"
		if sku != "" {
			if it, ok := catalog.Lookup(sku); ok {
				name = it.Name
			} else {
				name = sku
			}
		}
		line := fmt.Sprintf("%s%-6s  %s", prefix, slot, name)
		if i == m.equipIdx {
			b.WriteString(m.sty.Main.Render(line))
		} else {
			b.WriteString(m.sty.Text.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("Matrix / Make it Rain visuals: data + equip only (render stub)"))
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp(km(bind("j/k", "j/k", "slot"), bind("enter", "enter", "cycle"), bind("esc", "esc", "hub"))))
	return b.String()
}

func (m Model) viewBuyWait() string {
	var b strings.Builder
	item, _ := catalog.Lookup(m.buyOrder.SKU)
	b.WriteString(m.sty.Title.Render(fmt.Sprintf("Buy  %s  ·  %d sats", item.Name, m.buyOrder.Sats)))
	b.WriteString("\n\n")
	if m.buyOrder.State == persist.OrderGranted {
		b.WriteString(m.sty.Main.Render("paid · granted to Inventory"))
		b.WriteString("\n\n")
		b.WriteString(m.renderHelp(km(bind("esc", "esc", "shop"))))
		return b.String()
	}
	if m.buyQR != "" {
		b.WriteString(m.buyQR)
		b.WriteString("\n\n")
	}
	b.WriteString(m.sty.Sub.Render(ln.ShortBolt11(m.buyOrder.Bolt11)))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render(m.spin.View() + " waiting for payment…"))
	b.WriteString("\n")
	if m.buyErr != "" {
		b.WriteString(m.sty.Incorrect.Render(m.buyErr))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.renderHelp(km(bind("esc", "esc", "leave wait"))))
	return b.String()
}

// grantSoloXP awards Season XP after a finished solo race.
func (m *Model) grantSoloXP() {
	if m.app == nil || !m.isClaimed() || m.sess == nil || !m.sessionActive() {
		return
	}
	if m.sess.DNF {
		m.lastXPLine = "+0 xp · DNF"
		return
	}
	g, err := m.app.Progress.GrantFinish(m.claimedID, progress.FinishSolo, time.Now())
	if err != nil {
		m.lastXPLine = ""
		return
	}
	m.lastXPLine = progress.FormatGrantLine(g)
}

// grantMultiXP awards Season XP once per finished multi race for this client.
func (m *Model) grantMultiXP(raceNumber int, dnf bool) {
	if m.app == nil || !m.isClaimed() || !m.sessionActive() || m.multiView.YouAreSpectator {
		return
	}
	if raceNumber <= m.multiXPRace {
		return
	}
	m.multiXPRace = raceNumber
	if dnf {
		m.lastXPLine = "+0 xp · DNF"
		return
	}
	g, err := m.app.Progress.GrantFinish(m.claimedID, progress.FinishMulti, time.Now())
	if err != nil {
		return
	}
	m.lastXPLine = progress.FormatGrantLine(g)
}
