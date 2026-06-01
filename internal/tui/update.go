package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	case tea.PasteMsg:
		if m.screen == screenInputKey {
			m.sshKey = normalizeSSHKeyInput(m.sshKey + " " + msg.String())
			m.sshKeyErr = ""
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
		m.timezoneErr = ""
		if strings.TrimSpace(m.timezone) == "" {
			m.timezone = "UTC"
			m.goNext()
			return m, nil
		}
		if err := validateTimezone(m.timezone); err != nil {
			m.timezoneErr = err.Error()
			m.refreshTimezoneMatches()
			return m, nil
		}
		m.goNext()
	case "esc":
		m.timezoneErr = ""
		m.goBack()
	case "up", "k":
		if m.timezoneCursor > 0 {
			m.timezoneCursor--
			m.timezone = m.timezoneMatches[m.timezoneCursor]
		}
	case "down", "j":
		if m.timezoneCursor < len(m.timezoneMatches)-1 {
			m.timezoneCursor++
			m.timezone = m.timezoneMatches[m.timezoneCursor]
		}
	case "tab":
		if len(m.timezoneMatches) > 0 {
			m.timezone = m.timezoneMatches[m.timezoneCursor]
		}
	case "backspace":
		if len(m.timezone) > 0 {
			m.timezone = m.timezone[:len(m.timezone)-1]
		}
		m.timezoneErr = ""
		m.refreshTimezoneMatches()
	default:
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.timezone += s
			m.timezoneErr = ""
			m.refreshTimezoneMatches()
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
		if err := user.ValidateUsername(m.username); err != nil {
			m.usernameErr = err.Error()
			return m, nil
		}
		m.goNext()
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
		m.sshKey = normalizeSSHKeyInput(m.sshKey)
		if err := user.ValidateSSHKey(m.sshKey); err != nil {
			m.sshKeyErr = err.Error()
			return m, nil
		}
		m.goNext()
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
		} else if strings.Contains(s, "ssh-") || strings.Contains(s, "ecdsa-") || strings.Contains(s, "sk-") {
			m.sshKey = normalizeSSHKeyInput(m.sshKey + " " + s)
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
		if m.isChain() && m.steps[m.runningStepIndex()].status == stepFail {
			m.screen = screenRunning
			m.steps[m.runningStepIndex()].status = stepRunning
			m.steps[m.runningStepIndex()].output = ""
			return m, tea.Batch(runProvisioningStep(m), tickSpinner())
		}
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

func normalizeSSHKeyInput(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (m *model) refreshTimezoneMatches() {
	m.timezoneMatches = timezoneMatches(m.timezone, 6)
	m.timezoneCursor = 0
}

func timezoneMatches(query string, limit int) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	zones := availableTimezones()
	if query == "" {
		defaults := []string{"UTC", "America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles", "Europe/London"}
		if limit > len(defaults) {
			limit = len(defaults)
		}
		return defaults[:limit]
	}
	var matches []string
	for _, zone := range zones {
		if strings.Contains(strings.ToLower(zone), query) {
			matches = append(matches, zone)
			if len(matches) >= limit {
				break
			}
		}
	}
	return matches
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
		"posix": true, "right": true, "Etc": false,
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
