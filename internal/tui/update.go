package tui

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/sahilm/fuzzy"
	"github.com/sqamsqam/setup/internal/devtools"
	dockermaint "github.com/sqamsqam/setup/internal/docker"
	"github.com/sqamsqam/setup/internal/firewall"
	sysgroup "github.com/sqamsqam/setup/internal/group"
	"github.com/sqamsqam/setup/internal/service"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
)

const maxTimezoneMatches = 6

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case stepStatusMsg:
		return m.handleStepMsg(msg)

	case spinner.TickMsg:
		if m.screen != screenRunning {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.refreshSteps()
		return m, cmd

	case tea.PasteMsg:
		m.inputErr = ""
		switch m.screen {
		case screenInputTimezone:
			var cmd tea.Cmd
			m.timezoneInput, cmd = m.timezoneInput.Update(msg)
			m.refreshTimezoneMatches()
			return m, cmd
		case screenInputUser:
			var cmd tea.Cmd
			m.usernameInput, cmd = m.usernameInput.Update(msg)
			return m, cmd
		case screenInputGroupName:
			var cmd tea.Cmd
			m.groupNameInput, cmd = m.groupNameInput.Update(msg)
			return m, cmd
		case screenInputServiceUserGroups:
			var cmd tea.Cmd
			m.serviceGroupsInput, cmd = m.serviceGroupsInput.Update(msg)
			return m, cmd
		case screenInputServiceName:
			var cmd tea.Cmd
			m.serviceNameInput, cmd = m.serviceNameInput.Update(msg)
			return m, cmd
		case screenInputServiceWorkDir:
			var cmd tea.Cmd
			m.serviceWorkDir, cmd = m.serviceWorkDir.Update(msg)
			return m, cmd
		case screenInputServiceCommand:
			var cmd tea.Cmd
			m.serviceCommand, cmd = m.serviceCommand.Update(msg)
			return m, cmd
		case screenInputServiceEnvFile:
			var cmd tea.Cmd
			m.serviceEnvFile, cmd = m.serviceEnvFile.Update(msg)
			return m, cmd
		case screenInputFirewallRule:
			return m.updateFirewallRulePaste(msg)
		case screenInputNetworkRuleNumber:
			var cmd tea.Cmd
			m.networkRuleInput, cmd = m.networkRuleInput.Update(msg)
			return m, cmd
		case screenInputFail2BanOptions:
			return m.updateFail2BanPaste(msg)
		case screenInputDockerLogOptions:
			return m.updateDockerLogPaste(msg)
		case screenInputDockerPruneTargets:
			var cmd tea.Cmd
			m.pruneTargetsInput, cmd = m.pruneTargetsInput.Update(msg)
			return m, cmd
		case screenInputGuardIP:
			var cmd tea.Cmd
			m.guardIPInput, cmd = m.guardIPInput.Update(msg)
			return m, cmd
		case screenInputKey:
			var cmd tea.Cmd
			m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
			return m, cmd
		}

	case tea.KeyPressMsg:
		if m.quitting {
			return m, tea.Quit
		}

		switch m.screen {
		case screenHome:
			return m.updateHome(msg)
		case screenMainMenu:
			return m.updateMainMenu(msg)
		case screenInputTimezone:
			return m.updateInputTimezone(msg)
		case screenInputUser:
			return m.updateInputUser(msg)
		case screenInputGroupName:
			return m.updateInputGroupName(msg)
		case screenInputServiceUserGroups:
			return m.updateInputServiceUserGroups(msg)
		case screenInputServiceName:
			return m.updateInputServiceName(msg)
		case screenInputServiceWorkDir:
			return m.updateInputServiceWorkDir(msg)
		case screenInputServiceCommand:
			return m.updateInputServiceCommand(msg)
		case screenInputServiceEnvFile:
			return m.updateInputServiceEnvFile(msg)
		case screenInputFirewallRule:
			return m.updateInputFirewallRule(msg)
		case screenInputNetworkRuleNumber:
			return m.updateInputNetworkRuleNumber(msg)
		case screenInputFail2BanOptions:
			return m.updateInputFail2BanOptions(msg)
		case screenInputDockerLogOptions:
			return m.updateInputDockerLogOptions(msg)
		case screenInputDockerPruneTargets:
			return m.updateInputDockerPruneTargets(msg)
		case screenInputGuardIP:
			return m.updateInputGuardIP(msg)
		case screenInputKey:
			return m.updateInputKey(msg)
		case screenConfirm:
			return m.updateConfirm(msg)
		case screenRunning:
			return m.updateRunning(msg)
		case screenDone:
			return m.updateDone(msg)
		}

	case tea.MouseClickMsg:
		switch m.screen {
		case screenRunning, screenDone:
			return m.updateRunMouse(msg)
		}

	default:
		switch m.screen {
		case screenInputTimezone:
			var cmd tea.Cmd
			m.timezoneInput, cmd = m.timezoneInput.Update(msg)
			m.refreshTimezoneMatches()
			return m, cmd
		case screenInputUser:
			var cmd tea.Cmd
			m.usernameInput, cmd = m.usernameInput.Update(msg)
			return m, cmd
		case screenInputGroupName:
			var cmd tea.Cmd
			m.groupNameInput, cmd = m.groupNameInput.Update(msg)
			return m, cmd
		case screenInputServiceUserGroups:
			var cmd tea.Cmd
			m.serviceGroupsInput, cmd = m.serviceGroupsInput.Update(msg)
			return m, cmd
		case screenInputServiceName:
			var cmd tea.Cmd
			m.serviceNameInput, cmd = m.serviceNameInput.Update(msg)
			return m, cmd
		case screenInputServiceWorkDir:
			var cmd tea.Cmd
			m.serviceWorkDir, cmd = m.serviceWorkDir.Update(msg)
			return m, cmd
		case screenInputServiceCommand:
			var cmd tea.Cmd
			m.serviceCommand, cmd = m.serviceCommand.Update(msg)
			return m, cmd
		case screenInputServiceEnvFile:
			var cmd tea.Cmd
			m.serviceEnvFile, cmd = m.serviceEnvFile.Update(msg)
			return m, cmd
		case screenInputFirewallRule:
			return m.updateFirewallRulePaste(msg)
		case screenInputNetworkRuleNumber:
			var cmd tea.Cmd
			m.networkRuleInput, cmd = m.networkRuleInput.Update(msg)
			return m, cmd
		case screenInputFail2BanOptions:
			return m.updateFail2BanPaste(msg)
		case screenInputDockerLogOptions:
			return m.updateDockerLogPaste(msg)
		case screenInputDockerPruneTargets:
			var cmd tea.Cmd
			m.pruneTargetsInput, cmd = m.pruneTargetsInput.Update(msg)
			return m, cmd
		case screenInputGuardIP:
			var cmd tea.Cmd
			m.guardIPInput, cmd = m.guardIPInput.Update(msg)
			return m, cmd
		case screenInputKey:
			var cmd tea.Cmd
			m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
			return m, cmd
		case screenConfirm:
			var cmd tea.Cmd
			m.confirm, cmd = m.confirm.Update(msg)
			return m, cmd
		case screenRunning, screenDone:
			var cmd tea.Cmd
			m.output, cmd = m.output.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.homeList.SettingFilter() {
		var cmd tea.Cmd
		m.homeList, cmd = m.homeList.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Select):
		if item, ok := m.homeList.SelectedItem().(homeItem); ok {
			m.currentArea = item.area
			m.planErr = ""
			m.planList = m.newPlanList()
			if m.width > 0 && m.height > 0 {
				m.resize(m.width, m.height)
			}
			m.screen = screenMainMenu
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.homeList, cmd = m.homeList.Update(msg)
	return m, cmd
}

func (m model) updateMainMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.planList.SettingFilter() {
		var cmd tea.Cmd
		m.planList, cmd = m.planList.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.planErr = ""
		return m.goHome()
	case key.Matches(msg, keys.Toggle):
		m.planErr = ""
		if item, ok := m.planList.SelectedItem().(planItem); ok {
			m.togglePlanItem(item.id)
			return m, m.refreshPlanList()
		}
	case key.Matches(msg, keys.Continue):
		return m.startInputFlow()
	}

	var cmd tea.Cmd
	m.planList, cmd = m.planList.Update(msg)
	return m, cmd
}

func (m model) updateInputGroupName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if err := sysgroup.ValidateName(strings.TrimSpace(m.groupNameInput.Value())); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.groupNameInput, cmd = m.groupNameInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputTimezone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, m.timezoneInput.KeyMap.AcceptSuggestion):
		m.acceptTimezoneMatch()
		m.inputErr = ""
		return m, nil
	case key.Matches(msg, m.timezoneInput.KeyMap.NextSuggestion):
		m.selectTimezoneMatch(1)
		m.inputErr = ""
		return m, nil
	case key.Matches(msg, m.timezoneInput.KeyMap.PrevSuggestion):
		m.selectTimezoneMatch(-1)
		m.inputErr = ""
		return m, nil
	case key.Matches(msg, keys.Continue):
		tz := strings.TrimSpace(m.timezoneInput.Value())
		if tz == "" {
			tz = "UTC"
			m.timezoneInput.SetValue(tz)
			m.refreshTimezoneMatches()
		}
		if err := validateTimezone(tz); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.timezoneInput, cmd = m.timezoneInput.Update(msg)
		m.refreshTimezoneMatches()
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputUser(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if err := user.ValidateUsername(strings.TrimSpace(m.usernameInput.Value())); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.usernameInput, cmd = m.usernameInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputServiceUserGroups(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.serviceGroupsInput, cmd = m.serviceGroupsInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputServiceName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if err := service.ValidateName(strings.TrimSpace(m.serviceNameInput.Value())); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.serviceNameInput, cmd = m.serviceNameInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputServiceWorkDir(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		workdir := strings.TrimSpace(m.serviceWorkDir.Value())
		if workdir == "" || !filepath.IsAbs(workdir) {
			m.inputErr = "working directory must be an absolute path"
			return m, nil
		}
		if strings.ContainsAny(workdir, "\r\n") {
			m.inputErr = "working directory must be a single line"
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.serviceWorkDir, cmd = m.serviceWorkDir.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputServiceCommand(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		command := strings.TrimSpace(m.serviceCommand.Value())
		if command == "" || strings.ContainsAny(command, "\r\n") {
			m.inputErr = "command must be a non-empty single line"
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.serviceCommand, cmd = m.serviceCommand.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputServiceEnvFile(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		envFile := strings.TrimSpace(m.serviceEnvFile.Value())
		if envFile != "" && !filepath.IsAbs(envFile) {
			m.inputErr = "env file must be an absolute path"
			return m, nil
		}
		if strings.ContainsAny(envFile, "\r\n") {
			m.inputErr = "env file must be a single line"
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.serviceEnvFile, cmd = m.serviceEnvFile.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputFirewallRule(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.focusFirewallRuleField(m.inputSub + 1)
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.focusFirewallRuleField(m.inputSub - 1)
		return m, nil
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if err := firewall.ValidateRule(m.firewallRule()); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		return m.updateFirewallRuleInputs(msg)
	}
}

func (m model) updateInputNetworkRuleNumber(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if _, err := networkRuleNumber(m.networkRuleInput.Value()); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.networkRuleInput, cmd = m.networkRuleInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputFail2BanOptions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.focusFail2BanField(m.inputSub + 1)
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.focusFail2BanField(m.inputSub - 1)
		return m, nil
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if _, err := fail2banMaxRetry(m.fail2banMaxRetry.Value()); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		for _, field := range []struct {
			name  string
			value string
		}{
			{"bantime", m.fail2banBanTime.Value()},
			{"findtime", m.fail2banFindTime.Value()},
		} {
			value := strings.TrimSpace(field.value)
			if value == "" {
				m.inputErr = field.name + " must not be empty"
				return m, nil
			}
			if strings.ContainsAny(value, "\r\n") {
				m.inputErr = field.name + " must be a single line"
				return m, nil
			}
		}
		m.inputErr = ""
		return m.goNext()
	default:
		return m.updateFail2BanInputs(msg)
	}
}

func (m model) updateInputDockerLogOptions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.focusDockerLogField(m.inputSub + 1)
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.focusDockerLogField(m.inputSub - 1)
		return m, nil
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		for _, field := range []struct {
			name  string
			value string
		}{
			{"max size", m.dockerMaxSize.Value()},
			{"max file", m.dockerMaxFile.Value()},
		} {
			value := strings.TrimSpace(field.value)
			if value == "" {
				m.inputErr = field.name + " must not be empty"
				return m, nil
			}
			if strings.ContainsAny(value, "\r\n") {
				m.inputErr = field.name + " must be a single line"
				return m, nil
			}
		}
		m.inputErr = ""
		return m.goNext()
	default:
		return m.updateDockerLogInputs(msg)
	}
}

func (m model) updateInputDockerPruneTargets(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if _, err := parseDockerPruneTargets(m.pruneTargetsInput.Value()); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.pruneTargetsInput, cmd = m.pruneTargetsInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateInputGuardIP(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		if net.ParseIP(strings.TrimSpace(m.guardIPInput.Value())) == nil {
			m.inputErr = "invalid IP address"
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.guardIPInput, cmd = m.guardIPInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateFirewallRulePaste(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.updateFirewallRuleInputs(msg)
}

func (m model) updateFail2BanPaste(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.updateFail2BanInputs(msg)
}

func (m model) updateDockerLogPaste(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.updateDockerLogInputs(msg)
}

func (m model) updateFirewallRuleInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.inputSub {
	case 1:
		m.firewallProtoInput, cmd = m.firewallProtoInput.Update(msg)
	case 2:
		m.firewallFromInput, cmd = m.firewallFromInput.Update(msg)
	case 3:
		m.firewallComment, cmd = m.firewallComment.Update(msg)
	default:
		m.firewallPortInput, cmd = m.firewallPortInput.Update(msg)
	}
	m.inputErr = ""
	return m, cmd
}

func (m model) updateFail2BanInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.inputSub {
	case 1:
		m.fail2banFindTime, cmd = m.fail2banFindTime.Update(msg)
	case 2:
		m.fail2banMaxRetry, cmd = m.fail2banMaxRetry.Update(msg)
	default:
		m.fail2banBanTime, cmd = m.fail2banBanTime.Update(msg)
	}
	m.inputErr = ""
	return m, cmd
}

func (m model) updateDockerLogInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.inputSub == 1 {
		m.dockerMaxFile, cmd = m.dockerMaxFile.Update(msg)
	} else {
		m.dockerMaxSize, cmd = m.dockerMaxSize.Update(msg)
	}
	m.inputErr = ""
	return m, cmd
}

func (m model) updateInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		pubkey := normalizeSSHKeyInput(m.sshKeyInput.Value())
		m.sshKeyInput.SetValue(pubkey)
		if err := user.ValidateSSHKey(pubkey); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		m.inputErr = ""
		return m.goNext()
	default:
		var cmd tea.Cmd
		m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
		m.inputErr = ""
		return m, cmd
	}
}

func (m model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		flow := m.inputFlow()
		if len(flow) == 0 {
			m.screen = screenMainMenu
			return m, nil
		}
		m.inputPos = len(flow) - 1
		m.screen = flow[m.inputPos]
		return m, m.focusCurrentInput()
	case key.Matches(msg, keys.Continue):
		m.runSteps = m.buildRunSteps()
		if len(m.runSteps) == 0 {
			m.planErr = "select at least one action"
			m.screen = screenMainMenu
			return m, nil
		}
		if m.width > 0 && m.height > 0 {
			m.resize(m.width, m.height)
		}
		m.runningIndex = 0
		m.selectedRunStep = 0
		m.expandedRunStep = -1
		m.runSteps[m.runningIndex].status = stepRunning
		m.screen = screenRunning
		m.output.SetContent("")
		m.refreshSteps()
		m.refreshOutput()
		return m, tea.Batch(runProvisioningStep(m), tickSpinner(m.spinner))
	default:
		var cmd tea.Cmd
		m.confirm, cmd = m.confirm.Update(msg)
		return m, cmd
	}
}

func (m model) updateRunning(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case m.matchesRunStepUp(msg):
		m.selectRunStep(m.selectedRunStep - 1)
		return m, nil
	case m.matchesRunStepDown(msg):
		m.selectRunStep(m.selectedRunStep + 1)
		return m, nil
	case key.Matches(msg, keys.Expand):
		m.toggleExpandedRunStep(m.selectedRunStep)
		return m, nil
	}
	return m.updateRunViewports(msg)
}

