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
