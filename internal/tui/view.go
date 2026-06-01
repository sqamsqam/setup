package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/sqamsqam/setup/internal/user"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F4D35E"))
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A7C7E7"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8A8A8A"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DCFB6"))
	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F4D35E"))
	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DCFB6"))
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#686868"))
	logStepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A7C7E7"))
	logCommandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F4D35E"))
	logDoneStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DCFB6"))
	logErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B"))
	logPanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#E8E8E8"))
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E7D6B")).
			Padding(1, 2)
	runPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2E7D6B")).
			Padding(0, 1)
)

func (m model) mainMenuView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Ubuntu LXC Provisioning"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Edit the run plan, then continue through only the required inputs."))
	s.WriteString("\n\n")

	if os.Geteuid() != 0 {
		s.WriteString(errorStyle.Render("WARNING: not running as root. Provisioning may fail."))
		s.WriteString("\n")
	}
	if m.dryRun {
		s.WriteString(warnStyle.Render("DRY RUN: commands will be logged without changing the system."))
		s.WriteString("\n")
	}
	if m.planErr != "" {
		s.WriteString(errorStyle.Render(m.planErr))
		s.WriteString("\n")
	}
	if os.Geteuid() != 0 || m.dryRun || m.planErr != "" {
		s.WriteString("\n")
	}

	s.WriteString(m.planList.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("%d selected item(s)", m.selectedPlanCount())))
	return s.String()
}

func (m model) inputTimezoneView() string {
	body := "Timezone\n\n"
	body += m.timezoneInput.View()
	body += "\n\n"
	body += m.timezoneMatchesView()
	body += "\n\n"
	body += dimStyle.Render("Fuzzy search is supported. Use Tab to accept, Up/Down to choose. Blank defaults to UTC.")
	if m.inputErr != "" {
		body += "\n\n" + errorStyle.Render(m.inputErr)
	}
	return m.page("System Bootstrap", "Set the container timezone.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) timezoneMatchesView() string {
	if len(m.timezoneMatches) == 0 {
		if strings.TrimSpace(m.timezoneInput.Value()) == "" {
			return dimStyle.Render("Type to search timezones.")
		}
		return dimStyle.Render("No matching timezones.")
	}
	var b strings.Builder
	matches := m.timezoneMatches
	if len(matches) > maxTimezoneMatches {
		matches = matches[:maxTimezoneMatches]
	}
	for i, match := range matches {
		if i > 0 {
			b.WriteString("\n")
		}
		prefix := "  "
		style := dimStyle
		if i == m.timezoneMatchIndex {
			prefix = "> "
			style = accentStyle
		}
		b.WriteString(style.Render(prefix + match))
	}
	return b.String()
}

func (m model) inputUserView() string {
	body := "Username\n\n"
	body += m.usernameInput.View()
	body += "\n\n"
	body += dimStyle.Render("Must match ^[a-z_][a-z0-9_-]*$ and be 32 characters or fewer.")
	if m.inputErr != "" {
		body += "\n\n" + errorStyle.Render(m.inputErr)
	}
	return m.page("Target User", "Used for account creation and per-user Node.js tooling.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputKeyView() string {
	body := "SSH public key\n\n"
	body += m.sshKeyInput.View()
	pubkey := normalizeSSHKeyInput(m.sshKeyInput.Value())
	if m.inputErr != "" {
		body += "\n\n" + errorStyle.Render(m.inputErr)
	} else if pubkey != "" {
		if err := user.ValidateSSHKey(pubkey); err == nil {
			body += "\n\n" + dimStyle.Render(user.SSHKeySummary(pubkey))
		}
	}
	return m.page("Add User", "Paste the public key that should be installed for the user.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) confirmView() string {
	return m.page("Confirm Run", "Review the exact plan before provisioning starts.", m.confirm.View(), []key.Binding{keys.Continue, keys.Back, keys.Scroll, keys.Quit})
}

func (m model) confirmBody() string {
	var body strings.Builder
	body.WriteString("Selected plan\n\n")
	for _, line := range m.planSummaryLines() {
		body.WriteString("  ")
		body.WriteString(line)
		body.WriteString("\n")
	}

	body.WriteString("\nInputs\n")
	if m.selections.NeedsTimezone() {
		fmt.Fprintf(&body, "  Timezone: %s\n", strings.TrimSpace(m.timezoneInput.Value()))
	}
	if m.selections.NeedsUsername() {
		fmt.Fprintf(&body, "  Username: %s\n", strings.TrimSpace(m.usernameInput.Value()))
	}
	if m.selections.NeedsSSHKey() {
		pubkey := normalizeSSHKeyInput(m.sshKeyInput.Value())
		fmt.Fprintf(&body, "  SSH key: %s\n", user.SSHKeySummary(pubkey))
	}

	if m.selections.Bootstrap || m.selections.AddUser {
		body.WriteString("\nAccess changes\n")
	}
	if m.selections.Bootstrap {
		body.WriteString("  SSH will be hardened, root SSH login disabled, and the root password locked.\n")
	}
	if m.selections.AddUser {
		body.WriteString("  The user receives passwordless sudo and SSH AllowUsers is regenerated.\n")
	}
	if m.dryRun {
		body.WriteString("\n")
		body.WriteString(warnStyle.Render("DRY RUN: no changes will be made."))
		body.WriteString("\n")
	}

	return body.String()
}

func (m model) runningView() string {
	var s strings.Builder
	contentWidth := m.runContentWidth()

	s.WriteString(titleStyle.Render("Running Provisioning"))
	if summary := m.currentStepSummary(); summary != "" {
		s.WriteString("  ")
		s.WriteString(m.spinner.View())
		s.WriteString(" ")
		s.WriteString(subtitleStyle.Render(summary))
	}
	s.WriteString("\n\n")
	s.WriteString(m.progress.ViewAs(m.runProgress()))
	s.WriteString("\n\n")
	s.WriteString(m.runBodyView())
	s.WriteString("\n\n")
	m.help.SetWidth(contentWidth)
	s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.Scroll, keys.Quit}}))
	return s.String()
}

