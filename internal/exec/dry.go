package exec

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type DryRunner struct {
	Stdout io.Writer
	Demo   bool
}

func NewDryRunner() *DryRunner {
	return &DryRunner{Stdout: os.Stderr}
}

func NewDemoRunner() *DryRunner {
	return &DryRunner{Stdout: os.Stderr, Demo: true}
}

func (d *DryRunner) IsDryRun() bool { return true }

func (d *DryRunner) IsDemo() bool { return d.Demo }

func (d *DryRunner) log(cmd string) {
	if d.Demo {
		_, _ = fmt.Fprintln(d.Stdout, cmd)
		return
	}
	_, _ = fmt.Fprintf(d.Stdout, "[DRY-RUN] %s\n", cmd)
}

func (d *DryRunner) logCommand(cmd string) {
	if d.Demo {
		_, _ = fmt.Fprintf(d.Stdout, "$ %s\n", cmd)
		return
	}
	d.log(cmd)
}

func (d *DryRunner) Run(name string, args ...string) error {
	d.logCommand(name + " " + strings.Join(args, " "))
	return nil
}

func (d *DryRunner) Output(name string, args ...string) (string, error) {
	d.logCommand(name + " " + strings.Join(args, " "))

	switch name {
	case "dpkg":
		if len(args) > 0 && args[0] == "--print-architecture" {
			return "amd64", nil
		}
	case "getent":
		if len(args) > 0 && args[0] == "passwd" {
			username := "user"
			if len(args) > 1 && args[1] != "" {
				username = args[1]
			}
			return username + ":x:1000:1000:User:/home/" + username + ":/bin/bash", nil
		}
	case "id":
		return "uid=1000(user) gid=1000(user) groups=1000(user)", nil
	case "awk":
		return "user", nil
	case "bash":
		if len(args) >= 2 && strings.Contains(args[1], "VERSION_CODENAME") {
			return "resolute", nil
		}
		if len(args) >= 2 && strings.Contains(args[1], "PRETTY_NAME") {
			return "Ubuntu 26.04 LTS", nil
		}
		if len(args) >= 2 && strings.Contains(args[1], "reboot-required") {
			return "Reboot not required.", nil
		}
		if len(args) >= 2 && strings.Contains(args[1], "apt list --upgradable") {
			return "No upgradable packages reported.", nil
		}
		if len(args) >= 2 && strings.Contains(args[1], "fuser") {
			return "clear", nil
		}
	case "systemd-detect-virt":
		return "lxc", nil
	case "systemctl":
		if len(args) > 0 {
			switch args[0] {
			case "is-system-running":
				return "running", nil
			case "is-active":
				return "active", nil
			case "--failed":
				return "0 loaded units listed.", nil
			case "status":
				return "active (running)", nil
			}
		}
	case "stat":
		return "cgroup2fs", nil
	case "df":
		return "Filesystem      Size  Used Avail Use% Mounted on\n/dev/root        20G  5G   15G  25% /", nil
	case "ss":
		return "Netid State  Local Address:Port\ntcp   LISTEN 0.0.0.0:22", nil
	case "sshd":
		if len(args) > 0 && args[0] == "-T" {
			return "port 22", nil
		}
	case "ufw":
		return "Status: inactive", nil
	case "docker":
		if len(args) >= 2 && args[0] == "system" && args[1] == "df" {
			return "TYPE            TOTAL     ACTIVE    SIZE\nImages          0         0         0B", nil
		}
	case "fail2ban-client":
		return "Status for the jail: sshd", nil
	}

	return "", nil
}

func (d *DryRunner) RunAsUser(user, name string, args ...string) error {
	allArgs := append([]string{"sudo", "-iu", user, "--", name}, args...)
	d.logCommand(strings.Join(allArgs, " "))
	return nil
}

func (d *DryRunner) Shell(script string) error {
	d.logCommand("bash -c '" + script + "'")
	return nil
}

func (d *DryRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	d.log(fmt.Sprintf("WriteFile(%s, %d bytes, %o)", path, len(data), perm))
	return nil
}

func (d *DryRunner) ReadFile(path string) ([]byte, error) {
	d.log("ReadFile(" + path + ")")
	return nil, nil
}

func (d *DryRunner) ReadDir(path string) ([]os.DirEntry, error) {
	d.log("ReadDir(" + path + ")")
	return nil, os.ErrNotExist
}

func (d *DryRunner) CreateTemp(dir, pattern string) (string, error) {
	if d.Demo {
		path := strings.TrimRight(dir, "/") + "/" + strings.ReplaceAll(pattern, "*", "000000")
		d.log("CreateTemp(" + dir + ", " + pattern + ") -> " + path)
		return path, nil
	}
	path := strings.TrimRight(dir, "/") + "/.setup-dry-run-" + strings.Trim(pattern, "*")
	d.log("CreateTemp(" + dir + ", " + pattern + ") -> " + path)
	return path, nil
}

func (d *DryRunner) Rename(oldpath, newpath string) error {
	d.log(fmt.Sprintf("Rename(%s → %s)", oldpath, newpath))
	return nil
}

func (d *DryRunner) Chmod(path string, mode os.FileMode) error {
	d.log(fmt.Sprintf("Chmod(%s, %o)", path, mode))
	return nil
}

func (d *DryRunner) Chown(path string, uid, gid int) error {
	d.log(fmt.Sprintf("Chown(%s, uid=%d, gid=%d)", path, uid, gid))
	return nil
}

func (d *DryRunner) MkdirAll(path string, perm os.FileMode) error {
	d.log(fmt.Sprintf("MkdirAll(%s, %o)", path, perm))
	return nil
}

func (d *DryRunner) Remove(path string) error {
	d.log("Remove(" + path + ")")
	return nil
}

func (d *DryRunner) RemoveAll(path string) error {
	d.log("RemoveAll(" + path + ")")
	return nil
}

func (d *DryRunner) Stat(path string) (os.FileInfo, error) {
	d.log("Stat(" + path + ")")
	return nil, os.ErrNotExist
}

func (d *DryRunner) LookupUser(username string) (uid, gid int, err error) {
	d.log("LookupUser(" + username + ")")
	if username == "root" {
		return 0, 0, nil
	}
	return 1000, 1000, nil
}
