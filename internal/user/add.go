package user

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

const sudoersFormat = "%s ALL=(ALL) NOPASSWD:ALL\n"

type accountInfo struct {
	uid   int
	gid   int
	home  string
	shell string
}

func AddUser(runner setupexec.CmdRunner, username, pubkey string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := ValidateSSHKey(pubkey); err != nil {
		return err
	}

	acct, err := ensureUser(runner, username)
	if err != nil {
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

	if err := installSSHKey(runner, username, acct, pubkey); err != nil {
		return fmt.Errorf("install SSH key: %w", err)
	}

	if err := updateAllowUsers(runner); err != nil {
		return fmt.Errorf("update AllowUsers: %w", err)
	}

	setupexec.PrintDone(fmt.Sprintf("User provisioned: %s", username))
	return nil
}

func ensureUser(runner setupexec.CmdRunner, username string) (accountInfo, error) {
	exists := false
	_, err := runner.Output("id", username)
	if err == nil {
		exists = true
	}

	if !exists {
		setupexec.PrintStep(fmt.Sprintf("Creating user %s", username))
		if err := runner.Run("adduser", "--disabled-password", "--gecos", "", username); err != nil {
			return accountInfo{}, err
		}
	} else {
		setupexec.PrintStep(fmt.Sprintf("User %s already exists, skipping creation", username))
	}
	return lookupAccount(runner, username)
}

func writeSudoers(runner setupexec.CmdRunner, username string) error {
	path := "/etc/sudoers.d/" + username
	content := "# Managed by setup — do not edit\n" + fmt.Sprintf(sudoersFormat, username)

	setupexec.PrintStep(fmt.Sprintf("Writing %s", path))

	oldContent, _ := runner.ReadFile(path)
	if bytes.Equal(oldContent, []byte(content)) {
		return nil
	}

	if err := runner.MkdirAll("/etc/sudoers.d", 0755); err != nil {
		return err
	}

	tmpPath, err := runner.CreateTemp("/etc/sudoers.d", ".setup-sudoers-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(content), 0440); err != nil {
		return fmt.Errorf("write temp sudoers: %w", err)
	}
	if err := runner.Chmod(tmpPath, 0440); err != nil {
		return err
	}
	if err := runner.Chown(tmpPath, 0, 0); err != nil {
		return err
	}

	if err := runner.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func installSSHKey(runner setupexec.CmdRunner, username string, acct accountInfo, pubkey string) error {
	homeDir := acct.home
	sshDir := homeDir + "/.ssh"
	authPath := sshDir + "/authorized_keys"

	setupexec.PrintStep(fmt.Sprintf("Configuring SSH for %s", username))

	// Create .ssh directory with correct ownership
	if err := runner.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create .ssh dir: %w", err)
	}
	if err := runner.Chown(sshDir, acct.uid, acct.gid); err != nil {
		return fmt.Errorf("chown .ssh dir: %w", err)
	}

	// Read existing authorized_keys if any
	existing, readErr := runner.ReadFile(authPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read authorized_keys: %w", readErr)
	}
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
	tmpPath, err := runner.CreateTemp(sshDir, ".setup-authorized-keys-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(newData), 0600); err != nil {
		return fmt.Errorf("write authorized_keys: %w", err)
	}
	if err := runner.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("chmod authorized_keys: %w", err)
	}
	if err := runner.Chown(tmpPath, acct.uid, acct.gid); err != nil {
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

	oldContent, readErr := runner.ReadFile(allowFile)
	if bytes.Equal(oldContent, []byte(newContent)) {
		return nil
	}

	if err := runner.MkdirAll(filepath.Dir(allowFile), 0755); err != nil {
		return err
	}

	tmpPath, err := runner.CreateTemp(filepath.Dir(allowFile), ".setup-allow-users-*")
	if err != nil {
		return fmt.Errorf("create temp AllowUsers file: %w", err)
	}
	defer func() { _ = runner.Remove(tmpPath) }()

	if err := runner.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write temp AllowUsers: %w", err)
	}
	if err := runner.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("chmod temp AllowUsers: %w", err)
	}

	if err := runner.Rename(tmpPath, allowFile); err != nil {
		return err
	}
	if err := runner.Run("sshd", "-t"); err != nil {
		if rollbackErr := rollbackFile(runner, allowFile, oldContent, readErr == nil, 0644); rollbackErr != nil {
			return fmt.Errorf("sshd configuration test failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
		}
		return fmt.Errorf("sshd configuration test failed; AllowUsers rolled back and SSH not restarted: %w", err)
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

func lookupAccount(runner setupexec.CmdRunner, username string) (accountInfo, error) {
	out, err := runner.Output("getent", "passwd", username)
	if err != nil {
		return accountInfo{}, fmt.Errorf("lookup passwd entry for %s: %w", username, err)
	}
	parts := strings.Split(out, ":")
	if len(parts) < 7 || parts[0] != username {
		return accountInfo{}, fmt.Errorf("invalid passwd entry for %s", username)
	}
	uid, err := strconv.Atoi(parts[2])
	if err != nil {
		return accountInfo{}, fmt.Errorf("parse uid for %s: %w", username, err)
	}
	gid, err := strconv.Atoi(parts[3])
	if err != nil {
		return accountInfo{}, fmt.Errorf("parse gid for %s: %w", username, err)
	}
	if uid < 1000 {
		return accountInfo{}, fmt.Errorf("refusing to manage %s: uid %d is below 1000", username, uid)
	}
	home := strings.TrimSpace(parts[5])
	if home == "" || !filepath.IsAbs(home) {
		return accountInfo{}, fmt.Errorf("refusing to manage %s: invalid home directory %q", username, home)
	}
	return accountInfo{uid: uid, gid: gid, home: home, shell: strings.TrimSpace(parts[6])}, nil
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

func rollbackFile(runner setupexec.CmdRunner, path string, oldContent []byte, hadOld bool, perm os.FileMode) error {
	if !hadOld {
		return runner.Remove(path)
	}
	return atomicWriteFile(runner, path, oldContent, perm)
}
