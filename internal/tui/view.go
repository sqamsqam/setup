package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/sqamsqam/setup/internal/user"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinnerChar(frame int) string {
	return spinnerFrames[frame%len(spinnerFrames)]
}

func drawProgressBar(pct, width int) string {
	if width <= 0 {
		width = 20
	}
	filled := (pct * width) / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return progressStyle.Render(strings.Repeat("█", filled)) +
		dimStyle.Render(strings.Repeat("░", width-filled)) +
		fmt.Sprintf(" %d%%", pct)
}

func (m model) mainMenuView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Ubuntu LXC Provisioning Tool"))
	s.WriteString("\n\n")

	if os.Geteuid() != 0 {
		s.WriteString(errorStyle.Render("WARNING: Not running as root. Provisioning may fail.\n"))
		s.WriteString("\n")
	}

	for i, item := range m.menuItems {
		prefix := "  "
		if m.menuCursor == i {
			prefix = cursorStyle.Render("► ")
		}
		s.WriteString(prefix + stepNameStyle.Render(item.label))
		s.WriteString("\n")
		s.WriteString("     " + dimStyle.Render(item.desc))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓ move · enter select · q quit"))
	if m.dryRun {
		s.WriteString("\n")
		s.WriteString(cursorStyle.Render("  DRY RUN — no changes will be made"))
	}
	return s.String()
}

func (m model) inputTimezoneView() string {
	s := titleStyle.Render("Timezone")
	s += "\n\n"
	s += "Search timezone for " + m.actionLabel() + ":\n\n"
	s += "  " + cursorStyle.Render(m.timezone)
	if m.timezone == "" {
		s += dimStyle.Render(" (UTC)")
	}
	s += "\n\n"
	if m.timezoneErr != "" {
		s += errorStyle.Render("  ✗ "+m.timezoneErr) + "\n\n"
	}
	matches := m.timezoneMatches
	if len(matches) == 0 {
		matches = timezoneMatches(m.timezone, 6)
	}
	if len(matches) > 0 {
		s += dimStyle.Render("Matches:") + "\n"
		for i, zone := range matches {
			prefix := "  "
			if i == m.timezoneCursor {
				prefix = cursorStyle.Render("► ")
			}
			s += prefix + zone + "\n"
		}
		s += "\n"
	}
	s += helpStyle.Render("type search · ↑/↓ choose · tab fill · enter confirm · esc back · q quit")
	return s
}

func (m model) inputUserView() string {
	s := titleStyle.Render(m.actionLabel() + " — Username")
	s += "\n\n"
	s += "Username: " + cursorStyle.Render(m.username)
	if m.username == "" {
		s += dimStyle.Render(" (type to enter)")
	}
	s += "\n"
	if m.usernameErr != "" {
		s += errorStyle.Render("  ✗ "+m.usernameErr) + "\n"
	}
	s += "\n"
	s += helpStyle.Render("enter confirm · esc back · q quit")
	return s
}

func (m model) inputKeyView() string {
	s := titleStyle.Render(m.actionLabel() + " — SSH Public Key")
	s += "\n\n"

	display := m.sshKey
	if len(display) == 0 {
		display = dimStyle.Render("(paste your public key)")
	}

	s += "Key: " + truncateKey(display, maxPreviewWidth(m.width))
	if m.sshKeyErr != "" {
		s += "\n" + errorStyle.Render("  ✗ "+m.sshKeyErr)
	} else if m.sshKey != "" {
		s += "\n" + dimStyle.Render("  "+user.SSHKeySummary(m.sshKey))
	}
	s += "\n\n"
	s += helpStyle.Render("paste key, then press enter · esc back · q quit")
	return s
}

