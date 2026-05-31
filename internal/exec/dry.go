package exec

import (
	"fmt"
	"os"
	"strings"
)

type DryRunner struct {
	Stdout *os.File
}

func NewDryRunner() *DryRunner {
	return &DryRunner{Stdout: os.Stderr}
}

func (d *DryRunner) IsDryRun() bool { return true }

func (d *DryRunner) log(cmd string) {
	fmt.Fprintf(d.Stdout, "[DRY-RUN] %s\n", cmd)
}

func (d *DryRunner) Run(name string, args ...string) error {
	d.log(name + " " + strings.Join(args, " "))
	return nil
}

func (d *DryRunner) Output(name string, args ...string) (string, error) {
	d.log(name + " " + strings.Join(args, " "))
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
