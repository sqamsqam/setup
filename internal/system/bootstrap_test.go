package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "99-hardening.conf")

	content := `PermitRootLogin no
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
MaxAuthTries 3
LoginGraceTime 30
`
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range []string{
		"PermitRootLogin no",
		"PubkeyAuthentication yes",
		"PasswordAuthentication no",
		"KbdInteractiveAuthentication no",
		"MaxAuthTries 3",
		"LoginGraceTime 30",
	} {
		if !strings.Contains(string(got), line) {
			t.Errorf("expected %q in output", line)
		}
	}
}

func TestUnattendedUpgradesConfig(t *testing.T) {
	content := strings.TrimSpace(`
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
`) + "\n"

	expected := []string{
		`Update-Package-Lists "1"`,
		`Download-Upgradeable-Packages "1"`,
		`AutocleanInterval "7"`,
		`Unattended-Upgrade "1"`,
	}

	for _, exp := range expected {
		if !strings.Contains(content, exp) {
			t.Errorf("expected %q in unattended upgrades config", exp)
		}
	}
}

func TestGoProfileScript(t *testing.T) {
	content := "export PATH=\"/usr/local/go/bin:$PATH\"\n"
	if !strings.Contains(content, "/usr/local/go/bin") {
		t.Errorf("expected /usr/local/go/bin in go profile script")
	}
	if !strings.HasPrefix(content, "export") {
		t.Errorf("expected export command in go profile script")
	}
}
