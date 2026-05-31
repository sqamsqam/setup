package tui

import (
	"io"

	tea "charm.land/bubbletea/v2"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/devtools"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
)

func newWizardRunner(dryRun bool) setupexec.CmdRunner {
	if dryRun {
		dr := setupexec.NewDryRunner()
		dr.Stdout = io.Discard
		return dr
	}
	real := setupexec.NewRealRunner()
	real.Env = append(real.Env, "DEBIAN_FRONTEND=noninteractive")
	return real
}

func runProvisioningStep(m model) tea.Cmd {
	return func() tea.Msg {
		runner := newWizardRunner(m.dryRun)
		act := m.effectiveAction()
		stepIdx := m.runningStepIndex()

		var err error
		switch act {
		case actionBootstrap:
			err = system.Bootstrap(runner, m.timezone)
		case actionAddUser:
			err = user.AddUser(runner, m.username, m.sshKey)
		case actionInstallTools:
			err = tools.InstallAll(runner)
		case actionInstallDevTools:
			err = devtools.InstallAllDevTools(runner, m.username)
		}

		if m.dryRun && err == nil {
			return stepStatusMsg{index: stepIdx, status: stepOK, output: "(dry run)"}
		}
		if err != nil {
			return stepStatusMsg{index: stepIdx, status: stepFail, output: err.Error()}
		}
		return stepStatusMsg{index: stepIdx, status: stepOK}
	}
}
