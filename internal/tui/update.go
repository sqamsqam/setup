package tui

import tea "charm.land/bubbletea/v2"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case stepStatusMsg:
		return m.handleStepMsg(msg)

	case tea.KeyPressMsg:
		if m.quitting {
			return m, tea.Quit
		}

		switch m.screen {
		case screenWelcome:
			return m.updateWelcome(msg)
		case screenStepSelect:
			return m.updateStepSelect(msg)
		case screenInputUser:
			return m.updateInputUser(msg)
		case screenInputKey:
			return m.updateInputKey(msg)
		case screenInputTimezone:
			return m.updateInputTimezone(msg)
		case screenConfirm:
			return m.updateConfirm(msg)
		case screenRunning:
			return m, nil
		case screenDone:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) handleStepMsg(msg stepStatusMsg) (tea.Model, tea.Cmd) {
	if msg.quitting {
		for i := range m.steps {
			if m.stepFlags[i] && m.steps[i].status == stepPending {
				m.steps[i].status = stepOK
			}
		}
		m.screen = screenDone
		return m, nil
	}

	m.steps[msg.index].status = msg.status
	m.steps[msg.index].output = msg.output
	m.screen = screenDone
	return m, nil
}

func (m model) updateWelcome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	default:
		m.screen = screenStepSelect
	}
	return m, nil
}

func (m model) updateStepSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.steps)-1 {
			m.cursor++
		}
	case " ", "enter":
		if m.cursor < len(m.stepFlags) {
			m.stepFlags[m.cursor] = !m.stepFlags[m.cursor]
		}
	case "c":
		if m.hasSelections() {
			if m.needsUserInput() {
				m.screen = screenInputUser
			} else if m.needsTimezoneInput() {
				m.screen = screenInputTimezone
			} else {
				m.screen = screenConfirm
			}
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
		if len(m.username) > 0 {
			if m.needsKeyInput() {
				m.screen = screenInputKey
			} else if m.needsTimezoneInput() {
				m.screen = screenInputTimezone
			} else {
				m.screen = screenConfirm
			}
		}
	case "backspace":
		if len(m.username) > 0 {
			m.username = m.username[:len(m.username)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.username += s
		}
	}
	return m, nil
}

func (m model) updateInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		if len(m.sshKey) > 0 {
			if m.needsTimezoneInput() {
				m.screen = screenInputTimezone
			} else {
				m.screen = screenConfirm
			}
		}
	case "backspace":
		if len(m.sshKey) > 0 {
			m.sshKey = m.sshKey[:len(m.sshKey)-1]
		}
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.sshKey += s
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
		m.screen = screenConfirm
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

func (m model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		m.screen = screenRunning
		m.resetSteps()
		return m, runProvisioning(m)
	case "esc":
		m.screen = screenStepSelect
	}
	return m, nil
}

func (m model) hasSelections() bool {
	for _, f := range m.stepFlags {
		if f {
			return true
		}
	}
	return false
}
