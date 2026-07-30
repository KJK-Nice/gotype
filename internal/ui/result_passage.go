package ui

import (
	"strings"

	"github.com/kjkusap/monkeytype-clone/internal/game"
)

// renderResultPassage shows the typed prompt with correctness coloring.
// Quotes show the full passage; time/words show what was on the prompt.
func (m Model) renderResultPassage(width, maxLines int) string {
	if m.sess == nil || len(m.sess.Chars) == 0 {
		return ""
	}
	if width < 12 {
		width = 12
	}
	if maxLines < 2 {
		maxLines = 2
	}

	var line, out strings.Builder
	col := 0
	lineCount := 1

	flush := func() {
		out.WriteString(line.String())
		line.Reset()
		col = 0
	}

	for _, ch := range m.sess.Chars {
		base := m.sty.Pending
		switch ch.State {
		case game.CharCorrect:
			base = m.sty.Correct
		case game.CharIncorrect:
			base = m.sty.Incorrect
		case game.CharExtra:
			base = m.sty.Extra
		}

		glyph := string(ch.R)
		if ch.R == ' ' && col >= width {
			flush()
			out.WriteByte('\n')
			lineCount++
			if lineCount > maxLines {
				out.WriteString(m.sty.Sub.Render("…"))
				return out.String()
			}
			continue
		}
		line.WriteString(base.Render(glyph))
		col++
		if col >= width && ch.R != ' ' {
			// hard-wrap long tokens
			flush()
			out.WriteByte('\n')
			lineCount++
			if lineCount > maxLines {
				out.WriteString(m.sty.Sub.Render("…"))
				return out.String()
			}
		}
	}
	flush()
	return out.String()
}
