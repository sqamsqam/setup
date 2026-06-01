package tui

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sqamsqam/setup/internal/devtools"
	"github.com/sqamsqam/setup/internal/tools"
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

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepOK
	stepFail
)

type stepStatusMsg struct {
	index  int
	status stepStatus
	output string
}

type selectionState struct {
	Bootstrap bool
	AddUser   bool
	Tools     tools.InstallOptions
	DevTools  devtools.InstallOptions
}

func defaultSelections() selectionState {
	return selectionState{
		Bootstrap: true,
		AddUser:   true,
		Tools:     tools.AllInstallOptions(),
		DevTools:  devtools.AllInstallOptions(),
	}
}

func (s selectionState) Any() bool {
	return s.Bootstrap || s.AddUser || s.Tools.Any() || s.DevTools.Any()
}

func (s selectionState) NeedsTimezone() bool {
	return s.Bootstrap
}

func (s selectionState) NeedsUsername() bool {
	return s.AddUser || s.DevTools.Node
}

func (s selectionState) NeedsSSHKey() bool {
	return s.AddUser
}

type planItemID string

const (
	itemBootstrap planItemID = "bootstrap"
	itemAddUser   planItemID = "add-user"
	itemCLIAll    planItemID = "cli-all"
	itemRipgrep   planItemID = "ripgrep"
	itemFd        planItemID = "fd"
	itemBat       planItemID = "bat"
	itemYq        planItemID = "yq"
	itemGlow      planItemID = "glow"
	itemGh        planItemID = "gh"
	itemDevAll    planItemID = "dev-all"
	itemGo        planItemID = "go"
	itemNode      planItemID = "node"
)

type planItem struct {
	id    planItemID
	title string
	desc  string
}

func (p planItem) FilterValue() string {
	return p.title + " " + p.desc
}

func (p planItem) Title() string {
	return p.title
}

func (p planItem) Description() string {
	return p.desc
}

type runStepID string

const (
	runBootstrap runStepID = "bootstrap"
	runAddUser   runStepID = "add-user"
	runToolDeps  runStepID = "tool-deps"
	runTool      runStepID = "tool"
	runGo        runStepID = "go"
	runNode      runStepID = "node"
)

type runStep struct {
	id     runStepID
	tool   tools.Tool
	name   string
	desc   string
	status stepStatus
	output string
}

type tuiKeys struct {
	Toggle   key.Binding
	Select   key.Binding
	Continue key.Binding
	Back     key.Binding
	Retry    key.Binding
	Quit     key.Binding
	Scroll   key.Binding
}

var keys = tuiKeys{
	Toggle:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Continue: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Retry:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "retry")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Scroll:   key.NewBinding(key.WithKeys("pgup/pgdn"), key.WithHelp("pgup/pgdn", "scroll output")),
}

type helpKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return k.short
}

func (k helpKeyMap) FullHelp() [][]key.Binding {
	if len(k.full) > 0 {
		return k.full
	}
	return [][]key.Binding{k.short}
}

type model struct {
	screen   screen
	inputPos int

	selections selectionState
	planErr    string
	inputErr   string

	planList           list.Model
	help               help.Model
	usernameInput      textinput.Model
	timezoneInput      textinput.Model
	timezones          []string
	timezoneMatches    []string
	timezoneMatchIndex int
	sshKeyInput        textarea.Model
	spinner            spinner.Model
	progress           progress.Model
	output             viewport.Model

	runSteps     []runStep
	runningIndex int

	width, height int
	dryRun        bool
	quitting      bool
}

func InitialModel(dryRun bool) model {
	m := model{
		screen:       screenMainMenu,
		selections:   defaultSelections(),
		dryRun:       dryRun,
		help:         help.New(),
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(accentStyle)),
		progress:     progress.New(progress.WithWidth(36), progress.WithColors(lipgloss.Color("#2E7D6B"))),
		output:       viewport.New(),
		runningIndex: -1,
	}
	m.output.SoftWrap = true
	m.initInputs()
	m.planList = m.newPlanList()
	return m
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
	return v
}

func (m *model) initInputs() {
	m.usernameInput = textinput.New()
	m.usernameInput.Prompt = ""
	m.usernameInput.Placeholder = "dev"
	m.usernameInput.CharLimit = 32
	m.usernameInput.SetWidth(48)

	m.timezoneInput = textinput.New()
	m.timezoneInput.Prompt = ""
	m.timezoneInput.Placeholder = "UTC"
	m.timezoneInput.SetValue("UTC")
	m.timezoneInput.SetWidth(48)
	m.timezones = availableTimezones()
	m.refreshTimezoneMatches()

	m.sshKeyInput = textarea.New()
	m.sshKeyInput.Prompt = ""
	m.sshKeyInput.Placeholder = "paste an ssh-ed25519, ssh-rsa, ecdsa-*, or sk-* public key"
	m.sshKeyInput.ShowLineNumbers = false
	m.sshKeyInput.SetWidth(72)
	m.sshKeyInput.SetHeight(4)
	m.sshKeyInput.MaxHeight = 4
}

func (m model) newPlanList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderForeground(lipgloss.Color("#2E7D6B")).
		Foreground(lipgloss.Color("#F4D35E"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#7DCFB6"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("#E8E8E8"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("#8A8A8A"))

	l := list.New(m.planItems(), delegate, 80, 18)
	l.Title = "Provisioning plan"
	l.SetStatusBarItemName("step", "steps")
	l.DisableQuitKeybindings()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Toggle, keys.Continue, keys.Quit}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Toggle, keys.Continue, keys.Quit}
	}
	return l
}

