package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(false)

	if m.dryRun {
		t.Error("expected dryRun to be false")
	}
	if m.timezone != "UTC" {
		t.Errorf("expected default timezone UTC, got: %s", m.timezone)
	}
	if m.screen != screenMainMenu {
		t.Errorf("expected screenMainMenu, got: %d", m.screen)
	}
	if m.chainIdx != -1 {
		t.Errorf("expected chainIdx -1, got: %d", m.chainIdx)
	}
	if len(m.menuItems) != 5 {
		t.Errorf("expected 5 menu items, got: %d", len(m.menuItems))
	}
	if m.menuItems[0].label != "Full Setup" {
		t.Errorf("expected first menu item 'Full Setup', got: %q", m.menuItems[0].label)
	}
	if m.menuItems[0].action != actionFullSetup {
		t.Errorf("expected first menu item action actionFullSetup, got: %d", m.menuItems[0].action)
	}
}

func TestInitialModelDryRun(t *testing.T) {
	m := InitialModel(true)
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestEffectiveActionStandalone(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	m.chainIdx = -1

	got := m.effectiveAction()
	if got != actionBootstrap {
		t.Errorf("expected actionBootstrap, got %d", got)
	}
}

func TestEffectiveActionChain(t *testing.T) {
	m := InitialModel(false)
	m.action = actionFullSetup
	m.chainIdx = 1

	got := m.effectiveAction()
	if got != actionAddUser {
		t.Errorf("expected actionAddUser for chainIdx 1, got %d", got)
	}

	m.chainIdx = 3
	got = m.effectiveAction()
	if got != actionInstallDevTools {
		t.Errorf("expected actionInstallDevTools for chainIdx 3, got %d", got)
	}
}

func TestIsChain(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	if m.isChain() {
		t.Error("standalone action should not be a chain")
	}

	m.action = actionFullSetup
	if !m.isChain() {
		t.Error("actionFullSetup should be a chain")
	}
}

func TestActionFlowScreenCounts(t *testing.T) {
	tests := []struct {
		action   action
		expected int
	}{
		{actionBootstrap, 2},       // timezone + confirm
		{actionAddUser, 3},         // user + key + confirm
		{actionInstallTools, 1},    // confirm only
		{actionInstallDevTools, 2}, // user + confirm
	}

	for _, tt := range tests {
		flow := actionFlow[tt.action]
		if len(flow) != tt.expected {
			t.Errorf("action %d expected %d screens, got %d", tt.action, tt.expected, len(flow))
		}
		// Last screen should always be confirm
		if flow[len(flow)-1] != screenConfirm {
			t.Errorf("action %d last screen should be confirm, got %d", tt.action, flow[len(flow)-1])
		}
	}
}

func TestGoNext(t *testing.T) {
	m := InitialModel(false)
	m.action = actionAddUser
	m.chainIdx = -1
	m.flowPos = 0
	m.screen = m.currentFlow()[0]

	if m.screen != screenInputUser {
		t.Errorf("expected screenInputUser at flowPos 0, got %d", m.screen)
	}

	m.goNext()
	if m.flowPos != 1 {
		t.Errorf("expected flowPos 1 after goNext, got %d", m.flowPos)
	}
	if m.screen != screenInputKey {
		t.Errorf("expected screenInputKey after goNext, got %d", m.screen)
	}

	m.goNext()
	if m.flowPos != 2 {
		t.Errorf("expected flowPos 2 after goNext, got %d", m.flowPos)
	}
	if m.screen != screenConfirm {
		t.Errorf("expected screenConfirm after goNext, got %d", m.screen)
	}
}

func TestGoBack(t *testing.T) {
	m := InitialModel(false)
	m.action = actionAddUser
	m.chainIdx = -1
	m.flowPos = 2
	m.screen = screenConfirm

	m.goBack()
	if m.flowPos != 1 {
		t.Errorf("expected flowPos 1 after goBack, got %d", m.flowPos)
	}
	if m.screen != screenInputKey {
		t.Errorf("expected screenInputKey after goBack, got %d", m.screen)
	}

	m.goBack()
	if m.flowPos != 0 {
		t.Errorf("expected flowPos 0 after goBack, got %d", m.flowPos)
	}
	if m.screen != screenInputUser {
		t.Errorf("expected screenInputUser after goBack, got %d", m.screen)
	}

	// Going back from first screen returns to menu
	m.goBack()
	if m.screen != screenMainMenu {
		t.Errorf("expected screenMainMenu after goBack from first screen, got %d", m.screen)
	}
}

func TestResetToMenu(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenDone
	m.action = actionAddUser
	m.chainIdx = 0
	m.username = "testuser"
	m.sshKey = "ssh-ed25519 test"
	m.timezone = "Europe/Paris"
	m.steps = []step{{name: "test", status: stepOK}}

	m.resetToMenu()

	if m.screen != screenMainMenu {
		t.Errorf("expected screenMainMenu, got %d", m.screen)
	}
	if m.menuCursor != 0 {
		t.Errorf("expected menuCursor 0, got %d", m.menuCursor)
	}
	if m.username != "" {
		t.Errorf("expected empty username, got %q", m.username)
	}
	if m.sshKey != "" {
		t.Errorf("expected empty sshKey, got %q", m.sshKey)
	}
	if m.timezone != "UTC" {
		t.Errorf("expected UTC timezone, got %q", m.timezone)
	}
	if m.chainIdx != -1 {
		t.Errorf("expected chainIdx -1, got %d", m.chainIdx)
	}
	if m.steps != nil {
		t.Errorf("expected nil steps, got %v", m.steps)
	}
}

func TestBuildStepsChain(t *testing.T) {
	m := InitialModel(false)
	m.action = actionFullSetup
	m.chainIdx = 0

	m.buildSteps()

	if len(m.steps) != 4 {
		t.Errorf("expected 4 steps for full setup chain, got %d", len(m.steps))
	}
	if m.steps[0].status != stepPending {
		t.Errorf("expected step 0 pending, got %d", m.steps[0].status)
	}
}

func TestBuildStepsStandalone(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	m.chainIdx = -1

	m.buildSteps()

	if len(m.steps) != 1 {
		t.Errorf("expected 1 step for standalone, got %d", len(m.steps))
	}
	if m.steps[0].status != stepPending {
		t.Errorf("expected step 0 pending, got %d", m.steps[0].status)
	}
}

func TestRunningStepIndex(t *testing.T) {
	m := InitialModel(false)

	m.action = actionBootstrap
	m.chainIdx = -1
	if m.runningStepIndex() != 0 {
		t.Errorf("standalone should return 0, got %d", m.runningStepIndex())
	}

	m.action = actionFullSetup
	m.chainIdx = 2
	if m.runningStepIndex() != 2 {
		t.Errorf("chainIdx 2 should return 2, got %d", m.runningStepIndex())
	}
}

func TestChainProgress(t *testing.T) {
	m := InitialModel(false)
	m.action = actionFullSetup
	m.chainIdx = 0
	m.buildSteps()

	if m.chainProgress() != 0 {
		t.Errorf("expected 0%% progress, got %d", m.chainProgress())
	}

	m.steps[0].status = stepOK
	if m.chainProgress() != 25 {
		t.Errorf("expected 25%% progress, got %d", m.chainProgress())
	}

	m.steps[1].status = stepOK
	m.steps[2].status = stepOK
	if m.chainProgress() != 75 {
		t.Errorf("expected 75%% progress, got %d", m.chainProgress())
	}

	m.steps[3].status = stepOK
	if m.chainProgress() != 100 {
		t.Errorf("expected 100%% progress, got %d", m.chainProgress())
	}
}

func TestChainProgressStandalone(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	m.chainIdx = -1

	if m.chainProgress() != 0 {
		t.Errorf("standalone should have 0 progress, got %d", m.chainProgress())
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

func TestTruncateKeyTrimmed(t *testing.T) {
	withSpace := "  key-content  "
	got := truncateKey(withSpace, 40)
	if got != "key-content" {
		t.Errorf("expected trimmed key, got: %q", got)
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status stepStatus
		want   string
	}{
		{stepPending, "  "},
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

func TestPasteSSHKey(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputKey
	msg := tea.PasteMsg{Content: "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo= pasted@example\n"}

	updated, _ := m.Update(msg)
	got := updated.(model)
	if got.sshKey != "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo= pasted@example" {
		t.Fatalf("unexpected pasted key: %q", got.sshKey)
	}
}

func TestEmptyUsernameShowsError(t *testing.T) {
	m := InitialModel(false)
	m.screen = screenInputUser

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.usernameErr == "" {
		t.Fatal("expected username error")
	}
	if got.screen != screenInputUser {
		t.Fatalf("expected to stay on username screen, got %d", got.screen)
	}
}

func TestBlankTimezoneDefaultsToUTC(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	m.screen = screenInputTimezone
	m.timezone = ""

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.timezone != "UTC" {
		t.Fatalf("expected UTC, got %q", got.timezone)
	}
	if got.screen != screenConfirm {
		t.Fatalf("expected confirm screen, got %d", got.screen)
	}
}

func TestFailedChainStepRetriesInsteadOfAdvancing(t *testing.T) {
	m := InitialModel(false)
	m.action = actionFullSetup
	m.chainIdx = 0
	m.screen = screenDone
	m.buildSteps()
	m.steps[0].status = stepFail

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	got := updated.(model)
	if got.chainIdx != 0 {
		t.Fatalf("expected chain index to stay at 0, got %d", got.chainIdx)
	}
	if got.screen != screenRunning {
		t.Fatalf("expected retry to enter running screen, got %d", got.screen)
	}
}

func TestSpinnerChar(t *testing.T) {
	frames := len(spinnerFrames)
	for i := 0; i < frames*3; i++ {
		got := spinnerChar(i)
		expected := spinnerFrames[i%frames]
		if got != expected {
			t.Errorf("spinnerChar(%d) = %q, want %q", i, got, expected)
		}
	}
}

func TestDrawProgressBar(t *testing.T) {
	tests := []struct {
		pct   int
		width int
	}{
		{0, 30},
		{25, 30},
		{50, 30},
		{75, 30},
		{100, 30},
	}

	for _, tt := range tests {
		got := drawProgressBar(tt.pct, tt.width)
		if len(got) == 0 {
			t.Errorf("drawProgressBar(%d, %d) returned empty", tt.pct, tt.width)
		}
		if !strings.Contains(got, "%") {
			t.Errorf("drawProgressBar(%d, %d) missing %% sign: %q", tt.pct, tt.width, got)
		}
	}
}

func TestActionLabel(t *testing.T) {
	m := InitialModel(false)
	m.action = actionBootstrap
	m.chainIdx = -1

	if got := m.actionLabel(); got != "System Bootstrap" {
		t.Errorf("expected 'System Bootstrap', got %q", got)
	}

	m.action = actionAddUser
	if got := m.actionLabel(); got != "Add User" {
		t.Errorf("expected 'Add User', got %q", got)
	}
}
