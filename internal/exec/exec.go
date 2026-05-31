package exec

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"os/user"
	"strconv"
	"strings"
)

var printWriter io.Writer = os.Stderr

func SetPrintWriter(w io.Writer) {
	printWriter = w
}

type dryRunner interface {
	IsDryRun() bool
}

func IsDryRun(r CmdRunner) bool {
	if d, ok := r.(dryRunner); ok {
		return d.IsDryRun()
	}
	return false
}

type CmdRunner interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
	RunAsUser(user, name string, args ...string) error
	Shell(script string) error

	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
	Rename(oldpath, newpath string) error
	Chmod(path string, mode os.FileMode) error
	Chown(path string, uid, gid int) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Stat(path string) (os.FileInfo, error)

	LookupUser(username string) (uid, gid int, err error)
}

type RealRunner struct {
	Env []string
}

func NewRealRunner() *RealRunner {
	return &RealRunner{
		Env: os.Environ(),
	}
}

func (r *RealRunner) Run(name string, args ...string) error {
	cmd := osexec.Command(name, args...)
	cmd.Env = r.Env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *RealRunner) Output(name string, args ...string) (string, error) {
	cmd := osexec.Command(name, args...)
	cmd.Env = r.Env
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *RealRunner) RunAsUser(user, name string, args ...string) error {
	allArgs := append([]string{"-iu", user, "--", name}, args...)
	return r.Run("sudo", allArgs...)
}

func (r *RealRunner) Shell(script string) error {
	cmd := osexec.Command("bash", "-c", script)
	cmd.Env = r.Env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *RealRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *RealRunner) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *RealRunner) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (r *RealRunner) Chmod(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

func (r *RealRunner) Chown(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}

func (r *RealRunner) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *RealRunner) Remove(path string) error {
	return os.Remove(path)
}

func (r *RealRunner) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (r *RealRunner) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (r *RealRunner) LookupUser(username string) (uid, gid int, err error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, 0, err
	}
	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse uid %q: %w", u.Uid, err)
	}
	gid, err = strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse gid %q: %w", u.Gid, err)
	}
	return uid, gid, nil
}

const DefaultTimezone = "UTC"

func CheckCommand(cmd string) bool {
	_, err := osexec.LookPath(cmd)
	return err == nil
}

func PrintStep(msg string) {
	_, _ = fmt.Fprintf(printWriter, "→ %s\n", msg)
}

func PrintDone(msg string) {
	_, _ = fmt.Fprintf(printWriter, "✓ %s\n", msg)
}

func PrintError(msg string) {
	_, _ = fmt.Fprintf(printWriter, "✗ %s\n", msg)
}
