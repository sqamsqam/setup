package tools

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/github"
)

func TestRepoToFilename(t *testing.T) {
	tests := []struct {
		repo string
		want string
	}{
		{"BurntSushi/ripgrep", "BurntSushi-ripgrep"},
		{"sharkdp/fd", "sharkdp-fd"},
		{"sharkdp/bat", "sharkdp-bat"},
		{"single", "single"},
	}

	for _, tt := range tests {
		got := repoToFilename(tt.repo)
		if got != tt.want {
			t.Errorf("repoToFilename(%q) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestInstallOptionsSelectedTools(t *testing.T) {
	opts := InstallOptions{Yq: true, Gh: true}
	got := opts.SelectedTools()
	want := []Tool{ToolYq, ToolGh}
	if len(got) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tool %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestInstallSelectedYqDryRun(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	err := InstallSelected(runner, InstallOptions{Yq: true})
	if err != nil {
		t.Fatalf("InstallSelected with dry runner returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "apt update") {
		t.Error("expected dependency apt update in output")
	}
	if !strings.Contains(output, "yq_linux_amd64") {
		t.Errorf("expected yq download in dry-run output, got: %q", output)
	}
	if strings.Contains(output, "ripgrep") {
		t.Errorf("did not expect unselected ripgrep command, got: %q", output)
	}
}

func TestInstallToolRejectsUnknownTool(t *testing.T) {
	runner := &setupexec.DryRunner{Stdout: io.Discard}
	err := InstallTool(runner, Tool("unknown"))
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestInstallYqWithDryRunner(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	err := installYq(runner)
	if err != nil {
		t.Fatalf("installYq with dry runner returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "yq_linux_amd64") {
		t.Errorf("expected yq download in dry-run output, got: %q", output)
	}
	if !strings.Contains(output, "checksums") {
		t.Errorf("expected yq checksum download in dry-run output, got: %q", output)
	}
}

func TestInstallYqCleansChecksumTempOnDownloadError(t *testing.T) {
	runner := newYqCleanupRunner("checksums")

	err := installYq(runner)
	if err == nil {
		t.Fatal("expected checksum download error")
	}
	for _, want := range []string{
		"remove:/usr/local/bin/.setup-yq-1",
		"remove:/usr/local/bin/.setup-yq-2.sha256",
	} {
		if !containsString(runner.ops, want) {
			t.Fatalf("missing cleanup operation %q from %v", want, runner.ops)
		}
	}
}

func TestInstallYqCleansOrderTempOnDownloadError(t *testing.T) {
	runner := newYqCleanupRunner("checksums_hashes_order")

	err := installYq(runner)
	if err == nil {
		t.Fatal("expected checksum order download error")
	}
	for _, want := range []string{
		"remove:/usr/local/bin/.setup-yq-1",
		"remove:/usr/local/bin/.setup-yq-2.sha256",
		"remove:/usr/local/bin/.setup-yq-order-3",
	} {
		if !containsString(runner.ops, want) {
			t.Fatalf("missing cleanup operation %q from %v", want, runner.ops)
		}
	}
}

func TestInstallGitHubDebDryRunShortCircuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets":[{"name":"ripgrep_14.1.0_amd64.deb","browser_download_url":"https://example.com/rg.deb"}]}`))
	}))
	defer server.Close()

	github.SetAPIBase(server.URL)
	github.SetHTTPClient(server.Client())
	t.Cleanup(func() {
		github.SetAPIBase("https://api.github.com")
		github.SetHTTPClient(&http.Client{Timeout: 30 * time.Second})
	})
	t.Setenv("GITHUB_TOKEN", "")

	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	err := installGitHubDeb(runner, "ripgrep", "ripgrep", "BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`)
	if err != nil {
		t.Fatalf("installGitHubDeb with dry runner returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "apt install -y ripgrep") {
		t.Errorf("expected dry-run package install, got: %q", output)
	}
}

func TestInstallGitHubDebCleansTempOnDownloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets":[{"name":"ripgrep_14.1.0_amd64.deb","browser_download_url":"https://example.com/rg.deb"}]}`))
	}))
	defer server.Close()

	github.SetAPIBase(server.URL)
	github.SetHTTPClient(server.Client())
	t.Cleanup(func() {
		github.SetAPIBase("https://api.github.com")
		github.SetHTTPClient(&http.Client{Timeout: 30 * time.Second})
	})
	t.Setenv("GITHUB_TOKEN", "")

	runner := newGitHubDebFailureRunner()
	err := installGitHubDeb(runner, "ripgrep", "ripgrep", "BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`)
	if err == nil {
		t.Fatal("expected download error")
	}
	if runner.debPath == "" {
		t.Fatalf("did not capture deb path; operations: %v", runner.ops)
	}
	if !containsString(runner.ops, "remove:"+runner.debPath) {
		t.Fatalf("missing temp deb cleanup for %s; operations: %v", runner.debPath, runner.ops)
	}
}

type githubDebFailureRunner struct {
	*aptKeyTestRunner
	debPath string
}

func newGitHubDebFailureRunner() *githubDebFailureRunner {
	return &githubDebFailureRunner{aptKeyTestRunner: newAptKeyTestRunner()}
}

func (r *githubDebFailureRunner) Output(name string, args ...string) (string, error) {
	r.ops = append(r.ops, "output:"+name+" "+strings.Join(args, " "))
	if name == "dpkg-query" {
		return "", errors.New("not installed")
	}
	return "", nil
}

func (r *githubDebFailureRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	if name == "wget" {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-O" {
				r.debPath = args[i+1]
			}
		}
		return errors.New("download failed")
	}
	return nil
}