func (m model) confirmView() string {
	var s strings.Builder

	if m.isChain() {
		s.WriteString(titleStyle.Render(fmt.Sprintf("Full Setup — %s (%d/%d)",
			stepNames[m.effectiveAction()], m.chainIdx+1, len(fullSetupChain))))
		s.WriteString("\n\n")

		for i, step := range m.steps {
			var prefix string
			if i < m.chainIdx {
				prefix = successStyle.Render("[✓]")
				line := successStyle.Render(fmt.Sprintf("  %s %s", prefix, step.name))
				s.WriteString(line + "\n")
			} else if i == m.chainIdx {
				prefix = cursorStyle.Render("►")
				fmt.Fprintf(&s, "  %s %s\n", prefix, stepNameStyle.Render(step.name))
			} else {
				prefix = dimStyle.Render("[ ]")
				line := dimStyle.Render(fmt.Sprintf("  %s %s", prefix, step.name))
				s.WriteString(line + "\n")
			}
		}
		s.WriteString("\n")
	} else {
		s.WriteString(titleStyle.Render(m.actionLabel()))
		s.WriteString("\n\n")
	}

	s.WriteString("Will perform:\n\n")
	effAct := m.effectiveAction()
	switch effAct {
	case actionBootstrap:
		s.WriteString("  • Configure locale, base packages, SSH hardening\n")
		s.WriteString("  • Set up unattended security upgrades\n")
		s.WriteString("  • Install Docker from the official apt repository\n")
		s.WriteString("\nAccess changes:\n")
		s.WriteString("  • Disable root/password SSH and lock the root password\n")
	case actionAddUser:
		s.WriteString("  • Create user account with passwordless sudo\n")
		s.WriteString("  • Install SSH public key\n")
		s.WriteString("  • Update SSH AllowUsers\n")
		s.WriteString("\nAccess changes:\n")
		s.WriteString("  • Grant passwordless sudo and restrict SSH to allowed users\n")
	case actionInstallTools:
		s.WriteString("  • Install ripgrep, fd, bat, yq, glow, gh\n")
	case actionInstallDevTools:
		s.WriteString("  • Install Go (system-wide)\n")
		s.WriteString("  • Install Node.js toolchain (per-user via fnm)\n")
	}

	s.WriteString("\n")
	if effAct == actionBootstrap {
		fmt.Fprintf(&s, "  Timezone: %s\n", m.timezone)
	}
	if effAct == actionAddUser || effAct == actionInstallDevTools {
		fmt.Fprintf(&s, "  Username: %s\n", m.username)
	}
	if effAct == actionAddUser && m.sshKey != "" {
		fmt.Fprintf(&s, "  SSH key: %s\n", user.SSHKeySummary(m.sshKey))
	}

	if m.dryRun {
		s.WriteString("\n" + cursorStyle.Render("  DRY RUN — no changes will be made"))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("enter run · esc back · q quit"))
	return s.String()
}

