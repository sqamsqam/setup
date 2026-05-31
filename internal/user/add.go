package user

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	_ = runner.Run("usermod", "-aG", "docker", username)

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
	content := fmt.Sprintf(sudoersFormat, username)

	setupexec.PrintStep(fmt.Sprintf("Writing %s", path))

	tmpPath := "/tmp/sudoers-" + username
	if err := os.WriteFile(tmpPath, []byte(content), 0440); err != nil {
		return fmt.Errorf("write temp sudoers: %w", err)
	}

	if err := runner.Run("chown", "root:root", tmpPath); err != nil {
		return err
	}

	oldContent, _ := os.ReadFile(path)
	if bytes.Equal(oldContent, []byte(content)) {
		os.Remove(tmpPath)
		return nil
	}

	if err := runner.Run("mv", tmpPath, path); err != nil {
		return err
	}
	return nil
}

func installSSHKey(runner setupexec.CmdRunner, username, pubkey string) error {
	homeDir := "/home/" + username
	sshDir := homeDir + "/.ssh"

	setupexec.PrintStep(fmt.Sprintf("Configuring SSH for %s", username))

	if err := runner.Run("install", "-d", "-m", "700", "-o", username, "-g", username, sshDir); err != nil {
		return fmt.Errorf("create .ssh dir: %w", err)
	}

	authPath := sshDir + "/authorized_keys"
	tmpPath := "/tmp/auth-" + username

	if err := os.WriteFile(tmpPath, []byte(pubkey+"\n"), 0600); err != nil {
		return fmt.Errorf("write temp authorized_keys: %w", err)
	}
	if err := runner.Run("chown", username+":"+username, tmpPath); err != nil {
		return err
	}
	if err := runner.Run("mv", tmpPath, authPath); err != nil {
		return err
	}
	return nil
}

func updateAllowUsers(runner setupexec.CmdRunner) error {
	allowFile := "/etc/ssh/sshd_config.d/98-allow-users.conf"

	setupexec.PrintStep("Updating SSH AllowUsers")

	users, err := listNonSystemUsers()
	if err != nil {
		return fmt.Errorf("list non-system users: %w", err)
	}

	newContent := "AllowUsers " + strings.Join(users, " ") + "\n"

	oldContent, _ := os.ReadFile(allowFile)
	if bytes.Equal(oldContent, []byte(newContent)) {
		return nil
	}

	tmpFile := "/tmp/ssh-allow-users.conf"
	if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write temp AllowUsers: %w", err)
	}

	if err := runner.Run("mv", tmpFile, allowFile); err != nil {
		return err
	}

	setupexec.PrintStep("Restarting SSH")
	return runner.Run("systemctl", "restart", "ssh")
}

func listNonSystemUsers() ([]string, error) {
	cmd := exec.Command("awk", "-F:", `$3 >= 1000 && $1 != "nobody" { print $1 }`, "/etc/passwd")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var users []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			users = append(users, line)
		}
	}
	return users, nil
}
