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
	Bootstrap         bool
	AddUser           bool
	FirewallBaseline  bool
	FirewallHTTP      bool
	FirewallHTTPS     bool
	FirewallMosh      bool
	Fail2Ban          bool
	DockerLogRotation bool
	Diagnostics       bool
	UpdatesCheck      bool
	Tools             tools.InstallOptions
	DevTools          devtools.InstallOptions
}

func defaultSelections() selectionState {
	return selectionState{
		Bootstrap: true,
		AddUser:   true,
		Tools:     tools.AllInstallOptions(),
		DevTools:  devtools.DefaultInstallOptions(),
	}
}

func (s selectionState) Any() bool {
	return s.Bootstrap || s.AddUser || s.FirewallBaseline || s.FirewallHTTP ||
		s.FirewallHTTPS || s.FirewallMosh || s.Fail2Ban ||
		s.DockerLogRotation || s.Diagnostics || s.UpdatesCheck ||
		s.Tools.Any() || s.DevTools.Any()
}

func (s selectionState) NeedsTimezone() bool {
	return s.Bootstrap
}

func (s selectionState) NeedsUsername() bool {
	return s.AddUser || s.DevTools.Node || s.DevTools.Rust || s.DevTools.Pnpm
}

func (s selectionState) NeedsSSHKey() bool {
	return s.AddUser
}

type planItemID string

const (
	itemBootstrap planItemID = "bootstrap"
	itemAddUser   planItemID = "add-user"
	itemManageAll planItemID = "manage-all"
	itemFirewall  planItemID = "firewall"
	itemHTTP      planItemID = "firewall-http"
	itemHTTPS     planItemID = "firewall-https"
	itemMosh      planItemID = "firewall-mosh"
	itemFail2Ban  planItemID = "fail2ban"
	itemDockerLog planItemID = "docker-log"
	itemDoctor    planItemID = "doctor"
	itemUpdates   planItemID = "updates"
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
	itemRust      planItemID = "rust"
	itemGoLint    planItemID = "go-lint"
	itemGoRel     planItemID = "goreleaser"
	itemGoVuln    planItemID = "govulncheck"
	itemPnpm      planItemID = "pnpm"
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
	runFirewall  runStepID = "firewall"
	runHTTP      runStepID = "firewall-http"
	runHTTPS     runStepID = "firewall-https"
	runMosh      runStepID = "firewall-mosh"
	runFail2Ban  runStepID = "fail2ban"
	runDockerLog runStepID = "docker-log"
	runDoctor    runStepID = "doctor"
	runUpdates   runStepID = "updates"
	runToolDeps  runStepID = "tool-deps"
	runTool      runStepID = "tool"
	runGo        runStepID = "go"
	runNode      runStepID = "node"
	runRust      runStepID = "rust"
	runGoLint    runStepID = "go-lint"
	runGoRel     runStepID = "goreleaser"
	runGoVuln    runStepID = "govulncheck"
	runPnpm      runStepID = "pnpm"
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
	Toggle:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Continue: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Retry:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "retry")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Scroll:   key.NewBinding(key.WithKeys("pgup/pgdn"), key.WithHelp("pgup/pgdn", "scroll")),
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
	confirm            viewport.Model
	steps              viewport.Model
	output             viewport.Model

	runSteps     []runStep
	runningIndex int

	width, height int
	dryRun        bool
	demo          bool
	quitting      bool
}

func InitialModel(dryRun bool) model {
	return InitialModelWithMode(dryRun, false)
}

