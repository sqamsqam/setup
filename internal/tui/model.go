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
	screenInputServiceUserGroups
	screenInputServiceName
	screenInputServiceWorkDir
	screenInputServiceCommand
	screenInputServiceEnvFile
	screenInputFirewallRule
	screenInputNetworkRuleNumber
	screenInputFail2BanOptions
	screenInputDockerLogOptions
	screenInputDockerPruneTargets
	screenInputGuardIP
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
	UserCreateLogin   bool
	UserCreateService bool
	UserSSHKey        bool
	UserAllowSSH      bool
	UserSudo          bool
	UserLinger        bool
	UserDockerGroup   bool
	UserServiceGroups bool
	ServiceCreate     bool
	ServiceStatus     bool
	ServiceLogs       bool
	ServiceRestart    bool
	ServiceList       bool
	ServiceDisable    bool
	ServiceRemove     bool
	FirewallBaseline  bool
	FirewallHTTP      bool
	FirewallHTTPS     bool
	FirewallMosh      bool
	FirewallCustom    bool
	NetworkStatus     bool
	NetworkList       bool
	NetworkDelete     bool
	NetworkReset      bool
	Fail2Ban          bool
	Fail2BanStatus    bool
	Fail2BanUnban     bool
	DockerLogRotation bool
	ContainersDisk    bool
	ContainersPrune   bool
	Diagnostics       bool
	UpdatesCheck      bool
	UpdatesUpgrade    bool
	UpdatesRebootNeed bool
	UpdatesUnattended bool
	UpdatesFailed     bool
	UpdatesReboot     bool
	Tools             tools.InstallOptions
	DevTools          devtools.InstallOptions
}

func defaultSelections() selectionState {
	return selectionState{
		Bootstrap:       true,
		UserCreateLogin: true,
		UserSSHKey:      true,
		UserAllowSSH:    true,
		UserSudo:        true,
		UserLinger:      true,
		UserDockerGroup: true,
		Tools:           tools.AllInstallOptions(),
		DevTools:        devtools.DefaultInstallOptions(),
	}
}

func (s selectionState) Any() bool {
	return s.Bootstrap || s.UserManagementAny() || s.InstanceManagementAny() ||
		s.ServiceAny() || s.Tools.Any() || s.DevTools.Any()
}

func (s selectionState) NeedsTimezone() bool {
	return s.Bootstrap
}

func (s selectionState) NeedsUsername() bool {
	return s.UserManagementAny() || s.ServiceAny() || s.DevTools.Node || s.DevTools.Rust || s.DevTools.Pnpm
}

func (s selectionState) NeedsSSHKey() bool {
	return s.UserSSHKey
}

func (s selectionState) NeedsServiceUserGroups() bool {
	return s.UserCreateService
}

func (s selectionState) NeedsServiceName() bool {
	return s.ServiceCreate || s.ServiceStatus || s.ServiceLogs || s.ServiceRestart || s.ServiceDisable || s.ServiceRemove
}

func (s selectionState) NeedsServiceWorkDir() bool {
	return s.ServiceCreate
}

func (s selectionState) NeedsServiceCommand() bool {
	return s.ServiceCreate
}

func (s selectionState) NeedsServiceEnvFile() bool {
	return s.ServiceCreate
}

func (s selectionState) NeedsFirewallRule() bool {
	return s.FirewallCustom
}

func (s selectionState) NeedsNetworkRuleNumber() bool {
	return s.NetworkDelete
}

func (s selectionState) NeedsFail2BanOptions() bool {
	return s.Fail2Ban
}

func (s selectionState) NeedsDockerLogOptions() bool {
	return s.DockerLogRotation
}

func (s selectionState) NeedsDockerPruneTargets() bool {
	return s.ContainersPrune
}

func (s selectionState) NeedsGuardIP() bool {
	return s.Fail2BanUnban
}

func (s selectionState) UserLoginAny() bool {
	return s.UserCreateLogin || s.UserSSHKey || s.UserAllowSSH || s.UserSudo || s.UserLinger || s.UserDockerGroup
}