func (m model) planItems() []list.Item {
	cliAll := m.selections.Tools.Ripgrep && m.selections.Tools.Fd && m.selections.Tools.Bat &&
		m.selections.Tools.Yq && m.selections.Tools.Glow && m.selections.Tools.Gh
	devAll := m.selections.DevTools.Go && m.selections.DevTools.Node

	return []list.Item{
		planItem{itemBootstrap, checkbox(m.selections.Bootstrap, m.selections.Bootstrap) + " System Bootstrap", "Locale, apt upgrade, base packages, SSH hardening, unattended upgrades, Docker"},
		planItem{itemAddUser, checkbox(m.selections.AddUser, m.selections.AddUser) + " Add User", "Passwordless sudo, SSH public key, linger, and AllowUsers"},
		planItem{itemCLIAll, checkbox(cliAll, m.selections.Tools.Any()) + " CLI Tools", "Toggle all CLI tools below"},
		planItem{itemRipgrep, "  " + checkbox(m.selections.Tools.Ripgrep, m.selections.Tools.Ripgrep) + " ripgrep", "GitHub release .deb, apt fallback only when verification is unavailable"},
		planItem{itemFd, "  " + checkbox(m.selections.Tools.Fd, m.selections.Tools.Fd) + " fd", "GitHub release .deb, with Debian fd-find alias handling"},
		planItem{itemBat, "  " + checkbox(m.selections.Tools.Bat, m.selections.Tools.Bat) + " bat", "GitHub release .deb, with Debian batcat alias handling"},
		planItem{itemYq, "  " + checkbox(m.selections.Tools.Yq, m.selections.Tools.Yq) + " yq", "Verified linux/amd64 binary from mikefarah/yq"},
		planItem{itemGlow, "  " + checkbox(m.selections.Tools.Glow, m.selections.Tools.Glow) + " glow", "charm.sh apt repository with key fingerprint verification"},
		planItem{itemGh, "  " + checkbox(m.selections.Tools.Gh, m.selections.Tools.Gh) + " gh", "GitHub CLI apt repository with key fingerprint verification"},
		planItem{itemDevAll, checkbox(devAll, m.selections.DevTools.Any()) + " Development Tools", "Toggle Go and Node.js tooling below"},
		planItem{itemGo, "  " + checkbox(m.selections.DevTools.Go, m.selections.DevTools.Go) + " Go", "System-wide install from go.dev with SHA256 verification"},
		planItem{itemNode, "  " + checkbox(m.selections.DevTools.Node, m.selections.DevTools.Node) + " Node.js", "Per-user fnm, latest Node, corepack, TypeScript, and tsx"},
	}
}

func checkbox(checked, partial bool) string {
	if checked {
		return "[x]"
	}
	if partial {
		return "[-]"
	}
	return "[ ]"
}

func (m *model) refreshPlanList() tea.Cmd {
	return m.planList.SetItems(m.planItems())
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height

	contentWidth := width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentWidth > 96 {
		contentWidth = 96
	}

	listHeight := height - 9
	if listHeight < 12 {
		listHeight = 12
	}
	m.planList.SetSize(contentWidth, listHeight)
	m.help.SetWidth(contentWidth)
	m.usernameInput.SetWidth(contentWidth - 4)
	m.timezoneInput.SetWidth(contentWidth - 4)
	m.sshKeyInput.SetWidth(contentWidth - 4)
	m.output.SetWidth(contentWidth)

	outputHeight := height - 16
	if outputHeight < 6 {
		outputHeight = 6
	}
	m.output.SetHeight(outputHeight)

	progressWidth := contentWidth - 12
	if progressWidth < 18 {
		progressWidth = 18
	}
	if progressWidth > 52 {
		progressWidth = 52
	}
	m.progress.SetWidth(progressWidth)
}

func (m model) inputFlow() []screen {
	var flow []screen
	if m.selections.NeedsTimezone() {
		flow = append(flow, screenInputTimezone)
	}
	if m.selections.NeedsUsername() {
		flow = append(flow, screenInputUser)
	}
	if m.selections.NeedsSSHKey() {
		flow = append(flow, screenInputKey)
	}
	return flow
}

func (m model) selectedPlanCount() int {
	count := 0
	if m.selections.Bootstrap {
		count++
	}
	if m.selections.AddUser {
		count++
	}
	count += len(m.selections.Tools.SelectedTools())
	if m.selections.DevTools.Go {
		count++
	}
	if m.selections.DevTools.Node {
		count++
	}
	return count
}

func (m model) runProgress() float64 {
	if len(m.runSteps) == 0 {
		return 0
	}
	done := 0
	for _, step := range m.runSteps {
		if step.status == stepOK {
			done++
		}
	}
	return float64(done) / float64(len(m.runSteps))
}

func (m model) completedRunSteps() int {
	done := 0
	for _, step := range m.runSteps {
		if step.status == stepOK {
			done++
		}
	}
	return done
}

func statusIcon(s stepStatus) string {
	switch s {
	case stepPending:
		return "[ ]"
	case stepRunning:
		return "[*]"
	case stepOK:
		return "[✓]"
	case stepFail:
		return "[✗]"
	default:
		return "[ ]"
	}
}

func (m model) currentStepSummary() string {
	if m.runningIndex < 0 || m.runningIndex >= len(m.runSteps) {
		return ""
	}
	return fmt.Sprintf("%d/%d %s", m.runningIndex+1, len(m.runSteps), m.runSteps[m.runningIndex].name)
}