func (r *githubDebFailureRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return os.Remove(path)
}

func TestEnsureDepsWithDryRunner(t *testing.T) {
	var buf bytes.Buffer
	runner := &setupexec.DryRunner{Stdout: &buf}

	setupexec.SetPrintWriter(io.Discard)

	err := ensureDeps(runner)
	if err != nil {
		t.Fatalf("ensureDeps with dry runner returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "apt update") {
		t.Error("expected apt update in output")
	}
	if !strings.Contains(output, "apt install") {
		t.Error("expected apt install in output")
	}
}

func TestChecksumForAsset(t *testing.T) {
	order := "CRC32\nSHA-1\nSHA-256\n"
	checksums := "yq_linux_amd64 deadbeef abcdef 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
	got, err := checksumForAsset(checksums, order, "yq_linux_amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if got != want {
		t.Fatalf("checksumForAsset = %q, want %q", got, want)
	}
}

func TestInstallAptKeyDoesNotOverwriteFinalKeyringOnFingerprintMismatch(t *testing.T) {
	tests := []struct {
		name      string
		install   func(setupexec.CmdRunner) error
		finalPath string
	}{
		{
			name:      "glow",
			install:   installGlow,
			finalPath: "/etc/apt/keyrings/charm.gpg",
		},
		{
			name:      "gh",
			install:   installGh,
			finalPath: "/etc/apt/keyrings/githubcli-archive-keyring.gpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newAptKeyTestRunner()
			runner.files[tt.finalPath] = []byte("existing trusted key")
			runner.fingerprint = "BADFINGERPRINT"

			err := tt.install(runner)
			if err == nil {
				t.Fatal("expected fingerprint verification error")
			}
			if got := string(runner.files[tt.finalPath]); got != "existing trusted key" {
				t.Fatalf("final keyring was overwritten with %q", got)
			}
			if runner.renamedTo(tt.finalPath) {
				t.Fatalf("renamed unverified temp keyring to %s; operations: %v", tt.finalPath, runner.ops)
			}
			for path := range runner.files {
				if strings.Contains(path, ".setup-") {
					t.Fatalf("temp file %s was not cleaned up; operations: %v", path, runner.ops)
				}
			}
		})
	}
}

