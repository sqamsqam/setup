package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

func (m model) welcomeView() string {
	s := titleStyle.Render("Ubuntu LXC Provisioning Tool")
	s += "\n\n"
	s += "Target: Ubuntu 26.04 LXC container (amd64)\n"
	s += "Press any key to continue, or q to quit.\n"
	s += "\n"
	s += helpStyle.Render("This tool will guide you through setting up a fresh container.\n")
	s += "\n"
	if os.Geteuid() != 0 {
		s += errorStyle.Render("Requires root privileges — run with sudo.\n")
	}
	return s
}

func (m model) stepSelectView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Select steps to run"))
	s.WriteString("\n\n")

	for i, step := range m.steps {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("❯ ")
		}

		checked := "[ ]"
		if m.stepFlags[i] {
			checked = successStyle.Render("[✓]")
		}

		line := cursor + checked + " " + stepNameStyle.Render(step.name)
		if step.status != stepPending {
			switch step.status {
			case stepOK:
				line += "  " + successStyle.Render("done")
			case stepFail:
				line += "  " + errorStyle.Render("failed")
			case stepRunning:
				line += "  " + cursorStyle.Render("running...")
			}
		}
		s.WriteString(line + "\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓ move · space toggle · c continue · q quit"))
	return s.String()
}

func (m model) inputUserView() string {
	s := titleStyle.Render("Add user")
	s += "\n\n"
	s += "Username: " + cursorStyle.Render(m.username)
	if m.username == "" {
		s += dimStyle.Render(" (type to enter)")
	}
	s += "\n\n"
	s += helpStyle.Render("enter confirm · backspace delete · q quit")
	return s
}

func (m model) inputKeyView() string {
	s := titleStyle.Render("SSH public key")
	s += "\n\n"

	display := m.sshKey
	if len(display) == 0 {
		display = dimStyle.Render("(paste your public key)")
	}

	s += "Key: " + truncateKey(display, 40)
	s += "\n\n"
	s += helpStyle.Render("paste key, then press enter · backspace delete · q quit")
	return s
}

func (m model) inputTimezoneView() string {
	s := titleStyle.Render("Timezone")
	s += "\n\n"
	s += "Timezone: " + cursorStyle.Render(m.timezone)
	s += "\n\n"
	s += helpStyle.Render("enter confirm · backspace edit · q quit")
	return s
}

func (m model) confirmView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Confirm provisioning"))

	s.WriteString("\n\n")
	s.WriteString("The following steps will run:\n\n")

	for i, f := range m.stepFlags {
		if f {
			s.WriteString("  • " + m.steps[i].name + "\n")
		}
	}

	if m.needsUserInput() {
		s.WriteString(fmt.Sprintf("\n  Username: %s\n", m.username))
	}
	if m.needsKeyInput() {
		s.WriteString(fmt.Sprintf("  SSH key: %s...\n", truncateKey(m.sshKey, 40)))
	}
	if m.needsTimezoneInput() {
		s.WriteString(fmt.Sprintf("\n  Timezone: %s\n", m.timezone))
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
	s.WriteString(titleStyle.Render("Running provisioning..."))
	s.WriteString("\n\n")

	for i, step := range m.steps {
		if !m.stepFlags[i] {
			s.WriteString(dimStyle.Render("  " + step.name + " (skipped)"))
			s.WriteString("\n")
			continue
		}
		icon := statusIcon(step.status)
		line := fmt.Sprintf("  %s %s", icon, step.name)
		switch step.status {
		case stepOK:
			line = successStyle.Render(line)
		case stepFail:
			line = errorStyle.Render(line)
		case stepRunning:
			line = cursorStyle.Render(line)
		default:
			line = dimStyle.Render(line)
		}
		s.WriteString(line + "\n")
		if step.output != "" && step.status == stepFail {
			s.WriteString("    " + errorStyle.Render(step.output) + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Running... please wait"))
	return s.String()
}

func (m model) doneView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Provisioning complete"))
	s.WriteString("\n\n")

	allOK := true
	for i, f := range m.stepFlags {
		if !f {
			continue
		}
		icon := statusIcon(m.steps[i].status)
		switch m.steps[i].status {
		case stepOK:
			s.WriteString(successStyle.Render(fmt.Sprintf("  %s %s", icon, m.steps[i].name)) + "\n")
		case stepFail:
			s.WriteString(errorStyle.Render(fmt.Sprintf("  %s %s — %s", icon, m.steps[i].name, m.steps[i].output)) + "\n")
			allOK = false
		default:
			s.WriteString(dimStyle.Render(fmt.Sprintf("  %s %s", icon, m.steps[i].name)) + "\n")
		}
	}

	s.WriteString("\n")
	if allOK {
		s.WriteString(successStyle.Render("All steps completed successfully."))
	} else {
		s.WriteString(errorStyle.Render("Some steps failed. Check the output above."))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Press any key to exit"))
	return s.String()
}

func truncateKey(key string, max int) string {
	key = strings.TrimSpace(key)
	if len(key) <= max {
		return key
	}
	return key[:max]
}
