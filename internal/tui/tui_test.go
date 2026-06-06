package tui

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sqamsqam/setup/internal/tools"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(false)

	if m.dryRun {
		t.Error("expected dryRun to be false")
	}
	if m.timezoneInput.Value() != "UTC" {
		t.Errorf("expected default timezone UTC, got: %s", m.timezoneInput.Value())
	}
	if m.screen != screenHome {
		t.Errorf("expected screenHome, got: %d", m.screen)
	}
	if m.selections.Any() {
		t.Fatalf("expected no default selection: %#v", m.selections)
	}
	if len(m.homeItems()) != 8 {
		t.Errorf("expected 8 home items, got: %d", len(m.homeItems()))
	}
	if len(m.planItems(areaFreshSetup)) != 1 {
		t.Errorf("expected 1 fresh setup item, got: %d", len(m.planItems(areaFreshSetup)))
	}
}

func TestInitialModelDryRun(t *testing.T) {
	m := InitialModel(true)
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestInitialModelDemoMode(t *testing.T) {
	m := InitialModelWithMode(false, true)
	if !m.demo {
		t.Error("expected demo to be true")
	}
	if strings.Contains(m.homeView(), "DRY RUN") {
		t.Fatal("demo mode should not render dry-run banner")
	}
}

func TestSelectionRequirements(t *testing.T) {
	s := selectionState{}
	if s.Any() || s.NeedsTimezone() || s.NeedsUsername() || s.NeedsSSHKey() {
		t.Fatalf("empty selection should not need inputs: %#v", s)
	}

	s.Bootstrap = true
	if !s.NeedsTimezone() {
		t.Fatal("bootstrap should require timezone")
	}

	s = selectionState{}
	s.DevTools.Node = true
	if !s.NeedsUsername() || s.NeedsSSHKey() {
		t.Fatal("node should require username but not SSH key")
	}

	s = selectionState{UserCreateLogin: true, UserSSHKey: true}
	if !s.NeedsUsername() || !s.NeedsSSHKey() {
		t.Fatal("add user should require username and SSH key")
	}

	s = selectionState{UserCreateService: true}
	if !s.NeedsUsername() || s.NeedsSSHKey() {
		t.Fatal("service user should require username but not SSH key")
	}

	s = selectionState{ServiceList: true}
	if !s.NeedsUsername() || s.NeedsServiceName() || s.NeedsSSHKey() {
		t.Fatal("service list should require username only")
	}

	s = selectionState{ServiceRestart: true}
	if !s.NeedsUsername() || !s.NeedsServiceName() || s.NeedsServiceWorkDir() || s.NeedsSSHKey() {
		t.Fatal("service restart should require username and service name only")
	}

	s = selectionState{ServiceCreate: true}
	if !s.NeedsUsername() || !s.NeedsServiceName() || !s.NeedsServiceWorkDir() ||
		!s.NeedsServiceCommand() || !s.NeedsServiceEnvFile() || s.NeedsSSHKey() {
		t.Fatal("service create should require username, name, workdir, command, and optional env-file screen")
	}

	s = selectionState{FirewallCustom: true}
	if !s.NeedsFirewallRule() {
		t.Fatal("custom firewall rule should require firewall rule input")
	}

	s = selectionState{NetworkDelete: true}
	if !s.NeedsNetworkRuleNumber() {
		t.Fatal("network delete should require rule number input")
	}

	s = selectionState{Fail2Ban: true, DockerLogRotation: true, ContainersPrune: true, Fail2BanUnban: true}
	if !s.NeedsFail2BanOptions() || !s.NeedsDockerLogOptions() || !s.NeedsDockerPruneTargets() || !s.NeedsGuardIP() {
		t.Fatal("admin option selections should require matching inputs")
	}
}

func TestToggleCLIAll(t *testing.T) {
	m := InitialModel(false)

	m.togglePlanItem(itemCLIAll)
	if got := len(m.selections.Tools.SelectedTools()); got != 6 {
		t.Fatalf("expected all CLI tools selected, got %d", got)
	}

	m.togglePlanItem(itemCLIAll)
	if m.selections.Tools.Any() {
		t.Fatalf("expected CLI tools to be disabled: %#v", m.selections.Tools)
	}
}

func TestToggleIndividualTool(t *testing.T) {
	m := InitialModel(false)
	m.selections.Tools = tools.InstallOptions{}

	m.togglePlanItem(itemYq)
	if !m.selections.Tools.Yq {
		t.Fatal("expected yq to be selected")
	}
	if got := len(m.selections.Tools.SelectedTools()); got != 1 {
		t.Fatalf("expected one selected tool, got %d", got)
	}
}

func TestToggleManagedServices(t *testing.T) {
	m := InitialModel(false)

	m.togglePlanItem(itemServiceAll)
	if !m.selections.ServiceAll() {
		t.Fatalf("expected all managed services selected: %#v", m.selections)
	}

	m.togglePlanItem(itemServiceAll)
	if m.selections.ServiceAny() {
		t.Fatalf("expected managed services disabled: %#v", m.selections)
	}
}

func TestToggleIndividualManagedService(t *testing.T) {
	m := InitialModel(false)

	m.togglePlanItem(itemServiceList)
	if !m.selections.ServiceList {
		t.Fatal("expected service list to be selected")
	}
	if !m.selections.ServiceAny() || m.selections.ServiceAll() {
		t.Fatalf("expected partial managed service selection: %#v", m.selections)
	}
}

func TestToggleInstanceManagementAll(t *testing.T) {
	m := InitialModel(false)

	m.togglePlanItem(itemManageAll)
	if !m.selections.InstanceManagementAll() {
		t.Fatalf("expected all instance-management actions selected: %#v", m.selections)
	}

	m.togglePlanItem(itemManageAll)
	if m.selections.InstanceManagementAny() {
		t.Fatalf("expected instance-management actions disabled: %#v", m.selections)
	}
}

func TestMainMenuSpaceTogglesSelectedPlanItem(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenMainMenu
	m.currentArea = areaFreshSetup
	m.planList = m.newPlanList()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Text: " "}))
	got := updated.(model)
	if !got.selections.Bootstrap {
		t.Fatal("expected space to toggle the selected bootstrap item on")
	}

	item, ok := got.planList.SelectedItem().(planItem)
	if !ok {
		t.Fatal("expected selected plan item")
	}
	_, state, title := splitPlanTitle(item.Title())
	if state != toggleOn || title != "System Bootstrap" {
		t.Fatalf("expected checked System Bootstrap item, got state=%d title=%q", state, title)
	}
}

