package ui

import (
	"fmt"

	"charm.land/bubbles/v2/list"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
)

// menuItem is a generic bubbles list row with an action id.
type menuItem struct {
	title, desc, action string
}

func (i menuItem) FilterValue() string { return i.title + " " + i.desc }
func (i menuItem) Title() string     { return i.title }
func (i menuItem) Description() string {
	return i.desc
}

func newMenuList(width, height int, title string, items []menuItem) list.Model {
	rows := make([]list.Item, len(items))
	for i, it := range items {
		rows[i] = it
	}
	l := list.New(rows, list.NewDefaultDelegate(), width, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	return l
}

func (m *Model) listSize() (int, int) {
	w := min(52, max(28, m.width-10))
	h := min(12, max(5, m.height-14))
	return w, h
}

func (m *Model) syncListSizes() {
	w, h := m.listSize()
	m.tipList.SetSize(w, min(8, h))
	m.shopList.SetSize(w, h)
	m.invList.SetSize(w, h)
	m.equipList.SetSize(w, min(8, h))
	m.multiMenuList.SetSize(w, min(6, h))
	m.claimList.SetSize(w, min(5, h))
}

func (m *Model) syncShopList() {
	items := catalog.ShopItems()
	rows := make([]list.Item, len(items))
	for i, it := range items {
		desc := fmt.Sprintf("%d sats", it.Sats)
		if it.Kind == catalog.KindPremium {
			desc += " · season unlock"
		} else {
			desc += " · consumable"
		}
		rows[i] = menuItem{title: it.Name, desc: desc, action: it.SKU}
	}
	m.shopList.SetItems(rows)
}

func (m *Model) syncInvList() {
	if m.app == nil || !m.isClaimed() {
		m.invList.SetItems(nil)
		return
	}
	eq := map[string]string{}
	for _, e := range m.app.Store.ListEquipment(m.claimedID) {
		eq[e.Slot] = e.SKU
	}
	var rows []list.Item
	for _, it := range catalog.NamedCosmetics() {
		qty := m.app.Store.InventoryQty(m.claimedID, it.SKU)
		mark := ""
		if eq[string(it.Slot)] == it.SKU {
			mark = "★ equipped"
		}
		desc := string(it.Slot)
		if qty < 1 {
			desc = "locked"
		} else if mark != "" {
			desc = mark + " · " + desc
		}
		title := it.Name
		if qty < 1 {
			title += " (locked)"
		}
		rows = append(rows, menuItem{title: title, desc: desc, action: it.SKU})
	}
	for _, it := range catalog.ShopItems() {
		if it.Kind != catalog.KindConsumable {
			continue
		}
		qty := m.app.Store.InventoryQty(m.claimedID, it.SKU)
		rows = append(rows, menuItem{
			title:  it.Name,
			desc:   fmt.Sprintf("x%d · use in race (ctrl+u)", qty),
			action: it.SKU,
		})
	}
	m.invList.SetItems(rows)
}

func (m *Model) syncEquipList() {
	slots := []catalog.Slot{catalog.SlotTheme, catalog.SlotCaret, catalog.SlotTitle, catalog.SlotFX}
	eq := map[string]string{}
	if m.app != nil && m.isClaimed() {
		for _, e := range m.app.Store.ListEquipment(m.claimedID) {
			eq[e.Slot] = e.SKU
		}
	}
	rows := make([]list.Item, len(slots))
	for i, slot := range slots {
		sku := eq[string(slot)]
		name := "default"
		if sku != "" {
			if it, ok := catalog.Lookup(sku); ok {
				name = it.Name
			} else {
				name = sku
			}
		}
		rows[i] = menuItem{
			title:  string(slot),
			desc:   name + " · enter to cycle",
			action: string(slot),
		}
	}
	m.equipList.SetItems(rows)
}

func (m *Model) syncProgLists() {
	m.syncShopList()
	m.syncInvList()
	m.syncEquipList()
	m.syncListSizes()
}

func newClaimList() list.Model {
	return newMenuList(40, 4, "", []menuItem{
		{title: "Register new Player", desc: "display name + one-time Claim Code", action: "register"},
		{title: "Reclaim existing Player", desc: "name + Claim Code from registration", action: "reclaim"},
	})
}

func newMultiMenuList() list.Model {
	return newMenuList(44, 5, "", []menuItem{
		{title: "Create room", desc: "host a best-of-3 race · key c", action: "create"},
		{title: "Join room", desc: "enter a 4-letter code · key j", action: "join"},
		{title: "Spectate live / demo", desc: "watch a race in progress · key d", action: "demo"},
	})
}

func selectedAction(l list.Model) string {
	if it, ok := l.SelectedItem().(menuItem); ok {
		return it.action
	}
	return ""
}
