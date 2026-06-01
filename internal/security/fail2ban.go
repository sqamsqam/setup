package security

import (
	"fmt"
	"net"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
)

const fail2banJailPath = "/etc/fail2ban/jail.d/setup-sshd.local"

type Fail2BanOptions struct {
	BanTime  string
	FindTime string
	MaxRetry int
}

func DefaultFail2BanOptions() Fail2BanOptions {
	return Fail2BanOptions{BanTime: "1h", FindTime: "10m", MaxRetry: 5}
}

func InstallFail2Ban(runner setupexec.CmdRunner, opts Fail2BanOptions) error {
	if opts.BanTime == "" {
		opts.BanTime = "1h"
	}
	if opts.FindTime == "" {
		opts.FindTime = "10m"
	}
	if opts.MaxRetry == 0 {
		opts.MaxRetry = 5
	}
	if opts.MaxRetry < 1 {
		return fmt.Errorf("max retry must be 1 or greater")
	}

	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	if err := runner.Run("apt", "install", "-y", "fail2ban"); err != nil {
		return err
	}

	changed, err := managed.WriteFileIfChanged(runner, fail2banJailPath, []byte(Fail2BanJailContent(opts)), 0644)
	if err != nil {
		return err
	}
	if err := runner.Run("systemctl", "enable", "--now", "fail2ban"); err != nil {
		return err
	}
	if changed {
		return runner.Run("systemctl", "restart", "fail2ban")
	}
	return nil
}

func Fail2BanJailContent(opts Fail2BanOptions) string {
	return managed.Marker + strings.TrimSpace(fmt.Sprintf(`
[sshd]
enabled = true
backend = systemd
bantime = %s
findtime = %s
maxretry = %d
`, opts.BanTime, opts.FindTime, opts.MaxRetry)) + "\n"
}

func Fail2BanStatus(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("fail2ban-client", "status", "sshd")
}

func UnbanIP(runner setupexec.CmdRunner, ip string) error {
	ip = strings.TrimSpace(ip)
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address %q", ip)
	}
	return runner.Run("fail2ban-client", "set", "sshd", "unbanip", ip)
}
