package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/managed"
)

type serviceTestRunner struct {
	*setupexec.DryRunner
	dirs  map[string][]os.DirEntry
	files map[string][]byte
	ops   []string
	errs  map[string]error
	outs  map[string]string
}

func newServiceTestRunner() *serviceTestRunner {
	return &serviceTestRunner{
		DryRunner: setupexec.NewDryRunner(),
		dirs:      make(map[string][]os.DirEntry),
		files:     make(map[string][]byte),
		errs:      make(map[string]error),
		outs:      make(map[string]string),
	}
}

func (r *serviceTestRunner) Run(name string, args ...string) error {
	op := "run:" + name + " " + strings.Join(args, " ")
	r.ops = append(r.ops, op)
	return r.errs[op]
}

func (r *serviceTestRunner) Output(name string, args ...string) (string, error) {
	op := "output:" + name + " " + strings.Join(args, " ")
	r.ops = append(r.ops, op)
	if name == "getent" && len(args) >= 2 && args[0] == "passwd" {
		return args[1] + ":x:1000:1000:User:/home/" + args[1] + ":/bin/bash", nil
	}
	if out, ok := r.outs[op]; ok {
		return out, nil
	}
	return "", nil
}

func (r *serviceTestRunner) RunAsUser(user, name string, args ...string) error {
	r.ops = append(r.ops, "run-as-user:"+user+":"+name+" "+strings.Join(args, " "))
	return nil
}

func (r *serviceTestRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *serviceTestRunner) ReadDir(path string) ([]os.DirEntry, error) {
	r.ops = append(r.ops, "read-dir:"+path)
	entries, ok := r.dirs[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]os.DirEntry(nil), entries...), nil
}

func (r *serviceTestRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, "write:"+path)
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *serviceTestRunner) CreateTemp(dir, pattern string) (string, error) {
	path := filepath.Join(dir, ".setup-test")
	r.ops = append(r.ops, "create-temp:"+path)
	r.files[path] = nil
	return path, nil
}

func (r *serviceTestRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	r.files[newpath] = append([]byte(nil), r.files[oldpath]...)
	delete(r.files, oldpath)
	return nil
}

func (r *serviceTestRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, "chmod:"+path)
	return nil
}

func (r *serviceTestRunner) Chown(path string, uid, gid int) error {
	r.ops = append(r.ops, "chown:"+path)
	return nil
}

func (r *serviceTestRunner) MkdirAll(path string, perm os.FileMode) error {
	r.ops = append(r.ops, "mkdir:"+path)
	return nil
}

func (r *serviceTestRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

func TestUnitName(t *testing.T) {
	if got := UnitName("app"); got != "setup-app.service" {
		t.Fatalf("unexpected unit name: %s", got)
	}
	if got := UnitName("setup-api"); got != "setup-api.service" {
		t.Fatalf("unexpected prefixed unit name: %s", got)
	}
}

func TestUnitContent(t *testing.T) {
	content := UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
		EnvFile: "/home/dev/app/.env",
	})
	for _, want := range []string{
		"Managed by setup",
		"WorkingDirectory=\"/home/dev/app\"",
		"EnvironmentFile=-\"/home/dev/app/.env\"",
		"ExecStart=/bin/bash -lc \"npm start\"",
		"Restart=on-failure",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in %q", want, content)
		}
	}
}

func TestCreateRefusesUnmanagedExistingUnit(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	runner.files[path] = []byte("[Service]\nExecStart=/bin/true\n")

	err := Create(runner, Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	})
	if err == nil {
		t.Fatal("expected unmanaged unit error")
	}
	if got := string(runner.files[path]); got != "[Service]\nExecStart=/bin/true\n" {
		t.Fatalf("unit changed to %q", got)
	}
}

func TestCreateReplacesManagedExistingUnit(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	runner.files[path] = []byte(managed.Marker + "[Service]\nExecStart=/bin/true\n")

	err := Create(runner, Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(runner.files[path]); !strings.Contains(got, "ExecStart=/bin/bash -lc \"npm start\"") {
		t.Fatalf("unit not replaced with managed content:\n%s", got)
	}
	if !hasPrefixOp(runner.ops, "run:systemd-analyze verify /home/dev/.config/systemd/user/setup-app.service") {
		t.Fatalf("expected systemd unit validation: %v", runner.ops)
	}
	if !hasPrefixOp(runner.ops, "run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user restart setup-app.service") {
		t.Fatalf("expected restart after changed existing unit: %v", runner.ops)
	}
}

func TestUnitContentEscapesSystemdPercentSpecifiers(t *testing.T) {
	content := UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "date +%F",
	})
	if !strings.Contains(content, "date +%%F") {
		t.Fatalf("expected percent specifier to be escaped:\n%s", content)
	}
}