func InitialModelWithMode(dryRun, demo bool) model {
	m := model{
		screen:       screenMainMenu,
		selections:   defaultSelections(),
		dryRun:       dryRun,
		demo:         demo,
		help:         help.New(),
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(accentStyle)),
		progress:     progress.New(progress.WithWidth(36), progress.WithColors(lipgloss.Color(colorAccent))),
		confirm:      viewport.New(),
		steps:        viewport.New(),
		output:       viewport.New(),
		runningIndex: -1,
	}
	m.confirm.SoftWrap = true
	m.confirm.FillHeight = true
	m.steps.SoftWrap = true
	m.steps.FillHeight = true
	m.output.SoftWrap = false
	m.output.FillHeight = true
	m.initInputs()
	m.planList = m.newPlanList()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() tea.View {
	var content string
	switch m.screen {
	case screenMainMenu:
		content = m.mainMenuView()
	case screenInputTimezone:
		content = m.inputTimezoneView()
	case screenInputUser:
		content = m.inputUserView()
	case screenInputKey:
		content = m.inputKeyView()
	case screenConfirm:
		content = m.confirmView()
	case screenRunning:
		content = m.runningView()
	case screenDone:
		content = m.doneView()
	default:
		content = "Unknown screen\n"
	}
	if m.width > 0 && m.height > 0 {
		content = lipgloss.NewStyle().Width(m.width).Height(m.height).Render(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
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
	l := list.New(m.planItems(), planDelegate{}, 80, 18)
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
	manageAny := m.selections.FirewallBaseline || m.selections.FirewallHTTP || m.selections.FirewallHTTPS ||
		m.selections.FirewallMosh || m.selections.Fail2Ban || m.selections.DockerLogRotation ||
		m.selections.Diagnostics || m.selections.UpdatesCheck
	manageAll := m.selections.FirewallBaseline && m.selections.FirewallHTTP && m.selections.FirewallHTTPS &&
		m.selections.FirewallMosh && m.selections.Fail2Ban && m.selections.DockerLogRotation &&
		m.selections.Diagnostics && m.selections.UpdatesCheck
	devAll := m.selections.DevTools.Go && m.selections.DevTools.Node && m.selections.DevTools.Rust &&
		m.selections.DevTools.GoLint && m.selections.DevTools.GoReleaser &&
		m.selections.DevTools.GoVulnCheck && m.selections.DevTools.Pnpm

	return []list.Item{
		planItem{itemBootstrap, checkbox(m.selections.Bootstrap, m.selections.Bootstrap) + " System Bootstrap", "Locale, apt, base packages, SSH hardening, unattended upgrades, Docker"},
		planItem{itemAddUser, checkbox(m.selections.AddUser, m.selections.AddUser) + " Add User", "Passwordless sudo, SSH key, linger, managed AllowUsers"},
		planItem{itemManageAll, checkbox(manageAll, manageAny) + " Instance Management", "UFW, common ports, fail2ban, Docker logs, diagnostics, updates"},
		planItem{itemFirewall, "  " + checkbox(m.selections.FirewallBaseline, m.selections.FirewallBaseline) + " UFW Firewall Baseline", "Default deny incoming, allow outgoing, preserve SSH access"},
		planItem{itemHTTP, "  " + checkbox(m.selections.FirewallHTTP, m.selections.FirewallHTTP) + " Allow HTTP", "Open tcp/80 through UFW"},
		planItem{itemHTTPS, "  " + checkbox(m.selections.FirewallHTTPS, m.selections.FirewallHTTPS) + " Allow HTTPS", "Open tcp/443 through UFW"},
		planItem{itemMosh, "  " + checkbox(m.selections.FirewallMosh, m.selections.FirewallMosh) + " Allow Mosh", "Open udp/60000:61000 through UFW"},
		planItem{itemFail2Ban, "  " + checkbox(m.selections.Fail2Ban, m.selections.Fail2Ban) + " fail2ban SSH Jail", "Install fail2ban and manage the setup SSH jail"},
		planItem{itemDockerLog, "  " + checkbox(m.selections.DockerLogRotation, m.selections.DockerLogRotation) + " Docker Log Rotation", "Set json-file max-size=10m and max-file=3"},
		planItem{itemDoctor, "  " + checkbox(m.selections.Diagnostics, m.selections.Diagnostics) + " Doctor Diagnostics", "Read-only checks for LXC, services, SSH, UFW, Docker"},
		planItem{itemUpdates, "  " + checkbox(m.selections.UpdatesCheck, m.selections.UpdatesCheck) + " Update Check", "Refresh apt metadata and list available upgrades"},
		planItem{itemCLIAll, checkbox(cliAll, m.selections.Tools.Any()) + " CLI Tools", "ripgrep, fd, bat, yq, glow, gh"},
		planItem{itemRipgrep, "  " + checkbox(m.selections.Tools.Ripgrep, m.selections.Tools.Ripgrep) + " ripgrep", "Verified GitHub release .deb with apt fallback"},
		planItem{itemFd, "  " + checkbox(m.selections.Tools.Fd, m.selections.Tools.Fd) + " fd", "Verified release .deb with Debian fd-find alias handling"},
		planItem{itemBat, "  " + checkbox(m.selections.Tools.Bat, m.selections.Tools.Bat) + " bat", "Verified release .deb with Debian batcat alias handling"},
		planItem{itemYq, "  " + checkbox(m.selections.Tools.Yq, m.selections.Tools.Yq) + " yq", "Verified linux/amd64 binary from mikefarah/yq"},
		planItem{itemGlow, "  " + checkbox(m.selections.Tools.Glow, m.selections.Tools.Glow) + " glow", "charm.sh apt repository with fingerprint verification"},
		planItem{itemGh, "  " + checkbox(m.selections.Tools.Gh, m.selections.Tools.Gh) + " gh", "GitHub CLI apt repository with fingerprint verification"},
		planItem{itemDevAll, checkbox(devAll, m.selections.DevTools.Any()) + " Development Tools", "Go, Node.js, Rust, linters, release tooling, pnpm"},
		planItem{itemGo, "  " + checkbox(m.selections.DevTools.Go, m.selections.DevTools.Go) + " Go", "System-wide go.dev install with SHA256 verification"},
		planItem{itemNode, "  " + checkbox(m.selections.DevTools.Node, m.selections.DevTools.Node) + " Node.js", "Per-user fnm, Node.js, corepack, TypeScript, tsx"},
		planItem{itemRust, "  " + checkbox(m.selections.DevTools.Rust, m.selections.DevTools.Rust) + " Rust", "Per-user stable rustup toolchain and components"},
		planItem{itemGoLint, "  " + checkbox(m.selections.DevTools.GoLint, m.selections.DevTools.GoLint) + " golangci-lint", "Verified release archive to /usr/local/bin"},
		planItem{itemGoRel, "  " + checkbox(m.selections.DevTools.GoReleaser, m.selections.DevTools.GoReleaser) + " GoReleaser", "Verified release archive to /usr/local/bin"},
		planItem{itemGoVuln, "  " + checkbox(m.selections.DevTools.GoVulnCheck, m.selections.DevTools.GoVulnCheck) + " govulncheck", "Official Go vulnerability scanner via go install"},
		planItem{itemPnpm, "  " + checkbox(m.selections.DevTools.Pnpm, m.selections.DevTools.Pnpm) + " pnpm", "Per-user pnpm through Corepack"},
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

	pageWidth := width - 6
	if pageWidth < 40 {
		pageWidth = 40
	}
	if pageWidth > 96 {
		pageWidth = 96
	}

	listHeight := height - 9
	if listHeight < 12 {
		listHeight = 12
	}
	m.planList.SetSize(pageWidth, listHeight)
	m.help.SetWidth(pageWidth)
	m.usernameInput.SetWidth(pageWidth - 4)
	m.timezoneInput.SetWidth(pageWidth - 4)
	m.sshKeyInput.SetWidth(pageWidth - 4)
	m.confirm.SetWidth(pageWidth - 4)

	confirmHeight := height - 9
	if confirmHeight < 6 {
		confirmHeight = 6
	}
	m.confirm.SetHeight(confirmHeight)

	stepsWidth, stepsHeight := m.stepsSize()
	m.steps.SetWidth(stepsWidth)
	m.steps.SetHeight(stepsHeight)
	if len(m.runSteps) > 0 {
		m.refreshSteps()
	}

	outputWidth, outputHeight := m.outputSize()
	m.output.SetWidth(outputWidth)
	m.output.SetHeight(outputHeight)

	progressWidth := m.runContentWidth() - 12
	if progressWidth < 18 {
		progressWidth = 18
	}
	if progressWidth > 72 {
		progressWidth = 72
	}
	m.progress.SetWidth(progressWidth)
}

func (m model) runContentWidth() int {
	width := m.width - 4
	if width < 40 {
		return 40
	}
	return width
}

func (m model) usesRunColumns() bool {
	return m.runContentWidth() >= 92
}

func (m model) stepPanelWidth() int {
	width := 36
	if len(m.runSteps) > 9 {
		width = 40
	}
	if max := m.runContentWidth() / 2; width > max {
		width = max
	}
	if width < 28 {
		return 28
	}
	return width
}

func (m model) outputSize() (int, int) {
	contentWidth := m.runContentWidth()
	if m.usesRunColumns() {
		outputWidth := contentWidth - m.stepPanelWidth() - 8
		if outputWidth < 32 {
			outputWidth = 32
		}
		outputHeight := m.height - 22
		if outputHeight < 6 {
			outputHeight = 6
		}
		return outputWidth, outputHeight
	}

	outputWidth := contentWidth - 6
	if outputWidth < 32 {
		outputWidth = 32
	}
	_, stepsHeight := m.stepsSize()
	outputHeight := m.height - stepsHeight - 26
	if outputHeight < 6 {
		outputHeight = 6
	}
	return outputWidth, outputHeight
}

func (m model) stepsSize() (int, int) {
	if m.usesRunColumns() {
		stepsWidth := m.stepPanelWidth() - 2
		if stepsWidth < 24 {
			stepsWidth = 24
		}
		_, outputHeight := m.outputSize()
		return stepsWidth, outputHeight
	}

	stepsWidth := m.runContentWidth() - 6
	if stepsWidth < 32 {
		stepsWidth = 32
	}
	stepsHeight := (m.height - 11) / 2
	if stepsHeight < 4 {
		stepsHeight = 4
	}
	if max := len(m.runSteps) + 1; max > 0 && stepsHeight > max {
		stepsHeight = max
	}
	return stepsWidth, stepsHeight
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
	if m.selections.FirewallBaseline {
		count++
	}
	if m.selections.FirewallHTTP {
		count++
	}
	if m.selections.FirewallHTTPS {
		count++
	}
	if m.selections.FirewallMosh {
		count++
	}
	if m.selections.Fail2Ban {
		count++
	}
	if m.selections.DockerLogRotation {
		count++
	}
	if m.selections.Diagnostics {
		count++
	}
	if m.selections.UpdatesCheck {
		count++
	}
	for _, selected := range []bool{
		m.selections.DevTools.Go,
		m.selections.DevTools.Node,
		m.selections.DevTools.Rust,
		m.selections.DevTools.GoLint,
		m.selections.DevTools.GoReleaser,
		m.selections.DevTools.GoVulnCheck,
		m.selections.DevTools.Pnpm,
	} {
		if selected {
			count++
		}
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
