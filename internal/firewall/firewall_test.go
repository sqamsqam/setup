package firewall

import (
	"bytes"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestValidateRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{name: "tcp port", rule: Rule{Port: "443", Proto: "tcp"}},
		{name: "udp range with cidr", rule: Rule{Port: "60000:61000", Proto: "udp", From: "10.0.0.0/24"}},
		{name: "bad port", rule: Rule{Port: "70000", Proto: "tcp"}, wantErr: true},
		{name: "bad proto", rule: Rule{Port: "443", Proto: "icmp"}, wantErr: true},
		{name: "bad source", rule: Rule{Port: "443", Proto: "tcp", From: "not-an-ip"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRule(tt.rule)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDetectSSHPortsReturnsAllValidPorts(t *testing.T) {
	runner := sshPortRunner{out: "port 22\nport 2222\npasswordauthentication no"}

	ports, err := DetectSSHPorts(runner)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"22", "2222"}
	if strings.Join(ports, ",") != strings.Join(want, ",") {
		t.Fatalf("ports = %v, want %v", ports, want)
	}
}

func TestEnableBaselineFailsWhenSSHDetectionFails(t *testing.T) {
	var buf bytes.Buffer
	runner := sshPortRunner{
		DryRunner: &setupexec.DryRunner{Stdout: &buf},
		err:       errSSHDetect,
	}

	err := EnableBaseline(runner, true)
	if err == nil {
		t.Fatal("expected SSH detection error")
	}
	if strings.Contains(buf.String(), "ufw --force enable") {
		t.Fatalf("should not enable UFW after detection failure: %s", buf.String())
	}
}

func TestAllowRuleBuildsCIDRCommand(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	err := AllowRule(runner, Rule{
		Port:    "443",
		Proto:   "tcp",
		From:    "10.0.0.0/24",
		Comment: "web",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	for _, want := range []string{"ufw allow from 10.0.0.0/24 to any port 443 proto tcp comment web"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

func TestEnableBaselineAllowsSSHBeforeEnable(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	if err := EnableBaseline(runner, true); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	allowIdx := strings.Index(got, "ufw allow 22/tcp")
	enableIdx := strings.Index(got, "ufw --force enable")
	if allowIdx == -1 || enableIdx == -1 {
		t.Fatalf("expected allow SSH and enable commands in %q", got)
	}
	if allowIdx > enableIdx {
		t.Fatalf("expected SSH allow before enable, got %q", got)
	}
}

func TestEnableBaselineAllowsExplicitSSHPort(t *testing.T) {
	var buf bytes.Buffer
	runner := sshPortRunner{
		DryRunner: &setupexec.DryRunner{Stdout: &buf},
		err:       errSSHDetect,
	}

	if err := EnableBaselineWithOptions(runner, EnableOptions{AllowSSH: true, SSHPorts: []string{"2222"}}); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "ufw allow 2222/tcp comment setup ssh") {
		t.Fatalf("expected explicit SSH allow: %s", got)
	}
}

func TestDenyAndLimitRulesBuildCommands(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	if err := DenyRule(runner, Rule{Port: "25", Proto: "tcp", From: "10.0.0.0/24", Comment: "mail"}); err != nil {
		t.Fatal(err)
	}
	if err := LimitRule(runner, Rule{Port: "22", Proto: "tcp", Comment: "ssh"}); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	for _, want := range []string{
		"ufw deny from 10.0.0.0/24 to any port 25 proto tcp comment mail",
		"ufw limit 22/tcp comment ssh",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

var errSSHDetect = errTest("sshd failed")

type errTest string

func (e errTest) Error() string {
	return string(e)
}

type sshPortRunner struct {
	*setupexec.DryRunner
	out string
	err error
}

func (s sshPortRunner) Output(name string, args ...string) (string, error) {
	if name == "sshd" && len(args) > 0 && args[0] == "-T" {
		if s.err != nil {
			return "", s.err
		}
		return s.out, nil
	}
	if s.DryRunner == nil {
		return "", nil
	}
	return s.DryRunner.Output(name, args...)
}
