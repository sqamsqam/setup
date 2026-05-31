package tui

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func Run(dryRun bool) {
	m := InitialModel(dryRun)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
