package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestVersionCommand(t *testing.T) {
	SetVersion("test-version-1.0")
	app := BuildApp(false, nil)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	err := app.Run(context.Background(), []string{"setup", "version"})
	if err != nil {
		t.Fatal(err)
	}

	_ = w.Close()
	os.Stdout = old
	<-done

	if !strings.Contains(buf.String(), "test-version-1.0") {
		t.Errorf("expected version in output, got: %s", buf.String())
	}
}

func TestVersionFlag(t *testing.T) {
	SetVersion("test-version-1.0")
	app := BuildApp(false, nil)

	var buf bytes.Buffer
	app.Writer = &buf

	err := app.Run(context.Background(), []string{"setup", "--version"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "test-version-1.0") {
		t.Errorf("expected version in output, got: %s", buf.String())
	}
}

func TestHelpOutput(t *testing.T) {
	app := BuildApp(false, nil)

	var buf bytes.Buffer
	app.Writer = &buf

	err := app.Run(context.Background(), []string{"setup", "--help"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "USAGE:") {
		t.Errorf("expected help output, got: %s", buf.String())
	}
}

func TestRunBootstrap(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{"setup", "bootstrap"})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
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
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{"setup", "bootstrap", "--timezone=America/New_York"})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "America/New_York") {
		t.Errorf("expected America/New_York in output, got: %s", output)
	}
}

func TestRunBootstrapWithTimezoneSpaceFlag(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{"setup", "bootstrap", "--timezone", "America/New_York"})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "America/New_York") {
		t.Errorf("expected America/New_York in output, got: %s", output)
	}
}

func TestRunBootstrapWithShortFlag(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{"setup", "bootstrap", "-t", "Europe/London"})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "Europe/London") {
		t.Errorf("expected Europe/London in output, got: %s", output)
	}
}

func TestRunAddUser(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{
		"setup", "add-user",
		"--user", "testuser",
		"--key", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo=",
	})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
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

func TestRunAddUserShortFlags(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{
		"setup", "add-user",
		"-u", "shorty",
		"-k", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo=",
	})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "shorty") {
		t.Errorf("expected shorty in output, got: %s", output)
	}
}

func TestRunDryRunFull(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{
		"setup", "full",
		"--user", "test",
		"--key", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo=",
	})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run output")
	}
	if !strings.Contains(output, "apt update") {
		t.Error("expected apt update in output")
	}
}

func TestRunAddUserMissingFlags(t *testing.T) {
	app := BuildApp(false, nil)

	err := app.Run(context.Background(), []string{"setup", "add-user"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRunAddUserMissingKey(t *testing.T) {
	app := BuildApp(false, nil)

	err := app.Run(context.Background(), []string{"setup", "add-user", "--user", "test"})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	app := BuildApp(false, nil)

	err := app.Run(context.Background(), []string{"setup", "unknown_cmd"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRunBootstrapUnknownFlag(t *testing.T) {
	app := BuildApp(false, nil)

	err := app.Run(context.Background(), []string{"setup", "bootstrap", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestInstallTools(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}

	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })

	err := app.Run(context.Background(), []string{"setup", "install-tools"})
	if err != nil {
		t.Fatal(err)
	}

	output := dryBuf.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run output")
	}
}

func TestDevToolsMissingUser(t *testing.T) {
	app := BuildApp(false, nil)

	err := app.Run(context.Background(), []string{"setup", "devtools"})
	if err == nil {
		t.Fatal("error expected for missing --user")
	}
}