func (m model) updateDone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.resetToPlan()
		return m, nil
	case m.matchesRunStepUp(msg):
		m.selectRunStep(m.selectedRunStep - 1)
		return m, nil
	case m.matchesRunStepDown(msg):
		m.selectRunStep(m.selectedRunStep + 1)
		return m, nil
	case key.Matches(msg, keys.Show):
		m.toggleExpandedRunStep(m.selectedRunStep)
		return m, nil
	case key.Matches(msg, keys.Retry):
		if m.currentStepFailed() {
			m.runSteps[m.runningIndex].status = stepRunning
			m.runSteps[m.runningIndex].output = ""
			m.selectedRunStep = m.runningIndex
			m.expandedRunStep = -1
			m.screen = screenRunning
			m.refreshSteps()
			m.refreshOutput()
			return m, tea.Batch(runProvisioningStep(m), tickSpinner(m.spinner))
		}
		return m.updateRunViewports(msg)
	case key.Matches(msg, keys.Continue):
		m.resetToPlan()
		return m, nil
	default:
		return m.updateRunViewports(msg)
	}
}

func (m model) updateRunViewports(msg tea.Msg) (tea.Model, tea.Cmd) {
	var stepsCmd, outputCmd tea.Cmd
	m.steps, stepsCmd = m.steps.Update(msg)
	m.output, outputCmd = m.output.Update(msg)
	return m, tea.Batch(stepsCmd, outputCmd)
}

