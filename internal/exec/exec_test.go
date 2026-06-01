package exec

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

func TestDryRunnerWriteFile(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.WriteFile("/tmp/test", []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "WriteFile") {
		t.Errorf("expected WriteFile in output, got: %q", got)
	}
	if !strings.Contains(got, "/tmp/test") {
		t.Errorf("expected path in output, got: %q", got)
	}
}

func TestDryRunnerReadFile(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	data, err := runner.ReadFile("/etc/some-file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data in dry-run, got: %v", data)
	}
	if !strings.Contains(buf.String(), "ReadFile") {
		t.Errorf("expected ReadFile in output")
	}
}

func TestDryRunnerRename(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.Rename("/tmp/src", "/etc/dst")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "Rename") {
		t.Errorf("expected Rename in output")
	}
	if !strings.Contains(got, "/tmp/src") || !strings.Contains(got, "/etc/dst") {
		t.Errorf("expected both paths in output")
	}
}

func TestDryRunnerChmod(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.Chmod("/usr/local/bin/yq", 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "Chmod") {
		t.Errorf("expected Chmod in output")
	}
}

func TestDryRunnerMkdirAll(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.MkdirAll("/etc/ssh/sshd_config.d", 0755)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "MkdirAll") {
		t.Errorf("expected MkdirAll in output")
	}
}

func TestDryRunnerRemove(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.Remove("/tmp/some-temp-file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Remove") {
		t.Errorf("expected Remove in output")
	}
}

func TestDryRunnerRemoveAll(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	err := runner.RemoveAll("/usr/local/go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "RemoveAll") {
		t.Errorf("expected RemoveAll in output")
	}
}

func TestDryRunnerStat(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	_, err := runner.Stat("/some/file")
	if err == nil {
		t.Error("expected error from dry-run Stat")
	}
	if !strings.Contains(buf.String(), "Stat") {
		t.Errorf("expected Stat in output")
	}
}

func TestDryRunnerLookupUser(t *testing.T) {
	var buf bytes.Buffer
	runner := &DryRunner{Stdout: &buf}

	uid, gid, err := runner.LookupUser("root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != 0 || gid != 0 {
		t.Errorf("expected root uid=0,gid=0, got uid=%d,gid=%d", uid, gid)
	}

	buf.Reset()
	uid, gid, err = runner.LookupUser("dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != 1000 || gid != 1000 {
		t.Errorf("expected dev uid=1000,gid=1000, got uid=%d,gid=%d", uid, gid)
	}

	if !strings.Contains(buf.String(), "LookupUser") {
		t.Errorf("expected LookupUser in output")
	}
}

func TestRealRunnerLookupUser(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		runner := NewRealRunner()
		uid, gid, err := runner.LookupUser("root")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uid != 0 || gid != 0 {
			t.Errorf("expected root uid=0,gid=0, got uid=%d,gid=%d", uid, gid)
		}
	})

	t.Run("nonexistent", func(t *testing.T) {
		runner := NewRealRunner()
		_, _, err := runner.LookupUser("this-user-does-not-exist-12345")
		if err == nil {
			t.Error("expected error for nonexistent user")
		}
	})
}

func TestRealRunnerUsesSafePathForOutput(t *testing.T) {
	fakeDir := t.TempDir()
	writeExecutable(t, filepath.Join(fakeDir, "sh"), "#!/bin/sh\nprintf hijacked\n")
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	runner := NewRealRunner()
	out, err := runner.Output("sh", "-c", "printf safe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "safe" {
		t.Fatalf("expected safe command output, got %q", out)
	}
}

func TestRealRunnerUsesSafePathForShell(t *testing.T) {
	fakeDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "hijacked")
	writeExecutable(t, filepath.Join(fakeDir, "bash"), "#!/bin/sh\ntouch "+marker+"\n")
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	runner := NewRealRunner()
	if err := runner.Shell("true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expected fake bash not to run, stat err=%v", err)
	}
}

func TestRealRunnerSetsSafePathAndPreservesOtherEnv(t *testing.T) {
	fakeDir := t.TempDir()
	t.Setenv("PATH", fakeDir)
	t.Setenv("SETUP_TEST_ENV", "preserved")

	runner := NewRealRunner()
	if got := envValue(runner.Env, "PATH"); got != safePath {
		t.Fatalf("expected PATH %q, got %q", safePath, got)
	}
	if got := envValue(runner.Env, "SETUP_TEST_ENV"); got != "preserved" {
		t.Fatalf("expected SETUP_TEST_ENV to be preserved, got %q", got)
	}

	out, err := runner.Output("sh", "-c", "printf %s \"$SETUP_TEST_ENV\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "preserved" {
		t.Fatalf("expected child env to be preserved, got %q", out)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}
