package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
	if m.screen != screenMainMenu {
		t.Errorf("expected screenMainMenu, got: %d", m.screen)
	}
	if !m.selections.Bootstrap || !m.selections.AddUser || !m.selections.Tools.Any() || !m.selections.DevTools.Any() {
		t.Fatalf("expected full default selection: %#v", m.selections)
	}
	if len(m.planItems()) != 12 {
		t.Errorf("expected 12 plan items, got: %d", len(m.planItems()))
	}
}

func TestInitialModelDryRun(t *testing.T) {
	m := InitialModel(true)
	if !m.dryRun {
		t.Error("expected dryRun to be true")
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

	s = selectionState{AddUser: true}
	if !s.NeedsUsername() || !s.NeedsSSHKey() {
		t.Fatal("add user should require username and SSH key")
	}
}

func TestToggleCLIAll(t *testing.T) {
	m := InitialModel(false)

	m.togglePlanItem(itemCLIAll)
	if m.selections.Tools.Any() {
		t.Fatalf("expected CLI tools to be disabled: %#v", m.selections.Tools)
	}

	m.togglePlanItem(itemCLIAll)
	if got := len(m.selections.Tools.SelectedTools()); got != 6 {
		t.Fatalf("expected all CLI tools selected, got %d", got)
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

func TestStartInputFlowDefaultStartsWithTimezone(t *testing.T) {
	m := InitialModel(false)

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

	updated, _ := m.startInputFlow()
	got := updated.(model)
	if got.screen != screenConfirm {
		t.Fatalf("expected confirm screen, got %d", got.screen)
	}
}

func TestStartInputFlowRejectsEmptySelection(t *testing.T) {
	m := InitialModel(false)
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
	if got := len(m.inputFlow()); got != 3 {
		t.Fatalf("expected timezone, username, key screens, got %d", got)
	}

	m.selections = selectionState{}
	m.selections.DevTools.Node = true
	flow := m.inputFlow()
	if len(flow) != 1 || flow[0] != screenInputUser {
		t.Fatalf("expected only username flow, got %#v", flow)
	}
}

func TestBuildRunStepsDefault(t *testing.T) {
	m := InitialModel(false)
	steps := m.buildRunSteps()

	if len(steps) != 11 {
		t.Fatalf("expected 11 run steps, got %d", len(steps))
	}
	if steps[0].id != runBootstrap || steps[1].id != runAddUser || steps[2].id != runToolDeps {
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
		{stepPending, "[ ]"},
		{stepRunning, "[*]"},
		{stepOK, "[✓]"},
		{stepFail, "[✗]"},
	}

	for _, tt := range tests {
		got := statusIcon(tt.status)
		if got != tt.want {
			t.Errorf("statusIcon(%v) = %q, want %q", tt.status, got, tt.want)
		}
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
