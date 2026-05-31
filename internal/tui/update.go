package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sqamsqam/setup/internal/user"
)

func tickSpinner() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case stepStatusMsg:
		return m.handleStepMsg(msg)

	case spinnerTickMsg:
		m.spinnerFrame++
		if m.screen == screenRunning {
			return m, tickSpinner()
		}
		return m, nil

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
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		case screenDone:
			return m.updateDone(msg)
		}
	}

	return m, nil
}

func (m model) handleStepMsg(msg stepStatusMsg) (tea.Model, tea.Cmd) {
	m.steps[msg.index].status = msg.status
	m.steps[msg.index].output = msg.output
	m.screen = screenDone
	return m, nil
}

func (m model) updateMainMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}
	case "enter":
		if m.menuCursor >= 0 && m.menuCursor < len(m.menuItems) {
			item := m.menuItems[m.menuCursor]
			m.action = item.action
			if m.action == actionFullSetup {
				m.chainIdx = 0
			} else {
				m.chainIdx = -1
			}
			m.buildSteps()
			m.flowPos = 0
			m.screen = m.currentFlow()[0]
		}
	}
	return m, nil
}

func (m model) updateInputTimezone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		m.goNext()
	case "esc":
		m.goBack()
	case "backspace":
		if len(m.timezone) > 0 {
			m.timezone = m.timezone[:len(m.timezone)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.timezone += s
		}
	}
	return m, nil
}

func (m model) updateInputUser(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		m.usernameErr = ""
		if len(m.username) > 0 {
			if err := user.ValidateUsername(m.username); err != nil {
				m.usernameErr = err.Error()
				return m, nil
			}
			m.goNext()
		}
	case "esc":
		m.usernameErr = ""
		m.goBack()
	case "backspace":
		if len(m.username) > 0 {
			m.username = m.username[:len(m.username)-1]
		}
		m.usernameErr = ""
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.username += s
		}
		m.usernameErr = ""
	}
	return m, nil
}

func (m model) updateInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		m.sshKeyErr = ""
		if len(m.sshKey) > 0 {
			if err := user.ValidateSSHKey(m.sshKey); err != nil {
				m.sshKeyErr = err.Error()
				return m, nil
			}
			m.goNext()
		}
	case "esc":
		m.sshKeyErr = ""
		m.goBack()
	case "backspace":
		if len(m.sshKey) > 0 {
			m.sshKey = m.sshKey[:len(m.sshKey)-1]
		}
		m.sshKeyErr = ""
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.sshKey += s
		}
		m.sshKeyErr = ""
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		m.screen = screenRunning
		m.steps[m.runningStepIndex()].status = stepRunning
		m.steps[m.runningStepIndex()].output = ""
		return m, tea.Batch(runProvisioningStep(m), tickSpinner())
	case "esc":
		m.goBack()
	}
	return m, nil
}

func (m model) updateDone(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "q" || key == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	if key == "enter" {
		if m.isChain() && m.chainIdx < len(fullSetupChain)-1 {
			m.chainIdx++
			m.spinnerFrame = 0
			m.flowPos = 0
			m.screen = m.currentFlow()[0]
			return m, nil
		}
		m.resetToMenu()
		return m, nil
	}

	if key == "esc" {
		m.resetToMenu()
		return m, nil
	}

	return m, nil
}
