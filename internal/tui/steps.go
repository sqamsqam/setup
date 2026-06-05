package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sqamsqam/setup/internal/devtools"
	"github.com/sqamsqam/setup/internal/diagnostics"
	dockermaint "github.com/sqamsqam/setup/internal/docker"
	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/firewall"
	"github.com/sqamsqam/setup/internal/security"
	"github.com/sqamsqam/setup/internal/service"
	"github.com/sqamsqam/setup/internal/system"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/updates"
	"github.com/sqamsqam/setup/internal/user"
)

func newWizardRunner(dryRun, demo bool, output io.Writer) setupexec.CmdRunner {
	if output == nil {
		output = io.Discard
	}
	if demo {
		dr := setupexec.NewDemoRunner()
		dr.Stdout = output
		return dr
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
	return loggingRunner{CmdRunner: real, output: output}
}

type loggingRunner struct {
	setupexec.CmdRunner
	output io.Writer
}

func (r loggingRunner) Run(name string, args ...string) error {
	r.logCommand(commandString(name, args...))
	return r.CmdRunner.Run(name, args...)
}

func (r loggingRunner) Output(name string, args ...string) (string, error) {
	r.logCommand(commandString(name, args...))
	return r.CmdRunner.Output(name, args...)
}

func (r loggingRunner) RunAsUser(user, name string, args ...string) error {
	allArgs := append([]string{"-iu", user, "--", name}, args...)
	r.logCommand(commandString("sudo", allArgs...))
	return r.CmdRunner.RunAsUser(user, name, args...)
}

func (r loggingRunner) Shell(script string) error {
	r.logCommand(commandString("bash", "-c", script))
	return r.CmdRunner.Shell(script)
}

func (r loggingRunner) logCommand(cmd string) {
	_, _ = fmt.Fprintf(r.output, "$ %s\n", cmd)
}

func commandString(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`|&;()<>*?[]{}!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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

		runner := newWizardRunner(m.dryRun, m.demo, &out)
		step := m.runSteps[stepIdx]
		err := runStepWithRunner(runner, m, step)
		output := strings.TrimSpace(out.String())

		if (m.dryRun || m.demo) && err == nil && output == "" {
			output = "(dry run)"
			if m.demo {
				output = "(no output)"
			}
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
	case runUserCreateLogin:
		return user.CreateLoginUser(runner, username)
	case runUserSSHKey:
		return user.AddAuthorizedKey(runner, username, normalizeSSHKeyInput(m.sshKeyInput.Value()))
	case runUserAllowSSH:
		return user.AllowSSH(runner, username)
	case runUserSudo:
		return user.EnablePasswordlessSudo(runner, username)
	case runUserLinger:
		return user.EnableLinger(runner, username)
	case runUserDockerGroup:
		return user.AddGroup(runner, username, "docker")
	case runServiceUser:
		return user.CreateServiceUser(runner, username, parseServiceGroups(m.serviceGroupsInput.Value()))
	case runServiceCreate:
		return service.Create(runner, service.Config{
			User:    username,
			Name:    strings.TrimSpace(m.serviceNameInput.Value()),
			WorkDir: strings.TrimSpace(m.serviceWorkDir.Value()),
			Command: strings.TrimSpace(m.serviceCommand.Value()),
			EnvFile: strings.TrimSpace(m.serviceEnvFile.Value()),
		})
	case runServiceStatus:
		out, err := service.Status(runner, username, strings.TrimSpace(m.serviceNameInput.Value()))
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runServiceLogs:
		out, err := service.Logs(runner, username, strings.TrimSpace(m.serviceNameInput.Value()))
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runServiceRestart:
		return service.Restart(runner, username, strings.TrimSpace(m.serviceNameInput.Value()))
	case runServiceList:
		units, err := service.List(runner, username)
		if err != nil {
			return err
		}
		if len(units) == 0 {
			setupexec.PrintOutput("No setup-managed services found.")
			return nil
		}
		setupexec.PrintOutput(strings.Join(units, "\n"))
		return nil
	case runServiceDisable:
		return service.Disable(runner, username, strings.TrimSpace(m.serviceNameInput.Value()))
	case runServiceRemove:
		return service.Remove(runner, username, strings.TrimSpace(m.serviceNameInput.Value()))
	case runFirewall:
		return firewall.EnableBaseline(runner, true)
	case runHTTP:
		return firewall.AllowRule(runner, firewall.Rule{Port: "80", Proto: "tcp", Comment: "setup http"})
	case runHTTPS:
		return firewall.AllowRule(runner, firewall.Rule{Port: "443", Proto: "tcp", Comment: "setup https"})
	case runMosh:
		return firewall.AllowRule(runner, firewall.Rule{Port: "60000:61000", Proto: "udp", Comment: "setup mosh"})
	case runFirewallCustom:
		return firewall.AllowRule(runner, m.firewallRule())
	case runNetworkStatus:
		out, err := firewall.Status(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runNetworkList:
		out, err := firewall.StatusNumbered(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runNetworkDelete:
		number, err := networkRuleNumber(m.networkRuleInput.Value())
		if err != nil {
			return err
		}
		return firewall.DeleteRule(runner, number)
	case runNetworkReset:
		return firewall.Reset(runner)
	case runFail2Ban:
		maxRetry, err := fail2banMaxRetry(m.fail2banMaxRetry.Value())
		if err != nil {
			return err
		}
		return security.InstallFail2Ban(runner, security.Fail2BanOptions{
			BanTime:  strings.TrimSpace(m.fail2banBanTime.Value()),
			FindTime: strings.TrimSpace(m.fail2banFindTime.Value()),
			MaxRetry: maxRetry,
		})
	case runFail2BanStatus:
		out, err := security.Fail2BanStatus(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runFail2BanUnban:
		return security.UnbanIP(runner, strings.TrimSpace(m.guardIPInput.Value()))
	case runDockerLog:
		return dockermaint.ConfigureLogRotation(runner, dockermaint.LogRotationOptions{
			MaxSize: strings.TrimSpace(m.dockerMaxSize.Value()),
			MaxFile: strings.TrimSpace(m.dockerMaxFile.Value()),
		})
	case runContainersDisk:
		out, err := dockermaint.DiskUsage(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runContainersPrune:
		opts, err := parseDockerPruneTargets(m.pruneTargetsInput.Value())
		if err != nil {
			return err
		}
		return dockermaint.Prune(runner, opts)
	case runDoctor:
		setupexec.PrintOutput(diagnostics.Format(diagnostics.Run(runner)))
		return nil
	case runUpdates:
		out, err := updates.Check(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runUpdatesUpgrade:
		return updates.Upgrade(runner)
	case runUpdatesRebootN:
		out, err := updates.RebootRequired(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runUpdatesUnattend:
		out, err := updates.UnattendedStatus(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runUpdatesFailed:
		out, err := updates.FailedUnits(runner)
		if err != nil {
			return err
		}
		setupexec.PrintOutput(out)
		return nil
	case runUpdatesReboot:
		return updates.Reboot(runner, true)
	case runToolDeps:
		return tools.InstallDependencies(runner)
	case runTool:
		return tools.InstallTool(runner, step.tool)
	case runGo:
		return devtools.InstallGo(runner)
	case runNode:
		return devtools.InstallNode(runner, username)
	case runRust:
		return devtools.InstallRust(runner, username)
	case runGoLint:
		return devtools.InstallGoLint(runner)
	case runGoRel:
		return devtools.InstallGoReleaser(runner)
	case runGoVuln:
		if err := devtools.InstallGo(runner); err != nil {
			return err
		}
		return devtools.InstallGoVulnCheck(runner)
	case runPnpm:
		if !m.selections.DevTools.Node {
			if err := devtools.InstallNode(runner, username); err != nil {
				return err
			}
		}
		return devtools.InstallPnpm(runner, username)
	default:
		return nil
	}
}