func (m model) doneView() string {
	var s strings.Builder
	if m.currentStepFailed() {
		s.WriteString(errorStyle.Render("Provisioning stopped"))
		s.WriteString("\n")
		s.WriteString(subtitleStyle.Render("Fix the issue, then retry the failed step or go back to the plan."))
	} else {
		s.WriteString(successStyle.Render("Provisioning complete"))
		s.WriteString("\n")
		s.WriteString(subtitleStyle.Render(fmt.Sprintf("%d step(s) completed successfully.", m.completedRunSteps())))
	}
	s.WriteString("\n\n")
	s.WriteString(m.runBodyView())

	s.WriteString("\n\n")
	m.help.SetWidth(m.runContentWidth())
	if m.currentStepFailed() {
		s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.Retry, keys.Back, keys.Scroll, keys.Quit}}))
	} else {
		s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.Continue, keys.Scroll, keys.Quit}}))
	}
	return s.String()
}

func (m model) runBodyView() string {
	if m.usesRunColumns() {
		steps := runPanelStyle.
			Width(m.stepPanelWidth()).
			Height(m.output.Height()).
			MaxHeight(m.output.Height()).
			Render(m.steps.View())
		log := runPanelStyle.
			Width(m.output.Width() + 2).
			Height(m.output.Height()).
			MaxHeight(m.output.Height()).
			Render(m.logPanelView())
		return lipgloss.JoinHorizontal(lipgloss.Top, steps, "  ", log)
	}

	var s strings.Builder
	s.WriteString(runPanelStyle.
		Width(m.steps.Width() + 2).
		Height(m.steps.Height()).
		MaxHeight(m.steps.Height()).
		Render(m.steps.View()))
	s.WriteString("\n")
	s.WriteString(runPanelStyle.
		Width(m.output.Width() + 2).
		Height(m.output.Height()).
		MaxHeight(m.output.Height()).
		Render(m.logPanelView()))
	return s.String()
}

