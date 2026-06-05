package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
)

type fail2banRunner struct {
	*setupexec.DryRunner
	files map[string][]byte
	ops   []string
}

func newFail2banRunner() *fail2banRunner {
	return &fail2banRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     make(map[string][]byte),
	}
}

func (r *fail2banRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	return nil
}

func (r *fail2banRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *fail2banRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, "write:"+path)
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *fail2banRunner) CreateTemp(dir, pattern string) (string, error) {
	path := filepath.Join(dir, ".setup-test")
	r.ops = append(r.ops, "create-temp:"+path)
	r.files[path] = nil
	return path, nil
}

func (r *fail2banRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	r.files[newpath] = append([]byte(nil), r.files[oldpath]...)
	delete(r.files, oldpath)
	return nil
}

func (r *fail2banRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, "chmod:"+path)
	return nil
}

func (r *fail2banRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

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

func TestInstallFail2BanRefusesUnmanagedJail(t *testing.T) {
	runner := newFail2banRunner()
	runner.files[fail2banJailPath] = []byte("[sshd]\nlocal = true\n")

	err := InstallFail2Ban(runner, DefaultFail2BanOptions())
	if err == nil {
		t.Fatal("expected unmanaged file error")
	}
	if got := string(runner.files[fail2banJailPath]); got != "[sshd]\nlocal = true\n" {
		t.Fatalf("jail file changed to %q", got)
	}
}

func TestInstallFail2BanSkipsUnchangedManagedJailValidationAndRestart(t *testing.T) {
	runner := newFail2banRunner()
	runner.files[fail2banJailPath] = []byte(Fail2BanJailContent(DefaultFail2BanOptions()))

	if err := InstallFail2Ban(runner, DefaultFail2BanOptions()); err != nil {
		t.Fatal(err)
	}
	for _, op := range runner.ops {
		if op == "run:fail2ban-client -t" || op == "run:systemctl restart fail2ban" {
			t.Fatalf("unexpected validation/restart for unchanged config: %v", runner.ops)
		}
	}
}

func TestInstallFail2BanValidatesChangedConfigBeforeRestart(t *testing.T) {
	runner := newFail2banRunner()
	runner.files[fail2banJailPath] = []byte(managed.Marker + "[sshd]\nmaxretry = 4\n")

	if err := InstallFail2Ban(runner, DefaultFail2BanOptions()); err != nil {
		t.Fatal(err)
	}
	validateAt := indexOp(runner.ops, "run:fail2ban-client -t")
	restartAt := indexOp(runner.ops, "run:systemctl restart fail2ban")
	if validateAt == -1 || restartAt == -1 || validateAt > restartAt {
		t.Fatalf("expected validation before restart: %v", runner.ops)
	}
}

func TestInstallFail2BanRejectsMultilineTimes(t *testing.T) {
	err := InstallFail2Ban(newFail2banRunner(), Fail2BanOptions{
		BanTime:  "1h\nignoreip = 0.0.0.0/0",
		FindTime: "10m",
		MaxRetry: 5,
	})
	if err == nil {
		t.Fatal("expected multiline option error")
	}
}

func indexOp(ops []string, want string) int {
	for i, op := range ops {
		if op == want {
			return i
		}
	}
	return -1
}