func TestPlanDelegateTruncatesRowsToListWidth(t *testing.T) {
	m := InitialModel(false)
	m.resize(52, 24)

	var out bytes.Buffer
	item := planItem{
		id:    itemBootstrap,
		title: checkbox(true, true) + " System Bootstrap With A Very Long Name",
		desc:  "This description is intentionally long enough that it would wrap if the delegate did not truncate it.",
	}
	planDelegate{}.Render(&out, m.planList, 0, item)

	for _, line := range strings.Split(out.String(), "\n") {
		if got := ansi.StringWidth(ansi.Strip(line)); got > m.planList.Width() {
			t.Fatalf("rendered line width = %d, want <= %d: %q", got, m.planList.Width(), line)
		}
	}
}

func TestPlanDelegateUsesToggleGlyphs(t *testing.T) {
	m := InitialModel(false)
	m.resize(80, 24)

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"on", checkbox(true, true) + " System Bootstrap", "●"},
		{"partial", checkbox(false, true) + " Instance Management", "◐"},
		{"off", checkbox(false, false) + " Allow HTTP", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			planDelegate{}.Render(&out, m.planList, 0, planItem{title: tt.title, desc: "desc"})
			rendered := ansi.Strip(out.String())
			if !strings.Contains(rendered, tt.want) {
				t.Fatalf("expected rendered toggle %q in %q", tt.want, rendered)
			}
			if strings.Contains(rendered, "[x]") || strings.Contains(rendered, "[-]") || strings.Contains(rendered, "[ ]") {
				t.Fatalf("rendered old checkbox syntax: %q", rendered)
			}
		})
	}
}

func TestStartInputFlowWithFreshSetupStartsWithTimezone(t *testing.T) {
	m := InitialModel(false)
	m.selections.Bootstrap = true

	updated, _ := m.startInputFlow()
	got := updated.(model)
	if got.screen != screenInputTimezone {
		t.Fatalf("expected timezone screen, got %d", got.screen)
	}
}

