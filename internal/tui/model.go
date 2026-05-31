package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type screen int

const (
	screenWelcome screen = iota
	screenStepSelect
	screenInputUser
	screenInputKey
	screenInputTimezone
	screenConfirm
	screenRunning
	screenDone
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

type model struct {
	screen    screen
	width     int
	height    int
	cursor    int
	steps     []step
	stepFlags []bool

	username  string
	sshKey    string
	timezone  string
	dryRun    bool
	quitting  bool
}

func InitialModel(dryRun bool) model {
	return model{
		screen: screenWelcome,
		steps: []step{
			{name: "Root bootstrap (locale, base packages, SSH, Docker)"},
			{name: "Add user with sudo access and SSH key"},
			{name: "Install CLI tools (ripgrep, fd, bat, yq, glow)"},
			{name: "Install development tools (Go, Node.js toolchain)"},
		},
		stepFlags: []bool{true, false, false, false},
		timezone:  "Australia/Sydney",
		dryRun:    dryRun,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() tea.View {
	switch m.screen {
	case screenWelcome:
		return m.welcomeView()
	case screenStepSelect:
		return m.stepSelectView()
	case screenInputUser:
		return m.inputUserView()
	case screenInputKey:
		return m.inputKeyView()
	case screenInputTimezone:
		return m.inputTimezoneView()
	case screenConfirm:
		return m.confirmView()
	case screenRunning:
		return m.runningView()
	case screenDone:
		return m.doneView()
	}
	return tea.NewView("Unknown screen\n")
}

func (m model) selectedSteps() []int {
	var sel []int
	for i, f := range m.stepFlags {
		if f {
			sel = append(sel, i)
		}
	}
	return sel
}

func (m model) needsUserInput() bool {
	return m.stepFlags[1] || m.stepFlags[3]
}

func (m model) needsKeyInput() bool {
	return m.stepFlags[1]
}

func (m model) needsTimezoneInput() bool {
	return m.stepFlags[0]
}

func (m *model) resetSteps() {
	for i := range m.steps {
		m.steps[i].status = stepPending
		m.steps[i].output = ""
	}
}

func statusIcon(s stepStatus) string {
	switch s {
	case stepPending:
		return "  "
	case stepRunning:
		return "⏳"
	case stepOK:
		return "✓ "
	case stepFail:
		return "✗ "
	}
	return "  "
}

func centerText(width int, text string) string {
	padding := (width - len(text)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}