func (m model) updateRunMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return m.updateRunViewports(msg)
	}
	if index, ok := m.runStepIndexAt(mouse.X, mouse.Y); ok {
		m.selectRunStep(index)
		m.toggleExpandedRunStep(index)
		return m, nil
	}
	return m.updateRunViewports(msg)
}

func tickSpinner(s spinner.Model) tea.Cmd {
	return func() tea.Msg {
		return s.Tick()
	}
}

func (m model) handleStepMsg(msg stepStatusMsg) (tea.Model, tea.Cmd) {
	if msg.index < 0 || msg.index >= len(m.runSteps) {
		return m, nil
	}
	m.runSteps[msg.index].status = msg.status
	m.runSteps[msg.index].output = msg.output
	m.selectedRunStep = msg.index
	if msg.status == stepFail {
		m.expandedRunStep = msg.index
	}
	m.refreshSteps()
	m.refreshOutput()

	if msg.status == stepFail {
		m.runningIndex = msg.index
		m.screen = screenDone
		return m, nil
	}

	next := msg.index + 1
	if next >= len(m.runSteps) {
		m.runningIndex = msg.index
		m.screen = screenDone
		return m, nil
	}

	m.runningIndex = next
	m.selectedRunStep = next
	m.runSteps[next].status = stepRunning
	m.refreshSteps()
	m.refreshOutput()
	return m, runProvisioningStep(m)
}

