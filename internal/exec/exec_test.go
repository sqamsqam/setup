package exec

import (
	"bytes"
	"strings"
	"testing"
)

func TestDryRunnerRun(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.Run("apt", "update")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "[DRY-RUN]") {
		t.Errorf("expected [DRY-RUN] prefix, got: %q", got)
	}
	if !strings.Contains(got, "apt update") {
		t.Errorf("expected command in output, got: %q", got)
	}
}

func TestDryRunnerRunAsUser(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.RunAsUser("dev", "bash", "-c", "echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "sudo") {
		t.Errorf("expected sudo in output, got: %q", got)
	}
	if !strings.Contains(got, "dev") {
		t.Errorf("expected username in output, got: %q", got)
	}
}

func TestDryRunnerShell(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.Shell("curl -fsSL https://get.docker.com | sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "[DRY-RUN]") {
		t.Errorf("expected [DRY-RUN] prefix, got: %q", got)
	}
	if !strings.Contains(got, "bash -c") {
		t.Errorf("expected bash -c in output, got: %q", got)
	}
}
