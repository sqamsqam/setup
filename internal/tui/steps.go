package tui

import (
	tea "charm.land/bubbletea/v2"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
	"github.com/sqamsqam/setup/internal/devtools"
)

func runProvisioning(m model, startIdx int) tea.Cmd {
	return func() tea.Msg {
		steps := []struct {
			idx int
			fn  func() error
		}{
			{0, func() error {
				runner := newTuiRunner()
				return system.Bootstrap(runner, m.timezone)
			}},
			{1, func() error {
				runner := newTuiRunner()
				return user.AddUser(runner, m.username, m.sshKey)
			}},
			{2, func() error {
				runner := newTuiRunner()
				return tools.InstallAll(runner)
			}},
			{3, func() error {
				runner := newTuiRunner()
				return devtools.InstallAllDevTools(runner, m.username)
			}},
		}

		for _, step := range steps {
			if step.idx < startIdx || step.idx >= len(m.stepFlags) || !m.stepFlags[step.idx] {
				continue
			}
			if m.dryRun {
				// In dry-run mode, show each step as "would run"
				return stepStatusMsg{index: step.idx, status: stepOK, output: "(dry run)"}
			}
			if err := step.fn(); err != nil {
				return stepStatusMsg{index: step.idx, status: stepFail, output: err.Error()}
			}
			return stepStatusMsg{index: step.idx, status: stepOK}
		}

		return stepStatusMsg{quitting: true}
	}
}

type tuiRunner struct {
	*setupexec.RealRunner
}

func newTuiRunner() *tuiRunner {
	real := setupexec.NewRealRunner()
	real.Env = append(real.Env, "DEBIAN_FRONTEND=noninteractive")
	return &tuiRunner{RealRunner: real}
}