func (m *model) togglePlanItem(id planItemID) {
	switch id {
	case itemBootstrap:
		m.selections.Bootstrap = !m.selections.Bootstrap
	case itemUserAll:
		if m.selections.UserManagementAny() {
			m.selections.UserCreateLogin = false
			m.selections.UserSSHKey = false
			m.selections.UserAllowSSH = false
			m.selections.UserSudo = false
			m.selections.UserLinger = false
			m.selections.UserDockerGroup = false
			m.selections.UserCreateService = false
			m.selections.UserServiceGroups = false
		} else {
			m.selections.UserCreateLogin = true
			m.selections.UserSSHKey = true
			m.selections.UserAllowSSH = true
			m.selections.UserSudo = true
			m.selections.UserLinger = true
			m.selections.UserDockerGroup = true
			m.selections.UserCreateService = true
			m.selections.UserServiceGroups = true
		}
	case itemAddUser:
		if m.selections.UserLoginAll() {
			m.selections.UserCreateLogin = false
			m.selections.UserSSHKey = false
			m.selections.UserAllowSSH = false
			m.selections.UserSudo = false
			m.selections.UserLinger = false
			m.selections.UserDockerGroup = false
		} else {
			m.selections.UserCreateLogin = true
			m.selections.UserSSHKey = true
			m.selections.UserAllowSSH = true
			m.selections.UserSudo = true
			m.selections.UserLinger = true
			m.selections.UserDockerGroup = true
		}
	case itemUserCreateLogin:
		m.selections.UserCreateLogin = !m.selections.UserCreateLogin
	case itemUserSSHKey:
		m.selections.UserSSHKey = !m.selections.UserSSHKey
	case itemUserAllowSSH:
		m.selections.UserAllowSSH = !m.selections.UserAllowSSH
	case itemUserDenySSH:
		m.selections.UserDenySSH = !m.selections.UserDenySSH
	case itemUserSudo:
		m.selections.UserSudo = !m.selections.UserSudo
	case itemUserSudoDisable:
		m.selections.UserSudoDisable = !m.selections.UserSudoDisable
	case itemUserLinger:
		m.selections.UserLinger = !m.selections.UserLinger
	case itemUserLingerDis:
		m.selections.UserLingerDisable = !m.selections.UserLingerDisable
	case itemUserDockerGroup:
		m.selections.UserDockerGroup = !m.selections.UserDockerGroup
	case itemUserDisable:
		m.selections.UserDisable = !m.selections.UserDisable
	case itemUserDelete:
		m.selections.UserDelete = !m.selections.UserDelete
	case itemServiceUser:
		m.selections.UserCreateService = !m.selections.UserCreateService
		if !m.selections.UserCreateService {
			m.selections.UserServiceGroups = false
		}
	case itemServiceGroups:
		m.selections.UserServiceGroups = !m.selections.UserServiceGroups
		if m.selections.UserServiceGroups {
			m.selections.UserCreateService = true
		}
	case itemGroupCreate:
		m.selections.GroupCreate = !m.selections.GroupCreate
	case itemGroupDelete:
		m.selections.GroupDelete = !m.selections.GroupDelete
	case itemGroupList:
		m.selections.GroupList = !m.selections.GroupList
	case itemGroupAddUser:
		m.selections.GroupAddUser = !m.selections.GroupAddUser
	case itemGroupRemoveUser:
		m.selections.GroupRemoveUser = !m.selections.GroupRemoveUser
	case itemServiceAll:
		if m.selections.ServiceAll() {
			m.selections.ServiceCreate = false
			m.selections.ServiceStatus = false
			m.selections.ServiceLogs = false
			m.selections.ServiceRestart = false
			m.selections.ServiceList = false
			m.selections.ServiceDisable = false
			m.selections.ServiceRemove = false
		} else {
			m.selections.ServiceCreate = true
			m.selections.ServiceStatus = true
			m.selections.ServiceLogs = true
			m.selections.ServiceRestart = true
			m.selections.ServiceList = true
			m.selections.ServiceDisable = true
			m.selections.ServiceRemove = true
		}
	case itemServiceCreate:
		m.selections.ServiceCreate = !m.selections.ServiceCreate
	case itemServiceStatus:
		m.selections.ServiceStatus = !m.selections.ServiceStatus
	case itemServiceLogs:
		m.selections.ServiceLogs = !m.selections.ServiceLogs
	case itemServiceRestart:
		m.selections.ServiceRestart = !m.selections.ServiceRestart
	case itemServiceList:
		m.selections.ServiceList = !m.selections.ServiceList
	case itemServiceDisable:
		m.selections.ServiceDisable = !m.selections.ServiceDisable
	case itemServiceRemove:
		m.selections.ServiceRemove = !m.selections.ServiceRemove
	case itemManageAll:
		if m.selections.InstanceManagementAll() {
			m.selections.FirewallBaseline = false
			m.selections.FirewallHTTP = false
			m.selections.FirewallHTTPS = false
			m.selections.FirewallMosh = false
			m.selections.FirewallCustom = false
			m.selections.NetworkStatus = false
			m.selections.NetworkList = false
			m.selections.NetworkDelete = false
			m.selections.NetworkReset = false
			m.selections.Fail2Ban = false
			m.selections.Fail2BanStatus = false
			m.selections.Fail2BanUnban = false
			m.selections.DockerLogRotation = false
			m.selections.ContainersDisk = false
			m.selections.ContainersPrune = false
			m.selections.Diagnostics = false
			m.selections.UpdatesCheck = false
			m.selections.UpdatesUpgrade = false
			m.selections.UpdatesRebootNeed = false
			m.selections.UpdatesUnattended = false
			m.selections.UpdatesFailed = false
			m.selections.UpdatesReboot = false
		} else {
			m.selections.FirewallBaseline = true
			m.selections.FirewallHTTP = true
			m.selections.FirewallHTTPS = true
			m.selections.FirewallMosh = true
			m.selections.FirewallCustom = true
			m.selections.NetworkStatus = true
			m.selections.NetworkList = true
			m.selections.NetworkDelete = true
			m.selections.NetworkReset = true
			m.selections.Fail2Ban = true
			m.selections.Fail2BanStatus = true
			m.selections.Fail2BanUnban = true
			m.selections.DockerLogRotation = true
			m.selections.ContainersDisk = true
			m.selections.ContainersPrune = true
			m.selections.Diagnostics = true
			m.selections.UpdatesCheck = true
			m.selections.UpdatesUpgrade = true
			m.selections.UpdatesRebootNeed = true
			m.selections.UpdatesUnattended = true
			m.selections.UpdatesFailed = true
			m.selections.UpdatesReboot = true
		}
	case itemFirewall:
		m.selections.FirewallBaseline = !m.selections.FirewallBaseline
	case itemHTTP:
		m.selections.FirewallHTTP = !m.selections.FirewallHTTP
	case itemHTTPS:
		m.selections.FirewallHTTPS = !m.selections.FirewallHTTPS
	case itemMosh:
		m.selections.FirewallMosh = !m.selections.FirewallMosh
	case itemFirewallCustom:
		m.selections.FirewallCustom = !m.selections.FirewallCustom
	case itemNetworkStatus:
		m.selections.NetworkStatus = !m.selections.NetworkStatus
	case itemNetworkList:
		m.selections.NetworkList = !m.selections.NetworkList
	case itemNetworkDelete:
		m.selections.NetworkDelete = !m.selections.NetworkDelete
	case itemNetworkReset:
		m.selections.NetworkReset = !m.selections.NetworkReset
	case itemFail2Ban:
		m.selections.Fail2Ban = !m.selections.Fail2Ban
	case itemFail2BanStatus:
		m.selections.Fail2BanStatus = !m.selections.Fail2BanStatus
	case itemFail2BanUnban:
		m.selections.Fail2BanUnban = !m.selections.Fail2BanUnban
	case itemDockerLog:
		m.selections.DockerLogRotation = !m.selections.DockerLogRotation
	case itemContainersDisk:
		m.selections.ContainersDisk = !m.selections.ContainersDisk
	case itemContainersPrune:
		m.selections.ContainersPrune = !m.selections.ContainersPrune
	case itemDoctor:
		m.selections.Diagnostics = !m.selections.Diagnostics
	case itemUpdates:
		m.selections.UpdatesCheck = !m.selections.UpdatesCheck
	case itemUpdatesUpgrade:
		m.selections.UpdatesUpgrade = !m.selections.UpdatesUpgrade
	case itemUpdatesRebootN:
		m.selections.UpdatesRebootNeed = !m.selections.UpdatesRebootNeed
	case itemUpdatesUnattend:
		m.selections.UpdatesUnattended = !m.selections.UpdatesUnattended
	case itemUpdatesFailed:
		m.selections.UpdatesFailed = !m.selections.UpdatesFailed
	case itemUpdatesReboot:
		m.selections.UpdatesReboot = !m.selections.UpdatesReboot
	case itemCLIAll:
		if m.selections.Tools.Ripgrep && m.selections.Tools.Fd && m.selections.Tools.Bat &&
			m.selections.Tools.Yq && m.selections.Tools.Glow && m.selections.Tools.Gh {
			m.selections.Tools = tools.InstallOptions{}
		} else {
			m.selections.Tools = tools.AllInstallOptions()
		}
	case itemRipgrep:
		m.selections.Tools.Ripgrep = !m.selections.Tools.Ripgrep
	case itemFd:
		m.selections.Tools.Fd = !m.selections.Tools.Fd
	case itemBat:
		m.selections.Tools.Bat = !m.selections.Tools.Bat
	case itemYq:
		m.selections.Tools.Yq = !m.selections.Tools.Yq
	case itemGlow:
		m.selections.Tools.Glow = !m.selections.Tools.Glow
	case itemGh:
		m.selections.Tools.Gh = !m.selections.Tools.Gh
	case itemDevAll:
		if m.selections.DevTools.Go && m.selections.DevTools.Node && m.selections.DevTools.Rust &&
			m.selections.DevTools.GoLint && m.selections.DevTools.GoReleaser &&
			m.selections.DevTools.GoVulnCheck && m.selections.DevTools.Pnpm {
			m.selections.DevTools = devtools.InstallOptions{}
		} else {
			m.selections.DevTools = devtools.AllInstallOptions()
		}
	case itemGo:
		m.selections.DevTools.Go = !m.selections.DevTools.Go
	case itemNode:
		m.selections.DevTools.Node = !m.selections.DevTools.Node
	case itemRust:
		m.selections.DevTools.Rust = !m.selections.DevTools.Rust
	case itemGoLint:
		m.selections.DevTools.GoLint = !m.selections.DevTools.GoLint
	case itemGoRel:
		m.selections.DevTools.GoReleaser = !m.selections.DevTools.GoReleaser
	case itemGoVuln:
		m.selections.DevTools.GoVulnCheck = !m.selections.DevTools.GoVulnCheck
	case itemPnpm:
		m.selections.DevTools.Pnpm = !m.selections.DevTools.Pnpm
	}
}

