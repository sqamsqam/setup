package managed

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type managedTestRunner struct {
	*setupexec.DryRunner
	files map[string][]byte
	ops   []string
	errs  map[string]error
}

func newManagedTestRunner() *managedTestRunner {
	return &managedTestRunner{
		DryRunner: setupexec.NewDryRunner(),
		files:     make(map[string][]byte),
		errs:      make(map[string]error),
	}
}

func (r *managedTestRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *managedTestRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, "write:"+path)
	if err := r.errs["write"]; err != nil {
		return err
	}
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *managedTestRunner) CreateTemp(dir, pattern string) (string, error) {
	r.ops = append(r.ops, "create-temp:"+dir)
	if err := r.errs["create-temp"]; err != nil {
		return "", err
	}
	path := filepath.Join(dir, ".setup-test")
	r.files[path] = nil
	return path, nil
}

func (r *managedTestRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	if err := r.errs["rename"]; err != nil {
		return err
	}
	data, ok := r.files[oldpath]
	if !ok {
		return os.ErrNotExist
	}
	r.files[newpath] = data
	delete(r.files, oldpath)
	return nil
}

func (r *managedTestRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, "chmod:"+path)
	if err := r.errs["chmod"]; err != nil {
		return err
	}
	return nil
}

func (r *managedTestRunner) MkdirAll(path string, perm os.FileMode) error {
	r.ops = append(r.ops, "mkdir:"+path)
	if err := r.errs["mkdir"]; err != nil {
		return err
	}
	return nil
}

func (r *managedTestRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

func TestWriteFileIfChangedSkipsIdenticalContent(t *testing.T) {
	runner := newManagedTestRunner()
	path := "/etc/example.conf"
	runner.files[path] = []byte("same")

	changed, err := WriteFileIfChanged(runner, path, []byte("same"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unchanged result")
	}
	for _, op := range runner.ops {
		if strings.HasPrefix(op, "write:") || strings.HasPrefix(op, "rename:") {
			t.Fatalf("unexpected write operation for unchanged file: %v", runner.ops)
		}
	}
}

func TestWriteFileIfChangedWritesAtomically(t *testing.T) {
	runner := newManagedTestRunner()
	path := "/etc/example.conf"

	changed, err := WriteFileIfChanged(runner, path, []byte("new"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed result")
	}
	if got := string(runner.files[path]); got != "new" {
		t.Fatalf("final file = %q, want new", got)
	}
	wantOrder := []string{"mkdir:/etc", "create-temp:/etc", "write:/etc/.setup-test", "chmod:/etc/.setup-test", "rename:/etc/.setup-test->/etc/example.conf", "remove:/etc/.setup-test"}
	for _, want := range wantOrder {
		if !containsOp(runner.ops, want) {
			t.Fatalf("missing operation %q from %v", want, runner.ops)
		}
	}
}

func TestWriteFileIfChangedCleansTempOnWriteError(t *testing.T) {
	runner := newManagedTestRunner()
	runner.errs["write"] = errors.New("disk full")

	changed, err := WriteFileIfChanged(runner, "/etc/example.conf", []byte("new"), 0644)
	if err == nil {
		t.Fatal("expected write error")
	}
	if changed {
		t.Fatal("expected changed=false on error")
	}
	if _, ok := runner.files["/etc/.setup-test"]; ok {
		t.Fatal("expected temp file cleanup")
	}
}

func containsOp(ops []string, want string) bool {
	for _, op := range ops {
		if op == want {
			return true
		}
	}
	return false
}
