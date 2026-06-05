package user

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
)

const sudoersFormat = "%s ALL=(ALL) NOPASSWD:ALL\n"
const allowUsersFile = "/etc/ssh/sshd_config.d/98-allow-users.conf"
const emptyAllowUsersSentinel = "setup-no-ssh-users"

type accountInfo struct {
	uid   int
	gid   int
	home  string
	shell string
}

func AddUser(runner setupexec.CmdRunner, username, pubkey string) error {
	if err := CreateLoginUser(runner, username); err != nil {
		return err
	}
	if err := EnableLinger(runner, username); err != nil {
		setupexec.PrintError("loginctl enable-linger failed (non-fatal)")
	}
	if err := AddGroup(runner, username, "sudo"); err != nil {
		return fmt.Errorf("add %s to sudo group: %w", username, err)
	}
	if err := EnablePasswordlessSudo(runner, username); err != nil {
		return fmt.Errorf("configure sudoers: %w", err)
	}
	if err := AddAuthorizedKey(runner, username, pubkey); err != nil {
		return fmt.Errorf("install SSH key: %w", err)
	}
	if err := AllowSSH(runner, username); err != nil {
		return fmt.Errorf("update AllowUsers: %w", err)
	}
	if err := AddGroup(runner, username, "docker"); err != nil {
		setupexec.PrintError("Failed to add user to docker group (non-fatal)")
	}
	setupexec.PrintDone(fmt.Sprintf("User provisioned: %s", username))
	return nil
}