func (m model) startInputFlow() (tea.Model, tea.Cmd) {
	m.planErr = ""
	if m.selectedAreaCount(m.currentArea) == 0 {
		m.planErr = "select at least one " + strings.ToLower(areaTitle(m.currentArea)) + " action"
		return m, nil
	}

	flow := m.inputFlow()
	if len(flow) == 0 {
		m.screen = screenConfirm
		m.refreshConfirm()
		return m, nil
	}
	m.inputPos = 0
	m.inputSub = 0
	m.screen = flow[0]
	return m, m.focusCurrentInput()
}

func (m model) goNext() (tea.Model, tea.Cmd) {
	flow := m.inputFlow()
	if m.inputPos < len(flow)-1 {
		m.inputPos++
		m.inputSub = 0
		m.screen = flow[m.inputPos]
		return m, m.focusCurrentInput()
	}
	m.screen = screenConfirm
	m.refreshConfirm()
	return m, nil
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	if m.inputPos > 0 {
		m.inputPos--
		m.inputSub = 0
		m.screen = m.inputFlow()[m.inputPos]
		return m, m.focusCurrentInput()
	}
	m.screen = screenMainMenu
	return m, nil
}

func (m model) goHome() (tea.Model, tea.Cmd) {
	m.screen = screenHome
	return m, m.refreshHomeList()
}

func (m *model) focusCurrentInput() tea.Cmd {
	m.usernameInput.Blur()
	m.groupNameInput.Blur()
	m.serviceGroupsInput.Blur()
	m.serviceNameInput.Blur()
	m.serviceWorkDir.Blur()
	m.serviceCommand.Blur()
	m.serviceEnvFile.Blur()
	m.firewallPortInput.Blur()
	m.firewallProtoInput.Blur()
	m.firewallFromInput.Blur()
	m.firewallComment.Blur()
	m.networkRuleInput.Blur()
	m.fail2banBanTime.Blur()
	m.fail2banFindTime.Blur()
	m.fail2banMaxRetry.Blur()
	m.dockerMaxSize.Blur()
	m.dockerMaxFile.Blur()
	m.pruneTargetsInput.Blur()
	m.guardIPInput.Blur()
	m.timezoneInput.Blur()
	m.sshKeyInput.Blur()

	switch m.screen {
	case screenInputTimezone:
		return m.timezoneInput.Focus()
	case screenInputUser:
		return m.usernameInput.Focus()
	case screenInputGroupName:
		return m.groupNameInput.Focus()
	case screenInputServiceUserGroups:
		return m.serviceGroupsInput.Focus()
	case screenInputServiceName:
		return m.serviceNameInput.Focus()
	case screenInputServiceWorkDir:
		return m.serviceWorkDir.Focus()
	case screenInputServiceCommand:
		return m.serviceCommand.Focus()
	case screenInputServiceEnvFile:
		return m.serviceEnvFile.Focus()
	case screenInputFirewallRule:
		return m.focusFirewallRuleField(0)
	case screenInputNetworkRuleNumber:
		return m.networkRuleInput.Focus()
	case screenInputFail2BanOptions:
		return m.focusFail2BanField(0)
	case screenInputDockerLogOptions:
		return m.focusDockerLogField(0)
	case screenInputDockerPruneTargets:
		return m.pruneTargetsInput.Focus()
	case screenInputGuardIP:
		return m.guardIPInput.Focus()
	case screenInputKey:
		return m.sshKeyInput.Focus()
	default:
		return nil
	}
}