func TestInstallAptKeyVerifiesTempBeforeFinalRename(t *testing.T) {
	tests := []struct {
		name        string
		install     func(setupexec.CmdRunner) error
		finalPath   string
		sourcePath  string
		sourceWants []string
		fingerprint string
	}{
		{
			name:        "glow",
			install:     installGlow,
			finalPath:   "/etc/apt/keyrings/charm.gpg",
			sourcePath:  "/etc/apt/sources.list.d/charm.list",
			sourceWants: []string{"deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *"},
			fingerprint: "F506F2D602D1C400A1E45D967E2E87C71D5E9D67",
		},
		{
			name:        "gh",
			install:     installGh,
			finalPath:   "/etc/apt/keyrings/githubcli-archive-keyring.gpg",
			sourcePath:  "/etc/apt/sources.list.d/github-cli.list",
			sourceWants: []string{"deb [arch=amd64 signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main"},
			fingerprint: "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newAptKeyTestRunner()
			runner.files[tt.finalPath] = []byte("existing trusted key")
			runner.fingerprint = tt.fingerprint

			if err := tt.install(runner); err != nil {
				t.Fatalf("install returned error: %v\noperations: %v", err, runner.ops)
			}
			if got := string(runner.files[tt.finalPath]); got != "downloaded key" {
				t.Fatalf("final keyring = %q, want downloaded key", got)
			}

			verifyAt := runner.indexOf("output:gpg")
			chmodAt := runner.indexOf("chmod:" + runner.verifiedKeyring)
			renameAt := runner.indexOf("rename:" + runner.verifiedKeyring + "->" + tt.finalPath)
			if verifyAt == -1 || chmodAt == -1 || renameAt == -1 {
				t.Fatalf("missing verify/chmod/rename operations: %v", runner.ops)
			}
			if verifyAt >= chmodAt || chmodAt >= renameAt {
				t.Fatalf("want verify before chmod before rename, got operations: %v", runner.ops)
			}

			source := string(runner.files[tt.sourcePath])
			for _, want := range tt.sourceWants {
				if !strings.Contains(source, want) {
					t.Fatalf("%s source missing %q:\n%s", tt.name, want, source)
				}
			}

			sourceTemp, sourceRenameAt := runner.renamedFrom(tt.sourcePath)
			if sourceTemp == "" {
				t.Fatalf("missing source list rename to %s; operations: %v", tt.sourcePath, runner.ops)
			}
			sourceWriteAt := runner.indexOf("write:" + sourceTemp + ":644")
			sourceChmodAt := runner.indexOf("chmod:" + sourceTemp + ":644")
			if sourceWriteAt == -1 || sourceChmodAt == -1 {
				t.Fatalf("missing source list write/chmod operations for %s; operations: %v", sourceTemp, runner.ops)
			}
			if sourceWriteAt >= sourceChmodAt || sourceChmodAt >= sourceRenameAt {
				t.Fatalf("want source list write before chmod before rename, got operations: %v", runner.ops)
			}
		})
	}
}

func TestVerifyAptGPGFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
		outputErr   error
		expected    string
		wantErr     bool
	}{
		{
			name:        "match",
			fingerprint: "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			expected:    "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
		},
		{
			name:        "match with spaces",
			fingerprint: "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			expected:    "23F3 D1D8 6577 3DE1 7D9D  8C30 A7B6 3A2B 8F85 411F",
		},
		{
			name:        "mismatch",
			fingerprint: "BADFINGERPRINT",
			expected:    "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			wantErr:     true,
		},
		{
			name:     "missing fingerprint",
			expected: "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			wantErr:  true,
		},
		{
			name:        "gpg error",
			fingerprint: "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			outputErr:   errors.New("gpg failed"),
			expected:    "23F3D1D865773DE17D9D8C30A7B63A2B8F85411F",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newAptKeyTestRunner()
			runner.fingerprint = tt.fingerprint
			runner.gpgErr = tt.outputErr
			err := verifyGPGFingerprint(runner, "/tmp/key", tt.expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("verifyGPGFingerprint error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type aptKeyTestRunner struct {
	files           map[string][]byte
	ops             []string
	tempN           int
	fingerprint     string
	gpgErr          error
	verifiedKeyring string
}

func newAptKeyTestRunner() *aptKeyTestRunner {
	return &aptKeyTestRunner{files: make(map[string][]byte)}
}

func (r *aptKeyTestRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	if name == "wget" {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-O" {
				r.files[args[i+1]] = []byte("downloaded key")
				return nil
			}
		}
	}
	return nil
}

func (r *aptKeyTestRunner) Output(name string, args ...string) (string, error) {
	r.ops = append(r.ops, "output:"+name+" "+strings.Join(args, " "))
	switch name {
	case "gpg":
		if r.gpgErr != nil {
			return "", r.gpgErr
		}
		if len(args) == 0 {
			return "", errors.New("missing gpg args")
		}
		keyringPath := args[len(args)-1]
		r.verifiedKeyring = keyringPath
		return strings.Join([]string{"fpr", "", "", "", "", "", "", "", "", r.fingerprint, ""}, ":"), nil
	case "dpkg":
		return "amd64\n", nil
	default:
		return "", nil
	}
}

func (r *aptKeyTestRunner) RunAsUser(user, name string, args ...string) error {
	r.ops = append(r.ops, "run-as-user:"+user+":"+name+" "+strings.Join(args, " "))
	return nil
}

func (r *aptKeyTestRunner) Shell(script string) error {
	r.ops = append(r.ops, "shell:"+script)
	const marker = " -o "
	idx := strings.LastIndex(script, marker)
	if idx == -1 {
		return fmt.Errorf("script missing output path: %s", script)
	}
	path := strings.TrimSpace(script[idx+len(marker):])
	path = strings.Trim(path, "'")
	r.files[path] = []byte("downloaded key")
	return nil
}

func (r *aptKeyTestRunner) WriteFile(path string, data []byte, perm os.FileMode) error {
	r.ops = append(r.ops, fmt.Sprintf("write:%s:%o", path, perm))
	r.files[path] = append([]byte(nil), data...)
	return nil
}

func (r *aptKeyTestRunner) ReadFile(path string) ([]byte, error) {
	r.ops = append(r.ops, "read:"+path)
	data, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (r *aptKeyTestRunner) ReadDir(path string) ([]os.DirEntry, error) {
	r.ops = append(r.ops, "read-dir:"+path)
	return nil, os.ErrNotExist
}

func (r *aptKeyTestRunner) CreateTemp(dir, pattern string) (string, error) {
	r.tempN++
	path := filepath.Join(dir, strings.Replace(pattern, "*", fmt.Sprintf("%d", r.tempN), 1))
	r.ops = append(r.ops, "create-temp:"+path)
	r.files[path] = nil
	return path, nil
}

func (r *aptKeyTestRunner) Rename(oldpath, newpath string) error {
	r.ops = append(r.ops, "rename:"+oldpath+"->"+newpath)
	data, ok := r.files[oldpath]
	if !ok {
		return os.ErrNotExist
	}
	r.files[newpath] = data
	delete(r.files, oldpath)
	return nil
}

func (r *aptKeyTestRunner) Chmod(path string, mode os.FileMode) error {
	r.ops = append(r.ops, fmt.Sprintf("chmod:%s:%o", path, mode))
	if _, ok := r.files[path]; !ok {
		return os.ErrNotExist
	}
	return nil
}

func (r *aptKeyTestRunner) Chown(path string, uid, gid int) error {
	r.ops = append(r.ops, fmt.Sprintf("chown:%s:%d:%d", path, uid, gid))
	return nil
}

func (r *aptKeyTestRunner) MkdirAll(path string, perm os.FileMode) error {
	r.ops = append(r.ops, fmt.Sprintf("mkdir:%s:%o", path, perm))
	return nil
}

func (r *aptKeyTestRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	delete(r.files, path)
	return nil
}

func (r *aptKeyTestRunner) RemoveAll(path string) error {
	r.ops = append(r.ops, "remove-all:"+path)
	for file := range r.files {
		if file == path || strings.HasPrefix(file, path+"/") {
			delete(r.files, file)
		}
	}
	return nil
}

func (r *aptKeyTestRunner) Stat(path string) (os.FileInfo, error) {
	r.ops = append(r.ops, "stat:"+path)
	return nil, os.ErrNotExist
}

func (r *aptKeyTestRunner) LookupUser(username string) (uid, gid int, err error) {
	r.ops = append(r.ops, "lookup-user:"+username)
	return 1000, 1000, nil
}

func (r *aptKeyTestRunner) renamedTo(path string) bool {
	for _, op := range r.ops {
		if strings.HasPrefix(op, "rename:") && strings.HasSuffix(op, "->"+path) {
			return true
		}
	}
	return false
}

func (r *aptKeyTestRunner) renamedFrom(path string) (string, int) {
	for i, op := range r.ops {
		if strings.HasPrefix(op, "rename:") && strings.HasSuffix(op, "->"+path) {
			return strings.TrimSuffix(strings.TrimPrefix(op, "rename:"), "->"+path), i
		}
	}
	return "", -1
}

func (r *aptKeyTestRunner) indexOf(prefix string) int {
	for i, op := range r.ops {
		if strings.HasPrefix(op, prefix) {
			return i
		}
	}
	return -1
}

type yqCleanupRunner struct {
	*setupexec.DryRunner
	ops        []string
	tempN      int
	failNeedle string
}

func newYqCleanupRunner(failNeedle string) *yqCleanupRunner {
	return &yqCleanupRunner{
		DryRunner:  setupexec.NewDryRunner(),
		failNeedle: failNeedle,
	}
}

func (r *yqCleanupRunner) Run(name string, args ...string) error {
	r.ops = append(r.ops, "run:"+name+" "+strings.Join(args, " "))
	if name == "wget" && strings.Contains(strings.Join(args, " "), r.failNeedle) {
		return errors.New("download failed")
	}
	return nil
}

func (r *yqCleanupRunner) CreateTemp(dir, pattern string) (string, error) {
	r.tempN++
	path := filepath.Join(dir, strings.Replace(pattern, "*", fmt.Sprintf("%d", r.tempN), 1))
	r.ops = append(r.ops, "create-temp:"+path)
	return path, nil
}

func (r *yqCleanupRunner) Remove(path string) error {
	r.ops = append(r.ops, "remove:"+path)
	return nil
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
