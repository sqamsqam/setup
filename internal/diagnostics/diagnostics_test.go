package diagnostics

import (
	"fmt"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestFormatReport(t *testing.T) {
	got := Format(Report{{Name: "Virtualization", Status: "ok", Detail: "lxc"}})
	if !strings.Contains(got, "Virtualization: ok") || !strings.Contains(got, "  lxc") {
		t.Fatalf("unexpected report: %q", got)
	}
}

func TestUFWCheckReportsMissingBinaryClearly(t *testing.T) {
	runner := &diagnosticsRunner{ufwErr: fmt.Errorf(`executable "ufw" not found in safe PATH`)}

	got := ufwCheck(runner)

	if got.Status != "warning" {
		t.Fatalf("status = %q, want warning", got.Status)
	}
	if !strings.Contains(got.Detail, "ufw is not installed") {
		t.Fatalf("detail = %q, want clear missing UFW warning", got.Detail)
	}
	if strings.Contains(got.Detail, "safe PATH") {
		t.Fatalf("detail exposes raw command failure: %q", got.Detail)
	}
}

func TestDockerUFWCheckWarnsWhenBothActive(t *testing.T) {
	got := dockerUFWCheck(
		Check{Name: "UFW", Status: "ok", Detail: "Status: active"},
		Check{Name: "Docker service", Status: "ok", Detail: "active"},
	)

	if got.Status != "warning" {
		t.Fatalf("status = %q, want warning", got.Status)
	}
	for _, want := range []string{"Docker-published ports can bypass UFW", "DOCKER-USER"} {
		if !strings.Contains(got.Detail, want) {
			t.Fatalf("detail = %q, missing %q", got.Detail, want)
		}
	}
}

type diagnosticsRunner struct {
	*setupexec.DryRunner
	ufwErr error
}

func (r *diagnosticsRunner) Output(name string, args ...string) (string, error) {
	if name == "ufw" && r.ufwErr != nil {
		return "", r.ufwErr
	}
	if r.DryRunner == nil {
		r.DryRunner = setupexec.NewDryRunner()
	}
	return r.DryRunner.Output(name, args...)
}
