package tui

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func Run(dryRun bool) {
	RunWithMode(dryRun, false)
}

func RunWithMode(dryRun, demo bool) {
	setupexec.SetPrintWriter(io.Discard)
	m := InitialModelWithMode(dryRun, demo)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
