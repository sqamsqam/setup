package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
	setupuser "github.com/sqamsqam/setup/internal/user"
)

var serviceNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

type Config struct {
	User    string
	Name    string
	WorkDir string
	Command string
	EnvFile string
}

type account struct {
	uid  int
	gid  int
	home string
}

func Create(runner setupexec.CmdRunner, cfg Config) error {
	acct, err := validateConfig(runner, cfg)
	if err != nil {
		return err
	}
	unit := UnitName(cfg.Name)
	dir := filepath.Join(acct.home, ".config", "systemd", "user")
	path := filepath.Join(dir, unit)
	if err := runner.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := runner.Chown(dir, acct.uid, acct.gid); err != nil {
		return err
	}
	changed, err := managed.WriteManagedFileIfChanged(runner, path, []byte(UnitContent(cfg)), 0644)
	if err != nil {
		return err
	}
	if changed {
		if err := runner.Chown(path, acct.uid, acct.gid); err != nil {
			return err
		}
	}
	if err := runner.Run("loginctl", "enable-linger", cfg.User); err != nil {
		return err
	}
	if err := runner.RunAsUser(cfg.User, "systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	return runner.RunAsUser(cfg.User, "systemctl", "--user", "enable", "--now", unit)
}

func Status(runner setupexec.CmdRunner, user, name string) (string, error) {
	if err := validateUserAndName(user, name); err != nil {
		return "", err
	}
	if _, err := requireManagedUnit(runner, user, name); err != nil {
		return "", err
	}
	return runner.Output("sudo", "-iu", user, "--", "systemctl", "--user", "status", UnitName(name), "--no-pager")
}

func Logs(runner setupexec.CmdRunner, user, name string) (string, error) {
	if err := validateUserAndName(user, name); err != nil {
		return "", err
	}
	if _, err := requireManagedUnit(runner, user, name); err != nil {
		return "", err
	}
	return runner.Output("sudo", "-iu", user, "--", "journalctl", "--user", "-u", UnitName(name), "--no-pager", "-n", "100")
}

func Restart(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	if _, err := requireManagedUnit(runner, user, name); err != nil {
		return err
	}
	return runner.RunAsUser(user, "systemctl", "--user", "restart", UnitName(name))
}

func List(runner setupexec.CmdRunner, user string) ([]string, error) {
	if err := setupuser.ValidateUsername(user); err != nil {
		return nil, err
	}
	acct, err := lookupAccount(runner, user)
	if err != nil {
		return nil, err
	}
	dir := userUnitDir(acct)
	entries, err := runner.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var units []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "setup-") || !strings.HasSuffix(name, ".service") {
			continue
		}
		content, err := runner.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if managed.IsMarked(content) {
			units = append(units, name)
		}
	}
	sort.Strings(units)
	return units, nil
}

func Disable(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	if _, err := requireManagedUnit(runner, user, name); err != nil {
		return err
	}
	return runner.RunAsUser(user, "systemctl", "--user", "disable", "--now", UnitName(name))
}

func Remove(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	path, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return err
	}
	if err := runner.RunAsUser(user, "systemctl", "--user", "disable", "--now", UnitName(name)); err != nil {
		return err
	}
	if err := runner.Remove(path); err != nil {
		return err
	}
	return runner.RunAsUser(user, "systemctl", "--user", "daemon-reload")
}

func UnitName(name string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".service")
	if strings.HasPrefix(name, "setup-") {
		return name + ".service"
	}
	return "setup-" + name + ".service"
}

func UnitContent(cfg Config) string {
	var b strings.Builder
	b.WriteString(managed.Marker)
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=setup managed service %s\n", cfg.Name)
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", systemdQuote(cfg.WorkDir))
	if strings.TrimSpace(cfg.EnvFile) != "" {
		fmt.Fprintf(&b, "EnvironmentFile=-%s\n", systemdQuote(cfg.EnvFile))
	}
	fmt.Fprintf(&b, "ExecStart=/bin/bash -lc %s\n", systemdQuote(cfg.Command))
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=default.target\n")
	return b.String()
}

func validateConfig(runner setupexec.CmdRunner, cfg Config) (account, error) {
	if err := validateUserAndName(cfg.User, cfg.Name); err != nil {
		return account{}, err
	}
	if strings.TrimSpace(cfg.WorkDir) == "" || !filepath.IsAbs(cfg.WorkDir) {
		return account{}, fmt.Errorf("working directory must be an absolute path")
	}
	if strings.ContainsAny(cfg.WorkDir, "\r\n") {
		return account{}, fmt.Errorf("working directory must be a single line")
	}
	if strings.TrimSpace(cfg.Command) == "" || strings.ContainsAny(cfg.Command, "\r\n") {
		return account{}, fmt.Errorf("command must be a non-empty single line")
	}
	if strings.TrimSpace(cfg.EnvFile) != "" {
		if !filepath.IsAbs(cfg.EnvFile) {
			return account{}, fmt.Errorf("env file must be an absolute path")
		}
		if strings.ContainsAny(cfg.EnvFile, "\r\n") {
			return account{}, fmt.Errorf("env file must be a single line")
		}
	}
	return lookupAccount(runner, cfg.User)
}

func validateUserAndName(username, name string) error {
	if err := setupuser.ValidateUsername(username); err != nil {
		return err
	}
	return ValidateName(name)
}

func ValidateName(name string) error {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".service")
	name = strings.TrimPrefix(name, "setup-")
	if !serviceNameRe.MatchString(name) {
		return fmt.Errorf("service name must match %s", serviceNameRe.String())
	}
	return nil
}

func requireManagedUnit(runner setupexec.CmdRunner, username, name string) (string, error) {
	acct, err := lookupAccount(runner, username)
	if err != nil {
		return "", err
	}
	path := filepath.Join(userUnitDir(acct), UnitName(name))
	content, err := runner.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("managed service unit %s does not exist", path)
		}
		return "", err
	}
	if setupexec.IsDryRun(runner) && len(content) == 0 {
		return path, nil
	}
	if !managed.IsMarked(content) {
		return "", fmt.Errorf("refusing to operate on unmanaged service unit %s", path)
	}
	return path, nil
}

func userUnitDir(acct account) string {
	return filepath.Join(acct.home, ".config", "systemd", "user")
}

func lookupAccount(runner setupexec.CmdRunner, username string) (account, error) {
	out, err := runner.Output("getent", "passwd", username)
	if err != nil {
		return account{}, fmt.Errorf("lookup passwd entry for %s: %w", username, err)
	}
	parts := strings.Split(out, ":")
	if len(parts) < 7 || parts[0] != username {
		return account{}, fmt.Errorf("invalid passwd entry for %s", username)
	}
	uid, err := strconv.Atoi(parts[2])
	if err != nil {
		return account{}, fmt.Errorf("parse uid for %s: %w", username, err)
	}
	gid, err := strconv.Atoi(parts[3])
	if err != nil {
		return account{}, fmt.Errorf("parse gid for %s: %w", username, err)
	}
	if uid < 1000 {
		return account{}, fmt.Errorf("refusing to manage %s: uid %d is below 1000", username, uid)
	}
	home := strings.TrimSpace(parts[5])
	if home == "" || !filepath.IsAbs(home) {
		return account{}, fmt.Errorf("invalid home directory for %s: %q", username, home)
	}
	return account{uid: uid, gid: gid, home: home}, nil
}

func systemdQuote(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
	return `"` + replacer.Replace(s) + `"`
}
