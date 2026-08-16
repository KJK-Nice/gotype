package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
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

func (m *Model) openProg(s progSurface) tea.Cmd {
	if m.app == nil {
		m.statusErr = "progression unavailable"
		return nil
	}
	if !m.isClaimed() {
		return m.openLogin()
	}
	if !m.sessionActive() {
		m.claimedID = ""
		m.statusErr = "session logged in elsewhere — login"
		return m.openLogin()
	}
	m.prog = s
	m.buyErr = ""
	m.statusErr = ""
	m.syncProgLists()
	return nil
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
	if m.loginMode != loginIdle || m.tipPhase != tipNone || m.chatMode {
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
	if m.prog != progNone && m.prog != progBuyWait {
		switch msg.String() {
		case "i":
			return m, m.openProg(progInventory), true
		case "s":
			return m, m.openProg(progShop), true
		case "p":
			return m, m.openProg(progPass), true
		case "e":
			return m, m.openProg(progEquip), true
		}
	}
	switch msg.String() {
	case "i":
		return m, m.openProg(progInventory), true
	case "s":
		return m, m.openProg(progShop), true
	case "p":
		return m, m.openProg(progPass), true
	case "e":
		return m, m.openProg(progEquip), true
	}
	return m, nil, false
}

func (m Model) progKeys(extra ...key.Binding) phaseKeyMap {
	b := []key.Binding{
		bind("i/s/p/e", "i/s/p/e", "tabs"),
		bind("esc", "esc", "close"),
	}
	b = append(b, extra...)
	return km(b...)
}

func (m Model) updateProg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.prog {
	case progInventory, progShop, progEquip:
		if msg.String() == "esc" {
			m.clearProg()
			return m, nil
		}
		if msg.String() == "q" {
			return m, tea.Quit
		}
	case progPass:
		switch msg.String() {
		case "esc":
			m.clearProg()
		case "q":
			return m, tea.Quit
		}
		return m, nil
	case progBuyWait:
		switch msg.String() {
		case "esc":
			m.clearProg()
			_ = m.openProg(progShop)
		case "q":
			return m, tea.Quit
		}
		return m, nil
	}

	switch m.prog {
	case progShop:
		if msg.String() == "enter" {
			sku := selectedAction(m.shopList)
			if sku == "" {
				return m, nil
			}
			m.prog = progBuyWait
			m.buyErr = ""
			return m, tea.Batch(m.buyCreateCmd(sku), m.spin.Tick)
		}
		var cmd tea.Cmd
		m.shopList, cmd = m.shopList.Update(msg)
		return m, cmd
	case progInventory:
		var cmd tea.Cmd
		m.invList, cmd = m.invList.Update(msg)
		return m, cmd
	case progEquip:
		if msg.String() == "enter" {
			slot := catalog.Slot(selectedAction(m.equipList))
			if slot != "" {
				m.cycleEquip(slot)
				m.syncEquipList()
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.equipList, cmd = m.equipList.Update(msg)
		return m, cmd
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
			return buyMsg{err: "login first"}
		}
		if !m.app.Store.HasActiveSession(m.claimedID, m.sessionID) {
			return buyMsg{err: "session logged in elsewhere — login"}
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
		m.syncInvList()
		return nil
	}
	return m.buyPollCmd()
}

func (m Model) viewProg() string {
	switch m.prog {
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

func (m Model) viewInventory() string {
	var body strings.Builder
	body.WriteString(m.progTabBar())
	body.WriteString("\n\n")
	body.WriteString(m.invList.View())
	return m.renderScreen(Screen{
		Title:    "inventory",
		Subtitle: "cosmetics + consumables",
		Meta:     m.progMeta(),
		Body:     body.String(),
		Keys:     m.progKeys(bind("enter", "enter", "details")),
	})
}

func (m Model) viewShop() string {
	var body strings.Builder
	body.WriteString(m.progTabBar())
	body.WriteString("\n\n")
	body.WriteString(m.shopList.View())
	status := m.buyErr
	if m.app != nil && !shop.ShopConfigured() {
		if status != "" {
			status += " · "
		}
		status += "set PHOENIXD_URL + PHOENIXD_PASSWORD for tip + Buy"
	}
	return m.renderScreen(Screen{
		Title:    "shop",
		Subtitle: "sats · lightning invoice",
		Meta:     m.progMeta(),
		Body:     body.String(),
		Status:   status,
		Keys:     m.progKeys(bind("enter", "enter", "Buy")),
	})
}

func (m Model) viewPass() string {
	var body strings.Builder
	body.WriteString(m.progTabBar())
	body.WriteString("\n\n")
	if m.app == nil {
		return m.renderScreen(Screen{Title: "season pass", Body: body.String(), Keys: m.progKeys()})
	}
	pv, err := m.app.Progress.ViewPass(m.claimedID, time.Now())
	if err != nil {
		return m.renderScreen(Screen{
			Title:  "season pass",
			Meta:   m.progMeta(),
			Status: err.Error(),
			Keys:   m.progKeys(),
		})
	}
	prem := "off"
	if pv.PremiumUnlocked {
		prem = "on"
	}
	body.WriteString(m.sty.Main.Render(fmt.Sprintf("season %d  ·  %dd left  ·  premium %s", pv.SeasonID, pv.DaysLeft, prem)))
	body.WriteString("\n")
	body.WriteString(m.sty.Text.Render(fmt.Sprintf("xp %d  ·  tier %d/%d  ·  next @ %d xp", pv.XP, pv.Tier, progress.MaxTier, pv.NextTierXP)))
	body.WriteString("\n\n")

	barW := min(40, max(20, m.width-16))
	tierBar := m.sty.TierProgress(barW)
	if pv.NextTierXP > 0 {
		pct := float64(pv.XP%pv.NextTierXP) / float64(pv.NextTierXP)
		if pv.Tier >= progress.MaxTier {
			pct = 1
		}
		_ = tierBar.SetPercent(pct)
	}
	body.WriteString(tierBar.View())
	body.WriteString("\n\n")

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
	body.WriteString(m.sty.Text.Render(fmt.Sprintf("t10 Matrix (Theme)     %s", matrix)))
	body.WriteString("\n")
	body.WriteString(m.sty.Text.Render(fmt.Sprintf("t15 Make it Rain (FX)  %s", rain)))
	body.WriteString("\n")
	body.WriteString(m.sty.Sub.Render("more tier rewards coming soon"))

	return m.renderScreen(Screen{
		Title:    "season pass",
		Subtitle: "free + premium tracks",
		Meta:     m.progMeta(),
		Body:     body.String(),
		Keys:     m.progKeys(),
	})
}

func (m Model) viewEquip() string {
	var body strings.Builder
	body.WriteString(m.progTabBar())
	body.WriteString("\n\n")
	body.WriteString(m.equipList.View())
	body.WriteString("\n")
	body.WriteString(m.sty.Sub.Render("Matrix / Make it Rain: equip data only (render stub)"))
	return m.renderScreen(Screen{
		Title:    "equip",
		Subtitle: "theme · caret · title · fx",
		Meta:     m.progMeta(),
		Body:     body.String(),
		Keys:     m.progKeys(bind("enter", "enter", "cycle")),
	})
}

func (m Model) viewBuyWait() string {
	item, _ := catalog.Lookup(m.buyOrder.SKU)
	title := "Buy"
	if item.Name != "" {
		title = "Buy " + item.Name
	}
	if m.buyOrder.State == persist.OrderGranted {
		return m.renderPayment(PaymentView{
			Title:  title,
			Sats:   m.buyOrder.Sats,
			Done:   true,
			Status: "paid · granted to inventory",
		}, km(bind("esc", "esc", "shop")))
	}
	return m.renderPayment(PaymentView{
		Title:    title,
		Subtitle: "scan with a lightning wallet",
		Sats:     m.buyOrder.Sats,
		QR:       m.buyQR,
		Bolt11:   m.buyOrder.Bolt11,
		Spinner:  m.spin.View(),
		Status:   "waiting for payment…",
		Err:      m.buyErr,
	}, km(bind("esc", "esc", "leave wait")))
}

// grantSoloXP awards Season XP after a finished solo race.
func (m *Model) grantSoloXP() {
	if m.app == nil || !m.isClaimed() || m.sess == nil || !m.sessionActive() {
		return
	}
	best := m.sess.Snapshot(m.now).BestCombo
	if m.sess.DNF {
		g, err := m.app.Progress.NoteBestCombo(m.claimedID, best)
		if err == nil {
			m.lastComboPB = g.ComboPB
			m.comboPBNew = g.ComboPBNew
		}
		m.lastXPLine = "+0 xp · DNF"
		return
	}
	g, err := m.app.Progress.GrantFinish(m.claimedID, progress.FinishSolo, best, time.Now())
	if err != nil {
		m.lastXPLine = ""
		return
	}
	m.lastComboPB = g.ComboPB
	m.comboPBNew = g.ComboPBNew
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
	best := 0
	if m.sess != nil {
		best = m.sess.Snapshot(m.now).BestCombo
	}
	if dnf {
		g, err := m.app.Progress.NoteBestCombo(m.claimedID, best)
		if err == nil {
			m.lastComboPB = g.ComboPB
			m.comboPBNew = g.ComboPBNew
		}
		m.lastXPLine = "+0 xp · DNF"
		return
	}
	g, err := m.app.Progress.GrantFinish(m.claimedID, progress.FinishMulti, best, time.Now())
	if err != nil {
		return
	}
	m.lastComboPB = g.ComboPB
	m.comboPBNew = g.ComboPBNew
	m.lastXPLine = progress.FormatGrantLine(g)
}
