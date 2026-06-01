package diagnostics

import (
	"fmt"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type Check struct {
	Name   string
	Status string
	Detail string
}

type Report []Check

func Run(runner setupexec.CmdRunner) Report {
	var checks Report
	checks = append(checks,
		outputCheck(runner, "Virtualization", "systemd-detect-virt"),
		outputCheck(runner, "Ubuntu", "bash", "-c", `. /etc/os-release && echo "$PRETTY_NAME"`),
		outputCheck(runner, "Architecture", "dpkg", "--print-architecture"),
		outputCheck(runner, "Systemd", "systemctl", "is-system-running"),
		outputCheck(runner, "Cgroup filesystem", "stat", "-fc", "%T", "/sys/fs/cgroup"),
		outputCheck(runner, "Disk free", "df", "-h", "/"),
		outputCheck(runner, "Apt locks", "bash", "-c", `fuser /var/lib/dpkg/lock-frontend /var/lib/apt/lists/lock /var/cache/apt/archives/lock >/dev/null 2>&1 && echo locked || echo clear`),
		outputCheck(runner, "Reboot required", "bash", "-c", `test -f /var/run/reboot-required && cat /var/run/reboot-required || echo "not required"`),
		outputCheck(runner, "Failed units", "systemctl", "--failed", "--no-pager", "--plain"),
		outputCheck(runner, "Listening ports", "ss", "-tulpen"),
		runCheck(runner, "SSH config", "sshd", "-t"),
		outputCheck(runner, "UFW", "ufw", "status", "verbose"),
		outputCheck(runner, "Docker service", "systemctl", "is-active", "docker"),
		outputCheck(runner, "Docker disk", "docker", "system", "df"),
	)
	return checks
}

func Format(report Report) string {
	var b strings.Builder
	for _, check := range report {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s: %s", check.Name, check.Status)
		if strings.TrimSpace(check.Detail) != "" {
			b.WriteString("\n")
			b.WriteString(indent(strings.TrimSpace(check.Detail), "  "))
		}
	}
	return b.String()
}

func outputCheck(runner setupexec.CmdRunner, name, cmd string, args ...string) Check {
	out, err := runner.Output(cmd, args...)
	if err != nil {
		return Check{Name: name, Status: "warning", Detail: err.Error()}
	}
	out = strings.TrimSpace(out)
	if out == "" {
		out = "(no output)"
	}
	return Check{Name: name, Status: "ok", Detail: out}
}

func runCheck(runner setupexec.CmdRunner, name, cmd string, args ...string) Check {
	if err := runner.Run(cmd, args...); err != nil {
		return Check{Name: name, Status: "warning", Detail: err.Error()}
	}
	return Check{Name: name, Status: "ok"}
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
