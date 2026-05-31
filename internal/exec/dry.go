package exec

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type DryRunner struct {
	Stdout io.Writer
}

func NewDryRunner() *DryRunner {
	return &DryRunner{Stdout: os.Stderr}
}

func (d *DryRunner) IsDryRun() bool { return true }

func (d *DryRunner) log(cmd string) {
	_, _ = fmt.Fprintf(d.Stdout, "[DRY-RUN] %s\n", cmd)
}

func (d *DryRunner) Run(name string, args ...string) error {
	d.log(name + " " + strings.Join(args, " "))
	return nil
}

func (d *DryRunner) Output(name string, args ...string) (string, error) {
	d.log(name + " " + strings.Join(args, " "))

	switch name {
	case "dpkg":
		if len(args) > 0 && args[0] == "--print-architecture" {
			return "amd64", nil
		}
	case "getent":
		if len(args) > 0 && args[0] == "passwd" {
			return "user:x:1000:1000:User:/home/user:/bin/bash", nil
		}
	case "id":
		return "uid=1000(user) gid=1000(user) groups=1000(user)", nil
	}

	return "", nil
}

func (d *DryRunner) RunAsUser(user, name string, args ...string) error {
	allArgs := append([]string{"sudo", "-iu", user, "--", name}, args...)
	d.log(strings.Join(allArgs, " "))
	return nil
}

func (d *DryRunner) Shell(script string) error {
	d.log("bash -c '" + script + "'")
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
