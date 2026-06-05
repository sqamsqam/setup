package system

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dockermaint "github.com/sqamsqam/setup/internal/docker"
	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
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
		{"Configuring Docker log rotation", func() error { return dockermaint.ConfigureLogRotation(runner, dockermaint.DefaultLogRotationOptions()) }},
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

	oldContent, err := runner.ReadFile(autoUpgradesConfig)
	if err == nil {
		if bytes.Equal(oldContent, []byte(content)) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	_, err = managed.WriteManagedFileIfChanged(runner, autoUpgradesConfig, []byte(content), 0644)
	return err
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
	if err := runner.MkdirAll("/etc/ssh/sshd_config.d", 0755); err != nil {
		return err
	}

	newContent := []byte(content)
	oldContent, readErr := runner.ReadFile(sshdHardeningConfig)

	// If content unchanged and current config is valid, skip
	if readErr == nil {
		if bytes.Equal(oldContent, newContent) && sshdConfigValid(runner) {
			return nil
		}
		if !managed.IsMarked(oldContent) && (!setupexec.IsDryRun(runner) || len(oldContent) != 0) {
			return fmt.Errorf("refusing to replace unmanaged SSH hardening file %s", sshdHardeningConfig)
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}

	if err := installSSHDropIn(runner, sshdHardeningConfig, newContent); err != nil {
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
	keyringPath := "/etc/apt/keyrings/docker.asc"
	sourcePath := "/etc/apt/sources.list.d/docker.sources"

	if err := runner.MkdirAll("/etc/apt/keyrings", 0755); err != nil {
		return err
	}

	keyTmp, err := runner.CreateTemp("/etc/apt/keyrings", ".setup-docker-*.asc")
	if err != nil {
		return fmt.Errorf("create temp Docker keyring: %w", err)
	}
	defer func() { _ = runner.Remove(keyTmp) }()

	if err := runner.Run("curl", "-fsSL", "https://download.docker.com/linux/ubuntu/gpg", "-o", keyTmp); err != nil {
		return fmt.Errorf("download Docker GPG key: %w", err)
	}
	if !setupexec.IsDryRun(runner) {
		const dockerFingerprint = "9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
		if err := verifyGPGFingerprint(runner, keyTmp, dockerFingerprint); err != nil {
			return fmt.Errorf("verify Docker GPG key: %w", err)
		}
	}
	if err := runner.Chmod(keyTmp, 0644); err != nil {
		return err
	}
	if err := runner.Rename(keyTmp, keyringPath); err != nil {
		return err
	}

	arch, err := runner.Output("dpkg", "--print-architecture")
	if err != nil {
		return fmt.Errorf("get architecture: %w", err)
	}
	codename, err := runner.Output("bash", "-c", `. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}"`)
	if err != nil {
		return fmt.Errorf("get Ubuntu codename: %w", err)
	}
	if strings.TrimSpace(codename) == "" {
		return fmt.Errorf("ubuntu codename is empty")
	}

	sourceContent := fmt.Sprintf(`Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: %s
Components: stable
Architectures: %s
Signed-By: %s
`, strings.TrimSpace(codename), strings.TrimSpace(arch), keyringPath)

	if err := atomicWriteFile(runner, sourcePath, []byte(sourceContent), 0644); err != nil {
		return fmt.Errorf("write Docker apt source: %w", err)
	}
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin")
}

func startSSH(runner setupexec.CmdRunner) error {
	return runner.Run("systemctl", "enable", "--now", "ssh")
}

func atomicWriteFile(runner setupexec.CmdRunner, path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := runner.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpPath, err := runner.CreateTemp(dir, ".setup-*")
	if err != nil {
		return err
	}
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	if err := runner.Chmod(tmpPath, perm); err != nil {
		return err
	}
	return runner.Rename(tmpPath, path)
}

func installSSHDropIn(runner setupexec.CmdRunner, path string, data []byte) error {
	oldContent, readErr := runner.ReadFile(path)
	hadOld := readErr == nil
	if readErr == nil {
		if !managed.IsMarked(oldContent) && (!setupexec.IsDryRun(runner) || len(oldContent) != 0) {
			return fmt.Errorf("refusing to replace unmanaged SSH drop-in %s", path)
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}

	if err := atomicWriteFile(runner, path, data, 0644); err != nil {
		return err
	}
	if err := runner.Run("sshd", "-t"); err != nil {
		if rollbackErr := rollbackFile(runner, path, oldContent, hadOld, 0644); rollbackErr != nil {
			return fmt.Errorf("sshd configuration test failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
		}
		return fmt.Errorf("sshd configuration test failed; candidate rolled back and SSH not restarted: %w", err)
	}
	return nil
}

func rollbackFile(runner setupexec.CmdRunner, path string, oldContent []byte, hadOld bool, perm os.FileMode) error {
	if !hadOld {
		return runner.Remove(path)
	}
	return atomicWriteFile(runner, path, oldContent, perm)
}

func verifyGPGFingerprint(runner setupexec.CmdRunner, keyPath, expected string) error {
	out, err := runner.Output("gpg", "--show-keys", "--with-fingerprint", "--with-colons", keyPath)
	if err != nil {
		return err
	}
	normalizedExpected := strings.ReplaceAll(expected, " ", "")
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "fpr:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 10 && parts[9] == normalizedExpected {
				return nil
			}
		}
	}
	return fmt.Errorf("fingerprint mismatch for %s", keyPath)
}