func (s selectionState) UserLoginAll() bool {
	return s.UserCreateLogin && s.UserSSHKey && s.UserAllowSSH && s.UserSudo && s.UserLinger && s.UserDockerGroup
}

func (s selectionState) UserManagementAny() bool {
	return s.UserLoginAny() || s.UserCreateService
}

func (s selectionState) ServiceAny() bool {
	return s.ServiceCreate || s.ServiceStatus || s.ServiceLogs || s.ServiceRestart ||
		s.ServiceList || s.ServiceDisable || s.ServiceRemove
}

func (s selectionState) ServiceAll() bool {
	return s.ServiceCreate && s.ServiceStatus && s.ServiceLogs && s.ServiceRestart &&
		s.ServiceList && s.ServiceDisable && s.ServiceRemove
}

func (s selectionState) InstanceManagementAny() bool {
	return s.FirewallBaseline || s.FirewallHTTP || s.FirewallHTTPS || s.FirewallMosh ||
		s.FirewallCustom || s.NetworkStatus || s.NetworkList || s.NetworkDelete || s.NetworkReset ||
		s.Fail2Ban || s.Fail2BanStatus || s.Fail2BanUnban || s.DockerLogRotation ||
		s.ContainersDisk || s.ContainersPrune || s.Diagnostics || s.UpdatesCheck ||
		s.UpdatesUpgrade || s.UpdatesRebootNeed || s.UpdatesUnattended || s.UpdatesFailed || s.UpdatesReboot
}

func (s selectionState) InstanceManagementAll() bool {
	return s.FirewallBaseline && s.FirewallHTTP && s.FirewallHTTPS && s.FirewallMosh &&
		s.FirewallCustom && s.NetworkStatus && s.NetworkList && s.NetworkDelete && s.NetworkReset &&
		s.Fail2Ban && s.Fail2BanStatus && s.Fail2BanUnban && s.DockerLogRotation &&
		s.ContainersDisk && s.ContainersPrune && s.Diagnostics && s.UpdatesCheck &&
		s.UpdatesUpgrade && s.UpdatesRebootNeed && s.UpdatesUnattended && s.UpdatesFailed && s.UpdatesReboot
}

type planItemID string

