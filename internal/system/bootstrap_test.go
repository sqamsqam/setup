package system

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

func TestGeneratedSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "99-hardening.conf")

	content := `# Managed by setup — do not edit
PermitRootLogin no
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
MaxAuthTries 3
LoginGraceTime 30
X11Forwarding no
`
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range []string{
		"Managed by setup",
		"PermitRootLogin no",
		"PubkeyAuthentication yes",
		"PasswordAuthentication no",
		"KbdInteractiveAuthentication no",
		"ChallengeResponseAuthentication no",
		"MaxAuthTries 3",
		"LoginGraceTime 30",
		"X11Forwarding no",
	} {
		if !strings.Contains(string(got), line) {
			t.Errorf("expected %q in output", line)
		}
	}
}

func TestUnattendedUpgradesConfig(t *testing.T) {
	content := "# Managed by setup — do not edit\n" + strings.TrimSpace(`
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
`) + "\n"

	expected := []string{
		"Managed by setup",
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
	content := "# Managed by setup — do not edit\nexport PATH=\"/usr/local/go/bin:$PATH\"\n"
	if !strings.Contains(content, "/usr/local/go/bin") {
		t.Errorf("expected /usr/local/go/bin in go profile script")
	}
	if !strings.HasPrefix(content, "# Managed by setup") {
		t.Errorf("expected managed marker in go profile script")
	}
}

func TestBootstrapWithDryRunner(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	err := Bootstrap(runner, "UTC")
	if err != nil {
		t.Fatalf("Bootstrap with dry runner returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run log output")
	}
	if !strings.Contains(output, "apt update") {
		t.Error("expected apt update in bootstrap steps")
	}
	if !strings.Contains(output, "locale-gen") {
		t.Error("expected locale-gen in bootstrap steps")
	}
	if !strings.Contains(output, "passwd -l root") {
		t.Error("expected lock root password in bootstrap steps")
	}
	if !strings.Contains(output, "sshd -t") {
		t.Error("expected sshd -t in bootstrap steps")
	}
	if !strings.Contains(output, "timedatectl set-timezone UTC") {
		t.Error("expected timezone setting in bootstrap steps")
	}
	if !strings.Contains(output, "systemctl enable --now ssh") {
		t.Error("expected enabling ssh in bootstrap steps")
	}
}