func (m *model) focusFirewallRuleField(index int) tea.Cmd {
	m.inputSub = normalizeInputSub(index, 4)
	m.firewallPortInput.Blur()
	m.firewallProtoInput.Blur()
	m.firewallFromInput.Blur()
	m.firewallComment.Blur()
	switch m.inputSub {
	case 1:
		return m.firewallProtoInput.Focus()
	case 2:
		return m.firewallFromInput.Focus()
	case 3:
		return m.firewallComment.Focus()
	default:
		return m.firewallPortInput.Focus()
	}
}

func (m *model) focusFail2BanField(index int) tea.Cmd {
	m.inputSub = normalizeInputSub(index, 3)
	m.fail2banBanTime.Blur()
	m.fail2banFindTime.Blur()
	m.fail2banMaxRetry.Blur()
	switch m.inputSub {
	case 1:
		return m.fail2banFindTime.Focus()
	case 2:
		return m.fail2banMaxRetry.Focus()
	default:
		return m.fail2banBanTime.Focus()
	}
}

func (m *model) focusDockerLogField(index int) tea.Cmd {
	m.inputSub = normalizeInputSub(index, 2)
	m.dockerMaxSize.Blur()
	m.dockerMaxFile.Blur()
	if m.inputSub == 1 {
		return m.dockerMaxFile.Focus()
	}
	return m.dockerMaxSize.Focus()
}

func normalizeInputSub(index, count int) int {
	if count <= 0 {
		return 0
	}
	index %= count
	if index < 0 {
		index += count
	}
	return index
}

func (m model) firewallRule() firewall.Rule {
	return firewall.Rule{
		Port:    strings.TrimSpace(m.firewallPortInput.Value()),
		Proto:   strings.TrimSpace(m.firewallProtoInput.Value()),
		From:    strings.TrimSpace(m.firewallFromInput.Value()),
		Comment: strings.TrimSpace(m.firewallComment.Value()),
	}
}

func networkRuleNumber(value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("rule number must be 1 or greater")
	}
	return n, nil
}

func fail2banMaxRetry(value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("max retry must be 1 or greater")
	}
	return n, nil
}

func parseServiceGroups(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	groups := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		group := strings.TrimSpace(field)
		if group == "" || seen[group] {
			continue
		}
		seen[group] = true
		groups = append(groups, group)
	}
	return groups
}

func parseDockerPruneTargets(value string) (dockermaint.PruneOptions, error) {
	var opts dockermaint.PruneOptions
	fields := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	for _, field := range fields {
		switch strings.TrimSpace(field) {
		case "", "none":
		case "container", "containers":
			opts.Containers = true
		case "image", "images":
			opts.Images = true
		case "build-cache", "buildcache", "builder", "cache":
			opts.BuildCache = true
		default:
			return dockermaint.PruneOptions{}, fmt.Errorf("unknown prune target %q", field)
		}
	}
	if !opts.Containers && !opts.Images && !opts.BuildCache {
		return dockermaint.PruneOptions{}, fmt.Errorf("select at least one prune target")
	}
	return opts, nil
}

func (m *model) resetToPlan() {
	m.screen = screenMainMenu
	m.inputPos = 0
	m.inputSub = 0
	m.inputErr = ""
	m.planErr = ""
	m.runSteps = nil
	m.runningIndex = -1
	m.selectedRunStep = -1
	m.expandedRunStep = -1
	m.steps.SetContent("")
	m.output.SetContent("")
}

func (m *model) refreshConfirm() {
	m.confirm.SetContent(m.confirmBody())
	m.confirm.GotoTop()
}

func (m *model) refreshSteps() {
	m.steps.SetContent(m.stepsContent())
	if m.selectedRunStep >= 0 {
		m.steps.EnsureVisible(m.runStepStartLine(m.selectedRunStep), 0, 0)
	}
}

func (m model) currentStepFailed() bool {
	return m.runningIndex >= 0 &&
		m.runningIndex < len(m.runSteps) &&
		m.runSteps[m.runningIndex].status == stepFail
}