func (m model) runningView() string {
	var s strings.Builder

	if m.isChain() {
		s.WriteString(titleStyle.Render(fmt.Sprintf("Full Setup — Step %d/%d",
			m.chainIdx+1, len(fullSetupChain))))
		s.WriteString("\n\n")

		for i, step := range m.steps {
			var icon string
			if i == m.chainIdx && step.status == stepRunning {
				icon = "[" + cursorStyle.Render(spinnerChar(m.spinnerFrame)) + "]"
			} else {
				switch step.status {
				case stepOK:
					icon = successStyle.Render("[✓]")
				case stepFail:
					icon = errorStyle.Render("[✗]")
				default:
					icon = dimStyle.Render("[ ]")
				}
			}
			line := fmt.Sprintf("  %s %s", icon, step.name)
			switch step.status {
			case stepOK:
				line = successStyle.Render(line)
			case stepFail:
				line = errorStyle.Render(line)
			default:
				if i != m.chainIdx {
					line = dimStyle.Render(line)
				}
			}
			s.WriteString(line + "\n")
			if step.status == stepOK && step.output != "" {
				s.WriteString(dimStyle.Render(indentLines(truncateOutput(step.output, 1800), "      ")) + "\n")
			}
		}

		s.WriteString("\n")
		s.WriteString(drawProgressBar(m.chainProgress(), 30))
	} else {
		act := m.effectiveAction()
		s.WriteString(titleStyle.Render(stepNames[act]))
		s.WriteString("\n\n")

		for _, step := range m.steps {
			spinner := cursorStyle.Render(spinnerChar(m.spinnerFrame))
			line := fmt.Sprintf("  %s %s", spinner, step.name)
			switch step.status {
			case stepOK:
				line = successStyle.Render(fmt.Sprintf("  [✓] %s", step.name))
			case stepFail:
				line = errorStyle.Render(fmt.Sprintf("  [✗] %s — %s", step.name, step.output))
			}
			s.WriteString(line + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Running... please wait"))
	return s.String()
}

func (m model) doneView() string {
	var s strings.Builder

	if m.isChain() {
		actionLabel := stepNames[m.effectiveAction()]
		lastStep := m.chainIdx >= len(fullSetupChain)-1
		s.WriteString(titleStyle.Render(fmt.Sprintf("Full Setup — %s complete", actionLabel)))
		fmt.Fprintf(&s, " (%d/%d)", m.chainIdx+1, len(fullSetupChain))
		s.WriteString("\n\n")

		for _, step := range m.steps {
			var icon string
			switch step.status {
			case stepOK:
				icon = successStyle.Render("[✓]")
			case stepFail:
				icon = errorStyle.Render("[✗]")
			default:
				icon = dimStyle.Render("[ ]")
			}
			line := fmt.Sprintf("  %s %s", icon, step.name)
			switch step.status {
			case stepOK:
				line = successStyle.Render(line)
			case stepFail:
				line = errorStyle.Render(line + " — " + step.output)
			default:
				line = dimStyle.Render(line)
			}
			s.WriteString(line + "\n")
		}

		s.WriteString("\n")
		if lastStep {
			allOK := true
			for _, step := range m.steps {
				if step.status == stepFail {
					allOK = false
					break
				}
			}
			if allOK {
				s.WriteString(successStyle.Render("All steps completed successfully."))
			} else {
				s.WriteString(errorStyle.Render("Full setup stopped after a failed step."))
			}
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("enter back to menu · q quit"))
		} else if m.steps[m.runningStepIndex()].status == stepFail {
			s.WriteString(errorStyle.Render("This step failed. Fix the issue or retry before continuing."))
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("enter retry · esc back to menu · q quit"))
		} else {
			nextAct := fullSetupChain[m.chainIdx+1]
			fmt.Fprintf(&s, "Next: %s\n\n", stepNames[nextAct])
			s.WriteString(helpStyle.Render("enter continue · esc back to menu · q quit"))
		}
	} else {
		s.WriteString(titleStyle.Render("Task complete"))
		s.WriteString("\n\n")

		for _, step := range m.steps {
			var icon string
			switch step.status {
			case stepOK:
				icon = successStyle.Render("[✓]")
			case stepFail:
				icon = errorStyle.Render("[✗]")
			default:
				icon = dimStyle.Render("[ ]")
			}
			line := fmt.Sprintf("  %s %s", icon, step.name)
			switch step.status {
			case stepOK:
				line = successStyle.Render(line)
			case stepFail:
				line = errorStyle.Render(line + " — " + step.output)
			}
			s.WriteString(line + "\n")
			if step.status == stepOK && step.output != "" {
				s.WriteString(dimStyle.Render(indentLines(truncateOutput(step.output, 1800), "      ")) + "\n")
			}
		}

		s.WriteString("\n")
		allOK := true
		for _, step := range m.steps {
			if step.status == stepFail {
				allOK = false
				break
			}
		}
		if allOK {
			s.WriteString(successStyle.Render("Completed successfully."))
		}
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("enter back to menu · q quit"))
	}

	return s.String()
}

func truncateKey(key string, max int) string {
	key = strings.TrimSpace(key)
	if len(key) <= max {
		return key
	}
	if max <= 1 {
		return key[:max]
	}
	if max <= 3 {
		return key[:max]
	}
	return key[:max-3] + "..."
}

func maxPreviewWidth(width int) int {
	if width <= 0 {
		return 72
	}
	if width < 50 {
		return 32
	}
	return min(width-10, 96)
}

func truncateOutput(output string, max int) string {
	output = strings.TrimSpace(output)
	if len(output) <= max {
		return output
	}
	return output[:max] + "\n..."
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