func TestStartInputFlowWithOnlyCLIShowsConfirm(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{}
	m.selections.Tools.Yq = true
	m.currentArea = areaTools

	updated, _ := m.startInputFlow()
	got := updated.(model)
	if got.screen != screenConfirm {
		t.Fatalf("expected confirm screen, got %d", got.screen)
	}
}

func TestStartInputFlowWithServiceListStartsWithUser(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{ServiceList: true}
	m.currentArea = areaServices

	updated, _ := m.startInputFlow()
	got := updated.(model)
	if got.screen != screenInputUser {
		t.Fatalf("expected username screen, got %d", got.screen)
	}
}

func TestStartInputFlowRejectsEmptySelection(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenMainMenu
	m.selections = selectionState{}

	updated, _ := m.startInputFlow()
	got := updated.(model)
	if got.screen != screenMainMenu {
		t.Fatalf("expected menu screen, got %d", got.screen)
	}
	if got.planErr == "" {
		t.Fatal("expected plan error")
	}
}

func TestInputFlow(t *testing.T) {
	m := InitialModel(false)
	if got := len(m.inputFlow()); got != 0 {
		t.Fatalf("expected no default input screens, got %d", got)
	}

	m.selections = selectionState{}
	m.selections.DevTools.Node = true
	flow := m.inputFlow()
	if len(flow) != 1 || flow[0] != screenInputUser {
		t.Fatalf("expected only username flow, got %#v", flow)
	}

	m.selections = selectionState{ServiceCreate: true}
	flow = m.inputFlow()
	want := []screen{
		screenInputUser,
		screenInputServiceName,
		screenInputServiceWorkDir,
		screenInputServiceCommand,
		screenInputServiceEnvFile,
	}
	if len(flow) != len(want) {
		t.Fatalf("service create flow = %#v, want %#v", flow, want)
	}
	for i := range want {
		if flow[i] != want[i] {
			t.Fatalf("service create flow = %#v, want %#v", flow, want)
		}
	}

	m.selections = selectionState{ServiceRemove: true}
	flow = m.inputFlow()
	if len(flow) != 2 || flow[0] != screenInputUser || flow[1] != screenInputServiceName {
		t.Fatalf("expected username + service name flow, got %#v", flow)
	}

	m.selections = selectionState{
		FirewallCustom:    true,
		NetworkDelete:     true,
		Fail2Ban:          true,
		DockerLogRotation: true,
		ContainersPrune:   true,
		Fail2BanUnban:     true,
	}
	flow = m.inputFlow()
	want = []screen{
		screenInputFirewallRule,
		screenInputNetworkRuleNumber,
		screenInputFail2BanOptions,
		screenInputDockerLogOptions,
		screenInputDockerPruneTargets,
		screenInputGuardIP,
	}
	if len(flow) != len(want) {
		t.Fatalf("admin flow = %#v, want %#v", flow, want)
	}
	for i := range want {
		if flow[i] != want[i] {
			t.Fatalf("admin flow = %#v, want %#v", flow, want)
		}
	}

	m.selections = selectionState{GroupAddUser: true}
	flow = m.inputFlow()
	want = []screen{screenInputUser, screenInputGroupName}
	if len(flow) != len(want) {
		t.Fatalf("group membership flow = %#v, want %#v", flow, want)
	}
	for i := range want {
		if flow[i] != want[i] {
			t.Fatalf("group membership flow = %#v, want %#v", flow, want)
		}
	}
}

func TestBuildRunStepsFreshUserTools(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	steps := m.buildRunSteps()

	if len(steps) != 16 {
		t.Fatalf("expected 16 run steps, got %d", len(steps))
	}
	if steps[0].id != runBootstrap || steps[1].id != runUserCreateLogin || steps[2].id != runUserSSHKey {
		t.Fatalf("unexpected leading steps: %#v", steps[:3])
	}
	if steps[len(steps)-2].id != runGo || steps[len(steps)-1].id != runNode {
		t.Fatalf("expected Go and Node last, got %#v", steps[len(steps)-2:])
	}
}

