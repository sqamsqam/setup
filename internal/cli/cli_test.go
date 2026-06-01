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

func TestDefaultRunnerPreservesSafePathWhenAddingAptEnv(t *testing.T) {
	t.Setenv("PATH", "/tmp/attacker")

	runner := defaultRunner(false)
	real, ok := runner.(*setupexec.RealRunner)
	if !ok {
		t.Fatalf("defaultRunner(false) = %T, want *exec.RealRunner", runner)
	}

	if got := envValue(real.Env, "PATH"); strings.HasPrefix(got, "/tmp/attacker") {
		t.Fatalf("default runner preserved unsafe PATH %q", got)
	}
	if got := envValue(real.Env, "DEBIAN_FRONTEND"); got != "noninteractive" {
		t.Fatalf("DEBIAN_FRONTEND = %q, want noninteractive", got)
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
	if !strings.Contains(buf.String(), "dry-run") {
		t.Errorf("expected global dry-run flag in help, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "demo") {
		t.Errorf("expected global demo flag in help, got: %s", buf.String())
	}
}

func TestDefaultRunnerForDemoMode(t *testing.T) {
	runner := defaultRunnerForMode(true, true)
	if !setupexec.IsDryRun(runner) {
		t.Fatal("expected demo runner to be dry-run safe")
	}
	if !setupexec.IsDemo(runner) {
		t.Fatal("expected demo runner")
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

func TestRunFirewallAllow(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}
	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })
	err := app.Run(context.Background(), []string{
		"setup", "firewall", "allow",
		"--port", "443",
		"--proto", "tcp",
		"--from", "10.0.0.0/24",
		"--comment", "web",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(dryBuf.String(), "ufw allow from 10.0.0.0/24 to any port 443 proto tcp comment web") {
		t.Fatalf("unexpected output: %s", dryBuf.String())
	}
}

func TestRunDockerLogsConfig(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}
	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })
	err := app.Run(context.Background(), []string{"setup", "docker", "logs-config"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dryBuf.String(), "systemctl restart docker") {
		t.Fatalf("expected docker restart in output: %s", dryBuf.String())
	}
}

func TestRunDevToolsPnpmInstallsNodeFirst(t *testing.T) {
	var dryBuf bytes.Buffer
	dryRunner := &setupexec.DryRunner{Stdout: &dryBuf}
	setupexec.SetPrintWriter(io.Discard)

	app := BuildApp(false, func(bool) setupexec.CmdRunner { return dryRunner })
	err := app.Run(context.Background(), []string{"setup", "devtools", "--user", "dev", "--pnpm"})
	if err != nil {
		t.Fatal(err)
	}
	output := dryBuf.String()
	if !strings.Contains(output, "fnm") || !strings.Contains(output, "pnpm") {
		t.Fatalf("expected node and pnpm commands in output: %s", output)
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

func TestRunAddUserRejectsKeyAndKeyFile(t *testing.T) {
	keyFile, err := os.CreateTemp(t.TempDir(), "key-*.pub")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := keyFile.WriteString("ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo="); err != nil {
		t.Fatal(err)
	}
	if err := keyFile.Close(); err != nil {
		t.Fatal(err)
	}

	app := BuildApp(false, nil)
	err = app.Run(context.Background(), []string{
		"setup", "add-user",
		"--user", "test",
		"--key", "ssh-ed25519 /B9dB00GY0f13kc2Y0uRBWRC6xXQDQUknL0Jkj1HxEo=",
		"--key-file", keyFile.Name(),
	})
	if err == nil {
		t.Fatal("expected error for conflicting key inputs")
	}
	if !strings.Contains(err.Error(), "either --key or --key-file") {
		t.Fatalf("unexpected error: %v", err)
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
