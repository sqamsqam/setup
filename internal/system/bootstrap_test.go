package system

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

type dockerAptTestRunner struct {
	files           map[string][]byte
	ops             []string
	tempN           int
	fingerprint     string
	gpgErr          error
	codename        string
	verifiedKeyring string
}

func newDockerAptTestRunner() *dockerAptTestRunner {
	return &dockerAptTestRunner{
		files:       make(map[string][]byte),
		fingerprint: "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
		codename:    "resolute",
	}
}

func (r *dockerAptTestRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	if name == "curl" {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-o" {
				r.files[args[i+1]] = []byte("downloaded docker key")
				return nil
			}
		}
	}
	return nil
}

func (r *dockerAptTestRunner) Output(name string, args ...string) (string, error) {
	r.ops = append(r.ops, "output:"+name+" "+strings.Join(args, " "))
	switch name {
	case "gpg":
		if r.gpgErr != nil {
			return "", r.gpgErr
		}
		if len(args) > 0 {
			r.verifiedKeyring = args[len(args)-1]
		}
		return strings.Join([]string{"fpr", "", "", "", "", "", "", "", "", r.fingerprint, ""}, ":"), nil
	case "dpkg":
		return "amd64\n", nil
	case "bash":
		return r.codename + "\n", nil
	default:
		return "", nil
	}
}

func (r *dockerAptTestRunner) RunAsUser(user, name string, args ...string) error {
	r.ops = append(r.ops, "run-as-user:"+user+":"+name+" "+strings.Join(args, " "))
	return nil
}

func (r *dockerAptTestRunner) Shell(script string) error {
	r.ops = append(r.ops, "shell:"+script)
	return nil
}

func (r *dockerAptTestRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, "write:"+path)
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *dockerAptTestRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *dockerAptTestRunner) CreateTemp(dir, pattern string) (string, error) {
	r.tempN++
	path := filepath.Join(dir, strings.Replace(pattern, "*", strconv.Itoa(r.tempN), 1))
	r.ops = append(r.ops, "create-temp:"+path)
	r.files[path] = nil
	return path, nil
}

func (r *dockerAptTestRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	data, ok := r.files[oldpath]
	if !ok {
		return os.ErrNotExist
	}
	r.files[newpath] = data
	delete(r.files, oldpath)
	return nil
}

func (r *dockerAptTestRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, "chmod:"+path)
	if _, ok := r.files[path]; !ok {
		return os.ErrNotExist
	}
	return nil
}

func (r *dockerAptTestRunner) Chown(path string, uid, gid int) error {
	r.ops = append(r.ops, "chown:"+path)
	return nil
}

func (r *dockerAptTestRunner) MkdirAll(path string, perm os.FileMode) error {
	r.ops = append(r.ops, "mkdir:"+path)
	return nil
}

