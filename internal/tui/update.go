package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/sqamsqam/setup/internal/devtools"
	"github.com/sqamsqam/setup/internal/tools"
	"github.com/sqamsqam/setup/internal/user"
)

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
		return m, cmd

	case tea.PasteMsg:
		m.inputErr = ""
		switch m.screen {
		case screenInputTimezone:
			var cmd tea.Cmd
			m.timezoneInput, cmd = m.timezoneInput.Update(msg)
			return m, cmd
		case screenInputUser:
			var cmd tea.Cmd
			m.usernameInput, cmd = m.usernameInput.Update(msg)
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
		case screenMainMenu:
			return m.updateMainMenu(msg)
		case screenInputTimezone:
			return m.updateInputTimezone(msg)
		case screenInputUser:
			return m.updateInputUser(msg)
		case screenInputKey:
			return m.updateInputKey(msg)
		case screenConfirm:
			return m.updateConfirm(msg)
		case screenRunning:
			return m.updateRunning(msg)
		case screenDone:
			return m.updateDone(msg)
		}

	default:
		switch m.screen {
		case screenInputTimezone:
			var cmd tea.Cmd
			m.timezoneInput, cmd = m.timezoneInput.Update(msg)
			return m, cmd
		case screenInputUser:
			var cmd tea.Cmd
			m.usernameInput, cmd = m.usernameInput.Update(msg)
			return m, cmd
		case screenInputKey:
			var cmd tea.Cmd
			m.sshKeyInput, cmd = m.sshKeyInput.Update(msg)
			return m, cmd
		case screenRunning, screenDone:
			var cmd tea.Cmd
			m.output, cmd = m.output.Update(msg)
			return m, cmd
		}
	}

	return m, nil
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

func (m model) updateInputTimezone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.inputErr = ""
		return m.goBack()
	case key.Matches(msg, keys.Continue):
		tz := strings.TrimSpace(m.timezoneInput.Value())
		if tz == "" {
			tz = "UTC"
			m.timezoneInput.SetValue(tz)
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
			m.planErr = "select at least one provisioning item"
			m.screen = screenMainMenu
			return m, nil
		}
		m.runningIndex = 0
		m.runSteps[m.runningIndex].status = stepRunning
		m.screen = screenRunning
		m.output.SetContent("")
		m.refreshOutput()
		return m, tea.Batch(runProvisioningStep(m), tickSpinner(m.spinner))
	}
	return m, nil
}

func (m model) updateRunning(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Quit) {
		m.quitting = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return m, cmd
}

func (m model) updateDone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Back):
		m.resetToPlan()
		return m, nil
	case key.Matches(msg, keys.Continue):
		if m.currentStepFailed() {
			m.runSteps[m.runningIndex].status = stepRunning
			m.runSteps[m.runningIndex].output = ""
			m.screen = screenRunning
			m.refreshOutput()
			return m, tea.Batch(runProvisioningStep(m), tickSpinner(m.spinner))
		}
		m.resetToPlan()
		return m, nil
	default:
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		return m, cmd
	}
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
	m.runSteps[next].status = stepRunning
	m.refreshOutput()
	return m, runProvisioningStep(m)
}

func (m *model) togglePlanItem(id planItemID) {
	switch id {
	case itemBootstrap:
		m.selections.Bootstrap = !m.selections.Bootstrap
	case itemAddUser:
		m.selections.AddUser = !m.selections.AddUser
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
		if m.selections.DevTools.Go && m.selections.DevTools.Node {
			m.selections.DevTools = devtools.InstallOptions{}
		} else {
			m.selections.DevTools = devtools.AllInstallOptions()
		}
	case itemGo:
		m.selections.DevTools.Go = !m.selections.DevTools.Go
	case itemNode:
		m.selections.DevTools.Node = !m.selections.DevTools.Node
	}
}

func (m model) startInputFlow() (tea.Model, tea.Cmd) {
	m.planErr = ""
	if !m.selections.Any() {
		m.planErr = "select at least one provisioning item"
		return m, nil
	}

	flow := m.inputFlow()
	if len(flow) == 0 {
		m.screen = screenConfirm
		return m, nil
	}
	m.inputPos = 0
	m.screen = flow[0]
	return m, m.focusCurrentInput()
}

func (m model) goNext() (tea.Model, tea.Cmd) {
	flow := m.inputFlow()
	if m.inputPos < len(flow)-1 {
		m.inputPos++
		m.screen = flow[m.inputPos]
		return m, m.focusCurrentInput()
	}
	m.screen = screenConfirm
	return m, nil
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	if m.inputPos > 0 {
		m.inputPos--
		m.screen = m.inputFlow()[m.inputPos]
		return m, m.focusCurrentInput()
	}
	m.screen = screenMainMenu
	return m, nil
}

func (m *model) focusCurrentInput() tea.Cmd {
	m.usernameInput.Blur()
	m.timezoneInput.Blur()
	m.sshKeyInput.Blur()

	switch m.screen {
	case screenInputTimezone:
		return m.timezoneInput.Focus()
	case screenInputUser:
		return m.usernameInput.Focus()
	case screenInputKey:
		return m.sshKeyInput.Focus()
	default:
		return nil
	}
}

func (m *model) resetToPlan() {
	m.screen = screenMainMenu
	m.inputPos = 0
	m.inputErr = ""
	m.planErr = ""
	m.runSteps = nil
	m.runningIndex = -1
	m.output.SetContent("")
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
	if m.selections.AddUser {
		steps = append(steps, runStep{
			id:   runAddUser,
			name: "Add User",
			desc: "Create sudo user, install SSH key, update AllowUsers",
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
	return steps
}

func (m *model) refreshOutput() {
	var b strings.Builder
	for _, step := range m.runSteps {
		if step.output == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "== %s ==\n%s", step.name, strings.TrimSpace(step.output))
	}
	m.output.SetContent(b.String())
	m.output.GotoBottom()
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
