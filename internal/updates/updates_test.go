package updates

import (
	"errors"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type updatesTestRunner struct {
	*setupexec.DryRunner
	ops       []string
	outputs   map[string]string
	runErr    map[string]error
	outputErr map[string]error
}

func newUpdatesTestRunner() *updatesTestRunner {
	return &updatesTestRunner{
		DryRunner: setupexec.NewDryRunner(),
		outputs:   make(map[string]string),
		runErr:    make(map[string]error),
		outputErr: make(map[string]error),
	}
}

func (r *updatesTestRunner) Run(name string, args ...string) error {
	key := commandKey(name, args...)
	r.ops = append(r.ops, "run:"+key)
	if err := r.runErr[key]; err != nil {
		return err
	}
	return nil
}

func (r *updatesTestRunner) Output(name string, args ...string) (string, error) {
	key := commandKey(name, args...)
	r.ops = append(r.ops, "output:"+key)
	if err := r.outputErr[key]; err != nil {
		return "", err
	}
	return r.outputs[key], nil
}

func TestCheckReturnsDefaultMessageWhenNoUpgrades(t *testing.T) {
	runner := newUpdatesTestRunner()

	got, err := Check(runner)
	if err != nil {
		t.Fatal(err)
	}
	if got != "No upgradable packages reported." {
		t.Fatalf("Check = %q", got)
	}
	if !containsUpdateOp(runner.ops, "run:apt update") {
		t.Fatalf("expected apt update, got %v", runner.ops)
	}
}

func TestCheckReturnsUpgradableOutput(t *testing.T) {
	runner := newUpdatesTestRunner()
	key := commandKey("bash", "-c", `apt list --upgradable 2>/dev/null | sed -n '2,80p'`)
	runner.outputs[key] = "pkg/resolute 1 amd64 [upgradable]\n"

	got, err := Check(runner)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "pkg/resolute") {
		t.Fatalf("Check output = %q", got)
	}
}

func TestCheckPropagatesAptUpdateError(t *testing.T) {
	runner := newUpdatesTestRunner()
	runner.runErr["apt update"] = errors.New("apt locked")

	_, err := Check(runner)
	if err == nil {
		t.Fatal("expected apt update error")
	}
}

func TestUpgradeRunsFullUpgrade(t *testing.T) {
	runner := newUpdatesTestRunner()

	if err := Upgrade(runner); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"run:apt update", "run:apt full-upgrade -y"} {
		if !containsUpdateOp(runner.ops, want) {
			t.Fatalf("missing %q from %v", want, runner.ops)
		}
	}
}

func TestRebootRequiresConfirmation(t *testing.T) {
	runner := newUpdatesTestRunner()

	if err := Reboot(runner, false); err == nil {
		t.Fatal("expected confirmation error")
	}
	if err := Reboot(runner, true); err != nil {
		t.Fatal(err)
	}
	if !containsUpdateOp(runner.ops, "run:systemctl reboot") {
		t.Fatalf("expected reboot command, got %v", runner.ops)
	}
}

func TestStatusHelpersUseExpectedCommands(t *testing.T) {
	runner := newUpdatesTestRunner()
	runner.outputs[commandKey("bash", "-c", `if test -f /var/run/reboot-required; then cat /var/run/reboot-required; if test -f /var/run/reboot-required.pkgs; then echo; cat /var/run/reboot-required.pkgs; fi; else echo "Reboot not required."; fi`)] = "Reboot not required."
	runner.outputs[commandKey("systemctl", "status", "unattended-upgrades", "--no-pager")] = "active"
	runner.outputs[commandKey("systemctl", "--failed", "--no-pager", "--plain")] = "0 loaded units listed."

	if got, err := RebootRequired(runner); err != nil || got == "" {
		t.Fatalf("RebootRequired = %q, %v", got, err)
	}
	if got, err := UnattendedStatus(runner); err != nil || got != "active" {
		t.Fatalf("UnattendedStatus = %q, %v", got, err)
	}
	if got, err := FailedUnits(runner); err != nil || !strings.Contains(got, "0 loaded") {
		t.Fatalf("FailedUnits = %q, %v", got, err)
	}
}

func commandKey(name string, args ...string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func containsUpdateOp(ops []string, want string) bool {
	for _, op := range ops {
		if op == want {
			return true
		}
	}
	return false
}
