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
	fmt.Fprintf(d.Stdout, "[DRY-RUN] %s\n", cmd)
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

func (d *DryRunner) CombinedOutput(name string, args ...string) (string, error) {
	d.log("CombinedOutput: " + name + " " + strings.Join(args, " "))
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