func TestCreateRollsBackInvalidUnit(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	old := []byte(UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "/bin/true",
	}))
	runner.files[path] = old
	runner.errs["run:systemd-analyze verify "+path] = os.ErrPermission

	err := Create(runner, Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := string(runner.files[path]); got != string(old) {
		t.Fatalf("expected rollback to old unit, got:\n%s", got)
	}
	if hasPrefixOp(runner.ops, "run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user enable") {
		t.Fatalf("should not start invalid unit: %v", runner.ops)
	}
}

func TestCreateReloadsAfterDaemonReloadRollback(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	old := []byte(UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "/bin/true",
	}))
	runner.files[path] = old
	reloadOp := "run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user daemon-reload"
	runner.errs[reloadOp] = os.ErrPermission

	err := Create(runner, Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	})
	if err == nil {
		t.Fatal("expected daemon-reload error")
	}
	if got := string(runner.files[path]); got != string(old) {
		t.Fatalf("expected rollback to old unit, got:\n%s", got)
	}
	if countPrefixOp(runner.ops, reloadOp) != 2 {
		t.Fatalf("expected daemon-reload before and after rollback: %v", runner.ops)
	}
}

func countPrefixOp(ops []string, want string) int {
	var count int
	for _, op := range ops {
		if strings.HasPrefix(op, want) {
			count++
		}
	}
	return count
}

func TestRestartRefusesUnmanagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	runner.files["/home/dev/.config/systemd/user/setup-app.service"] = []byte("[Service]\nExecStart=/bin/true\n")

	err := Restart(runner, "dev", "app")
	if err == nil {
		t.Fatal("expected unmanaged unit error")
	}
	for _, op := range runner.ops {
		if strings.HasPrefix(op, "run:sudo ") {
			t.Fatalf("unexpected service operation after unmanaged refusal: %v", runner.ops)
		}
	}
}

func TestRestartManagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	runner.files["/home/dev/.config/systemd/user/setup-app.service"] = []byte(UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	}))

	if err := Restart(runner, "dev", "app"); err != nil {
		t.Fatal(err)
	}
	if !hasPrefixOp(runner.ops, "run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user restart setup-app.service") {
		t.Fatalf("expected restart operation: %v", runner.ops)
	}
}

func TestRestartDryRunDoesNotRequireReadableUnitFile(t *testing.T) {
	if err := Restart(setupexec.NewDryRunner(), "dev", "app"); err != nil {
		t.Fatal(err)
	}
}

func TestListManagedUnits(t *testing.T) {
	runner := newServiceTestRunner()
	dir := "/home/dev/.config/systemd/user"
	runner.dirs[dir] = []os.DirEntry{
		testDirEntry{name: "setup-zed.service"},
		testDirEntry{name: "setup-app.service"},
		testDirEntry{name: "setup-unmanaged.service"},
		testDirEntry{name: "other.service"},
		testDirEntry{name: "setup-not-service.timer"},
		testDirEntry{name: "setup-dir.service", dir: true},
	}
	runner.files[filepath.Join(dir, "setup-zed.service")] = []byte(managed.Marker + "[Service]\nExecStart=/bin/true\n")
	runner.files[filepath.Join(dir, "setup-app.service")] = []byte(managed.Marker + "[Service]\nExecStart=/bin/true\n")
	runner.files[filepath.Join(dir, "setup-unmanaged.service")] = []byte("[Service]\nExecStart=/bin/true\n")

	units, err := List(runner, "dev")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(units, ",")
	if got != "setup-app.service,setup-zed.service" {
		t.Fatalf("units = %q", got)
	}
}

