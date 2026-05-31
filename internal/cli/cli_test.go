package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestVersion(t *testing.T) {
	SetVersion("test-version-1.0")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	Run([]string{"version"})

	w.Close()
	os.Stdout = old
	<-done

	if !strings.Contains(buf.String(), "test-version-1.0") {
		t.Errorf("expected version in output, got: %s", buf.String())
	}
}

func TestHelpOutput(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	Run([]string{"--help"})

	w.Close()
	os.Stdout = old
	<-done

	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("expected help output, got: %s", buf.String())
	}
}

func TestRunBootstrap(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	runBootstrap(runner, nil)

	output := buf.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run log output")
	}
	if !strings.Contains(output, "apt update") {
		t.Error("expected apt update in output")
	}
	if !strings.Contains(output, "locale-gen") {
		t.Error("expected locale-gen in output")
	}
}

func TestRunBootstrapWithTimezoneEqualsFlag(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	runBootstrap(runner, []string{"--timezone=America/New_York"})

	output := buf.String()
	if !strings.Contains(output, "America/New_York") {
		t.Errorf("expected America/New_York in output, got: %s", output)
	}
}

func TestRunBootstrapWithTimezoneSpaceFlag(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	runBootstrap(runner, []string{"--timezone", "America/New_York"})

	output := buf.String()
	if !strings.Contains(output, "America/New_York") {
		t.Errorf("expected America/New_York in output, got: %s", output)
	}
}

func TestRunAddUser(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	runAddUser(runner, []string{"--user", "testuser", "--key", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo="})

	output := buf.String()
	if !strings.Contains(output, "testuser") {
		t.Errorf("expected testuser in output, got: %s", output)
	}
	if !strings.Contains(output, "loginctl enable-linger") {
		t.Errorf("expected loginctl enable-linger in output, got: %s", output)
	}
	if !strings.Contains(output, "usermod -aG sudo") {
		t.Errorf("expected usermod in output, got: %s", output)
	}
}

func TestRunDryRunFull(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	runFull(runner, []string{"--user", "test", "--key", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo="})

	output := buf.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run output")
	}
	if !strings.Contains(output, "apt update") {
		t.Error("expected apt update in output")
	}
}

func TestRunAddUserMissingFlags(t *testing.T) {
	if os.Getenv("CLI_TEST_SUBPROCESS") == "1" {
		SetVersion("test")
		Run([]string{"add-user"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRunAddUserMissingFlags")
	cmd.Env = append(os.Environ(), "CLI_TEST_SUBPROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit code 1 for missing flags")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if os.Getenv("CLI_TEST_SUBPROCESS") == "1" {
		SetVersion("test")
		Run([]string{"unknown_cmd"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRunUnknownCommand")
	cmd.Env = append(os.Environ(), "CLI_TEST_SUBPROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit code 1 for unknown command")
	}
}

func TestRunBootstrapUnknownFlag(t *testing.T) {
	if os.Getenv("CLI_TEST_SUBPROCESS") == "1" {
		var buf bytes.Buffer
		runner := &setupexec.DryRunner{Stdout: &buf}
		runBootstrap(runner, []string{"--unknown-flag"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRunBootstrapUnknownFlag")
	cmd.Env = append(os.Environ(), "CLI_TEST_SUBPROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit code 1 for unknown flag")
	}
}
