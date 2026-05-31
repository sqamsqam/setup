package system

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

const (
	sshdHardeningConfig = "/etc/ssh/sshd_config.d/99-hardening.conf"
	autoUpgradesConfig  = "/etc/apt/apt.conf.d/20auto-upgrades"
	goProfileScript     = "/etc/profile.d/go.sh"
)

func Bootstrap(runner setupexec.CmdRunner, timezone string) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Setting up locale", func() error { return setupLocale(runner) }},
		{"Updating system packages", func() error { return updateSystem(runner) }},
		{"Installing base packages", func() error { return installBasePackages(runner) }},
		{"Configuring unattended upgrades", func() error { return configureUnattendedUpgrades(runner) }},
		{"Setting timezone to " + timezone, func() error { return setTimezone(runner, timezone) }},
		{"Hardening SSH", func() error { return hardenSSH(runner) }},
		{"Locking root password", func() error { return lockRootPassword(runner) }},
		{"Installing Docker", func() error { return installDocker(runner) }},
		{"Enabling and starting SSH", func() error { return startSSH(runner) }},
	}

	for _, step := range steps {
		setupexec.PrintStep(step.name)
		if err := step.fn(); err != nil {
			setupexec.PrintError(step.name)
			return fmt.Errorf("%s: %w", step.name, err)
		}
		setupexec.PrintDone(step.name)
	}

	setupexec.PrintDone("Root bootstrap complete")
	return nil
}

func setupLocale(runner setupexec.CmdRunner) error {
	if err := runner.Run("sed", "-i", `s/^# *en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/`, "/etc/locale.gen"); err != nil {
		return err
	}
	if err := runner.Run("locale-gen"); err != nil {
		return err
	}
	return runner.Run("update-locale", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
}

func updateSystem(runner setupexec.CmdRunner) error {
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "full-upgrade", "-y")
}

func installBasePackages(runner setupexec.CmdRunner) error {
	return runner.Run("apt", "install", "-y",
		"sudo", "openssh-server", "curl", "wget", "git", "zip", "unzip",
		"htop", "jq", "mosh", "tmux", "gpg", "vim",
		"ca-certificates", "unattended-upgrades", "systemd",
	)
}

func configureUnattendedUpgrades(runner setupexec.CmdRunner) error {
	content := "# Managed by setup — do not edit\n" + strings.TrimSpace(`
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
`) + "\n"

	oldContent, _ := runner.ReadFile(autoUpgradesConfig)
	if bytes.Equal(oldContent, []byte(content)) {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "setup-auto-upgrades-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer runner.Remove(tmpPath)

	if err := runner.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return err
	}
	return runner.Rename(tmpPath, autoUpgradesConfig)
}

func setTimezone(runner setupexec.CmdRunner, tz string) error {
	return runner.Run("timedatectl", "set-timezone", tz)
}

func hardenSSH(runner setupexec.CmdRunner) error {
	content := `# Managed by setup — do not edit
PermitRootLogin no
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
MaxAuthTries 3
LoginGraceTime 30
X11Forwarding no
ClientAliveInterval 300
ClientAliveCountMax 2
MaxSessions 10
MaxStartups 10:30:100
`
	tmpFile, err := os.CreateTemp("", "setup-99-hardening-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer runner.Remove(tmpPath)

	if err := runner.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return err
	}

	if err := runner.MkdirAll("/etc/ssh/sshd_config.d", 0755); err != nil {
		return err
	}

	oldContent, _ := runner.ReadFile(sshdHardeningConfig)
	newContent := []byte(content)

	// If content unchanged and current config is valid, skip
	if string(oldContent) == string(newContent) && sshdConfigValid(runner) {
		return nil
	}

	// Validate the new config against the temp file before installing
	if err := runner.Run("sshd", "-t", "-f", tmpPath); err != nil {
		return fmt.Errorf("sshd configuration test failed — new hardening config rejected, SSH not restarted")
	}

	if err := runner.Rename(tmpPath, sshdHardeningConfig); err != nil {
		return err
	}

	return runner.Run("systemctl", "restart", "ssh")
}

func sshdConfigValid(runner setupexec.CmdRunner) bool {
	err := runner.Run("sshd", "-t")
	return err == nil
}

func lockRootPassword(runner setupexec.CmdRunner) error {
	return runner.Run("passwd", "-l", "root")
}

func installDocker(runner setupexec.CmdRunner) error {
	return runner.Shell("set -euo pipefail; curl -fsSL https://get.docker.com | sh")
}

func startSSH(runner setupexec.CmdRunner) error {
	return runner.Run("systemctl", "enable", "--now", "ssh")
}