const (
	itemBootstrap       planItemID = "bootstrap"
	itemUserAll         planItemID = "user-all"
	itemAddUser         planItemID = "add-user"
	itemUserCreateLogin planItemID = "user-create-login"
	itemUserSSHKey      planItemID = "user-ssh-key"
	itemUserAllowSSH    planItemID = "user-allow-ssh"
	itemUserSudo        planItemID = "user-sudo"
	itemUserLinger      planItemID = "user-linger"
	itemUserDockerGroup planItemID = "user-docker-group"
	itemServiceUser     planItemID = "service-user"
	itemServiceGroups   planItemID = "service-user-groups"
	itemServiceAll      planItemID = "service-all"
	itemServiceCreate   planItemID = "service-create"
	itemServiceStatus   planItemID = "service-status"
	itemServiceLogs     planItemID = "service-logs"
	itemServiceRestart  planItemID = "service-restart"
	itemServiceList     planItemID = "service-list"
	itemServiceDisable  planItemID = "service-disable"
	itemServiceRemove   planItemID = "service-remove"
	itemManageAll       planItemID = "manage-all"
	itemFirewall        planItemID = "firewall"
	itemHTTP            planItemID = "firewall-http"
	itemHTTPS           planItemID = "firewall-https"
	itemMosh            planItemID = "firewall-mosh"
	itemFirewallCustom  planItemID = "firewall-custom"
	itemNetworkStatus   planItemID = "network-status"
	itemNetworkList     planItemID = "network-list"
	itemNetworkDelete   planItemID = "network-delete"
	itemNetworkReset    planItemID = "network-reset"
	itemFail2Ban        planItemID = "fail2ban"
	itemFail2BanStatus  planItemID = "fail2ban-status"
	itemFail2BanUnban   planItemID = "fail2ban-unban"
	itemDockerLog       planItemID = "docker-log"
	itemContainersDisk  planItemID = "containers-disk"
	itemContainersPrune planItemID = "containers-prune"
	itemDoctor          planItemID = "doctor"
	itemUpdates         planItemID = "updates"
	itemUpdatesUpgrade  planItemID = "updates-upgrade"
	itemUpdatesRebootN  planItemID = "updates-reboot-needed"
	itemUpdatesUnattend planItemID = "updates-unattended"
	itemUpdatesFailed   planItemID = "updates-failed"
	itemUpdatesReboot   planItemID = "updates-reboot"
	itemCLIAll          planItemID = "cli-all"
	itemRipgrep         planItemID = "ripgrep"
	itemFd              planItemID = "fd"
	itemBat             planItemID = "bat"
	itemYq              planItemID = "yq"
	itemGlow            planItemID = "glow"
	itemGh              planItemID = "gh"
	itemDevAll          planItemID = "dev-all"
	itemGo              planItemID = "go"
	itemNode            planItemID = "node"
	itemRust            planItemID = "rust"
	itemGoLint          planItemID = "go-lint"
	itemGoRel           planItemID = "goreleaser"
	itemGoVuln          planItemID = "govulncheck"
	itemPnpm            planItemID = "pnpm"
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
	runBootstrap       runStepID = "bootstrap"
	runUserCreateLogin runStepID = "user-create-login"
	runUserSSHKey      runStepID = "user-ssh-key"
	runUserAllowSSH    runStepID = "user-allow-ssh"
	runUserSudo        runStepID = "user-sudo"
	runUserLinger      runStepID = "user-linger"
	runUserDockerGroup runStepID = "user-docker-group"
	runServiceUser     runStepID = "service-user"
	runServiceCreate   runStepID = "service-create"
	runServiceStatus   runStepID = "service-status"
	runServiceLogs     runStepID = "service-logs"
	runServiceRestart  runStepID = "service-restart"
	runServiceList     runStepID = "service-list"
	runServiceDisable  runStepID = "service-disable"
	runServiceRemove   runStepID = "service-remove"
	runFirewall        runStepID = "firewall"
	runHTTP            runStepID = "firewall-http"
	runHTTPS           runStepID = "firewall-https"
	runMosh            runStepID = "firewall-mosh"
	runFirewallCustom  runStepID = "firewall-custom"
	runNetworkStatus   runStepID = "network-status"
	runNetworkList     runStepID = "network-list"
	runNetworkDelete   runStepID = "network-delete"
	runNetworkReset    runStepID = "network-reset"
	runFail2Ban        runStepID = "fail2ban"
	runFail2BanStatus  runStepID = "fail2ban-status"
	runFail2BanUnban   runStepID = "fail2ban-unban"
	runDockerLog       runStepID = "docker-log"
	runContainersDisk  runStepID = "containers-disk"
	runContainersPrune runStepID = "containers-prune"
	runDoctor          runStepID = "doctor"
	runUpdates         runStepID = "updates"
	runUpdatesUpgrade  runStepID = "updates-upgrade"
	runUpdatesRebootN  runStepID = "updates-reboot-needed"
	runUpdatesUnattend runStepID = "updates-unattended"
	runUpdatesFailed   runStepID = "updates-failed"
	runUpdatesReboot   runStepID = "updates-reboot"
	runToolDeps        runStepID = "tool-deps"
	runTool            runStepID = "tool"
	runGo              runStepID = "go"
	runNode            runStepID = "node"
	runRust            runStepID = "rust"
	runGoLint          runStepID = "go-lint"
	runGoRel           runStepID = "goreleaser"
	runGoVuln          runStepID = "govulncheck"
	runPnpm            runStepID = "pnpm"
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
	Expand   key.Binding
	Show     key.Binding
	StepNav  key.Binding
	Select   key.Binding
	Continue key.Binding
	Back     key.Binding
	Retry    key.Binding
	Quit     key.Binding
	Scroll   key.Binding
}

