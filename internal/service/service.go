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

type UnitState struct {
	Unit          string
	LoadState     string
	ActiveState   string
	SubState      string
	UnitFileState string
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
	oldContent, readErr := runner.ReadFile(path)
	hadOld := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	changed, err := managed.WriteManagedFileIfChanged(runner, path, []byte(UnitContent(cfg)), 0644)
	if err != nil {
		return err
	}
	if changed {
		if err := runner.Chown(path, acct.uid, acct.gid); err != nil {
			return err
		}
		if err := validateUnit(runner, path); err != nil {
			if rollbackErr := rollbackUnit(runner, path, oldContent, hadOld, acct); rollbackErr != nil {
				return fmt.Errorf("validate systemd unit failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
			}
			return fmt.Errorf("validate systemd unit failed; candidate rolled back: %w", err)
		}
	}
	if err := runner.Run("loginctl", "enable-linger", cfg.User); err != nil {
		return err
	}
	if err := runner.Run("systemctl", "start", fmt.Sprintf("user@%d.service", acct.uid)); err != nil {
		return err
	}
	if err := runUserSystemctl(runner, acct, cfg.User, "--user", "daemon-reload"); err != nil {
		if !changed {
			return err
		}
		if rollbackErr := rollbackUnit(runner, path, oldContent, hadOld, acct); rollbackErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", err, rollbackErr)
		}
		if reloadErr := runUserSystemctl(runner, acct, cfg.User, "--user", "daemon-reload"); reloadErr != nil {
			return fmt.Errorf("%w (rollback reload failed: %v)", err, reloadErr)
		}
		return err
	}
	if err := runUserSystemctl(runner, acct, cfg.User, "--user", "enable", "--now", unit); err != nil {
		return err
	}
	if changed && hadOld {
		if err := runUserSystemctl(runner, acct, cfg.User, "--user", "restart", unit); err != nil {
			return err
		}
	}
	return nil
}

func Status(runner setupexec.CmdRunner, user, name string) (string, error) {
	if err := validateUserAndName(user, name); err != nil {
		return "", err
	}
	acct, _, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return "", err
	}
	return outputUserSystemctl(runner, acct, user, "--user", "status", UnitName(name), "--no-pager")
}

func Logs(runner setupexec.CmdRunner, user, name string) (string, error) {
	if err := validateUserAndName(user, name); err != nil {
		return "", err
	}
	acct, _, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return "", err
	}
	return outputUserJournalctl(runner, acct, user, "--user", "-u", UnitName(name), "--no-pager", "-n", "100")
}

func Restart(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	acct, _, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return err
	}
	return runUserSystemctl(runner, acct, user, "--user", "restart", UnitName(name))
}

func List(runner setupexec.CmdRunner, user string) ([]string, error) {
	_, units, err := listManagedUnits(runner, user)
	return units, err
}

func ListWithState(runner setupexec.CmdRunner, user string) ([]UnitState, error) {
	acct, units, err := listManagedUnits(runner, user)
	if err != nil {
		return nil, err
	}
	states := make([]UnitState, 0, len(units))
	for _, unit := range units {
		state, err := getUnitState(runner, acct, user, unit)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func listManagedUnits(runner setupexec.CmdRunner, user string) (account, []string, error) {
	if err := setupuser.ValidateUsername(user); err != nil {
		return account{}, nil, err
	}
	acct, err := lookupAccount(runner, user)
	if err != nil {
		return account{}, nil, err
	}
	dir := userUnitDir(acct)
	entries, err := runner.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return acct, nil, nil
		}
		return account{}, nil, err
	}

	var units []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "setup-") || !strings.HasSuffix(name, ".service") {
			continue
		}
		content, err := runner.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return account{}, nil, err
		}
		if managed.IsMarked(content) {
			units = append(units, name)
		}
	}
	sort.Strings(units)
	return acct, units, nil
}

func Disable(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	acct, _, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return err
	}
	return runUserSystemctl(runner, acct, user, "--user", "disable", "--now", UnitName(name))
}

func Remove(runner setupexec.CmdRunner, user, name string) error {
	if err := validateUserAndName(user, name); err != nil {
		return err
	}
	acct, path, err := requireManagedUnit(runner, user, name)
	if err != nil {
		return err
	}
	if err := runUserSystemctl(runner, acct, user, "--user", "disable", "--now", UnitName(name)); err != nil {
		return err
	}
	if err := runner.Remove(path); err != nil {
		return err
	}
	return runUserSystemctl(runner, acct, user, "--user", "daemon-reload")
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

func requireManagedUnit(runner setupexec.CmdRunner, username, name string) (account, string, error) {
	acct, err := lookupAccount(runner, username)
	if err != nil {
		return account{}, "", err
	}
	path := filepath.Join(userUnitDir(acct), UnitName(name))
	content, err := runner.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return account{}, "", fmt.Errorf("managed service unit %s does not exist", path)
		}
		return account{}, "", err
	}
	if setupexec.IsDryRun(runner) && len(content) == 0 {
		return acct, path, nil
	}
	if !managed.IsMarked(content) {
		return account{}, "", fmt.Errorf("refusing to operate on unmanaged service unit %s", path)
	}
	return acct, path, nil
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
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`", "%", "%%")
	return `"` + replacer.Replace(s) + `"`
}

func validateUnit(runner setupexec.CmdRunner, path string) error {
	return runner.Run("systemd-analyze", "verify", path)
}

func rollbackUnit(runner setupexec.CmdRunner, path string, oldContent []byte, hadOld bool, acct account) error {
	if !hadOld {
		return runner.Remove(path)
	}
	if _, err := managed.WriteManagedFileIfChanged(runner, path, oldContent, 0644); err != nil {
		return err
	}
	return runner.Chown(path, acct.uid, acct.gid)
}

func runUserSystemctl(runner setupexec.CmdRunner, acct account, user string, args ...string) error {
	allArgs := appendUserEnvArgs(user, acct.uid, "systemctl", args...)
	return runner.Run("sudo", allArgs...)
}

func outputUserSystemctl(runner setupexec.CmdRunner, acct account, user string, args ...string) (string, error) {
	allArgs := appendUserEnvArgs(user, acct.uid, "systemctl", args...)
	return runner.Output("sudo", allArgs...)
}

func outputUserJournalctl(runner setupexec.CmdRunner, acct account, user string, args ...string) (string, error) {
	allArgs := appendUserEnvArgs(user, acct.uid, "journalctl", args...)
	return runner.Output("sudo", allArgs...)
}

func appendUserEnvArgs(user string, uid int, name string, args ...string) []string {
	allArgs := []string{"-iu", user, "--", "env", "XDG_RUNTIME_DIR=/run/user/" + strconv.Itoa(uid), name}
	return append(allArgs, args...)
}

func getUnitState(runner setupexec.CmdRunner, acct account, user, unit string) (UnitState, error) {
	out, err := outputUserSystemctl(runner, acct, user,
		"--user",
		"show",
		unit,
		"--property=LoadState",
		"--property=ActiveState",
		"--property=SubState",
		"--property=UnitFileState",
		"--no-pager",
	)
	if err != nil {
		return UnitState{}, err
	}
	state := UnitState{
		Unit:          unit,
		LoadState:     "unknown",
		ActiveState:   "unknown",
		SubState:      "unknown",
		UnitFileState: "unknown",
	}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(value) == "" {
			value = "unknown"
		}
		switch key {
		case "LoadState":
			state.LoadState = value
		case "ActiveState":
			state.ActiveState = value
		case "SubState":
			state.SubState = value
		case "UnitFileState":
			state.UnitFileState = value
		}
	}
	return state, nil
}