func (m model) buildRunSteps() []runStep {
	var steps []runStep
	if m.selections.Bootstrap {
		steps = append(steps, runStep{
			id:   runBootstrap,
			name: "System Bootstrap",
			desc: "Locale, packages, SSH hardening, unattended upgrades, Docker",
		})
	}
	if m.selections.UserCreateLogin {
		steps = append(steps, runStep{
			id:   runUserCreateLogin,
			name: "Create Login User",
			desc: "Create or reuse the target login account",
		})
	}
	if m.selections.UserSSHKey {
		steps = append(steps, runStep{
			id:   runUserSSHKey,
			name: "Add SSH Key",
			desc: "Append the provided public key to authorized_keys",
		})
	}
	if m.selections.UserAllowSSH {
		steps = append(steps, runStep{
			id:   runUserAllowSSH,
			name: "Allow SSH Login",
			desc: "Add the user to setup-managed AllowUsers",
		})
	}
	if m.selections.UserDenySSH {
		steps = append(steps, runStep{
			id:   runUserDenySSH,
			name: "Deny SSH Login",
			desc: "Remove the user from setup-managed AllowUsers",
		})
	}
	if m.selections.UserSudo {
		steps = append(steps, runStep{
			id:   runUserSudo,
			name: "Enable Passwordless Sudo",
			desc: "Write setup-managed sudoers file",
		})
	}
	if m.selections.UserSudoDisable {
		steps = append(steps, runStep{
			id:   runUserSudoDisable,
			name: "Disable Passwordless Sudo",
			desc: "Remove setup-managed sudoers file",
		})
	}
	if m.selections.UserLinger {
		steps = append(steps, runStep{
			id:   runUserLinger,
			name: "Enable User Linger",
			desc: "Enable systemd user lingering",
		})
	}
	if m.selections.UserLingerDisable {
		steps = append(steps, runStep{
			id:   runUserLingerDis,
			name: "Disable User Linger",
			desc: "Disable systemd user lingering",
		})
	}
	if m.selections.UserDockerGroup {
		steps = append(steps, runStep{
			id:   runUserDockerGroup,
			name: "Add Docker Group",
			desc: "Add the user to the existing docker group",
		})
	}
	if m.selections.UserDisable {
		steps = append(steps, runStep{
			id:   runUserDisable,
			name: "Disable User",
			desc: "Lock password and remove setup-managed access",
		})
	}
	if m.selections.UserDelete {
		steps = append(steps, runStep{
			id:   runUserDelete,
			name: "Delete User",
			desc: "Disable access and delete the account while preserving home",
		})
	}
	if m.selections.UserCreateService {
		steps = append(steps, runStep{
			id:   runServiceUser,
			name: "Create Service User",
			desc: "Create setup-owned no-login system account under /var/lib",
		})
	}
	if m.selections.GroupCreate {
		steps = append(steps, runStep{
			id:   runGroupCreate,
			name: "Create Group",
			desc: "Create the selected group if needed",
		})
	}
	if m.selections.GroupDelete {
		steps = append(steps, runStep{
			id:   runGroupDelete,
			name: "Delete Group",
			desc: "Delete the group after primary-group safety checks",
		})
	}
	if m.selections.GroupList {
		steps = append(steps, runStep{
			id:   runGroupList,
			name: "List Groups",
			desc: "List system groups",
		})
	}
	if m.selections.GroupAddUser {
		steps = append(steps, runStep{
			id:   runGroupAddUser,
			name: "Add User To Group",
			desc: "Add the target user to the selected group",
		})
	}
	if m.selections.GroupRemoveUser {
		steps = append(steps, runStep{
			id:   runGroupRemoveUser,
			name: "Remove User From Group",
			desc: "Remove the target user from the selected group",
		})
	}
	if m.selections.ServiceCreate {
		steps = append(steps, runStep{
			id:   runServiceCreate,
			name: "Create Managed Service",
			desc: "Create and start a setup-managed per-user systemd service",
		})
	}
	if m.selections.ServiceStatus {
		steps = append(steps, runStep{
			id:   runServiceStatus,
			name: "Show Service Status",
			desc: "Show systemd status for a managed service",
		})
	}
	if m.selections.ServiceLogs {
		steps = append(steps, runStep{
			id:   runServiceLogs,
			name: "Show Service Logs",
			desc: "Show recent journal output for a managed service",
		})
	}
	if m.selections.ServiceRestart {
		steps = append(steps, runStep{
			id:   runServiceRestart,
			name: "Restart Managed Service",
			desc: "Restart a setup-managed user service",
		})
	}
	if m.selections.ServiceList {
		steps = append(steps, runStep{
			id:   runServiceList,
			name: "List Managed Services",
			desc: "List setup-managed user service unit files",
		})
	}
	if m.selections.ServiceDisable {
		steps = append(steps, runStep{
			id:   runServiceDisable,
			name: "Disable Managed Service",
			desc: "Stop and disable a setup-managed user service",
		})
	}
	if m.selections.ServiceRemove {
		steps = append(steps, runStep{
			id:   runServiceRemove,
			name: "Remove Managed Service",
			desc: "Stop, disable, and delete a setup-managed unit file",
		})
	}
	if m.selections.FirewallBaseline {
		steps = append(steps, runStep{
			id:   runFirewall,
			name: "Configure UFW Firewall",
			desc: "Default deny incoming, allow outgoing, allow SSH, enable UFW",
		})
	}
	if m.selections.FirewallHTTP {
		steps = append(steps, runStep{
			id:   runHTTP,
			name: "Allow HTTP",
			desc: "Allow tcp/80 through UFW",
		})
	}
	if m.selections.FirewallHTTPS {
		steps = append(steps, runStep{
			id:   runHTTPS,
			name: "Allow HTTPS",
			desc: "Allow tcp/443 through UFW",
		})
	}
	if m.selections.FirewallMosh {
		steps = append(steps, runStep{
			id:   runMosh,
			name: "Allow Mosh",
			desc: "Allow udp/60000:61000 through UFW",
		})
	}
	if m.selections.FirewallCustom {
		steps = append(steps, runStep{
			id:   runFirewallCustom,
			name: "Allow Custom Firewall Rule",
			desc: "Allow the configured port or range through UFW",
		})
	}
	if m.selections.NetworkStatus {
		steps = append(steps, runStep{
			id:   runNetworkStatus,
			name: "Show Network Status",
			desc: "Show verbose UFW status",
		})
	}
	if m.selections.NetworkList {
		steps = append(steps, runStep{
			id:   runNetworkList,
			name: "List Network Rules",
			desc: "Show numbered UFW rules",
		})
	}
	if m.selections.NetworkDelete {
		steps = append(steps, runStep{
			id:   runNetworkDelete,
			name: "Delete Network Rule",
			desc: "Delete the configured numbered UFW rule",
		})
	}
	if m.selections.NetworkReset {
		steps = append(steps, runStep{
			id:   runNetworkReset,
			name: "Reset Firewall",
			desc: "Reset UFW rules",
		})
	}
	if m.selections.Fail2Ban {
		steps = append(steps, runStep{
			id:   runFail2Ban,
			name: "Configure fail2ban",
			desc: "Install fail2ban and enable the sshd jail",
		})
	}
	if m.selections.Fail2BanStatus {
		steps = append(steps, runStep{
			id:   runFail2BanStatus,
			name: "Show fail2ban Status",
			desc: "Show fail2ban SSH jail status",
		})
	}
	if m.selections.Fail2BanUnban {
		steps = append(steps, runStep{
			id:   runFail2BanUnban,
			name: "Unban fail2ban IP",
			desc: "Unban the configured IP address from the SSH jail",
		})
	}
	if m.selections.DockerLogRotation {
		steps = append(steps, runStep{
			id:   runDockerLog,
			name: "Configure Docker Log Rotation",
			desc: "Merge json-file log rotation into /etc/docker/daemon.json",
		})
	}
	if m.selections.ContainersDisk {
		steps = append(steps, runStep{
			id:   runContainersDisk,
			name: "Show Docker Disk Usage",
			desc: "Show Docker system disk usage",
		})
	}
	if m.selections.ContainersPrune {
		steps = append(steps, runStep{
			id:   runContainersPrune,
			name: "Prune Docker Resources",
			desc: "Prune selected Docker resources",
		})
	}
	if m.selections.Diagnostics {
		steps = append(steps, runStep{
			id:   runDoctor,
			name: "Run Doctor Diagnostics",
			desc: "Read-only instance health and environment checks",
		})
	}
	if m.selections.UpdatesCheck {
		steps = append(steps, runStep{
			id:   runUpdates,
			name: "Check Package Updates",
			desc: "Refresh apt metadata and list upgradable packages",
		})
	}
	if m.selections.UpdatesUpgrade {
		steps = append(steps, runStep{
			id:   runUpdatesUpgrade,
			name: "Upgrade Packages",
			desc: "Run apt update and full-upgrade",
		})
	}
	if m.selections.UpdatesRebootNeed {
		steps = append(steps, runStep{
			id:   runUpdatesRebootN,
			name: "Check Reboot Requirement",
			desc: "Show whether a reboot is required",
		})
	}
	if m.selections.UpdatesUnattended {
		steps = append(steps, runStep{
			id:   runUpdatesUnattend,
			name: "Show Unattended Status",
			desc: "Show unattended-upgrades service status",
		})
	}
	if m.selections.UpdatesFailed {
		steps = append(steps, runStep{
			id:   runUpdatesFailed,
			name: "Show Failed Units",
			desc: "Show failed systemd units",
		})
	}
	if m.selections.UpdatesReboot {
		steps = append(steps, runStep{
			id:   runUpdatesReboot,
			name: "Reboot Instance",
			desc: "Reboot the instance",
		})
	}
	if m.selections.Tools.Any() {
		steps = append(steps, runStep{
			id:   runToolDeps,
			name: "Prepare CLI Tool Dependencies",
			desc: "Install curl, wget, jq, gpg, and ca-certificates",
		})
		for _, tool := range m.selections.Tools.SelectedTools() {
			steps = append(steps, runStep{
				id:   runTool,
				tool: tool,
				name: "Install " + tools.ToolName(tool),
				desc: "Install selected CLI tool",
			})
		}
	}
	if m.selections.DevTools.Go {
		steps = append(steps, runStep{
			id:   runGo,
			name: "Install Go",
			desc: "System-wide Go from go.dev with SHA256 verification",
		})
	}
	if m.selections.DevTools.Node {
		steps = append(steps, runStep{
			id:   runNode,
			name: "Install Node.js",
			desc: "Per-user fnm, Node.js, corepack, TypeScript, and tsx",
		})
	}
	if m.selections.DevTools.Rust {
		steps = append(steps, runStep{
			id:   runRust,
			name: "Install Rust",
			desc: "Per-user stable Rust via rustup",
		})
	}
	if m.selections.DevTools.GoLint {
		steps = append(steps, runStep{
			id:   runGoLint,
			name: "Install golangci-lint",
			desc: "Verified release archive to /usr/local/bin",
		})
	}
	if m.selections.DevTools.GoReleaser {
		steps = append(steps, runStep{
			id:   runGoRel,
			name: "Install GoReleaser",
			desc: "Verified release archive to /usr/local/bin",
		})
	}
	if m.selections.DevTools.GoVulnCheck {
		steps = append(steps, runStep{
			id:   runGoVuln,
			name: "Install govulncheck",
			desc: "Official Go vulnerability scanner",
		})
	}
	if m.selections.DevTools.Pnpm {
		steps = append(steps, runStep{
			id:   runPnpm,
			name: "Install pnpm",
			desc: "Corepack-managed pnpm for the target user",
		})
	}
	return steps
}