func TestBuildRunStepsYqOnly(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{}
	m.selections.Tools.Yq = true

	steps := m.buildRunSteps()
	if len(steps) != 2 {
		t.Fatalf("expected deps + yq, got %d steps", len(steps))
	}
	if steps[0].id != runToolDeps || steps[1].tool != tools.ToolYq {
		t.Fatalf("unexpected steps: %#v", steps)
	}
}

func TestBuildRunStepsManagedServices(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{ServiceCreate: true, ServiceList: true, ServiceRemove: true}

	steps := m.buildRunSteps()
	if len(steps) != 3 {
		t.Fatalf("expected 3 service steps, got %d", len(steps))
	}
	want := []runStepID{runServiceCreate, runServiceList, runServiceRemove}
	for i := range want {
		if steps[i].id != want[i] {
			t.Fatalf("steps = %#v, want ids %#v", steps, want)
		}
	}
}

func TestBuildRunStepsGroups(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{GroupCreate: true, GroupList: true, GroupRemoveUser: true}

	steps := m.buildRunSteps()
	want := []runStepID{runGroupCreate, runGroupList, runGroupRemoveUser}
	if len(steps) != len(want) {
		t.Fatalf("expected %d group steps, got %d", len(want), len(steps))
	}
	for i := range want {
		if steps[i].id != want[i] {
			t.Fatalf("steps = %#v, want ids %#v", steps, want)
		}
	}
}

func TestBuildRunStepsInstanceManagementGaps(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{
		FirewallCustom:    true,
		NetworkStatus:     true,
		NetworkList:       true,
		NetworkDelete:     true,
		NetworkReset:      true,
		Fail2BanStatus:    true,
		Fail2BanUnban:     true,
		ContainersDisk:    true,
		ContainersPrune:   true,
		UpdatesUpgrade:    true,
		UpdatesRebootNeed: true,
		UpdatesUnattended: true,
		UpdatesFailed:     true,
		UpdatesReboot:     true,
	}

	steps := m.buildRunSteps()
	want := []runStepID{
		runFirewallCustom,
		runNetworkStatus,
		runNetworkList,
		runNetworkDelete,
		runNetworkReset,
		runFail2BanStatus,
		runFail2BanUnban,
		runContainersDisk,
		runContainersPrune,
		runUpdatesUpgrade,
		runUpdatesRebootN,
		runUpdatesUnattend,
		runUpdatesFailed,
		runUpdatesReboot,
	}
	if len(steps) != len(want) {
		t.Fatalf("expected %d steps, got %d: %#v", len(want), len(steps), steps)
	}
	for i := range want {
		if steps[i].id != want[i] {
			t.Fatalf("steps = %#v, want ids %#v", steps, want)
		}
	}
}

func TestBlankTimezoneDefaultsToUTC(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{Bootstrap: true}
	m.screen = screenInputTimezone
	m.timezoneInput.SetValue("")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.timezoneInput.Value() != "UTC" {
		t.Fatalf("expected UTC, got %q", got.timezoneInput.Value())
	}
	if got.screen != screenConfirm {
		t.Fatalf("expected confirm screen, got %d", got.screen)
	}
}

func TestFuzzyTimezoneMatches(t *testing.T) {
	zones := []string{
		"America/Chicago",
		"America/Los_Angeles",
		"America/New_York",
		"Australia/Sydney",
		"UTC",
	}

	matches := fuzzyTimezoneMatches("ny", zones, 3)
	if len(matches) == 0 || matches[0] != "America/New_York" {
		t.Fatalf("expected New York match first, got %#v", matches)
	}

	matches = fuzzyTimezoneMatches("los angeles", zones, 3)
	if len(matches) == 0 || matches[0] != "America/Los_Angeles" {
		t.Fatalf("expected Los Angeles match first, got %#v", matches)
	}
}

func TestTabAcceptsFuzzyTimezoneMatch(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputTimezone
	m.timezones = []string{"America/Chicago", "America/New_York", "UTC"}
	m.timezoneInput.SetValue("ny")
	m.refreshTimezoneMatches()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	got := updated.(model)
	if got.timezoneInput.Value() != "America/New_York" {
		t.Fatalf("expected accepted fuzzy match, got %q", got.timezoneInput.Value())
	}
}

func TestEmptyUsernameShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputUser
	m.usernameInput.SetValue("")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected username error")
	}
	if got.screen != screenInputUser {
		t.Fatalf("expected to stay on username screen, got %d", got.screen)
	}
}

func TestInvalidServiceNameShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputServiceName
	m.serviceNameInput.SetValue("bad/name")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected service name error")
	}
	if got.screen != screenInputServiceName {
		t.Fatalf("expected to stay on service name screen, got %d", got.screen)
	}
}

func TestInvalidServiceWorkDirShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputServiceWorkDir
	m.serviceWorkDir.SetValue("relative/path")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected service workdir error")
	}
	if got.screen != screenInputServiceWorkDir {
		t.Fatalf("expected to stay on service workdir screen, got %d", got.screen)
	}
}

func TestEmptyServiceCommandShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputServiceCommand
	m.serviceCommand.SetValue("")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected service command error")
	}
	if got.screen != screenInputServiceCommand {
		t.Fatalf("expected to stay on service command screen, got %d", got.screen)
	}
}

func TestInvalidServiceEnvFileShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputServiceEnvFile
	m.serviceEnvFile.SetValue("relative.env")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected service env file error")
	}
	if got.screen != screenInputServiceEnvFile {
		t.Fatalf("expected to stay on service env-file screen, got %d", got.screen)
	}
}

func TestInvalidSSHKeyShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputKey
	m.sshKeyInput.SetValue("not-a-key")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.inputErr == "" {
		t.Fatal("expected SSH key error")
	}
	if got.screen != screenInputKey {
		t.Fatalf("expected to stay on SSH key screen, got %d", got.screen)
	}
}

func TestConfirmBodyIncludesManagedServiceWarnings(t *testing.T) {
	m := InitialModel(false)
	m.selections = selectionState{ServiceDisable: true, ServiceRemove: true}
	m.usernameInput.SetValue("dev")
	m.serviceNameInput.SetValue("app")

	body := m.confirmBody()
	for _, want := range []string{
		"Managed Service: disable",
		"Managed Service: remove",
		"The setup-managed service will be stopped and disabled.",
		"The setup-managed service unit file will be removed",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in confirm body:\n%s", want, body)
		}
	}
}

func TestConfirmScreenScrolls(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	m.resize(80, 12)
	m.screen = screenConfirm
	m.refreshConfirm()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	got := updated.(model)
	if got.confirm.YOffset() == 0 {
		t.Fatal("expected confirm viewport to scroll")
	}
}

func TestRunningViewFitsTerminalHeight(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	m.runSteps = m.buildRunSteps()
	m.runningIndex = 0
	m.runSteps[0].status = stepRunning
	m.runSteps[0].output = strings.Repeat("log line\n", 80)
	m.resize(80, 24)
	m.screen = screenRunning
	m.refreshSteps()
	m.refreshOutput()

	if got := lipgloss.Height(m.runningView()); got > m.height {
		t.Fatalf("running view height = %d, want <= %d", got, m.height)
	}
}

func TestRunningViewFitsTerminalHeightWithLongLogLines(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	m.runSteps = m.buildRunSteps()
	m.runningIndex = 0
	m.runSteps[0].status = stepRunning
	m.runSteps[0].output = strings.Repeat("x", 2000)
	m.resize(80, 24)
	m.screen = screenRunning
	m.refreshSteps()
	m.refreshOutput()

	if got := lipgloss.Height(m.runningView()); got > m.height {
		t.Fatalf("running view height = %d, want <= %d", got, m.height)
	}
}

func TestRefreshOutputTruncatesLongLogLines(t *testing.T) {
	m := InitialModel(false)
	m.runSteps = []runStep{{
		name:   "Long output",
		status: stepOK,
		output: strings.Repeat("x", 200),
	}}
	m.expandedRunStep = 0
	m.output.SetWidth(40)
	m.refreshOutput()

	for _, line := range strings.Split(m.output.GetContent(), "\n") {
		if got := ansi.StringWidth(ansi.Strip(line)); got > m.output.Width() {
			t.Fatalf("log line width = %d, want <= %d: %q", got, m.output.Width(), line)
		}
	}
	if m.runSteps[0].output != strings.Repeat("x", 200) {
		t.Fatal("raw step output should remain unmodified")
	}
}

