package user

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type userFileRunner struct {
	*setupexec.DryRunner
	files map[string][]byte
	ops   []string
}

func newUserFileRunner() *userFileRunner {
	return &userFileRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     make(map[string][]byte),
	}
}

func (r *userFileRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	return nil
}

func (r *userFileRunner) Output(name string, args ...string) (string, error) {
	r.ops = append(r.ops, "output:"+name+" "+strings.Join(args, " "))
	if name == "awk" {
		return "dev\nops\n", nil
	}
	return r.DryRunner.Output(name, args...)
}

func (r *userFileRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *userFileRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, "write:"+path+":"+perm.String())
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *userFileRunner) CreateTemp(dir, pattern string) (string, error) {
	path := filepath.Join(dir, strings.Replace(pattern, "*", "test", 1))
	r.ops = append(r.ops, "create-temp:"+path)
	r.files[path] = nil
	return path, nil
}

func (r *userFileRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	data, ok := r.files[oldpath]
	if !ok {
		return os.ErrNotExist
	}
	r.files[newpath] = append([]byte(nil), data...)
	delete(r.files, oldpath)
	return nil
}

func (r *userFileRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, "chmod:"+path+":"+mode.String())
	return nil
}

func (r *userFileRunner) Chown(path string, uid, gid int) error {
	r.ops = append(r.ops, "chown:"+path)
	return nil
}

func (r *userFileRunner) MkdirAll(path string, perm os.FileMode) error {
	r.ops = append(r.ops, "mkdir:"+path)
	return nil
}

func (r *userFileRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

func TestWriteSudoersChmodsTempBeforeRename(t *testing.T) {
	runner := newUserFileRunner()

	if err := writeSudoers(runner, "dev"); err != nil {
		t.Fatal(err)
	}

	tmpPath := "/etc/sudoers.d/.setup-sudoers-test"
	renameOp := "rename:" + tmpPath + "->/etc/sudoers.d/dev"
	wantOpsBeforeRename := []string{
		"chmod:" + tmpPath + ":-r--r-----",
		"chown:" + tmpPath,
	}
	for _, want := range wantOpsBeforeRename {
		if indexOp(runner.ops, want) == -1 {
			t.Fatalf("missing operation %q from %v", want, runner.ops)
		}
		if indexOp(runner.ops, want) > indexOp(runner.ops, renameOp) {
			t.Fatalf("operation %q happened after rename: %v", want, runner.ops)
		}
	}
	if got := string(runner.files["/etc/sudoers.d/dev"]); got != "# Managed by setup — do not edit\ndev ALL=(ALL) NOPASSWD:ALL\n" {
		t.Fatalf("sudoers content = %q", got)
	}
}

func TestWriteSudoersSkipsTempWhenContentUnchanged(t *testing.T) {
	runner := newUserFileRunner()
	runner.files["/etc/sudoers.d/dev"] = []byte("# Managed by setup — do not edit\ndev ALL=(ALL) NOPASSWD:ALL\n")

	if err := writeSudoers(runner, "dev"); err != nil {
		t.Fatal(err)
	}

	for _, op := range runner.ops {
		if strings.HasPrefix(op, "create-temp:") || strings.HasPrefix(op, "write:") || strings.HasPrefix(op, "rename:") {
			t.Fatalf("unexpected write path for unchanged sudoers file: %v", runner.ops)
		}
	}
}

func TestUpdateAllowUsersChmodsTempBeforeRename(t *testing.T) {
	runner := newUserFileRunner()

	if err := updateAllowUsers(runner); err != nil {
		t.Fatal(err)
	}

	tmpPath := "/etc/ssh/sshd_config.d/.setup-allow-users-test"
	renameOp := "rename:" + tmpPath + "->/etc/ssh/sshd_config.d/98-allow-users.conf"
	chmodOp := "chmod:" + tmpPath + ":-rw-r--r--"
	if indexOp(runner.ops, chmodOp) == -1 {
		t.Fatalf("missing chmod operation %q from %v", chmodOp, runner.ops)
	}
	if indexOp(runner.ops, chmodOp) > indexOp(runner.ops, renameOp) {
		t.Fatalf("chmod happened after rename: %v", runner.ops)
	}

	got := string(runner.files["/etc/ssh/sshd_config.d/98-allow-users.conf"])
	if got != "# Managed by setup — do not edit\nAllowUsers dev ops\n" {
		t.Fatalf("AllowUsers content = %q", got)
	}
	if indexOp(runner.ops, "run:sshd -t") > indexOp(runner.ops, "run:systemctl restart ssh") {
		t.Fatalf("sshd validation should happen before restart: %v", runner.ops)
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