func CreateLoginUser(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	_, err := ensureUser(runner, username)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func CreateLoginUserSelected(runner setupexec.CmdRunner, username, pubkey string, allowSSH, sudo, linger bool, groups []string) error {
	if err := CreateLoginUser(runner, username); err != nil {
		return err
	}
	if pubkey != "" {
		if err := AddAuthorizedKey(runner, username, pubkey); err != nil {
			return err
		}
	}
	if allowSSH {
		if err := AllowSSH(runner, username); err != nil {
			return err
		}
	}
	if sudo {
		if err := EnablePasswordlessSudo(runner, username); err != nil {
			return err
		}
	}
	if linger {
		if err := EnableLinger(runner, username); err != nil {
			return err
		}
	}
	for _, group := range groups {
		if err := AddGroup(runner, username, group); err != nil {
			return err
		}
	}
	return nil
}

func CreateServiceUser(runner setupexec.CmdRunner, username string, groups []string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	for _, group := range groups {
		if err := validateGroupName(group); err != nil {
			return err
		}
		if err := ensureGroupExists(runner, group); err != nil {
			return err
		}
	}

	exists := false
	_, err := runner.Output("id", username)
	if err == nil {
		exists = true
	}
	if setupexec.IsDryRun(runner) {
		exists = false
	}

	if exists {
		acct, err := lookupAnyAccount(runner, username)
		if err != nil {
			return err
		}
		if !isSetupServiceAccount(username, acct) {
			return fmt.Errorf("refusing to treat existing %s as a setup-owned service user", username)
		}
		setupexec.PrintStep(fmt.Sprintf("Service user %s already exists, skipping creation", username))
	} else {
		home := "/var/lib/" + username
		setupexec.PrintStep(fmt.Sprintf("Creating service user %s", username))
		if err := runner.Run("adduser", "--system", "--group", "--home", home, "--shell", "/usr/sbin/nologin", username); err != nil {
			return err
		}
	}
	for _, group := range groups {
		if err := AddGroup(runner, username, group); err != nil {
			return err
		}
	}
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

func AddAuthorizedKey(runner setupexec.CmdRunner, username, pubkey string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := ValidateSSHKey(pubkey); err != nil {
		return err
	}
	acct, err := lookupAccount(runner, username)
	if err != nil {
		return err
	}
	return installSSHKey(runner, username, acct, pubkey)
}

func AllowSSH(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupAccount(runner, username); err != nil {
		return err
	}
	return updateAllowUsersList(runner, func(users []string) []string {
		if contains(users, username) {
			return users
		}
		return append(users, username)
	})
}

func DenySSH(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	return updateAllowUsersList(runner, func(users []string) []string {
		var next []string
		for _, user := range users {
			if user != username {
				next = append(next, user)
			}
		}
		return next
	})
}

func EnableLinger(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	setupexec.PrintStep(fmt.Sprintf("Enabling lingering for %s", username))
	return runner.Run("loginctl", "enable-linger", username)
}

func DisableLinger(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	setupexec.PrintStep(fmt.Sprintf("Disabling lingering for %s", username))
	return runner.Run("loginctl", "disable-linger", username)
}

func EnablePasswordlessSudo(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	return writeSudoers(runner, username)
}

func DisablePasswordlessSudo(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	path := "/etc/sudoers.d/" + username
	setupexec.PrintStep(fmt.Sprintf("Removing setup-managed %s", path))
	content, err := runner.ReadFile(path)
	if err == nil && len(content) == 0 && setupexec.IsDryRun(runner) {
		err = os.ErrNotExist
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !bytes.HasPrefix(content, []byte(managed.Marker)) {
		return fmt.Errorf("refusing to remove unmanaged sudoers file %s", path)
	}
	return runner.Remove(path)
}

func writeSudoers(runner setupexec.CmdRunner, username string) error {
	path := "/etc/sudoers.d/" + username
	content := managed.Marker + fmt.Sprintf(sudoersFormat, username)

	setupexec.PrintStep(fmt.Sprintf("Writing %s", path))

	oldContent, err := runner.ReadFile(path)
	if err == nil && len(oldContent) == 0 && setupexec.IsDryRun(runner) {
		err = os.ErrNotExist
	}
	if err == nil {
		if bytes.Equal(oldContent, []byte(content)) {
			return nil
		}
		if !bytes.HasPrefix(oldContent, []byte(managed.Marker)) {
			return fmt.Errorf("refusing to replace unmanaged sudoers file %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
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

func AddGroup(runner setupexec.CmdRunner, username, group string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := validateGroupName(group); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	if err := ensureGroupExists(runner, group); err != nil {
		return err
	}
	if member, err := userInGroup(runner, username, group); err != nil {
		return err
	} else if member {
		setupexec.PrintStep(fmt.Sprintf("%s is already in %s, skipping", username, group))
		return nil
	}
	setupexec.PrintStep(fmt.Sprintf("Adding %s to %s group", username, group))
	return runner.Run("usermod", "-aG", group, username)
}

func RemoveGroup(runner setupexec.CmdRunner, username, group string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := validateGroupName(group); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	if err := ensureGroupExists(runner, group); err != nil {
		return err
	}
	if member, err := userInGroup(runner, username, group); err != nil {
		return err
	} else if !member {
		setupexec.PrintStep(fmt.Sprintf("%s is not in %s, skipping", username, group))
		return nil
	}
	setupexec.PrintStep(fmt.Sprintf("Removing %s from %s group", username, group))
	return runner.Run("gpasswd", "-d", username, group)
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
	return updateAllowUsersList(runner, func(users []string) []string { return users })
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
	acct, err := lookupAnyAccount(runner, username)
	if err != nil {
		return accountInfo{}, err
	}
	if acct.uid < 1000 {
		return accountInfo{}, fmt.Errorf("refusing to manage %s: uid %d is below 1000", username, acct.uid)
	}
	return acct, nil
}

func lookupAnyAccount(runner setupexec.CmdRunner, username string) (accountInfo, error) {
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
	home := strings.TrimSpace(parts[5])
	if home == "" || !filepath.IsAbs(home) {
		return accountInfo{}, fmt.Errorf("refusing to manage %s: invalid home directory %q", username, home)
	}
	return accountInfo{uid: uid, gid: gid, home: home, shell: strings.TrimSpace(parts[6])}, nil
}

func lookupManageableAccount(runner setupexec.CmdRunner, username string) (accountInfo, error) {
	acct, err := lookupAnyAccount(runner, username)
	if err != nil {
		return accountInfo{}, err
	}
	if acct.uid >= 1000 || isSetupServiceAccount(username, acct) {
		return acct, nil
	}
	return accountInfo{}, fmt.Errorf("refusing to manage %s: uid %d is below 1000 and account is not setup-owned", username, acct.uid)
}

func isSetupServiceAccount(username string, acct accountInfo) bool {
	if acct.home != "/var/lib/"+username {
		return false
	}
	shell := strings.TrimSpace(acct.shell)
	return shell == "/usr/sbin/nologin" || shell == "/sbin/nologin" || shell == "/bin/false"
}

func updateAllowUsersList(runner setupexec.CmdRunner, mutate func([]string) []string) error {
	setupexec.PrintStep("Updating SSH AllowUsers")

	oldContent, readErr := runner.ReadFile(allowUsersFile)
	if readErr == nil && len(oldContent) == 0 && setupexec.IsDryRun(runner) {
		readErr = os.ErrNotExist
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	if readErr == nil && !bytes.HasPrefix(oldContent, []byte(managed.Marker)) {
		return fmt.Errorf("refusing to replace unmanaged AllowUsers file %s", allowUsersFile)
	}

	users, err := parseManagedAllowUsers(oldContent, readErr == nil)
	if err != nil {
		return err
	}
	next := normalizeAllowUsers(mutate(users))
	newContent := renderAllowUsers(next)
	if bytes.Equal(oldContent, []byte(newContent)) {
		return nil
	}

	if err := runner.MkdirAll(filepath.Dir(allowUsersFile), 0755); err != nil {
		return err
	}

	tmpPath, err := runner.CreateTemp(filepath.Dir(allowUsersFile), ".setup-allow-users-*")
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

	if err := runner.Rename(tmpPath, allowUsersFile); err != nil {
		return err
	}
	if err := runner.Run("sshd", "-t"); err != nil {
		if rollbackErr := rollbackFile(runner, allowUsersFile, oldContent, readErr == nil, 0644); rollbackErr != nil {
			return fmt.Errorf("sshd configuration test failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
		}
		return fmt.Errorf("sshd configuration test failed; AllowUsers rolled back and SSH not restarted: %w", err)
	}

	setupexec.PrintStep("Restarting SSH")
	return runner.Run("systemctl", "restart", "ssh")
}

func parseManagedAllowUsers(content []byte, exists bool) ([]string, error) {
	if !exists {
		return nil, nil
	}
	var users []string
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 || fields[0] != "AllowUsers" {
			continue
		}
		for _, field := range fields[1:] {
			if field != emptyAllowUsersSentinel {
				users = append(users, field)
			}
		}
	}
	return normalizeAllowUsers(users), nil
}

func renderAllowUsers(users []string) string {
	if len(users) == 0 {
		return managed.Marker + "# Empty setup-managed SSH login allow-list.\nAllowUsers " + emptyAllowUsersSentinel + "\n"
	}
	return managed.Marker + "AllowUsers " + strings.Join(users, " ") + "\n"
}

func normalizeAllowUsers(users []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, username := range users {
		username = strings.TrimSpace(username)
		if username == "" || username == emptyAllowUsersSentinel || seen[username] {
			continue
		}
		seen[username] = true
		out = append(out, username)
	}
	sort.Strings(out)
	return out
}

func DisableUser(runner setupexec.CmdRunner, username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	setupexec.PrintStep(fmt.Sprintf("Locking password for %s", username))
	if err := runner.Run("passwd", "-l", username); err != nil {
		return err
	}
	if err := DenySSH(runner, username); err != nil {
		return err
	}
	if err := DisableLinger(runner, username); err != nil {
		return err
	}
	return DisablePasswordlessSudo(runner, username)
}

func DeleteUser(runner setupexec.CmdRunner, username string, removeHome bool) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if _, err := lookupManageableAccount(runner, username); err != nil {
		return err
	}
	if err := DisableUser(runner, username); err != nil {
		return err
	}
	setupexec.PrintStep(fmt.Sprintf("Deleting user %s", username))
	if removeHome {
		return runner.Run("deluser", "--remove-home", username)
	}
	return runner.Run("deluser", username)
}

func ensureGroupExists(runner setupexec.CmdRunner, group string) error {
	if _, err := runner.Output("getent", "group", group); err != nil {
		return fmt.Errorf("group %q does not exist", group)
	}
	return nil
}

func userInGroup(runner setupexec.CmdRunner, username, group string) (bool, error) {
	out, err := runner.Output("id", "-nG", username)
	if err != nil {
		return false, err
	}
	for _, existing := range strings.Fields(out) {
		if existing == group {
			return true, nil
		}
	}
	return false, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
