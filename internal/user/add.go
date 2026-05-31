package user

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

const sudoersFormat = "%s ALL=(ALL) NOPASSWD:ALL\n"

func AddUser(runner setupexec.CmdRunner, username, pubkey string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := ValidateSSHKey(pubkey); err != nil {
		return err
	}

	if err := ensureUser(runner, username); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	setupexec.PrintStep(fmt.Sprintf("Enabling lingering for %s", username))
	if err := runner.Run("loginctl", "enable-linger", username); err != nil {
		setupexec.PrintError("loginctl enable-linger failed (non-fatal)")
	}

	setupexec.PrintStep(fmt.Sprintf("Adding %s to sudo and docker groups", username))
	if err := runner.Run("usermod", "-aG", "sudo", username); err != nil {
		return fmt.Errorf("add %s to sudo group: %w", username, err)
	}
	if err := runner.Run("usermod", "-aG", "docker", username); err != nil {
		setupexec.PrintError("Failed to add user to docker group (non-fatal)")
	}

	if err := writeSudoers(runner, username); err != nil {
		return fmt.Errorf("configure sudoers: %w", err)
	}

	if err := installSSHKey(runner, username, pubkey); err != nil {
		return fmt.Errorf("install SSH key: %w", err)
	}

	if err := updateAllowUsers(runner); err != nil {
		return fmt.Errorf("update AllowUsers: %w", err)
	}

	setupexec.PrintDone(fmt.Sprintf("User provisioned: %s", username))
	return nil
}

func ensureUser(runner setupexec.CmdRunner, username string) error {
	exists := false
	_, err := runner.Output("id", username)
	if err == nil {
		exists = true
	}

	if !exists {
		setupexec.PrintStep(fmt.Sprintf("Creating user %s", username))
		if err := runner.Run("adduser", "--disabled-password", "--gecos", "", username); err != nil {
			return err
		}
	} else {
		setupexec.PrintStep(fmt.Sprintf("User %s already exists, skipping creation", username))
	}
	return nil
}

func writeSudoers(runner setupexec.CmdRunner, username string) error {
	path := "/etc/sudoers.d/" + username
	content := "# Managed by setup — do not edit\n" + fmt.Sprintf(sudoersFormat, username)

	setupexec.PrintStep(fmt.Sprintf("Writing %s", path))

	tmpFile, err := os.CreateTemp("", "setup-sudoers-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(content), 0440); err != nil {
		return fmt.Errorf("write temp sudoers: %w", err)
	}

	if err := runner.Chown(tmpPath, 0, 0); err != nil {
		return err
	}

	oldContent, _ := runner.ReadFile(path)
	if bytes.Equal(oldContent, []byte(content)) {
		return nil
	}

	if err := runner.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func installSSHKey(runner setupexec.CmdRunner, username, pubkey string) error {
	homeDir := "/home/" + username
	sshDir := homeDir + "/.ssh"
	authPath := sshDir + "/authorized_keys"

	setupexec.PrintStep(fmt.Sprintf("Configuring SSH for %s", username))

	// Look up the user's UID/GID for Chown
	uid, gid, err := runner.LookupUser(username)
	if err != nil {
		return fmt.Errorf("lookup user %s: %w", username, err)
	}

	// Create .ssh directory with correct ownership
	if err := runner.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create .ssh dir: %w", err)
	}
	if err := runner.Chown(sshDir, uid, gid); err != nil {
		return fmt.Errorf("chown .ssh dir: %w", err)
	}

	// Read existing authorized_keys if any
	existing, _ := runner.ReadFile(authPath)
	existingKeys := strings.TrimSpace(string(existing))

	// Check if this key is already present (idempotency)
	if existingKeys != "" {
		for _, line := range strings.Split(existingKeys, "\n") {
			if strings.TrimSpace(line) == pubkey {
				setupexec.PrintStep("SSH key already installed, skipping")
				return nil
			}
		}
	}

	// Build new content: existing keys + new key
	var newData string
	if existingKeys != "" {
		newData = existingKeys + "\n" + pubkey + "\n"
	} else {
		newData = pubkey + "\n"
	}

	// Write atomically via temp file
	tmpFile, err := os.CreateTemp("", "setup-authorized-keys-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(newData), 0600); err != nil {
		return fmt.Errorf("write authorized_keys: %w", err)
	}
	if err := runner.Chown(tmpPath, uid, gid); err != nil {
		return fmt.Errorf("chown authorized_keys: %w", err)
	}
	if err := runner.Rename(tmpPath, authPath); err != nil {
		return fmt.Errorf("install authorized_keys: %w", err)
	}
	return nil
}

func updateAllowUsers(runner setupexec.CmdRunner) error {
	allowFile := "/etc/ssh/sshd_config.d/98-allow-users.conf"

	setupexec.PrintStep("Updating SSH AllowUsers")

	users, err := listNonSystemUsers(runner)
	if err != nil {
		return fmt.Errorf("list non-system users: %w", err)
	}

	newContent := "# Managed by setup — do not edit\nAllowUsers " + strings.Join(users, " ") + "\n"

	oldContent, _ := runner.ReadFile(allowFile)
	if bytes.Equal(oldContent, []byte(newContent)) {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "setup-allow-users-*")
	if err != nil {
		return fmt.Errorf("create temp AllowUsers file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write temp AllowUsers: %w", err)
	}

	// Validate the new config against the temp file before installing
	if err := runner.Run("sshd", "-t", "-f", tmpPath); err != nil {
		return fmt.Errorf("sshd configuration test failed — new AllowUsers config rejected, SSH not restarted")
	}

	if err := runner.Rename(tmpPath, allowFile); err != nil {
		return err
	}

	setupexec.PrintStep("Restarting SSH")
	return runner.Run("systemctl", "restart", "ssh")
}

func listNonSystemUsers(runner setupexec.CmdRunner) ([]string, error) {
	out, err := runner.Output("awk", "-F:", `$3 >= 1000 && $1 != "nobody" { print $1 }`, "/etc/passwd")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var users []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			users = append(users, line)
		}
	}
	return users, nil
}
