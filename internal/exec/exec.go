package exec

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
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
	CombinedOutput(name string, args ...string) (string, error)
	RunAsUser(user, name string, args ...string) error
	Shell(script string) error
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

func (r *RealRunner) CombinedOutput(name string, args ...string) (string, error) {
	cmd := osexec.Command(name, args...)
	cmd.Env = r.Env
	out, err := cmd.CombinedOutput()
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

const DefaultTimezone = "Australia/Sydney"

func CheckCommand(cmd string) bool {
	_, err := osexec.LookPath(cmd)
	return err == nil
}

func PrintStep(msg string) {
	fmt.Fprintf(printWriter, "→ %s\n", msg)
}

func PrintDone(msg string) {
	fmt.Fprintf(printWriter, "✓ %s\n", msg)
}

func PrintError(msg string) {
	fmt.Fprintf(printWriter, "✗ %s\n", msg)
}
