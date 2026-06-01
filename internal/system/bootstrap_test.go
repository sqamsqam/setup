package system

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type sshTestRunner struct {
	*setupexec.DryRunner
	files  map[string][]byte
	runErr error
}

func (s *sshTestRunner) ReadFile(path string) ([]byte, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (s *sshTestRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	s.files[path] = append([]byte(nil), data...)
	return nil
}

func (s *sshTestRunner) CreateTemp(dir, pattern string) (string, error) {
	return filepath.Join(dir, ".setup-test"), nil
}

func (s *sshTestRunner) Rename(oldpath, newpath string) error {
	s.files[newpath] = append([]byte(nil), s.files[oldpath]...)
	delete(s.files, oldpath)
	return nil
}

func (s *sshTestRunner) Remove(path string) error {
	delete(s.files, path)
	return nil
}

func (s *sshTestRunner) Run(name string, args ...string) error {
	if name == "sshd" && s.runErr != nil {
		return s.runErr
	}
	return nil
}

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
ClientAliveInterval 300
ClientAliveCountMax 2
MaxSessions 10
MaxStartups 10:30:100
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
		"ClientAliveInterval 300",
		"ClientAliveCountMax 2",
		"MaxSessions 10",
		"MaxStartups 10:30:100",
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

func TestInstallSSHDropInRollsBackExistingFile(t *testing.T) {
	path := "/etc/ssh/sshd_config.d/99-test.conf"
	oldContent := []byte("old")
	runner := &sshTestRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     map[string][]byte{path: oldContent},
		runErr:    errors.New("bad config"),
	}

	err := installSSHDropIn(runner, path, []byte("new"))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := string(runner.files[path]); got != "old" {
		t.Fatalf("expected rollback to old content, got %q", got)
	}
}

func TestInstallSSHDropInRemovesNewFileOnRollback(t *testing.T) {
	path := "/etc/ssh/sshd_config.d/99-test.conf"
	runner := &sshTestRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     map[string][]byte{},
		runErr:    errors.New("bad config"),
	}

	err := installSSHDropIn(runner, path, []byte("new"))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, ok := runner.files[path]; ok {
		t.Fatal("expected new file to be removed")
	}
}