func (m model) logPanelView() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logPanelTitleStyle.Render("Output log"),
		"  ",
		dimStyle.Render(logScrollHint(m.output)),
	)

	bodyHeight := m.output.Height() - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := m.logView(bodyHeight)
	return header + "\n" + dimStyle.Render(strings.Repeat("─", max(1, m.output.Width()))) + "\n" + body
}

func (m model) logView(height int) string {
	if strings.TrimSpace(m.output.GetContent()) == "" {
		return dimStyle.Render("Log output will appear here.")
	}
	return lipgloss.NewStyle().Height(height).MaxHeight(height).Render(m.output.View())
}

func logScrollHint(v viewport.Model) string {
	if strings.TrimSpace(v.GetContent()) == "" {
		return "waiting"
	}
	switch {
	case v.AtTop() && v.AtBottom():
		return "all output visible"
	case v.AtBottom():
		return "bottom"
	case v.AtTop():
		return "top"
	default:
		return fmt.Sprintf("%3.0f%%", v.ScrollPercent()*100)
	}
}

func (m model) page(title, subtitle, body string, bindings []key.Binding) string {
	width := m.width - 6
	if width < 50 {
		width = 50
	}
	if width > 96 {
		width = 96
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render(title))
	if subtitle != "" {
		s.WriteString("\n")
		s.WriteString(subtitleStyle.Render(subtitle))
	}
	s.WriteString("\n\n")
	s.WriteString(panelStyle.Width(width).Render(body))
	s.WriteString("\n\n")
	s.WriteString(m.help.View(helpKeyMap{short: bindings}))
	return s.String()
}

func (m model) stepsContent() string {
	var s strings.Builder
	for i, step := range m.runSteps {
		icon := statusIcon(step.status)
		if step.status == stepRunning {
			icon = "[" + m.spinner.View() + "]"
		}
		line := fmt.Sprintf("%s %s", icon, step.name)
		switch step.status {
		case stepOK:
			line = successStyle.Render(line)
		case stepFail:
			line = errorStyle.Render(line)
		case stepPending:
			line = dimStyle.Render(line)
		default:
			if i == m.runningIndex {
				line = accentStyle.Render(line)
			}
		}
		s.WriteString("  ")
		s.WriteString(line)
		if step.desc != "" && step.status == stepPending {
			s.WriteString("\n      ")
			s.WriteString(dimStyle.Render(step.desc))
		}
		s.WriteString("\n")
	}
	return s.String()
}

func (m model) planSummaryLines() []string {
	var lines []string
	if m.selections.Bootstrap {
		lines = append(lines, "System Bootstrap")
	}
	if m.selections.AddUser {
		lines = append(lines, "Add User")
	}
	for _, tool := range m.selections.Tools.SelectedTools() {
		lines = append(lines, "CLI Tool: "+string(tool))
	}
	if m.selections.DevTools.Go {
		lines = append(lines, "Development Tool: Go")
	}
	if m.selections.DevTools.Node {
		lines = append(lines, "Development Tool: Node.js")
	}
	return lines
}

func truncateKey(key string, max int) string {
	key = strings.TrimSpace(key)
	if len(key) <= max {
		return key
	}
	if max <= 3 {
		return key[:max]
	}
	return key[:max-3] + "..."
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

func colorizeLog(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "▶ "):
			lines[i] = logStepStyle.Render(line)
		case strings.HasPrefix(trimmed, "== ") && strings.HasSuffix(trimmed, " =="):
			lines[i] = logStepStyle.Render(line)
		case strings.HasPrefix(trimmed, "$ ") || strings.HasPrefix(trimmed, "[DRY-RUN]"):
			lines[i] = logCommandStyle.Render(line)
		case strings.HasPrefix(trimmed, "→ "):
			lines[i] = logStepStyle.Render(line)
		case strings.HasPrefix(trimmed, "✓ "):
			lines[i] = logDoneStyle.Render(line)
		case strings.HasPrefix(trimmed, "✗ "):
			lines[i] = logErrorStyle.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