func TestListWithStateReportsManagedUnitStates(t *testing.T) {
	runner := newServiceTestRunner()
	dir := "/home/dev/.config/systemd/user"
	runner.dirs[dir] = []os.DirEntry{
		testDirEntry{name: "setup-app.service"},
		testDirEntry{name: "setup-worker.service"},
		testDirEntry{name: "setup-local.service"},
	}
	runner.files[filepath.Join(dir, "setup-app.service")] = []byte(managed.Marker + "[Service]\nExecStart=/bin/true\n")
	runner.files[filepath.Join(dir, "setup-worker.service")] = []byte(managed.Marker + "[Service]\nExecStart=/bin/true\n")
	runner.files[filepath.Join(dir, "setup-local.service")] = []byte("[Service]\nExecStart=/bin/true\n")
	runner.outs[showUnitOp("dev", "setup-app.service")] = strings.Join([]string{
		"LoadState=loaded",
		"ActiveState=active",
		"SubState=running",
		"UnitFileState=enabled",
	}, "\n")
	runner.outs[showUnitOp("dev", "setup-worker.service")] = strings.Join([]string{
		"LoadState=loaded",
		"ActiveState=failed",
		"SubState=failed",
		"UnitFileState=disabled",
	}, "\n")

	states, err := ListWithState(runner, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 managed units, got %#v", states)
	}
	if states[0] != (UnitState{Unit: "setup-app.service", LoadState: "loaded", ActiveState: "active", SubState: "running", UnitFileState: "enabled"}) {
		t.Fatalf("unexpected app state: %#v", states[0])
	}
	if states[1] != (UnitState{Unit: "setup-worker.service", LoadState: "loaded", ActiveState: "failed", SubState: "failed", UnitFileState: "disabled"}) {
		t.Fatalf("unexpected worker state: %#v", states[1])
	}
}

func TestListMissingUnitDirReturnsEmpty(t *testing.T) {
	runner := newServiceTestRunner()
	units, err := List(runner, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 0 {
		t.Fatalf("expected empty units, got %v", units)
	}
}

func TestDisableRefusesUnmanagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	runner.files["/home/dev/.config/systemd/user/setup-app.service"] = []byte("[Service]\nExecStart=/bin/true\n")

	err := Disable(runner, "dev", "app")
	if err == nil {
		t.Fatal("expected unmanaged unit error")
	}
	for _, op := range runner.ops {
		if strings.HasPrefix(op, "run:sudo ") {
			t.Fatalf("unexpected service operation after unmanaged refusal: %v", runner.ops)
		}
	}
}

func TestDisableManagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	runner.files["/home/dev/.config/systemd/user/setup-app.service"] = []byte(UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	}))

	if err := Disable(runner, "dev", "app"); err != nil {
		t.Fatal(err)
	}
	if !hasPrefixOp(runner.ops, "run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user disable --now setup-app.service") {
		t.Fatalf("expected disable operation: %v", runner.ops)
	}
}

func TestRemoveRefusesUnmanagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	runner.files[path] = []byte("[Service]\nExecStart=/bin/true\n")

	err := Remove(runner, "dev", "app")
	if err == nil {
		t.Fatal("expected unmanaged unit error")
	}
	if _, ok := runner.files[path]; !ok {
		t.Fatal("unmanaged unit should not be removed")
	}
	for _, op := range runner.ops {
		if strings.HasPrefix(op, "run:sudo ") || strings.HasPrefix(op, "remove:") {
			t.Fatalf("unexpected operation after unmanaged refusal: %v", runner.ops)
		}
	}
}

func TestRemoveManagedUnit(t *testing.T) {
	runner := newServiceTestRunner()
	path := "/home/dev/.config/systemd/user/setup-app.service"
	runner.files[path] = []byte(UnitContent(Config{
		User:    "dev",
		Name:    "app",
		WorkDir: "/home/dev/app",
		Command: "npm start",
	}))

	if err := Remove(runner, "dev", "app"); err != nil {
		t.Fatal(err)
	}
	if _, ok := runner.files[path]; ok {
		t.Fatal("expected unit to be removed")
	}
	for _, want := range []string{
		"run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user disable --now setup-app.service",
		"remove:/home/dev/.config/systemd/user/setup-app.service",
		"run:sudo -iu dev -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user daemon-reload",
	} {
		if !hasPrefixOp(runner.ops, want) {
			t.Fatalf("missing %q from %v", want, runner.ops)
		}
	}
}

func hasPrefixOp(ops []string, want string) bool {
	for _, op := range ops {
		if strings.HasPrefix(op, want) {
			return true
		}
	}
	return false
}

func showUnitOp(user, unit string) string {
	return "output:sudo -iu " + user + " -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user show " + unit + " --property=LoadState --property=ActiveState --property=SubState --property=UnitFileState --no-pager"
}

type testDirEntry struct {
	name string
	dir  bool
}

func (e testDirEntry) Name() string               { return e.name }
func (e testDirEntry) IsDir() bool                { return e.dir }
func (e testDirEntry) Type() os.FileMode          { return 0 }
func (e testDirEntry) Info() (os.FileInfo, error) { return nil, nil }
