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
	s.WriteString(titleStyle.Render(areaTitle(m.currentArea)))
	s.WriteString(" ")
	s.WriteString(statusStyle.Render("ADMIN CONSOLE"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render(areaSubtitle(m.currentArea)))
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
	s.WriteString(helpStyle.Render(fmt.Sprintf("%d selected action(s) in %s", m.selectedAreaCount(m.currentArea), areaTitle(m.currentArea))))
	return s.String()
}

func (m model) homeView() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Setup Admin Console"))
	s.WriteString(" ")
	s.WriteString(statusStyle.Render("NO DEFAULTS"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Choose a management area. Nothing runs until you review the plan."))
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
	if (!m.dryRun && !m.demo && os.Geteuid() != 0) || (m.dryRun && !m.demo) {
		s.WriteString("\n")
	}

	s.WriteString(m.homeList.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("%d total selected action(s)", m.selectedPlanCount())))
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

func (m model) inputGroupNameView() string {
	body := fieldLabelStyle.Render("GROUP")
	body += "\n\n"
	body += m.groupNameInput.View()
	body += "\n\n"
	body += dimStyle.Render("Must match ^[a-z_][a-z0-9_-]*$ and be 32 characters or fewer.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Target Group", "Used for group creation, deletion, and membership changes.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputServiceUserGroupsView() string {
	body := fieldLabelStyle.Render("SERVICE USER GROUPS")
	body += "\n\n"
	body += m.serviceGroupsInput.View()
	body += "\n\n"
	body += dimStyle.Render("Optional. Use comma or space separated existing group names.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Service User Groups", "Add the no-login service account to existing supplementary groups.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputServiceNameView() string {
	body := fieldLabelStyle.Render("SERVICE NAME")
	body += "\n\n"
	body += m.serviceNameInput.View()
	body += "\n\n"
	body += dimStyle.Render("Names are stored as setup-<name>.service. Existing setup- prefixes are accepted.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Managed Service", "Choose the setup-managed user service to operate on.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputServiceWorkDirView() string {
	body := fieldLabelStyle.Render("WORKING DIRECTORY")
	body += "\n\n"
	body += m.serviceWorkDir.View()
	body += "\n\n"
	body += dimStyle.Render("Use an absolute path owned by or accessible to the target user.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Service Workdir", "Set the directory systemd should run the service from.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputServiceCommandView() string {
	body := fieldLabelStyle.Render("COMMAND")
	body += "\n\n"
	body += m.serviceCommand.View()
	body += "\n\n"
	body += dimStyle.Render("A single non-empty shell command run through /bin/bash -lc.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Service Command", "Set the command for the managed systemd service.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputServiceEnvFileView() string {
	body := fieldLabelStyle.Render("ENVIRONMENT FILE")
	body += "\n\n"
	body += m.serviceEnvFile.View()
	body += "\n\n"
	body += dimStyle.Render("Optional. Leave blank to skip EnvironmentFile.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Service Environment", "Optionally reference an absolute environment file path.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputFirewallRuleView() string {
	body := inputLine("PORT", m.firewallPortInput.View())
	body += "\n\n" + inputLine("PROTO", m.firewallProtoInput.View())
	body += "\n\n" + inputLine("FROM", m.firewallFromInput.View())
	body += "\n\n" + inputLine("COMMENT", m.firewallComment.View())
	body += "\n\n" + dimStyle.Render("Tab moves between fields. FROM and COMMENT are optional.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Custom Firewall Rule", "Configure one UFW allow rule.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputNetworkRuleNumberView() string {
	body := fieldLabelStyle.Render("RULE NUMBER")
	body += "\n\n"
	body += m.networkRuleInput.View()
	body += "\n\n"
	body += dimStyle.Render("Use the number shown by the numbered network rules action.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Delete Network Rule", "Choose the numbered UFW rule to delete.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputFail2BanOptionsView() string {
	body := inputLine("BANTIME", m.fail2banBanTime.View())
	body += "\n\n" + inputLine("FINDTIME", m.fail2banFindTime.View())
	body += "\n\n" + inputLine("MAX RETRY", m.fail2banMaxRetry.View())
	body += "\n\n" + dimStyle.Render("Tab moves between fields. Defaults match CLI mode.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("fail2ban Options", "Configure the setup-managed SSH jail.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputDockerLogOptionsView() string {
	body := inputLine("MAX SIZE", m.dockerMaxSize.View())
	body += "\n\n" + inputLine("MAX FILE", m.dockerMaxFile.View())
	body += "\n\n" + dimStyle.Render("Tab moves between fields. Defaults match CLI mode.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Docker Log Rotation", "Configure json-file log rotation values.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputDockerPruneTargetsView() string {
	body := fieldLabelStyle.Render("PRUNE TARGETS")
	body += "\n\n"
	body += m.pruneTargetsInput.View()
	body += "\n\n"
	body += dimStyle.Render("Use containers, images, and/or build-cache, separated by spaces or commas.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("Docker Prune", "Choose which Docker resources to prune.", body, []key.Binding{keys.Continue, keys.Back})
}

func (m model) inputGuardIPView() string {
	body := fieldLabelStyle.Render("IP ADDRESS")
	body += "\n\n"
	body += m.guardIPInput.View()
	body += "\n\n"
	body += dimStyle.Render("The IP will be unbanned from the fail2ban sshd jail.")
	if m.inputErr != "" {
		body += "\n\n" + errorBlock(m.inputErr)
	}
	return m.page("fail2ban Unban", "Choose the IP address to unban.", body, []key.Binding{keys.Continue, keys.Back})
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
	return m.page("SSH Key", "Paste the public key that should be installed for the user.", body, []key.Binding{keys.Continue, keys.Back})
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
	if m.selections.NeedsGroupName() {
		fmt.Fprintf(&body, "  Group: %s\n", strings.TrimSpace(m.groupNameInput.Value()))
	}
	if m.selections.UserServiceGroups && strings.TrimSpace(m.serviceGroupsInput.Value()) != "" {
		fmt.Fprintf(&body, "  Service user groups: %s\n", strings.TrimSpace(m.serviceGroupsInput.Value()))
	}
	if m.selections.NeedsServiceName() {
		fmt.Fprintf(&body, "  Service: %s\n", strings.TrimSpace(m.serviceNameInput.Value()))
	}
	if m.selections.NeedsServiceWorkDir() {
		fmt.Fprintf(&body, "  Service workdir: %s\n", strings.TrimSpace(m.serviceWorkDir.Value()))
	}
	if m.selections.NeedsServiceCommand() {
		fmt.Fprintf(&body, "  Service command: %s\n", strings.TrimSpace(m.serviceCommand.Value()))
	}
	if m.selections.NeedsServiceEnvFile() && strings.TrimSpace(m.serviceEnvFile.Value()) != "" {
		fmt.Fprintf(&body, "  Service env file: %s\n", strings.TrimSpace(m.serviceEnvFile.Value()))
	}
	if m.selections.NeedsFirewallRule() {
		fmt.Fprintf(&body, "  Firewall rule: %s/%s\n", strings.TrimSpace(m.firewallPortInput.Value()), strings.TrimSpace(m.firewallProtoInput.Value()))
	}
	if m.selections.NeedsNetworkRuleNumber() {
		fmt.Fprintf(&body, "  Network rule number: %s\n", strings.TrimSpace(m.networkRuleInput.Value()))
	}
	if m.selections.NeedsGuardIP() {
		fmt.Fprintf(&body, "  Unban IP: %s\n", strings.TrimSpace(m.guardIPInput.Value()))
	}
	if m.selections.NeedsDockerPruneTargets() {
		fmt.Fprintf(&body, "  Docker prune targets: %s\n", strings.TrimSpace(m.pruneTargetsInput.Value()))
	}
	if m.selections.NeedsSSHKey() {
		pubkey := normalizeSSHKeyInput(m.sshKeyInput.Value())
		fmt.Fprintf(&body, "  SSH key: %s\n", user.SSHKeySummary(pubkey))
	}

	if m.selections.Bootstrap || m.selections.UserManagementAny() {
		body.WriteString("\n")
		body.WriteString(sectionStyle.Render("Access changes"))
		body.WriteString("\n")
		body.WriteString(divider(48))
		body.WriteString("\n")
	}
	if m.selections.Bootstrap {
		body.WriteString("  SSH hardening, root SSH disabled, root password locked.\n")
	}
	if m.selections.UserSSHKey {
		body.WriteString("  The SSH public key will be appended for the target user.\n")
	}
	if m.selections.UserAllowSSH {
		body.WriteString("  The user will be added to the setup-managed SSH AllowUsers list.\n")
	}
	if m.selections.UserDenySSH {
		body.WriteString("  The user will be removed from the setup-managed SSH AllowUsers list.\n")
	}
	if m.selections.UserSudo {
		body.WriteString("  Setup-managed passwordless sudo will be enabled for the user.\n")
	}
	if m.selections.UserSudoDisable {
		body.WriteString("  Setup-managed passwordless sudo will be removed for the user.\n")
	}
	if m.selections.UserLinger {
		body.WriteString("  systemd user lingering will be enabled.\n")
	}
	if m.selections.UserLingerDisable {
		body.WriteString("  systemd user lingering will be disabled.\n")
	}
	if m.selections.UserDockerGroup {
		body.WriteString("  The user will be added to the existing docker group.\n")
	}
	if m.selections.UserCreateService {
		body.WriteString("  A setup-owned no-login service user will be created under /var/lib/<user>.\n")
		if m.selections.UserServiceGroups {
			body.WriteString("  The service user will be added to the selected existing groups.\n")
		}
	}
	if m.selections.UserDisable {
		body.WriteString("  The user password will be locked and setup-managed access removed.\n")
	}
	if m.selections.UserDelete {
		body.WriteString("  The user account will be deleted after access is disabled; the home directory is preserved.\n")
	}
	if m.selections.GroupAny() {
		body.WriteString("\n")
		body.WriteString(sectionStyle.Render("Groups"))
		body.WriteString("\n")
		body.WriteString(divider(48))
		body.WriteString("\n")
	}
	if m.selections.GroupCreate {
		body.WriteString("  The group will be created if needed.\n")
	}
	if m.selections.GroupDelete {
		body.WriteString("  The group will be deleted only if it is not a primary group for existing users.\n")
	}
	if m.selections.GroupList {
		body.WriteString("  System groups will be listed.\n")
	}
	if m.selections.GroupAddUser {
		body.WriteString("  The target user will be added to the selected group.\n")
	}
	if m.selections.GroupRemoveUser {
		body.WriteString("  The target user will be removed from the selected group.\n")
	}
	if m.selections.ServiceAny() {
		body.WriteString("\n")
		body.WriteString(sectionStyle.Render("Managed services"))
		body.WriteString("\n")
		body.WriteString(divider(48))
		body.WriteString("\n")
	}
	if m.selections.ServiceCreate {
		body.WriteString("  A setup-managed per-user systemd service will be created and started.\n")
	}
	if m.selections.ServiceStatus {
		body.WriteString("  Status will be read only after verifying the unit is setup-managed.\n")
	}
	if m.selections.ServiceLogs {
		body.WriteString("  Recent logs will be read only after verifying the unit is setup-managed.\n")
	}
	if m.selections.ServiceRestart {
		body.WriteString("  The setup-managed service will be restarted.\n")
	}
	if m.selections.ServiceList {
		body.WriteString("  Setup-managed service units will be listed for the target user.\n")
	}
	if m.selections.ServiceDisable {
		body.WriteString("  The setup-managed service will be stopped and disabled.\n")
	}
	if m.selections.ServiceRemove {
		body.WriteString("  The setup-managed service unit file will be removed after stopping and disabling it.\n")
	}
	if m.selections.FirewallBaseline {
		body.WriteString("  UFW will allow the detected SSH port before enabling default-deny incoming rules.\n")
	}
	if m.selections.FirewallHTTP || m.selections.FirewallHTTPS || m.selections.FirewallMosh {
		body.WriteString("  Selected common firewall ports will be opened through UFW.\n")
	}
	if m.selections.FirewallCustom {
		body.WriteString("  The custom firewall rule will be opened through UFW.\n")
	}
	if m.selections.NetworkStatus {
		body.WriteString("  Verbose UFW status will be displayed.\n")
	}
	if m.selections.NetworkList {
		body.WriteString("  Numbered UFW rules will be displayed.\n")
	}
	if m.selections.NetworkDelete {
		body.WriteString("  The selected numbered UFW rule will be deleted.\n")
	}
	if m.selections.NetworkReset {
		body.WriteString("  UFW rules will be reset.\n")
	}
	if m.selections.Fail2Ban {
		body.WriteString("  fail2ban will manage a setup-owned SSH jail.\n")
	}
	if m.selections.Fail2BanStatus {
		body.WriteString("  fail2ban SSH jail status will be displayed.\n")
	}
	if m.selections.Fail2BanUnban {
		body.WriteString("  The selected IP will be unbanned from fail2ban.\n")
	}
	if m.selections.DockerLogRotation {
		body.WriteString("  Docker daemon log rotation will be merged into daemon.json and Docker restarted only if changed.\n")
	}
	if m.selections.ContainersDisk {
		body.WriteString("  Docker disk usage will be displayed.\n")
	}
	if m.selections.ContainersPrune {
		body.WriteString("  Selected Docker resources will be pruned.\n")
	}
	if m.selections.UpdatesUpgrade {
		body.WriteString("  Apt metadata will be refreshed and packages upgraded.\n")
	}
	if m.selections.UpdatesRebootNeed {
		body.WriteString("  Reboot-required state will be displayed.\n")
	}
	if m.selections.UpdatesUnattended {
		body.WriteString("  unattended-upgrades service status will be displayed.\n")
	}
	if m.selections.UpdatesFailed {
		body.WriteString("  Failed systemd units will be displayed.\n")
	}
	if m.selections.UpdatesReboot {
		body.WriteString("  The instance will reboot.\n")
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
		s.WriteString(successStyle.Render("Setup actions complete"))
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
	if m.selections.UserCreateLogin {
		lines = append(lines, "User Management: create login user")
	}
	if m.selections.UserSSHKey {
		lines = append(lines, "User Management: add SSH key")
	}
	if m.selections.UserAllowSSH {
		lines = append(lines, "User Management: allow SSH login")
	}
	if m.selections.UserDenySSH {
		lines = append(lines, "User Management: deny SSH login")
	}
	if m.selections.UserSudo {
		lines = append(lines, "User Management: passwordless sudo")
	}
	if m.selections.UserSudoDisable {
		lines = append(lines, "User Management: disable sudo")
	}
	if m.selections.UserLinger {
		lines = append(lines, "User Management: enable linger")
	}
	if m.selections.UserLingerDisable {
		lines = append(lines, "User Management: disable linger")
	}
	if m.selections.UserDockerGroup {
		lines = append(lines, "User Management: docker group")
	}
	if m.selections.UserDisable {
		lines = append(lines, "User Management: disable user")
	}
	if m.selections.UserDelete {
		lines = append(lines, "User Management: delete user")
	}
	if m.selections.UserCreateService {
		lines = append(lines, "User Management: create service user")
		if m.selections.UserServiceGroups {
			lines = append(lines, "User Management: service user groups")
		}
	}
	if m.selections.GroupCreate {
		lines = append(lines, "Group Management: create group")
	}
	if m.selections.GroupDelete {
		lines = append(lines, "Group Management: delete group")
	}
	if m.selections.GroupList {
		lines = append(lines, "Group Management: list groups")
	}
	if m.selections.GroupAddUser {
		lines = append(lines, "Group Management: add user to group")
	}
	if m.selections.GroupRemoveUser {
		lines = append(lines, "Group Management: remove user from group")
	}
	if m.selections.ServiceCreate {
		lines = append(lines, "Managed Service: create")
	}
	if m.selections.ServiceStatus {
		lines = append(lines, "Managed Service: status")
	}
	if m.selections.ServiceLogs {
		lines = append(lines, "Managed Service: logs")
	}
	if m.selections.ServiceRestart {
		lines = append(lines, "Managed Service: restart")
	}
	if m.selections.ServiceList {
		lines = append(lines, "Managed Service: list")
	}
	if m.selections.ServiceDisable {
		lines = append(lines, "Managed Service: disable")
	}
	if m.selections.ServiceRemove {
		lines = append(lines, "Managed Service: remove")
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
	if m.selections.FirewallCustom {
		lines = append(lines, "Firewall Rule: custom")
	}
	if m.selections.NetworkStatus {
		lines = append(lines, "Network: status")
	}
	if m.selections.NetworkList {
		lines = append(lines, "Network: numbered rules")
	}
	if m.selections.NetworkDelete {
		lines = append(lines, "Network: delete rule")
	}
	if m.selections.NetworkReset {
		lines = append(lines, "Network: reset firewall")
	}
	if m.selections.Fail2Ban {
		lines = append(lines, "Instance Management: fail2ban SSH jail")
	}
	if m.selections.Fail2BanStatus {
		lines = append(lines, "Instance Management: fail2ban status")
	}
	if m.selections.Fail2BanUnban {
		lines = append(lines, "Instance Management: fail2ban unban")
	}
	if m.selections.DockerLogRotation {
		lines = append(lines, "Instance Management: Docker log rotation")
	}
	if m.selections.ContainersDisk {
		lines = append(lines, "Instance Management: Docker disk usage")
	}
	if m.selections.ContainersPrune {
		lines = append(lines, "Instance Management: Docker prune")
	}
	if m.selections.Diagnostics {
		lines = append(lines, "Instance Management: Doctor diagnostics")
	}
	if m.selections.UpdatesCheck {
		lines = append(lines, "Instance Management: Update check")
	}
	if m.selections.UpdatesUpgrade {
		lines = append(lines, "Instance Management: full upgrade")
	}
	if m.selections.UpdatesRebootNeed {
		lines = append(lines, "Instance Management: reboot needed")
	}
	if m.selections.UpdatesUnattended {
		lines = append(lines, "Instance Management: unattended status")
	}
	if m.selections.UpdatesFailed {
		lines = append(lines, "Instance Management: failed units")
	}
	if m.selections.UpdatesReboot {
		lines = append(lines, "Instance Management: reboot")
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

func inputLine(label, value string) string {
	return fieldLabelStyle.Render(label) + "\n" + value
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
