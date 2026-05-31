package tui

import (
	"strings"
	"testing"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(false)

	if m.dryRun {
		t.Error("expected dryRun to be false")
	}
	if m.timezone != "UTC" {
		t.Errorf("expected default timezone UTC, got: %s", m.timezone)
	}
	if m.screen != screenWelcome {
		t.Errorf("expected screenWelcome, got: %d", m.screen)
	}
	if len(m.steps) != 4 {
		t.Errorf("expected 4 steps, got: %d", len(m.steps))
	}
	if len(m.stepFlags) != 4 {
		t.Errorf("expected 4 step flags, got: %d", len(m.stepFlags))
	}
	if m.stepFlags[0] != true {
		t.Errorf("expected step 0 to be selected")
	}
}

func TestInitialModelDryRun(t *testing.T) {
	m := InitialModel(true)
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestSelectedStepsAllFalse(t *testing.T) {
	m := InitialModel(false)
	m.stepFlags = []bool{false, false, false, false}

	sel := m.selectedSteps()
	if len(sel) != 0 {
		t.Errorf("expected 0 selected steps, got: %d", len(sel))
	}
}

func TestSelectedStepsAllTrue(t *testing.T) {
	m := InitialModel(false)
	m.stepFlags = []bool{true, true, true, true}

	sel := m.selectedSteps()
	if len(sel) != 4 {
		t.Errorf("expected 4 selected steps, got: %d", len(sel))
	}
	for i, idx := range sel {
		if idx != i {
			t.Errorf("expected step %d at position %d, got %d", i, i, idx)
		}
	}
}

func TestSelectedStepsMixed(t *testing.T) {
	m := InitialModel(false)
	m.stepFlags = []bool{true, false, true, false}

	sel := m.selectedSteps()
	if len(sel) != 2 {
		t.Errorf("expected 2 selected steps, got: %d", len(sel))
	}
	if sel[0] != 0 || sel[1] != 2 {
		t.Errorf("expected steps [0, 2], got: %v", sel)
	}
}

func TestNeedsUserInput(t *testing.T) {
	tests := []struct {
		flags []bool
		want  bool
	}{
		{[]bool{true, false, false, false}, false},
		{[]bool{false, true, false, false}, true},
		{[]bool{false, false, false, true}, true},
		{[]bool{false, false, false, false}, false},
		{[]bool{true, true, true, true}, true},
	}

	for _, tt := range tests {
		m := InitialModel(false)
		m.stepFlags = tt.flags
		got := m.needsUserInput()
		if got != tt.want {
			t.Errorf("needsUserInput(%v) = %v, want %v", tt.flags, got, tt.want)
		}
	}
}

func TestNeedsKeyInput(t *testing.T) {
	tests := []struct {
		flags []bool
		want  bool
	}{
		{[]bool{true, false, false, false}, false},
		{[]bool{false, true, false, false}, true},
		{[]bool{false, true, true, true}, true},
		{[]bool{false, false, false, false}, false},
	}

	for _, tt := range tests {
		m := InitialModel(false)
		m.stepFlags = tt.flags
		got := m.needsKeyInput()
		if got != tt.want {
			t.Errorf("needsKeyInput(%v) = %v, want %v", tt.flags, got, tt.want)
		}
	}
}

func TestNeedsTimezoneInput(t *testing.T) {
	tests := []struct {
		flags []bool
		want  bool
	}{
		{[]bool{true, false, false, false}, true},
		{[]bool{false, true, false, false}, false},
		{[]bool{false, false, false, false}, false},
	}

	for _, tt := range tests {
		m := InitialModel(false)
		m.stepFlags = tt.flags
		got := m.needsTimezoneInput()
		if got != tt.want {
			t.Errorf("needsTimezoneInput(%v) = %v, want %v", tt.flags, got, tt.want)
		}
	}
}

func TestHasSelections(t *testing.T) {
	tests := []struct {
		flags []bool
		want  bool
	}{
		{[]bool{false, false, false, false}, false},
		{[]bool{true, false, false, false}, true},
		{[]bool{false, true, false, false}, true},
		{[]bool{true, true, true, true}, true},
	}

	for _, tt := range tests {
		m := InitialModel(false)
		m.stepFlags = tt.flags
		got := m.hasSelections()
		if got != tt.want {
			t.Errorf("hasSelections(%v) = %v, want %v", tt.flags, got, tt.want)
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
	if got != long[:40] {
		t.Errorf("expected first 40 chars, got: %q", got)
	}
}

func TestTruncateKeyTrimmed(t *testing.T) {
	withSpace := "  key-content  "
	got := truncateKey(withSpace, 40)
	if got != "key-content" {
		t.Errorf("expected trimmed key, got: %q", got)
	}
}

func TestResetSteps(t *testing.T) {
	m := InitialModel(false)

	m.steps[0].status = stepOK
	m.steps[0].output = "done"
	m.steps[1].status = stepFail
	m.steps[1].output = "error"

	m.resetSteps()

	for i, s := range m.steps {
		if s.status != stepPending {
			t.Errorf("step %d status expected pending, got: %v", i, s.status)
		}
		if s.output != "" {
			t.Errorf("step %d output expected empty, got: %q", i, s.output)
		}
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


