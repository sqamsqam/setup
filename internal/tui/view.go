package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

func (m model) welcomeView() string {
	s := titleStyle.Render("Ubuntu LXC Provisioning Tool")
	s += "\n\n"
	s += "Target: Ubuntu 26.04 LXC container (amd64)\n"
	s += "Press any key to continue, or q to quit.\n"
	s += "\n"
	s += helpStyle.Render("This tool will guide you through setting up a fresh container.\n")
	return s
}

func (m model) stepSelectView() string {
	s := titleStyle.Render("Select steps to run")
	s += "\n\n"

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
		s += line + "\n"
	}

	s += "\n"
	s += helpStyle.Render("↑/↓ move · space toggle · c continue · q quit")
	return s
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

	s += "Key: " + display
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
	s := titleStyle.Render("Confirm provisioning")

	s += "\n\n"
	s += "The following steps will run:\n\n"

	for i, f := range m.stepFlags {
		if f {
			s += "  • " + m.steps[i].name + "\n"
		}
	}

	if m.needsUserInput() {
		s += fmt.Sprintf("\n  Username: %s\n", m.username)
	}
	if m.needsKeyInput() {
		s += fmt.Sprintf("  SSH key: %s...\n", truncateKey(m.sshKey, 40))
	}
	if m.needsTimezoneInput() {
		s += fmt.Sprintf("\n  Timezone: %s\n", m.timezone)
	}

	if m.dryRun {
		s += "\n" + cursorStyle.Render("  DRY RUN — no changes will be made")
	}

	s += "\n\n"
	s += helpStyle.Render("enter run · esc back · q quit")
	return s
}

func (m model) runningView() string {
	s := titleStyle.Render("Running provisioning...")
	s += "\n\n"

	for i, step := range m.steps {
		if !m.stepFlags[i] {
			s += dimStyle.Render("  " + step.name + " (skipped)")
			s += "\n"
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
		s += line + "\n"
		if step.output != "" && step.status == stepFail {
			s += "    " + errorStyle.Render(step.output) + "\n"
		}
	}

	s += "\n"
	s += helpStyle.Render("Running... please wait")
	return s
}

func (m model) doneView() string {
	s := titleStyle.Render("Provisioning complete")
	s += "\n\n"

	allOK := true
	for i, f := range m.stepFlags {
		if !f {
			continue
		}
		icon := statusIcon(m.steps[i].status)
		switch m.steps[i].status {
		case stepOK:
			s += successStyle.Render(fmt.Sprintf("  %s %s", icon, m.steps[i].name)) + "\n"
		case stepFail:
			s += errorStyle.Render(fmt.Sprintf("  %s %s — %s", icon, m.steps[i].name, m.steps[i].output)) + "\n"
			allOK = false
		default:
			s += dimStyle.Render(fmt.Sprintf("  %s %s", icon, m.steps[i].name)) + "\n"
		}
	}

	s += "\n"
	if allOK {
		s += successStyle.Render("All steps completed successfully.")
	} else {
		s += errorStyle.Render("Some steps failed. Check the output above.")
	}

	s += "\n\n"
	s += helpStyle.Render("Press any key to exit")
	return s
}

func truncateKey(key string, max int) string {
	key = strings.TrimSpace(key)
	if len(key) <= max {
		return key
	}
	return key[:max]
}
