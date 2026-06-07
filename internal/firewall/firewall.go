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

type EnableOptions struct {
	AllowSSH bool
	SSHPorts []string
}

func Install(runner setupexec.CmdRunner) error {
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "ufw")
}

func EnableBaseline(runner setupexec.CmdRunner, allowSSH bool) error {
	return EnableBaselineWithOptions(runner, EnableOptions{AllowSSH: allowSSH})
}

func EnableBaselineWithOptions(runner setupexec.CmdRunner, opts EnableOptions) error {
	var sshPorts []string
	if opts.AllowSSH {
		var err error
		sshPorts, err = SSHPortsForEnable(runner, opts.SSHPorts)
		if err != nil {
			return err
		}
	}

	if err := Install(runner); err != nil {
		return err
	}
	if err := runner.Run("ufw", "default", "deny", "incoming"); err != nil {
		return err
	}
	if err := runner.Run("ufw", "default", "allow", "outgoing"); err != nil {
		return err
	}
	if opts.AllowSSH {
		for _, port := range sshPorts {
			if err := AllowRule(runner, Rule{Port: port, Proto: "tcp", Comment: "setup ssh"}); err != nil {
				return err
			}
		}
	}
	return runner.Run("ufw", "--force", "enable")
}

func SSHPortsForEnable(runner setupexec.CmdRunner, ports []string) ([]string, error) {
	if len(ports) > 0 {
		normalized, err := normalizePorts(ports)
		if err != nil {
			return nil, err
		}
		return normalized, nil
	}

	detected, err := DetectSSHPorts(runner)
	if err != nil {
		return nil, fmt.Errorf("detect SSH port: %w; pass --ssh-port to override", err)
	}
	return detected, nil
}

func AllowSSH(runner setupexec.CmdRunner) error {
	ports, err := DetectSSHPorts(runner)
	if err != nil {
		return fmt.Errorf("detect SSH port: %w", err)
	}
	for _, port := range ports {
		if err := AllowRule(runner, Rule{Port: port, Proto: "tcp", Comment: "setup ssh"}); err != nil {
			return err
		}
	}
	return nil
}

func DetectSSHPorts(runner setupexec.CmdRunner) ([]string, error) {
	out, err := runner.Output("sshd", "-T")
	if err != nil {
		return nil, err
	}
	var ports []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "port" && validPortSpec(fields[1]) == nil {
			ports = append(ports, fields[1])
		}
	}
	ports, err = normalizePorts(ports)
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no valid SSH ports found in sshd -T output")
	}
	return ports, nil
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

func DenyRule(runner setupexec.CmdRunner, rule Rule) error {
	return ruleCommand(runner, "deny", rule)
}

func LimitRule(runner setupexec.CmdRunner, rule Rule) error {
	if strings.TrimSpace(rule.From) != "" {
		return fmt.Errorf("limit does not support source filtering")
	}
	return ruleCommand(runner, "limit", rule)
}

func ruleCommand(runner setupexec.CmdRunner, action string, rule Rule) error {
	if err := ValidateRule(rule); err != nil {
		return err
	}

	proto := normalizedProto(rule.Proto)
	var args []string
	if strings.TrimSpace(rule.From) != "" {
		args = append(args, action, "from", strings.TrimSpace(rule.From), "to", "any", "port", rule.Port)
		if proto != "" {
			args = append(args, "proto", proto)
		}
	} else {
		target := rule.Port
		if proto != "" {
			target += "/" + proto
		}
		args = append(args, action, target)
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

func Reload(runner setupexec.CmdRunner) error {
	return runner.Run("ufw", "reload")
}

func Disable(runner setupexec.CmdRunner) error {
	return runner.Run("ufw", "disable")
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

func normalizePorts(ports []string) ([]string, error) {
	seen := make(map[string]bool, len(ports))
	var normalized []string
	for _, port := range ports {
		port = strings.TrimSpace(port)
		if err := validPortSpec(port); err != nil {
			return nil, err
		}
		if seen[port] {
			continue
		}
		seen[port] = true
		normalized = append(normalized, port)
	}
	return normalized, nil
}
