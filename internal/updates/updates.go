package updates

import (
	"fmt"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func Check(runner setupexec.CmdRunner) (string, error) {
	if err := runner.Run("apt", "update"); err != nil {
		return "", err
	}
	out, err := runner.Output("bash", "-c", `apt list --upgradable 2>/dev/null | sed -n '2,80p'`)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "No upgradable packages reported.", nil
	}
	return out, nil
}

func Upgrade(runner setupexec.CmdRunner) error {
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "full-upgrade", "-y")
}

func RebootRequired(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("bash", "-c", `if test -f /var/run/reboot-required; then cat /var/run/reboot-required; if test -f /var/run/reboot-required.pkgs; then echo; cat /var/run/reboot-required.pkgs; fi; else echo "Reboot not required."; fi`)
}

func UnattendedStatus(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("systemctl", "status", "unattended-upgrades", "--no-pager")
}

func FailedUnits(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("systemctl", "--failed", "--no-pager", "--plain")
}

func Reboot(runner setupexec.CmdRunner, yes bool) error {
	if !yes {
		return fmt.Errorf("refusing to reboot without explicit confirmation")
	}
	return runner.Run("systemctl", "reboot")
}