func (m *model) refreshOutput() {
	if m.expandedRunStep < 0 || m.expandedRunStep >= len(m.runSteps) {
		m.output.SetContent("")
		m.output.GotoTop()
		return
	}

	step := m.runSteps[m.expandedRunStep]
	output := strings.TrimSpace(step.output)
	if output == "" {
		m.output.SetContent("")
		m.output.GotoTop()
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "▶ %s\n%s", step.name, output)
	m.output.SetContent(colorizeLog(truncateLogLines(b.String(), m.output.Width())))
	m.output.GotoBottom()
}

func (m model) matchesRunStepUp(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("up", "k")))
}

func (m model) matchesRunStepDown(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("down", "j")))
}

func (m *model) selectRunStep(index int) {
	if len(m.runSteps) == 0 {
		m.selectedRunStep = -1
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.runSteps) {
		index = len(m.runSteps) - 1
	}
	m.selectedRunStep = index
	m.refreshSteps()
}

func (m *model) toggleExpandedRunStep(index int) {
	if index < 0 || index >= len(m.runSteps) {
		return
	}
	if m.expandedRunStep == index {
		m.expandedRunStep = -1
	} else {
		m.expandedRunStep = index
	}
	m.refreshSteps()
	m.refreshOutput()
}

func (m model) runStepIndexAt(x, y int) (int, bool) {
	left, top, width, height := m.runStepViewportBounds()
	if x < left || x >= left+width || y < top || y >= top+height {
		return -1, false
	}
	return m.runStepAtContentLine(m.steps.YOffset() + y - top)
}

func (m model) runStepViewportBounds() (left, top, width, height int) {
	top = m.runBodyTop() + 1
	left = 2
	width = m.steps.Width()
	height = m.steps.Height()
	return left, top, width, height
}

func (m model) runBodyTop() int {
	switch m.screen {
	case screenRunning:
		return 4
	case screenDone:
		return 3
	default:
		return 0
	}
}

func (m model) runStepAtContentLine(line int) (int, bool) {
	if line < 0 {
		return -1, false
	}
	for i := range m.runSteps {
		start := m.runStepStartLine(i)
		end := start + m.runStepRenderedLines(i)
		if line >= start && line < end {
			return i, true
		}
	}
	return -1, false
}

func (m model) runStepStartLine(index int) int {
	line := 0
	for i := 0; i < index && i < len(m.runSteps); i++ {
		line += m.runStepRenderedLines(i)
	}
	return line
}

func (m model) runStepRenderedLines(index int) int {
	if index < 0 || index >= len(m.runSteps) {
		return 0
	}
	step := m.runSteps[index]
	if step.desc != "" && step.status == stepPending {
		return 2
	}
	return 1
}

func normalizeSSHKeyInput(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func validateTimezone(tz string) error {
	tz = strings.TrimSpace(tz)
	if tz == "UTC" {
		return nil
	}
	for _, zone := range availableTimezones() {
		if zone == tz {
			return nil
		}
	}
	return fmt.Errorf("unknown timezone %q", tz)
}

func (m *model) refreshTimezoneMatches() {
	m.timezoneMatches = fuzzyTimezoneMatches(m.timezoneInput.Value(), m.timezones, maxTimezoneMatches)
	if m.timezoneMatchIndex >= len(m.timezoneMatches) {
		m.timezoneMatchIndex = 0
	}
}

func (m *model) acceptTimezoneMatch() {
	if len(m.timezoneMatches) == 0 {
		return
	}
	m.timezoneInput.SetValue(m.timezoneMatches[m.timezoneMatchIndex])
	m.timezoneInput.CursorEnd()
	m.refreshTimezoneMatches()
}

func (m *model) selectTimezoneMatch(delta int) {
	if len(m.timezoneMatches) == 0 {
		return
	}
	m.timezoneMatchIndex = (m.timezoneMatchIndex + delta + len(m.timezoneMatches)) % len(m.timezoneMatches)
}

func fuzzyTimezoneMatches(query string, zones []string, limit int) []string {
	query = timezoneSearchPattern(query)
	if query == "" || limit <= 0 {
		return nil
	}
	matches := fuzzy.Find(query, zones)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.Str)
	}
	return out
}

func timezoneSearchPattern(query string) string {
	query = strings.TrimSpace(query)
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '/', '_', '-':
			return -1
		default:
			return r
		}
	}, query)
}

func availableTimezones() []string {
	var zones []string
	root := "/usr/share/zoneinfo"
	skipDirs := map[string]bool{
		"posix": true,
		"right": true,
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(rel, ".") || strings.HasPrefix(rel, "leap-seconds") || strings.HasPrefix(rel, "tzdata") || strings.HasPrefix(rel, "localtime") {
			return nil
		}
		zones = append(zones, filepath.ToSlash(rel))
		return nil
	})
	if len(zones) == 0 {
		return []string{"UTC"}
	}
	sort.Strings(zones)
	return zones
}