func TestDoneViewFitsTerminalHeight(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	m.runSteps = m.buildRunSteps()
	for i := range m.runSteps {
		m.runSteps[i].status = stepOK
	}
	m.runningIndex = len(m.runSteps) - 1
	m.resize(80, 24)
	m.screen = screenDone
	m.refreshSteps()
	m.refreshOutput()

	if got := lipgloss.Height(m.doneView()); got > m.height {
		t.Fatalf("done view height = %d, want <= %d", got, m.height)
	}
}

func TestRunningStepListScrolls(t *testing.T) {
	m := InitialModel(false)
	m.selections = defaultSelections()
	m.runSteps = m.buildRunSteps()
	m.runningIndex = 0
	m.runSteps[0].status = stepRunning
	m.resize(80, 18)
	m.screen = screenRunning
	m.refreshSteps()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	got := updated.(model)
	if got.steps.YOffset() == 0 {
		t.Fatal("expected run step viewport to scroll")
	}
}

func TestViewsFitStandardTerminalHeight(t *testing.T) {
	tests := []struct {
		name  string
		model func() model
	}{
		{
			name: "main menu",
			model: func() model {
				m := InitialModel(true)
				m.resize(80, 24)
				m.planErr = "select at least one action"
				return m
			},
		},
		{
			name: "timezone input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputTimezone
				m.timezoneMatches = []string{
					"America/Anchorage",
					"America/Chicago",
					"America/Denver",
					"America/Los_Angeles",
					"America/New_York",
					"America/Phoenix",
					"America/Toronto",
					"America/Vancouver",
				}
				m.inputErr = `unknown timezone "America/Nowhere"`
				return m
			},
		},
		{
			name: "user input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputUser
				m.inputErr = "invalid username"
				return m
			},
		},
		{
			name: "service name input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputServiceName
				m.inputErr = "invalid service name"
				return m
			},
		},
		{
			name: "service workdir input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputServiceWorkDir
				m.inputErr = "invalid workdir"
				return m
			},
		},
		{
			name: "service command input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputServiceCommand
				m.inputErr = "invalid command"
				return m
			},
		},
		{
			name: "service env-file input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputServiceEnvFile
				m.inputErr = "invalid env file"
				return m
			},
		},
		{
			name: "ssh key input",
			model: func() model {
				m := InitialModel(false)
				m.resize(80, 24)
				m.screen = screenInputKey
				m.inputErr = "invalid SSH public key"
				return m
			},
		},
		{
			name: "confirm",
			model: func() model {
				m := InitialModel(true)
				m.selections = defaultSelections()
				m.resize(80, 24)
				m.screen = screenConfirm
				m.refreshConfirm()
				return m
			},
		},
		{
			name: "running",
			model: func() model {
				m := InitialModel(false)
				m.selections = defaultSelections()
				m.runSteps = m.buildRunSteps()
				m.runningIndex = 0
				m.runSteps[0].status = stepRunning
				m.runSteps[0].output = strings.Repeat("log line\n", 80)
				m.resize(80, 24)
				m.screen = screenRunning
				m.refreshSteps()
				m.refreshOutput()
				return m
			},
		},
		{
			name: "done",
			model: func() model {
				m := InitialModel(false)
				m.selections = defaultSelections()
				m.runSteps = m.buildRunSteps()
				for i := range m.runSteps {
					m.runSteps[i].status = stepOK
					m.runSteps[i].output = strings.Repeat("log line\n", 20)
				}
				m.runningIndex = len(m.runSteps) - 1
				m.resize(80, 24)
				m.screen = screenDone
				m.refreshSteps()
				m.refreshOutput()
				return m
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.model()
			if got := lipgloss.Height(m.View().Content); got > m.height {
				t.Fatalf("view height = %d, want <= %d", got, m.height)
			}
		})
	}
}

func TestNormalizeSSHKeyInput(t *testing.T) {
	got := normalizeSSHKeyInput("ssh-ed25519   abc123\nuser@example\n")
	if got != "ssh-ed25519 abc123 user@example" {
		t.Fatalf("unexpected normalized key: %q", got)
	}
}

