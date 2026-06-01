package tui

import (
	tea "charm.land/bubbletea/v2"
)

type screen int

const (
	screenMainMenu screen = iota
	screenInputTimezone
	screenInputUser
	screenInputKey
	screenConfirm
	screenRunning
	screenDone
)

type action int

const (
	actionFullSetup action = iota
	actionBootstrap
	actionAddUser
	actionInstallTools
	actionInstallDevTools
)

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepOK
	stepFail
)

type step struct {
	name   string
	status stepStatus
	output string
}

type stepStatusMsg struct {
	index  int
	status stepStatus
	output string
}

type spinnerTickMsg struct{}

type menuItem struct {
	label  string
	action action
	desc   string
}

var actionFlow = map[action][]screen{
	actionBootstrap:       {screenInputTimezone, screenConfirm},
	actionAddUser:         {screenInputUser, screenInputKey, screenConfirm},
	actionInstallTools:    {screenConfirm},
	actionInstallDevTools: {screenInputUser, screenConfirm},
}

var fullSetupChain = []action{
	actionBootstrap,
	actionAddUser,
	actionInstallTools,
	actionInstallDevTools,
}

var stepNames = map[action]string{
	actionBootstrap:       "System Bootstrap",
	actionAddUser:         "Add User",
	actionInstallTools:    "Install CLI Tools",
	actionInstallDevTools: "Install Dev Tools",
}

type model struct {
	screen   screen
	action   action
	chainIdx int
	flowPos  int

	menuCursor int
	menuItems  []menuItem

	username    string
	sshKey      string
	timezone    string
	usernameErr string
	sshKeyErr   string
	timezoneErr string

	timezoneMatches []string
	timezoneCursor  int

	steps        []step
	spinnerFrame int

	width, height int
	dryRun        bool
	quitting      bool
}

func InitialModel(dryRun bool) model {
	return model{
		screen:   screenMainMenu,
		chainIdx: -1,
		timezone: "UTC",
		dryRun:   dryRun,
		menuItems: []menuItem{
			{label: "Full Setup", action: actionFullSetup, desc: "Run all provisioning steps in sequence"},
			{label: "System Bootstrap", action: actionBootstrap, desc: "Configure locale, SSH, Docker, unattended upgrades"},
			{label: "Add User", action: actionAddUser, desc: "Create user with passwordless sudo and SSH key"},
			{label: "Install CLI Tools", action: actionInstallTools, desc: "Install ripgrep, fd, bat, yq, glow, gh"},
			{label: "Install Dev Tools", action: actionInstallDevTools, desc: "Install Go and Node.js toolchain"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() tea.View {
	var v tea.View
	switch m.screen {
	case screenMainMenu:
		v = tea.NewView(m.mainMenuView())
	case screenInputTimezone:
		v = tea.NewView(m.inputTimezoneView())
	case screenInputUser:
		v = tea.NewView(m.inputUserView())
	case screenInputKey:
		v = tea.NewView(m.inputKeyView())
	case screenConfirm:
		v = tea.NewView(m.confirmView())
	case screenRunning:
		v = tea.NewView(m.runningView())
	case screenDone:
		v = tea.NewView(m.doneView())
	default:
		v = tea.NewView("Unknown screen\n")
	}
	if m.screen == screenRunning && m.isChain() {
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarDefault, m.chainProgress())
	}
	return v
}

func (m model) effectiveAction() action {
	if m.isChain() && m.chainIdx >= 0 && m.chainIdx < len(fullSetupChain) {
		return fullSetupChain[m.chainIdx]
	}
	return m.action
}

func (m model) isChain() bool {
	return m.action == actionFullSetup
}

func (m model) currentFlow() []screen {
	return actionFlow[m.effectiveAction()]
}

func (m model) actionLabel() string {
	act := m.effectiveAction()
	if name, ok := stepNames[act]; ok {
		return name
	}
	return "Unknown"
}

func (m *model) goNext() {
	flow := m.currentFlow()
	if m.flowPos < len(flow)-1 {
		m.flowPos++
		m.screen = flow[m.flowPos]
	} else {
		m.screen = screenConfirm
	}
}

func (m *model) goBack() {
	if m.flowPos > 0 {
		m.flowPos--
		m.screen = m.currentFlow()[m.flowPos]
	} else {
		m.resetToMenu()
	}
}

func (m *model) resetToMenu() {
	m.screen = screenMainMenu
	m.action = 0
	m.chainIdx = -1
	m.flowPos = 0
	m.menuCursor = 0
	m.username = ""
	m.sshKey = ""
	m.timezone = "UTC"
	m.usernameErr = ""
	m.sshKeyErr = ""
	m.steps = nil
	m.spinnerFrame = 0
}

func (m *model) buildSteps() {
	if m.isChain() {
		m.steps = []step{
			{name: "System Bootstrap (locale, packages, SSH, Docker)"},
			{name: "Add user with sudo access and SSH key"},
			{name: "Install CLI tools (ripgrep, fd, bat, yq, glow, gh)"},
			{name: "Install development tools (Go, Node.js toolchain)"},
		}
	} else {
		switch m.effectiveAction() {
		case actionBootstrap:
			m.steps = []step{{name: "System Bootstrap (locale, packages, SSH, Docker)"}}
		case actionAddUser:
			m.steps = []step{{name: "Add user with sudo access and SSH key"}}
		case actionInstallTools:
			m.steps = []step{{name: "Install CLI tools (ripgrep, fd, bat, yq, glow, gh)"}}
		case actionInstallDevTools:
			m.steps = []step{{name: "Install development tools (Go, Node.js toolchain)"}}
		}
	}
}

func (m model) runningStepIndex() int {
	if m.isChain() {
		return m.chainIdx
	}
	return 0
}

func (m model) chainProgress() int {
	if !m.isChain() || len(m.steps) == 0 {
		return 0
	}
	completed := 0
	for _, s := range m.steps {
		if s.status == stepOK {
			completed++
		}
	}
	return (completed * 100) / len(m.steps)
}

func statusIcon(s stepStatus) string {
	switch s {
	case stepPending:
		return "  "
	case stepRunning:
		return "[*]"
	case stepOK:
		return "[✓]"
	case stepFail:
		return "[✗]"
	}
	return "  "
}
