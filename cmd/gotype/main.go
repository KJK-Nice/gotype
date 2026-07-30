package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/kjkusap/monkeytype-clone/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.New(), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			if ws.Width < 1 {
				ws.Width = 80
			}
			if ws.Height < 1 {
				ws.Height = 24
			}
			return ws
		}
		return msg
	}))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gotype: %v\n", err)
		os.Exit(1)
	}
}
