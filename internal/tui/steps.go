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

func newWizardRunner(dryRun bool, output io.Writer) setupexec.CmdRunner {
	if output == nil {
		output = io.Discard
	}
	if dryRun {
		dr := setupexec.NewDryRunner()
		dr.Stdout = output
		return dr
	}
	real := setupexec.NewRealRunner()
	real.Env = append(real.Env, "DEBIAN_FRONTEND=noninteractive")
	real.Stdin = nil
	real.Stdout = output
	real.Stderr = output
	return real
}

func runProvisioningStep(m model) tea.Cmd {
	return func() tea.Msg {
		stepIdx := m.runningIndex
		if stepIdx < 0 || stepIdx >= len(m.runSteps) {
			return stepStatusMsg{index: stepIdx, status: stepFail, output: "invalid run step"}
		}

		var out bytes.Buffer
		setupexec.SetPrintWriter(&out)
		defer setupexec.SetPrintWriter(io.Discard)

		runner := newWizardRunner(m.dryRun, &out)
		step := m.runSteps[stepIdx]
		err := runStepWithRunner(runner, m, step)
		output := strings.TrimSpace(out.String())

		if m.dryRun && err == nil && output == "" {
			output = "(dry run)"
		}
		if err != nil {
			if output != "" {
				output += "\n"
			}
			output += err.Error()
			return stepStatusMsg{index: stepIdx, status: stepFail, output: output}
		}
		return stepStatusMsg{index: stepIdx, status: stepOK, output: output}
	}
}

func runStepWithRunner(runner setupexec.CmdRunner, m model, step runStep) error {
	username := strings.TrimSpace(m.usernameInput.Value())
	timezone := strings.TrimSpace(m.timezoneInput.Value())
	if timezone == "" {
		timezone = "UTC"
	}

	switch step.id {
	case runBootstrap:
		return system.Bootstrap(runner, timezone)
	case runAddUser:
		return user.AddUser(runner, username, normalizeSSHKeyInput(m.sshKeyInput.Value()))
	case runToolDeps:
		return tools.InstallDependencies(runner)
	case runTool:
		return tools.InstallTool(runner, step.tool)
	case runGo:
		return devtools.InstallGo(runner)
	case runNode:
		return devtools.InstallNode(runner, username)
	default:
		return nil
	}
}
