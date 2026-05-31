package tui

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func Run(dryRun bool) {
	setupexec.SetPrintWriter(io.Discard)
	m := InitialModel(dryRun)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