var keys = tuiKeys{
	Toggle:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
	Expand:   key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("enter/space", "show output")),
	Show:     key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "show output")),
	StepNav:  key.NewBinding(key.WithKeys("up/down", "k/j"), key.WithHelp("up/down", "select step")),
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
	inputSub int

	selections selectionState
	planErr    string
	inputErr   string

	planList           list.Model
	help               help.Model
	usernameInput      textinput.Model
	serviceGroupsInput textinput.Model
	serviceNameInput   textinput.Model
	serviceWorkDir     textinput.Model
	serviceCommand     textinput.Model
	serviceEnvFile     textinput.Model
	firewallPortInput  textinput.Model
	firewallProtoInput textinput.Model
	firewallFromInput  textinput.Model
	firewallComment    textinput.Model
	networkRuleInput   textinput.Model
	fail2banBanTime    textinput.Model
	fail2banFindTime   textinput.Model
	fail2banMaxRetry   textinput.Model
	dockerMaxSize      textinput.Model
	dockerMaxFile      textinput.Model
	pruneTargetsInput  textinput.Model
	guardIPInput       textinput.Model
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

	runSteps        []runStep
	runningIndex    int
	selectedRunStep int
	expandedRunStep int

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
		screen:          screenMainMenu,
		selections:      defaultSelections(),
		dryRun:          dryRun,
		demo:            demo,
		help:            help.New(),
		spinner:         spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(accentStyle)),
		progress:        progress.New(progress.WithWidth(36), progress.WithColors(lipgloss.Color(colorAccent))),
		confirm:         viewport.New(),
		steps:           viewport.New(),
		output:          viewport.New(),
		runningIndex:    -1,
		selectedRunStep: -1,
		expandedRunStep: -1,
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
	case screenInputServiceUserGroups:
		content = m.inputServiceUserGroupsView()
	case screenInputServiceName:
		content = m.inputServiceNameView()
	case screenInputServiceWorkDir:
		content = m.inputServiceWorkDirView()
	case screenInputServiceCommand:
		content = m.inputServiceCommandView()
	case screenInputServiceEnvFile:
		content = m.inputServiceEnvFileView()
	case screenInputFirewallRule:
		content = m.inputFirewallRuleView()
	case screenInputNetworkRuleNumber:
		content = m.inputNetworkRuleNumberView()
	case screenInputFail2BanOptions:
		content = m.inputFail2BanOptionsView()
	case screenInputDockerLogOptions:
		content = m.inputDockerLogOptionsView()
	case screenInputDockerPruneTargets:
		content = m.inputDockerPruneTargetsView()
	case screenInputGuardIP:
		content = m.inputGuardIPView()
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
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *model) initInputs() {
	m.usernameInput = textinput.New()
	m.usernameInput.Prompt = ""
	m.usernameInput.Placeholder = "dev"
	m.usernameInput.CharLimit = 32
	m.usernameInput.SetWidth(48)

	m.serviceGroupsInput = textinput.New()
	m.serviceGroupsInput.Prompt = ""
	m.serviceGroupsInput.Placeholder = "www-data, docker"
	m.serviceGroupsInput.SetWidth(48)

	m.serviceNameInput = textinput.New()
	m.serviceNameInput.Prompt = ""
	m.serviceNameInput.Placeholder = "app"
	m.serviceNameInput.CharLimit = 64
	m.serviceNameInput.SetWidth(48)

	m.serviceWorkDir = textinput.New()
	m.serviceWorkDir.Prompt = ""
	m.serviceWorkDir.Placeholder = "/home/dev/app"
	m.serviceWorkDir.SetWidth(48)

	m.serviceCommand = textinput.New()
	m.serviceCommand.Prompt = ""
	m.serviceCommand.Placeholder = "npm start"
	m.serviceCommand.SetWidth(48)

	m.serviceEnvFile = textinput.New()
	m.serviceEnvFile.Prompt = ""
	m.serviceEnvFile.Placeholder = "/home/dev/app/.env"
	m.serviceEnvFile.SetWidth(48)

	m.firewallPortInput = textinput.New()
	m.firewallPortInput.Prompt = ""
	m.firewallPortInput.Placeholder = "443"
	m.firewallPortInput.SetWidth(48)

	m.firewallProtoInput = textinput.New()
	m.firewallProtoInput.Prompt = ""
	m.firewallProtoInput.Placeholder = "tcp"
	m.firewallProtoInput.SetValue("tcp")
	m.firewallProtoInput.SetWidth(48)

	m.firewallFromInput = textinput.New()
	m.firewallFromInput.Prompt = ""
	m.firewallFromInput.Placeholder = "203.0.113.0/24"
	m.firewallFromInput.SetWidth(48)

	m.firewallComment = textinput.New()
	m.firewallComment.Prompt = ""
	m.firewallComment.Placeholder = "setup custom"
	m.firewallComment.SetWidth(48)

	m.networkRuleInput = textinput.New()
	m.networkRuleInput.Prompt = ""
	m.networkRuleInput.Placeholder = "2"
	m.networkRuleInput.SetWidth(48)

	m.fail2banBanTime = textinput.New()
	m.fail2banBanTime.Prompt = ""
	m.fail2banBanTime.Placeholder = "1h"
	m.fail2banBanTime.SetValue("1h")
	m.fail2banBanTime.SetWidth(48)

	m.fail2banFindTime = textinput.New()
	m.fail2banFindTime.Prompt = ""
	m.fail2banFindTime.Placeholder = "10m"
	m.fail2banFindTime.SetValue("10m")
	m.fail2banFindTime.SetWidth(48)

	m.fail2banMaxRetry = textinput.New()
	m.fail2banMaxRetry.Prompt = ""
	m.fail2banMaxRetry.Placeholder = "5"
	m.fail2banMaxRetry.SetValue("5")
	m.fail2banMaxRetry.SetWidth(48)

	m.dockerMaxSize = textinput.New()
	m.dockerMaxSize.Prompt = ""
	m.dockerMaxSize.Placeholder = "10m"
	m.dockerMaxSize.SetValue("10m")
	m.dockerMaxSize.SetWidth(48)

	m.dockerMaxFile = textinput.New()
	m.dockerMaxFile.Prompt = ""
	m.dockerMaxFile.Placeholder = "3"
	m.dockerMaxFile.SetValue("3")
	m.dockerMaxFile.SetWidth(48)

	m.pruneTargetsInput = textinput.New()
	m.pruneTargetsInput.Prompt = ""
	m.pruneTargetsInput.Placeholder = "containers images build-cache"
	m.pruneTargetsInput.SetValue("containers images build-cache")
	m.pruneTargetsInput.SetWidth(48)

	m.guardIPInput = textinput.New()
	m.guardIPInput.Prompt = ""
	m.guardIPInput.Placeholder = "203.0.113.10"
	m.guardIPInput.SetWidth(48)

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
	manageAny := m.selections.InstanceManagementAny()
	manageAll := m.selections.InstanceManagementAll()
	devAll := m.selections.DevTools.Go && m.selections.DevTools.Node && m.selections.DevTools.Rust &&
		m.selections.DevTools.GoLint && m.selections.DevTools.GoReleaser &&
		m.selections.DevTools.GoVulnCheck && m.selections.DevTools.Pnpm
	userAny := m.selections.UserManagementAny()
	userAll := m.selections.UserLoginAll() && m.selections.UserCreateService
	serviceAny := m.selections.ServiceAny()
	serviceAll := m.selections.ServiceAll()

	return []list.Item{
		planItem{itemBootstrap, checkbox(m.selections.Bootstrap, m.selections.Bootstrap) + " System Bootstrap", "Locale, apt, base packages, SSH hardening, unattended upgrades, Docker"},
		planItem{itemUserAll, checkbox(userAll, userAny) + " User Management", "Login-user and setup-owned service-user workflows"},
		planItem{itemAddUser, "  " + checkbox(m.selections.UserLoginAll(), m.selections.UserLoginAny()) + " Login User Workflow", "Create/reuse login user with the classic default access set"},
		planItem{itemUserCreateLogin, "    " + checkbox(m.selections.UserCreateLogin, m.selections.UserCreateLogin) + " Create Login User", "Create or reuse the target login account"},
		planItem{itemUserSSHKey, "    " + checkbox(m.selections.UserSSHKey, m.selections.UserSSHKey) + " Add SSH Key", "Append the provided public key to authorized_keys"},
		planItem{itemUserAllowSSH, "    " + checkbox(m.selections.UserAllowSSH, m.selections.UserAllowSSH) + " Allow SSH Login", "Add the user to setup-managed AllowUsers"},
		planItem{itemUserSudo, "    " + checkbox(m.selections.UserSudo, m.selections.UserSudo) + " Passwordless Sudo", "Write setup-managed /etc/sudoers.d/<user>"},
		planItem{itemUserLinger, "    " + checkbox(m.selections.UserLinger, m.selections.UserLinger) + " Enable Linger", "Enable systemd user lingering"},
		planItem{itemUserDockerGroup, "    " + checkbox(m.selections.UserDockerGroup, m.selections.UserDockerGroup) + " Docker Group", "Add the login user to the existing docker group"},
		planItem{itemServiceUser, "  " + checkbox(m.selections.UserCreateService, m.selections.UserCreateService) + " Service User", "Create a setup-owned no-login system user under /var/lib/<user>"},
		planItem{itemServiceGroups, "    " + checkbox(m.selections.UserServiceGroups, m.selections.UserServiceGroups) + " Service User Groups", "Add the service user to existing supplementary groups"},
		planItem{itemServiceAll, checkbox(serviceAll, serviceAny) + " Managed Services", "Create, inspect, restart, disable, remove, and list setup-owned user services"},
		planItem{itemServiceCreate, "  " + checkbox(m.selections.ServiceCreate, m.selections.ServiceCreate) + " Create Service", "Create and start a setup-managed per-user systemd service"},
		planItem{itemServiceStatus, "  " + checkbox(m.selections.ServiceStatus, m.selections.ServiceStatus) + " Service Status", "Show systemd status for a managed service"},
		planItem{itemServiceLogs, "  " + checkbox(m.selections.ServiceLogs, m.selections.ServiceLogs) + " Service Logs", "Show recent journal output for a managed service"},
		planItem{itemServiceRestart, "  " + checkbox(m.selections.ServiceRestart, m.selections.ServiceRestart) + " Restart Service", "Restart a managed service"},
		planItem{itemServiceList, "  " + checkbox(m.selections.ServiceList, m.selections.ServiceList) + " List Services", "List setup-managed service unit files"},
		planItem{itemServiceDisable, "  " + checkbox(m.selections.ServiceDisable, m.selections.ServiceDisable) + " Disable Service", "Stop and disable a managed service after confirmation"},
		planItem{itemServiceRemove, "  " + checkbox(m.selections.ServiceRemove, m.selections.ServiceRemove) + " Remove Service", "Stop, disable, and delete a managed service unit after confirmation"},
		planItem{itemManageAll, checkbox(manageAll, manageAny) + " Instance Management", "UFW, common ports, fail2ban, Docker logs, diagnostics, updates"},
		planItem{itemFirewall, "  " + checkbox(m.selections.FirewallBaseline, m.selections.FirewallBaseline) + " UFW Firewall Baseline", "Default deny incoming, allow outgoing, preserve SSH access"},
		planItem{itemHTTP, "  " + checkbox(m.selections.FirewallHTTP, m.selections.FirewallHTTP) + " Allow HTTP", "Open tcp/80 through UFW"},
		planItem{itemHTTPS, "  " + checkbox(m.selections.FirewallHTTPS, m.selections.FirewallHTTPS) + " Allow HTTPS", "Open tcp/443 through UFW"},
		planItem{itemMosh, "  " + checkbox(m.selections.FirewallMosh, m.selections.FirewallMosh) + " Allow Mosh", "Open udp/60000:61000 through UFW"},
		planItem{itemFirewallCustom, "  " + checkbox(m.selections.FirewallCustom, m.selections.FirewallCustom) + " Custom Firewall Rule", "Open a custom TCP/UDP port or range through UFW"},
		planItem{itemNetworkStatus, "  " + checkbox(m.selections.NetworkStatus, m.selections.NetworkStatus) + " Network Status", "Show verbose UFW status"},
		planItem{itemNetworkList, "  " + checkbox(m.selections.NetworkList, m.selections.NetworkList) + " Numbered Network Rules", "Show numbered UFW rules"},
		planItem{itemNetworkDelete, "  " + checkbox(m.selections.NetworkDelete, m.selections.NetworkDelete) + " Delete Network Rule", "Delete a numbered UFW rule after confirmation"},
		planItem{itemNetworkReset, "  " + checkbox(m.selections.NetworkReset, m.selections.NetworkReset) + " Reset Firewall", "Reset UFW rules after confirmation"},
		planItem{itemFail2Ban, "  " + checkbox(m.selections.Fail2Ban, m.selections.Fail2Ban) + " fail2ban SSH Jail", "Install fail2ban and manage the setup SSH jail"},
		planItem{itemFail2BanStatus, "  " + checkbox(m.selections.Fail2BanStatus, m.selections.Fail2BanStatus) + " fail2ban Status", "Show fail2ban SSH jail status"},
		planItem{itemFail2BanUnban, "  " + checkbox(m.selections.Fail2BanUnban, m.selections.Fail2BanUnban) + " fail2ban Unban IP", "Unban an IP address from the SSH jail"},
		planItem{itemDockerLog, "  " + checkbox(m.selections.DockerLogRotation, m.selections.DockerLogRotation) + " Docker Log Rotation", "Set json-file max-size=10m and max-file=3"},
		planItem{itemContainersDisk, "  " + checkbox(m.selections.ContainersDisk, m.selections.ContainersDisk) + " Docker Disk Usage", "Show Docker system disk usage"},
		planItem{itemContainersPrune, "  " + checkbox(m.selections.ContainersPrune, m.selections.ContainersPrune) + " Docker Prune", "Prune selected Docker resources after confirmation"},
		planItem{itemDoctor, "  " + checkbox(m.selections.Diagnostics, m.selections.Diagnostics) + " Doctor Diagnostics", "Read-only checks for LXC, services, SSH, UFW, Docker"},
		planItem{itemUpdates, "  " + checkbox(m.selections.UpdatesCheck, m.selections.UpdatesCheck) + " Update Check", "Refresh apt metadata and list available upgrades"},
		planItem{itemUpdatesUpgrade, "  " + checkbox(m.selections.UpdatesUpgrade, m.selections.UpdatesUpgrade) + " Full Upgrade", "Run apt update and full-upgrade"},
		planItem{itemUpdatesRebootN, "  " + checkbox(m.selections.UpdatesRebootNeed, m.selections.UpdatesRebootNeed) + " Reboot Needed", "Show whether packages require a reboot"},
		planItem{itemUpdatesUnattend, "  " + checkbox(m.selections.UpdatesUnattended, m.selections.UpdatesUnattended) + " Unattended Status", "Show unattended-upgrades service status"},
		planItem{itemUpdatesFailed, "  " + checkbox(m.selections.UpdatesFailed, m.selections.UpdatesFailed) + " Failed Units", "Show failed systemd units"},
		planItem{itemUpdatesReboot, "  " + checkbox(m.selections.UpdatesReboot, m.selections.UpdatesReboot) + " Reboot Instance", "Reboot the instance after confirmation"},
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
	m.serviceGroupsInput.SetWidth(pageWidth - 4)
	m.serviceNameInput.SetWidth(pageWidth - 4)
	m.serviceWorkDir.SetWidth(pageWidth - 4)
	m.serviceCommand.SetWidth(pageWidth - 4)
	m.serviceEnvFile.SetWidth(pageWidth - 4)
	m.firewallPortInput.SetWidth(pageWidth - 4)
	m.firewallProtoInput.SetWidth(pageWidth - 4)
	m.firewallFromInput.SetWidth(pageWidth - 4)
	m.firewallComment.SetWidth(pageWidth - 4)
	m.networkRuleInput.SetWidth(pageWidth - 4)
	m.fail2banBanTime.SetWidth(pageWidth - 4)
	m.fail2banFindTime.SetWidth(pageWidth - 4)
	m.fail2banMaxRetry.SetWidth(pageWidth - 4)
	m.dockerMaxSize.SetWidth(pageWidth - 4)
	m.dockerMaxFile.SetWidth(pageWidth - 4)
	m.pruneTargetsInput.SetWidth(pageWidth - 4)
	m.guardIPInput.SetWidth(pageWidth - 4)
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
	if len(m.runSteps) > 0 {
		m.refreshOutput()
	}

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
	if m.selections.NeedsServiceUserGroups() && m.selections.UserServiceGroups {
		flow = append(flow, screenInputServiceUserGroups)
	}
	if m.selections.NeedsServiceName() {
		flow = append(flow, screenInputServiceName)
	}
	if m.selections.NeedsServiceWorkDir() {
		flow = append(flow, screenInputServiceWorkDir)
	}
	if m.selections.NeedsServiceCommand() {
		flow = append(flow, screenInputServiceCommand)
	}
	if m.selections.NeedsServiceEnvFile() {
		flow = append(flow, screenInputServiceEnvFile)
	}
	if m.selections.NeedsFirewallRule() {
		flow = append(flow, screenInputFirewallRule)
	}
	if m.selections.NeedsNetworkRuleNumber() {
		flow = append(flow, screenInputNetworkRuleNumber)
	}
	if m.selections.NeedsFail2BanOptions() {
		flow = append(flow, screenInputFail2BanOptions)
	}
	if m.selections.NeedsDockerLogOptions() {
		flow = append(flow, screenInputDockerLogOptions)
	}
	if m.selections.NeedsDockerPruneTargets() {
		flow = append(flow, screenInputDockerPruneTargets)
	}
	if m.selections.NeedsGuardIP() {
		flow = append(flow, screenInputGuardIP)
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
	for _, selected := range []bool{
		m.selections.UserCreateLogin,
		m.selections.UserSSHKey,
		m.selections.UserAllowSSH,
		m.selections.UserSudo,
		m.selections.UserLinger,
		m.selections.UserDockerGroup,
		m.selections.UserCreateService,
		m.selections.UserServiceGroups,
		m.selections.ServiceCreate,
		m.selections.ServiceStatus,
		m.selections.ServiceLogs,
		m.selections.ServiceRestart,
		m.selections.ServiceList,
		m.selections.ServiceDisable,
		m.selections.ServiceRemove,
	} {
		if selected {
			count++
		}
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
	if m.selections.FirewallCustom {
		count++
	}
	if m.selections.NetworkStatus {
		count++
	}
	if m.selections.NetworkList {
		count++
	}
	if m.selections.NetworkDelete {
		count++
	}
	if m.selections.NetworkReset {
		count++
	}
	if m.selections.Fail2Ban {
		count++
	}
	if m.selections.Fail2BanStatus {
		count++
	}
	if m.selections.Fail2BanUnban {
		count++
	}
	if m.selections.DockerLogRotation {
		count++
	}
	if m.selections.ContainersDisk {
		count++
	}
	if m.selections.ContainersPrune {
		count++
	}
	if m.selections.Diagnostics {
		count++
	}
	if m.selections.UpdatesCheck {
		count++
	}
	if m.selections.UpdatesUpgrade {
		count++
	}
	if m.selections.UpdatesRebootNeed {
		count++
	}
	if m.selections.UpdatesUnattended {
		count++
	}
	if m.selections.UpdatesFailed {
		count++
	}
	if m.selections.UpdatesReboot {
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
		return "○"
	case stepRunning:
		return "•"
	case stepOK:
		return "✓"
	case stepFail:
		return "✗"
	default:
		return "○"
	}
}

func (m model) currentStepSummary() string {
	if m.runningIndex < 0 || m.runningIndex >= len(m.runSteps) {
		return ""
	}
	return fmt.Sprintf("%d/%d %s", m.runningIndex+1, len(m.runSteps), m.runSteps[m.runningIndex].name)
}
