package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sqamsqam/setup/internal/user"
)

func (m model) mainMenuView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Fresh Ubuntu Instance Setup"))
	s.WriteString(" ")
	s.WriteString(statusStyle.Render("PROVISIONING CONSOLE"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Pick what this container needs, then review the plan."))
	s.WriteString("\n\n")

	if !m.dryRun && !m.demo && os.Geteuid() != 0 {
		s.WriteString(errorStyle.Render("ROOT CHECK"))
		s.WriteString(" ")
		s.WriteString(valueStyle.Render("Not running as root. Provisioning may fail."))
		s.WriteString("\n")
	}
	if m.dryRun && !m.demo {
		s.WriteString(warnStyle.Render("DRY RUN"))
		s.WriteString(" ")
		s.WriteString(valueStyle.Render("Commands will be logged without changing the system."))
		s.WriteString("\n")
	}
	if m.planErr != "" {
		s.WriteString(errorStyle.Render("PLAN CHECK"))
		s.WriteString(" ")
		s.WriteString(valueStyle.Render(m.planErr))
		s.WriteString("\n")
	}
	if (!m.dryRun && !m.demo && os.Geteuid() != 0) || (m.dryRun && !m.demo) || m.planErr != "" {
		s.WriteString("\n")
	}

	s.WriteString(m.planList.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("%d selected item(s)", m.selectedPlanCount())))
	return s.String()
}

func (m model) inputTimezoneView() string {
	body := fieldLabelStyle.Render("TIMEZONE")
	body += "\n\n"
	body += m.timezoneInput.View()
	body += "\n\n"
	body += m.timezoneMatchesView()
	body += "\n\n"
	body += dimStyle.Render("Fuzzy search is supported. Tab accepts a match; blank defaults to UTC.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
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
			prefix = "› "
			style = accentStyle
		}
		b.WriteString(style.Render(prefix + match))
	}
	return b.String()
}

func (m model) inputUserView() string {
	body := fieldLabelStyle.Render("USERNAME")
	body += "\n\n"
	body += m.usernameInput.View()
	body += "\n\n"
	body += dimStyle.Render("Must match ^[a-z_][a-z0-9_-]*$ and be 32 characters or fewer.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Target User", "Used for account creation and per-user Node.js tooling.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputKeyView() string {
	body := fieldLabelStyle.Render("SSH PUBLIC KEY")
	body += "\n\n"
	body += m.sshKeyInput.View()
	pubkey := normalizeSSHKeyInput(m.sshKeyInput.Value())
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	} else if pubkey != "" {
		if err := user.ValidateSSHKey(pubkey); err == nil {
			body += "\n\n" + accentStyle.Render("Verified key")
			body += " "
			body += dimStyle.Render(user.SSHKeySummary(pubkey))
		}
	}
	return m.page("Add User", "Paste the public key that should be installed for the user.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) confirmView() string {
	return m.page("Review Plan", "Check the exact plan before anything runs.", m.confirm.View(), []key.Binding{keys.Continue, keys.Back, keys.Scroll, keys.Quit})
}

func (m model) confirmBody() string {
	var body strings.Builder
	body.WriteString(sectionStyle.Render("Selected plan"))
	body.WriteString("\n")
	body.WriteString(divider(48))
	body.WriteString("\n")
	for _, line := range m.planSummaryLines() {
		body.WriteString(accentStyle.Render("  • "))
		body.WriteString(line)
		body.WriteString("\n")
	}

	body.WriteString("\n")
	body.WriteString(sectionStyle.Render("Inputs"))
	body.WriteString("\n")
	body.WriteString(divider(48))
	body.WriteString("\n")
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
		body.WriteString("\n")
		body.WriteString(sectionStyle.Render("Access changes"))
		body.WriteString("\n")
		body.WriteString(divider(48))
		body.WriteString("\n")
	}
	if m.selections.Bootstrap {
		body.WriteString("  SSH hardening, root SSH disabled, root password locked.\n")
	}
	if m.selections.AddUser {
		body.WriteString("  User receives passwordless sudo and managed SSH AllowUsers.\n")
	}
	if m.selections.FirewallBaseline {
		body.WriteString("  UFW will allow the detected SSH port before enabling default-deny incoming rules.\n")
	}
	if m.selections.FirewallHTTP || m.selections.FirewallHTTPS || m.selections.FirewallMosh {
		body.WriteString("  Selected common firewall ports will be opened through UFW.\n")
	}
	if m.selections.Fail2Ban {
		body.WriteString("  fail2ban will manage a setup-owned SSH jail.\n")
	}
	if m.selections.DockerLogRotation {
		body.WriteString("  Docker daemon log rotation will be merged into daemon.json and Docker restarted only if changed.\n")
	}
	if m.dryRun && !m.demo {
		body.WriteString("\n")
		body.WriteString(warnStyle.Render("DRY RUN"))
		body.WriteString(" ")
		body.WriteString(valueStyle.Render("No changes will be made."))
		body.WriteString("\n")
	}

	return body.String()
}

func (m model) runningView() string {
	var s strings.Builder
	contentWidth := m.runContentWidth()

	s.WriteString(titleStyle.Render("Getting Things Ready"))
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
	s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.StepNav, keys.Expand, keys.Scroll, keys.Quit}}))
	return s.String()
}

