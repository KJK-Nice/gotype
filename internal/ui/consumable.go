package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/consumable"
	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func (m *Model) resetConsumableRace() {
	m.usedConsumable = nil
	m.useStripOpen = false
	m.calmArmed = false
	m.useStatus = ""
}

func (m Model) consumableCtx() consumable.Context {
	ctx := consumable.Context{
		Solo:        m.roomCode == "",
		ThreeStrike: m.sess != nil && m.sess.Config.ThreeStrike,
		MatchPoint:  m.roomCode != "" && m.multiView.MatchPoint,
		Claimed:     m.isClaimed(),
		UsedClass:   m.usedConsumable,
	}
	if m.sess != nil {
		ctx.Finished = m.sess.Finished
		ctx.DNF = m.sess.DNF
		ctx.HP = m.sess.HP
		ctx.MaxHP = m.sess.MaxHP
	}
	return ctx
}

func (m *Model) tryUseConsumable(sku string) {
	if m.sess == nil || m.app == nil {
		return
	}
	if m.usedConsumable == nil {
		m.usedConsumable = map[string]bool{}
	}
	eff, err := consumable.TrySpend(m.app.Store, m.claimedID, sku, m.consumableCtx())
	if err != nil {
		m.useStatus = consumable.ErrMessage(err)
		m.useStatusUntil = m.now.Add(2 * time.Second)
		return
	}
	item, _ := catalog.Lookup(sku)
	if item.Class != "" {
		m.usedConsumable[item.Class] = true
	}
	m.useStripOpen = false
	switch eff {
	case consumable.EffectReveal:
		m.sess.ActivateReveal(game.RevealPeekWords)
		m.useStatus = "Reveal — peek ahead"
	case consumable.EffectCalm:
		m.calmArmed = true
		m.useStatus = "Calm — next typo won't shake"
	case consumable.EffectRetry:
		m.applyRetry()
		m.useStatus = "Retry — race reset"
	case consumable.EffectHeart:
		if m.sess.AddHeart() {
			m.useStatus = fmt.Sprintf("Heart — %s", heartHUD(m.sess.HP, m.sess.MaxHP))
			if m.roomCode != "" {
				m.maybeSyncMulti(false)
			}
		}
	}
	m.useStatusUntil = m.now.Add(2 * time.Second)
}

func (m *Model) applyRetry() {
	if m.sess == nil {
		return
	}
	started := m.sess.Started
	startAt := m.sess.StartedAt
	noAuto := m.sess.NoAutoFinish
	m.sess.ResetRace()
	if started {
		m.sess.Started = true
		m.sess.StartedAt = startAt
		m.sess.Stats.Start(startAt)
		m.sess.NoAutoFinish = noAuto
	}
	m.calmArmed = false
	m.shake = spring1D{}
	m.resetCaret()
}

func (m Model) useSlotKey(msg tea.KeyPressMsg) (sku string, ok bool) {
	key := msg.String()
	if strings.HasPrefix(key, "ctrl+") && len(key) == 6 && key[5] >= '1' && key[5] <= '4' {
		idx := int(key[5] - '1')
		if idx < len(consumable.UseOrder) {
			return consumable.UseOrder[idx], true
		}
	}
	if !m.useStripOpen {
		return "", false
	}
	if len(key) == 1 && key[0] >= '1' && key[0] <= '4' {
		idx := int(key[0] - '1')
		if idx < len(consumable.UseOrder) {
			return consumable.UseOrder[idx], true
		}
	}
	return "", false
}

func (m Model) handleConsumableKey(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.sess == nil || m.phase != phaseTyping {
		return false, nil
	}
	switch msg.String() {
	case "ctrl+u":
		m.useStripOpen = !m.useStripOpen
		return true, nil
	case "esc":
		if m.useStripOpen {
			m.useStripOpen = false
			return true, nil
		}
	}
	if sku, ok := m.useSlotKey(msg); ok {
		m.tryUseConsumable(sku)
		return true, nil
	}
	return false, nil
}

func (m Model) viewUseStrip() string {
	if !m.useStripOpen && m.useStatus == "" {
		return ""
	}
	var b strings.Builder
	if m.useStripOpen {
		b.WriteString(m.sty.Sub.Render("use "))
		for i, sku := range consumable.UseOrder {
			item, _ := catalog.Lookup(sku)
			qty := 0
			if m.app != nil && m.isClaimed() {
				qty = m.app.Store.InventoryQty(m.claimedID, sku)
			}
			used := m.usedConsumable != nil && m.usedConsumable[item.Class]
			label := fmt.Sprintf("%d %s×%d", i+1, item.Name, qty)
			switch {
			case used:
				b.WriteString(m.sty.Sub.Render("[" + label + " used] "))
			case qty < 1:
				b.WriteString(m.sty.Sub.Render("[" + label + "] "))
			default:
				b.WriteString(m.sty.Main.Render("[" + label + "] "))
			}
		}
		b.WriteString("\n")
	}
	if m.useStatus != "" && m.now.Before(m.useStatusUntil) {
		b.WriteString(m.sty.Main.Render(m.useStatus))
		b.WriteString("\n")
	} else if m.calmArmed {
		b.WriteString(m.sty.Sub.Render("calm armed"))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) typingHelp() phaseKeyMap {
	if m.roomCode != "" {
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