func TestFailedStepRetriesInsteadOfResetting(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenDone
	m.runningIndex = 0
	m.runSteps = []runStep{{id: runBootstrap, name: "System Bootstrap", status: stepFail}}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.runningIndex != 0 {
		t.Fatalf("expected run index to stay at 0, got %d", got.runningIndex)
	}
	if got.screen != screenRunning {
		t.Fatalf("expected retry to enter running screen, got %d", got.screen)
	}
	if got.runSteps[0].status != stepRunning {
		t.Fatalf("expected failed step to be running, got %d", got.runSteps[0].status)
	}
}

func TestHandleStepMsgAdvancesToNextStep(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runningIndex = 0
	m.runSteps = []runStep{
		{id: runBootstrap, name: "System Bootstrap", status: stepRunning},
		{id: runGo, name: "Install Go", status: stepPending},
	}

	updated, _ := m.handleStepMsg(stepStatusMsg{index: 0, status: stepOK, output: "done"})
	got := updated.(model)
	if got.runSteps[0].status != stepOK {
		t.Fatalf("expected first step OK, got %d", got.runSteps[0].status)
	}
	if got.runningIndex != 1 || got.runSteps[1].status != stepRunning {
		t.Fatalf("expected second step running, got index=%d status=%d", got.runningIndex, got.runSteps[1].status)
	}
	if got.screen != screenRunning {
		t.Fatalf("expected running screen, got %d", got.screen)
	}
}

func TestChainProgress(t *testing.T) {
	m := InitialModel(false)
	m.runSteps = []runStep{
		{status: stepOK},
		{status: stepPending},
		{status: stepPending},
		{status: stepOK},
	}
	if got := m.runProgress(); got != 0.5 {
		t.Fatalf("expected 50%% progress, got %f", got)
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status stepStatus
		want   string
	}{
		{stepPending, "○"},
		{stepRunning, "•"},
		{stepOK, "✓"},
		{stepFail, "✗"},
	}

	for _, tt := range tests {
		got := statusIcon(tt.status)
		if got != tt.want {
			t.Errorf("statusIcon(%v) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestRunOutputHiddenByDefault(t *testing.T) {
	m := InitialModel(false)
	m.runSteps = []runStep{{
		name:   "System Bootstrap",
		status: stepOK,
		output: "apt update",
	}}
	m.output.SetWidth(80)
	m.refreshOutput()

	if got := strings.TrimSpace(m.output.GetContent()); got != "" {
		t.Fatalf("expected collapsed output, got %q", got)
	}
}

func TestEnterTogglesSelectedRunStepOutput(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runSteps = []runStep{
		{name: "System Bootstrap", status: stepOK, output: "bootstrap output"},
		{name: "Install Go", status: stepOK, output: "go output"},
	}
	m.selectedRunStep = 1
	m.expandedRunStep = -1
	m.output.SetWidth(80)
	m.refreshSteps()
	m.refreshOutput()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.expandedRunStep != 1 {
		t.Fatalf("expected step 1 expanded, got %d", got.expandedRunStep)
	}
	if content := got.output.GetContent(); !strings.Contains(content, "go output") || strings.Contains(content, "bootstrap output") {
		t.Fatalf("unexpected expanded output: %q", content)
	}

	updated, _ = got.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got = updated.(model)
	if got.expandedRunStep != -1 {
		t.Fatalf("expected output collapsed, got %d", got.expandedRunStep)
	}
}

func TestRunStepNavigationDoesNotChangeExpandedStep(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runSteps = []runStep{
		{name: "System Bootstrap", status: stepOK, output: "bootstrap output"},
		{name: "Install Go", status: stepRunning},
	}
	m.selectedRunStep = 0
	m.expandedRunStep = 0
	m.refreshSteps()
	m.refreshOutput()

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got := updated.(model)
	if got.selectedRunStep != 1 {
		t.Fatalf("expected selected step 1, got %d", got.selectedRunStep)
	}
	if got.expandedRunStep != 0 {
		t.Fatalf("expected expanded step to remain 0, got %d", got.expandedRunStep)
	}
}

func TestFailedStepAutoExpandsOutput(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runningIndex = 0
	m.runSteps = []runStep{{name: "System Bootstrap", status: stepRunning}}
	m.output.SetWidth(80)

	updated, _ := m.handleStepMsg(stepStatusMsg{index: 0, status: stepFail, output: "boom"})
	got := updated.(model)
	if got.expandedRunStep != 0 {
		t.Fatalf("expected failed step expanded, got %d", got.expandedRunStep)
	}
	if !strings.Contains(got.output.GetContent(), "boom") {
		t.Fatalf("expected failed output visible, got %q", got.output.GetContent())
	}
}

func TestMouseClickTogglesRunStepOutput(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runSteps = []runStep{
		{name: "System Bootstrap", status: stepOK, output: "bootstrap output"},
		{name: "Install Go", status: stepOK, output: "go output"},
	}
	m.selectedRunStep = 0
	m.expandedRunStep = -1
	m.resize(100, 24)
	m.refreshSteps()
	m.refreshOutput()

	left, top, _, _ := m.runStepViewportBounds()
	updated, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: left, Y: top + 1, Button: tea.MouseLeft}))
	got := updated.(model)
	if got.selectedRunStep != 1 || got.expandedRunStep != 1 {
		t.Fatalf("expected clicked step selected and expanded, selected=%d expanded=%d", got.selectedRunStep, got.expandedRunStep)
	}
	if !strings.Contains(got.output.GetContent(), "go output") {
		t.Fatalf("expected clicked step output, got %q", got.output.GetContent())
	}
}