func (m model) doneView() string {
	var s strings.Builder
	if m.currentStepFailed() {
		s.WriteString(errorStyle.Render("Setup stopped"))
		s.WriteString("\n")
		s.WriteString(subtitleStyle.Render("Fix the issue, then retry the failed step or go back to the plan."))
	} else {
		s.WriteString(successStyle.Render("Fresh setup complete"))
		s.WriteString("\n")
		s.WriteString(subtitleStyle.Render(fmt.Sprintf("%d step(s) completed successfully.", m.completedRunSteps())))
	}
	s.WriteString("\n\n")
	s.WriteString(m.runBodyView())

	s.WriteString("\n\n")
	m.help.SetWidth(m.runContentWidth())
	if m.currentStepFailed() {
		s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.Retry, keys.Back, keys.StepNav, keys.Show, keys.Scroll, keys.Quit}}))
	} else {
		s.WriteString(m.help.View(helpKeyMap{short: []key.Binding{keys.Continue, keys.StepNav, keys.Show, keys.Scroll, keys.Quit}}))
	}
	return s.String()
}

func (m model) runBodyView() string {
	if m.usesRunColumns() {
		steps := runPanelStyle.
			Width(m.stepPanelWidth()).
			Height(m.output.Height()).
			Render(m.steps.View())
		log := runPanelStyle.
			Width(m.output.Width() + 2).
			Height(m.output.Height()).
			Render(m.logPanelView())
		return lipgloss.JoinHorizontal(lipgloss.Top, steps, "  ", log)
	}

	var s strings.Builder
	s.WriteString(runPanelStyle.
		Width(m.steps.Width() + 2).
		Height(m.steps.Height()).
		Render(m.steps.View()))
	s.WriteString("\n")
	s.WriteString(runPanelStyle.
		Width(m.output.Width() + 2).
		Height(m.output.Height()).
		Render(m.logPanelView()))
	return s.String()
}

func (m model) logPanelView() string {
	title := "STEP OUTPUT"
	if m.expandedRunStep >= 0 && m.expandedRunStep < len(m.runSteps) {
		titleWidth := m.output.Width() - 18
		if titleWidth < 1 {
			titleWidth = 1
		}
		title = ansi.Truncate(m.runSteps[m.expandedRunStep].name, titleWidth, "…")
		if title == "" {
			title = "STEP OUTPUT"
		}
	}
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logPanelTitleStyle.Render(title),
		"  ",
		statusStyle.Render(logScrollHint(m.output)),
	)

	bodyHeight := m.output.Height() - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := m.logView(bodyHeight)
	dividerWidth := m.output.Width() - 2
	if dividerWidth < 1 {
		dividerWidth = 1
	}
	return header + "\n" + divider(dividerWidth) + "\n" + body
}

func (m model) logView(height int) string {
	if strings.TrimSpace(m.output.GetContent()) == "" {
		if m.expandedRunStep >= 0 && m.expandedRunStep < len(m.runSteps) {
			return dimStyle.Render("This step has not produced output yet.")
		}
		if m.screen == screenDone {
			return dimStyle.Render("Select a step and press space to view output.")
		}
		return dimStyle.Render("Select a step and press enter or space to view output.")
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
		selector := "  "
		if i == m.selectedRunStep {
			selector = selectedStripeStyle.Render("▌") + " "
		}
		expander := " "
		if step.output != "" {
			expander = "▸"
			if i == m.expandedRunStep {
				expander = "▾"
			}
		}
		line := fmt.Sprintf("%s %s %s", runStatusIcon(step.status, m.spinner.View()), expander, step.name)
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
		s.WriteString(selector)
		s.WriteString(line)
		if step.desc != "" && step.status == stepPending {
			s.WriteString("\n      ")
			s.WriteString(faintStyle.Render(step.desc))
		}
		s.WriteString("\n")
	}
	return s.String()
}

func runStatusIcon(status stepStatus, spinnerView string) string {
	if status == stepRunning {
		return spinnerView
	}
	return statusIcon(status)
}

func errorBlock(message string) string {
	return errorStyle.Render("INPUT CHECK") + " " + valueStyle.Render(message)
}

func (m model) planSummaryLines() []string {
	var lines []string
	if m.selections.Bootstrap {
		lines = append(lines, "System Bootstrap")
	}
	if m.selections.AddUser {
		lines = append(lines, "Add User")
	}
	if m.selections.FirewallBaseline {
		lines = append(lines, "Instance Management: UFW firewall baseline")
	}
	if m.selections.FirewallHTTP {
		lines = append(lines, "Firewall Rule: allow HTTP")
	}
	if m.selections.FirewallHTTPS {
		lines = append(lines, "Firewall Rule: allow HTTPS")
	}
	if m.selections.FirewallMosh {
		lines = append(lines, "Firewall Rule: allow Mosh")
	}
	if m.selections.Fail2Ban {
		lines = append(lines, "Instance Management: fail2ban SSH jail")
	}
	if m.selections.DockerLogRotation {
		lines = append(lines, "Instance Management: Docker log rotation")
	}
	if m.selections.Diagnostics {
		lines = append(lines, "Instance Management: Doctor diagnostics")
	}
	if m.selections.UpdatesCheck {
		lines = append(lines, "Instance Management: Update check")
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
	if m.selections.DevTools.Rust {
		lines = append(lines, "Development Tool: Rust")
	}
	if m.selections.DevTools.GoLint {
		lines = append(lines, "Development Tool: golangci-lint")
	}
	if m.selections.DevTools.GoReleaser {
		lines = append(lines, "Development Tool: GoReleaser")
	}
	if m.selections.DevTools.GoVulnCheck {
		lines = append(lines, "Development Tool: govulncheck")
	}
	if m.selections.DevTools.Pnpm {
		lines = append(lines, "Development Tool: pnpm")
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

func truncateLogLines(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "…")
	}
	return strings.Join(lines, "\n")
}
