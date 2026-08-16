package ui

import (
	"fmt"
	"strings"

	"github.com/kjkusap/monkeytype-clone/internal/game"
	"github.com/kjkusap/monkeytype-clone/internal/quoteai"
	"github.com/kjkusap/monkeytype-clone/internal/words"
)

const (
	configLabelW = 10
	configHintW  = 10
)

// configKV renders "label  key  value" — label/key in Sub, value in Main.
func (m Model) configKV(label, keyHint, value string) string {
	var b strings.Builder
	if label != "" {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("%-*s", configLabelW, label)))
	}
	if keyHint != "" {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("%-*s", configHintW, keyHint)))
	} else if label != "" {
		b.WriteString(strings.Repeat(" ", configHintW))
	}
	b.WriteString(m.sty.Main.Render(value))
	return b.String()
}

func (m Model) configFieldLabel(label, keyHint string, focused bool) string {
	style := m.sty.Sub
	if focused {
		style = m.sty.Main
	}
	var b strings.Builder
	b.WriteString(style.Render(fmt.Sprintf("%-*s", configLabelW, label)))
	if keyHint != "" {
		b.WriteString(m.sty.Sub.Render(fmt.Sprintf("%-*s", configHintW, keyHint)))
	} else {
		b.WriteString(strings.Repeat(" ", configHintW))
	}
	return b.String()
}

func (m Model) renderModeOptions() string {
	modeTime := m.sty.Option.Render("time")
	modeWords := m.sty.Option.Render("words")
	modeQuote := m.sty.Option.Render("quote")
	switch m.cfg.Mode {
	case game.ModeTime:
		modeTime = m.sty.Selected.Render("time")
	case game.ModeWords:
		modeWords = m.sty.Selected.Render("words")
	case game.ModeQuotes:
		modeQuote = m.sty.Selected.Render("quote")
	}
	parts := []string{modeTime, modeWords, modeQuote}
	if quoteai.Configured() {
		modeAI := m.sty.Option.Render("ai")
		if m.cfg.Mode == game.ModeAI {
			modeAI = m.sty.Selected.Render("ai")
		}
		parts = append(parts, modeAI)
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderValueOptions() string {
	var b strings.Builder
	switch m.cfg.Mode {
	case game.ModeTime:
		for i, d := range game.TimeOptions {
			label := fmt.Sprintf("%d", int(d.Seconds()))
			if d == m.cfg.Duration {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.TimeOptions)-1 {
				b.WriteString("  ")
			}
		}
	case game.ModeQuotes, game.ModeAI:
		for i, qlen := range game.QuoteLenOptions {
			label := qlen.String()
			if qlen == m.cfg.QuoteLen {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.QuoteLenOptions)-1 {
				b.WriteString("  ")
			}
		}
	default:
		for i, n := range game.WordOptions {
			label := fmt.Sprintf("%d", n)
			if n == m.cfg.WordCount {
				b.WriteString(m.sty.Selected.Render(label))
			} else {
				b.WriteString(m.sty.Option.Render(label))
			}
			if i < len(game.WordOptions)-1 {
				b.WriteString("  ")
			}
		}
	}
	return b.String()
}

func (m Model) viewConfig() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.sty.Title.Render("gotype"))
	b.WriteString("\n")
	b.WriteString(m.sty.Sub.Render("typing races in your terminal"))
	b.WriteString("\n\n")

	modeHint := "t/w/o"
	if quoteai.Configured() {
		modeHint = "t/w/o/a"
	}
	if m.focus == focusMode {
		modeHint = "←/→"
	}
	b.WriteString(m.configFieldLabel("mode", modeHint, m.focus == focusMode))
	b.WriteString(m.renderModeOptions())
	b.WriteString("\n\n")

	valueHint := ""
	if m.focus == focusValue {
		valueHint = "←/→"
	}
	b.WriteString(m.configFieldLabel("value", valueHint, m.focus == focusValue))
	b.WriteString(m.renderValueOptions())
	b.WriteString("\n\n")

	b.WriteString(m.configKV("theme", "u", ThemeName(m.themeIdx)))
	b.WriteString("\n")
	b.WriteString(m.configKV("voice", "v", m.voice.String()))
	b.WriteString("\n")
	if m.ninjaCaret {
		b.WriteString(m.configKV("ninja", "n", "on"))
	} else {
		b.WriteString(m.configKV("ninja", "n", "off"))
	}
	b.WriteString("\n")
	if m.cfg.Daily {
		b.WriteString(m.configKV("daily", "y", words.DailyLabel(m.now)))
	} else {
		b.WriteString(m.configKV("daily", "y", "off"))
	}
	b.WriteString("\n")
	if m.ghostOn {
		b.WriteString(m.configKV("ghost", "g", "on"))
	} else {
		b.WriteString(m.configKV("ghost", "g", "off"))
	}
	b.WriteString("\n")
	if m.isClaimed() {
		b.WriteString(m.configKV("player", "l", m.playerName))
	} else {
		b.WriteString(m.configKV("player", "l", "guest"))
	}

	if m.multiEnabled() {
		b.WriteString("\n")
		b.WriteString(m.configKV("multi", "m", "menu"))
	}

	b.WriteString("\n\n")
	b.WriteString(m.sty.Divider.Render(strings.Repeat("─", 28)))
	b.WriteString("\n\n")

	b.WriteString(m.configKV("progress", "i/s/p/e", "inv · shop · pass · equip"))
	b.WriteString("\n")
	b.WriteString(m.configKV("start", "enter/space", configDetail(m.cfg, nil)))
	b.WriteString("\n")
	b.WriteString(m.configKV("quit", "q", "exit"))
	b.WriteString("\n")

	if m.statusErr != "" {
		b.WriteString("\n")
		b.WriteString(m.sty.Incorrect.Render(m.statusErr))
	}
	return b.String()
}
