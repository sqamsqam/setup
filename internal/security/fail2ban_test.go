package security

import (
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestFail2BanJailContent(t *testing.T) {
	content := Fail2BanJailContent(DefaultFail2BanOptions())
	for _, want := range []string{
		"Managed by setup",
		"[sshd]",
		"enabled = true",
		"backend = systemd",
		"bantime = 1h",
		"findtime = 10m",
		"maxretry = 5",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in %q", want, content)
		}
	}
}

func TestUnbanIPValidatesAddress(t *testing.T) {
	if err := UnbanIP(setupexec.NewDryRunner(), "not-an-ip"); err == nil {
		t.Fatal("expected invalid IP error")
	}
}
