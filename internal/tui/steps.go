package tui

import (
	"bytes"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sqamsqam/setup/internal/devtools"
	setupexec "github.com/sqamsqam/setup/internal/exec"
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
		var dryBuf *bytes.Buffer
		if dr, ok := runner.(*setupexec.DryRunner); ok {
			dryBuf = &bytes.Buffer{}
			dr.Stdout = dryBuf
		}
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
			output := "(dry run)"
			if dryBuf != nil && strings.TrimSpace(dryBuf.String()) != "" {
				output = dryBuf.String()
			}
			return stepStatusMsg{index: stepIdx, status: stepOK, output: output}
		}
		if err != nil {
			return stepStatusMsg{index: stepIdx, status: stepFail, output: err.Error()}
		}
		return stepStatusMsg{index: stepIdx, status: stepOK}
	}
}
