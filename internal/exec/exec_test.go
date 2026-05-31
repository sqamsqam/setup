package exec

import (
	"bytes"
	"io"
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

func TestDryRunnerIsDryRun(t *testing.T) {
	runner := &DryRunner{Stdout: io.Discard}
	if !runner.IsDryRun() {
		t.Error("expected DryRunner.IsDryRun() to return true")
	}
}

func TestDryRunnerOutput(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	out, err := runner.Output("dpkg", "--print-architecture")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "amd64" {
		t.Errorf("expected amd64, got: %q", out)
	}
	if !strings.Contains(buf.String(), "dpkg --print-architecture") {
		t.Errorf("expected command log in output")
	}

	buf.Reset()
	out, err = runner.Output("getent", "passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "user:x:1000:1000:User:/home/user:/bin/bash" {
		t.Errorf("unexpected getent output: %q", out)
	}

	buf.Reset()
	out, err = runner.Output("id", "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "uid=1000(user) gid=1000(user) groups=1000(user)" {
		t.Errorf("unexpected id output: %q", out)
	}
}

func TestDryRunnerCombinedOutput(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	out, err := runner.CombinedOutput("ls", "-la")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output, got: %q", out)
	}
	if !strings.Contains(buf.String(), "CombinedOutput: ls -la") {
		t.Errorf("expected CombinedOutput prefix in log, got: %q", buf.String())
	}
}