func (r *dockerAptTestRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

func (r *dockerAptTestRunner) RemoveAll(path string) error {
	r.ops = append(r.ops, "remove-all:"+path)
	for file := range r.files {
		if file == path || strings.HasPrefix(file, path+"/") {
			delete(r.files, file)
		}
	}
	return nil
}

func (r *dockerAptTestRunner) Stat(path string) (os.FileInfo, error) {
	r.ops = append(r.ops, "stat:"+path)
	return nil, os.ErrNotExist
}

func (r *dockerAptTestRunner) LookupUser(username string) (uid, gid int, err error) {
	r.ops = append(r.ops, "lookup-user:"+username)
	return 1000, 1000, nil
}

func (r *dockerAptTestRunner) renamedTo(path string) bool {
	for _, op := range r.ops {
		if strings.HasPrefix(op, "rename:") && strings.HasSuffix(op, "->"+path) {
			return true
		}
	}
	return false
}

func (r *dockerAptTestRunner) indexOf(prefix string) int {
	for i, op := range r.ops {
		if strings.HasPrefix(op, prefix) {
			return i
		}
	}
	return -1
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

func TestInstallDockerDoesNotOverwriteFinalKeyringOnFingerprintMismatch(t *testing.T) {
	runner := newDockerAptTestRunner()
	runner.files["/etc/apt/keyrings/docker.asc"] = []byte("existing trusted key")
	runner.fingerprint = "BADFINGERPRINT"

	err := installDocker(runner)
	if err == nil {
		t.Fatal("expected fingerprint verification error")
	}
	if !strings.Contains(err.Error(), "verify Docker GPG key") {
		t.Fatalf("expected contextual Docker key error, got %v", err)
	}
	if got := string(runner.files["/etc/apt/keyrings/docker.asc"]); got != "existing trusted key" {
		t.Fatalf("final keyring was overwritten with %q", got)
	}
	if runner.renamedTo("/etc/apt/keyrings/docker.asc") {
		t.Fatalf("renamed unverified temp keyring; operations: %v", runner.ops)
	}
	for path := range runner.files {
		if strings.Contains(path, ".setup-") {
			t.Fatalf("temp file %s was not cleaned up; operations: %v", path, runner.ops)
		}
	}
}

func TestInstallDockerVerifiesTempBeforeFinalRenameAndWritesResoluteSource(t *testing.T) {
	runner := newDockerAptTestRunner()
	runner.files["/etc/apt/keyrings/docker.asc"] = []byte("existing trusted key")

	if err := installDocker(runner); err != nil {
		t.Fatalf("installDocker returned error: %v\noperations: %v", err, runner.ops)
	}
	if got := string(runner.files["/etc/apt/keyrings/docker.asc"]); got != "downloaded docker key" {
		t.Fatalf("final keyring = %q, want downloaded docker key", got)
	}

	verifyAt := runner.indexOf("output:gpg")
	chmodAt := runner.indexOf("chmod:" + runner.verifiedKeyring)
	renameAt := runner.indexOf("rename:" + runner.verifiedKeyring + "->/etc/apt/keyrings/docker.asc")
	if verifyAt == -1 || chmodAt == -1 || renameAt == -1 {
		t.Fatalf("missing verify/chmod/rename operations: %v", runner.ops)
	}
	if verifyAt >= chmodAt || chmodAt >= renameAt {
		t.Fatalf("want verify before chmod before rename, got operations: %v", runner.ops)
	}

	source := string(runner.files["/etc/apt/sources.list.d/docker.sources"])
	for _, want := range []string{
		"Suites: resolute",
		"Architectures: amd64",
		"Signed-By: /etc/apt/keyrings/docker.asc",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("docker source missing %q:\n%s", want, source)
		}
	}
}

func TestInstallDockerRejectsEmptyCodename(t *testing.T) {
	runner := newDockerAptTestRunner()
	runner.codename = ""

	err := installDocker(runner)
	if err == nil {
		t.Fatal("expected empty codename error")
	}
	if !strings.Contains(err.Error(), "ubuntu codename is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyGPGFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
		outputErr   error
		expected    string
		wantErr     bool
	}{
		{
			name:        "match",
			fingerprint: "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			expected:    "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
		},
		{
			name:        "match with spaces",
			fingerprint: "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			expected:    "9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88",
		},
		{
			name:        "mismatch",
			fingerprint: "BADFINGERPRINT",
			expected:    "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			wantErr:     true,
		},
		{
			name:     "missing fingerprint",
			expected: "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			wantErr:  true,
		},
		{
			name:        "gpg error",
			outputErr:   errors.New("gpg failed"),
			expected:    "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			fingerprint: "9DC858229FC7DD38854AE2D88D81803C0EBFCD88",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newDockerAptTestRunner()
			runner.fingerprint = tt.fingerprint
			runner.gpgErr = tt.outputErr
			err := verifyGPGFingerprint(runner, "/tmp/key", tt.expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("verifyGPGFingerprint error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
