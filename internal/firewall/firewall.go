package firewall

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type Rule struct {
	Port    string
	Proto   string
	From    string
	Comment string
}

func Install(runner setupexec.CmdRunner) error {
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "ufw")
}

func EnableBaseline(runner setupexec.CmdRunner, allowSSH bool) error {
	if err := Install(runner); err != nil {
		return err
	}
	if err := runner.Run("ufw", "default", "deny", "incoming"); err != nil {
		return err
	}
	if err := runner.Run("ufw", "default", "allow", "outgoing"); err != nil {
		return err
	}
	if allowSSH {
		if err := AllowSSH(runner); err != nil {
			return err
		}
	}
	return runner.Run("ufw", "--force", "enable")
}

func AllowSSH(runner setupexec.CmdRunner) error {
	port := DetectSSHPort(runner)
	return AllowRule(runner, Rule{Port: port, Proto: "tcp", Comment: "setup ssh"})
}

func DetectSSHPort(runner setupexec.CmdRunner) string {
	out, err := runner.Output("sshd", "-T")
	if err != nil {
		return "22"
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "port" && validPortSpec(fields[1]) == nil {
			return fields[1]
		}
	}
	return "22"
}

func AllowRule(runner setupexec.CmdRunner, rule Rule) error {
	if err := ValidateRule(rule); err != nil {
		return err
	}

	proto := normalizedProto(rule.Proto)
	var args []string
	if strings.TrimSpace(rule.From) != "" {
		args = append(args, "allow", "from", strings.TrimSpace(rule.From), "to", "any", "port", rule.Port)
		if proto != "" {
			args = append(args, "proto", proto)
		}
	} else {
		target := rule.Port
		if proto != "" {
			target += "/" + proto
		}
		args = append(args, "allow", target)
	}
	if strings.TrimSpace(rule.Comment) != "" {
		args = append(args, "comment", strings.TrimSpace(rule.Comment))
	}
	return runner.Run("ufw", args...)
}

func DeleteRule(runner setupexec.CmdRunner, number int) error {
	if number < 1 {
		return fmt.Errorf("rule number must be 1 or greater")
	}
	return runner.Run("ufw", "--force", "delete", strconv.Itoa(number))
}

func Reset(runner setupexec.CmdRunner) error {
	return runner.Run("ufw", "--force", "reset")
}

func Status(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("ufw", "status", "verbose")
}

func StatusNumbered(runner setupexec.CmdRunner) (string, error) {
	return runner.Output("ufw", "status", "numbered")
}

func ValidateRule(rule Rule) error {
	if err := validPortSpec(strings.TrimSpace(rule.Port)); err != nil {
		return err
	}
	if err := validProto(rule.Proto); err != nil {
		return err
	}
	if err := validSource(rule.From); err != nil {
		return err
	}
	if strings.ContainsAny(rule.Comment, "\r\n") {
		return fmt.Errorf("comment must be a single line")
	}
	if len(rule.Comment) > 80 {
		return fmt.Errorf("comment must be 80 characters or fewer")
	}
	return nil
}

func validPortSpec(port string) error {
	if port == "" {
		return fmt.Errorf("port is required")
	}
	parts := strings.Split(port, ":")
	if len(parts) > 2 {
		return fmt.Errorf("port must be a number or range")
	}
	prev := 0
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > 65535 {
			return fmt.Errorf("invalid port %q", port)
		}
		if i == 1 && n < prev {
			return fmt.Errorf("port range must be ascending")
		}
		prev = n
	}
	return nil
}

func validProto(proto string) error {
	switch normalizedProto(proto) {
	case "", "tcp", "udp":
		return nil
	default:
		return fmt.Errorf("protocol must be tcp or udp")
	}
}

func normalizedProto(proto string) string {
	proto = strings.ToLower(strings.TrimSpace(proto))
	if proto == "any" {
		return ""
	}
	return proto
}

func validSource(source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	if ip := net.ParseIP(source); ip != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(source); err == nil {
		return nil
	}
	return fmt.Errorf("source must be an IP address or CIDR")
}