func TestMouseClickTogglesRunStepOutputStackedLayout(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenRunning
	m.runSteps = []runStep{
		{name: "System Bootstrap", status: stepOK, output: "bootstrap output"},
		{name: "Install Go", status: stepOK, output: "go output"},
	}
	m.selectedRunStep = 0
	m.expandedRunStep = -1
	m.resize(80, 24)
	m.refreshSteps()
	m.refreshOutput()

	left, top, _, _ := m.runStepViewportBounds()
	updated, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: left, Y: top + 1, Button: tea.MouseLeft}))
	got := updated.(model)
	if got.selectedRunStep != 1 || got.expandedRunStep != 1 {
		t.Fatalf("expected stacked clicked step selected and expanded, selected=%d expanded=%d", got.selectedRunStep, got.expandedRunStep)
	}
	if !strings.Contains(got.output.GetContent(), "go output") {
		t.Fatalf("expected clicked step output, got %q", got.output.GetContent())
	}
}

func TestMouseClickTogglesRunStepOutputFailedDoneLayout(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenDone
	m.runningIndex = 0
	m.runSteps = []runStep{
		{name: "System Bootstrap", status: stepFail, output: "boom"},
		{name: "Install Go", status: stepPending},
	}
	m.selectedRunStep = 0
	m.expandedRunStep = 0
	m.resize(80, 24)
	m.refreshSteps()
	m.refreshOutput()

	left, top, _, _ := m.runStepViewportBounds()
	updated, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: left, Y: top, Button: tea.MouseLeft}))
	got := updated.(model)
	if got.selectedRunStep != 0 || got.expandedRunStep != -1 {
		t.Fatalf("expected failed done click to collapse selected step, selected=%d expanded=%d", got.selectedRunStep, got.expandedRunStep)
	}
	if strings.TrimSpace(got.output.GetContent()) != "" {
		t.Fatalf("expected collapsed output, got %q", got.output.GetContent())
	}
}

func TestViewEnablesMouse(t *testing.T) {
	m := InitialModel(false)
	view := m.View()
	if view.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("expected mouse cell motion, got %v", view.MouseMode)
	}
}

func TestTruncateKey(t *testing.T) {
	short := "ssh-ed25519 test"
	if got := truncateKey(short, 40); got != short {
		t.Errorf("short key should not be truncated, got: %q", got)
	}

	long := strings.Repeat("a", 100)
	got := truncateKey(long, 40)
	if len(got) != 40 {
		t.Errorf("expected truncated key length 40, got: %d", len(got))
	}
	if got != long[:37]+"..." {
		t.Errorf("expected truncated preview with ellipsis, got: %q", got)
	}
}

func TestDrawOutputHelpers(t *testing.T) {
	if got := truncateOutput("abc", 10); got != "abc" {
		t.Fatalf("unexpected truncateOutput result: %q", got)
	}
	if got := indentLines("a\nb", "  "); got != "  a\n  b" {
		t.Fatalf("unexpected indentLines result: %q", got)
	}
}
