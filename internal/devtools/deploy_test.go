package devtools

import (
	"errors"
	"strings"
	"testing"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

type mockRunner struct {
	setupexec.CmdRunner
	outputFunc func(name string, args ...string) (string, error)
}

func (m *mockRunner) Output(name string, args ...string) (string, error) {
	return m.outputFunc(name, args...)
}

func TestInstallOptions(t *testing.T) {
	opts := AllInstallOptions()
	if !opts.Go || !opts.Node || !opts.Any() {
		t.Fatalf("expected all dev tools selected, got %#v", opts)
	}

	opts.Node = false
	if !opts.Any() {
		t.Fatal("go-only options should count as non-empty")
	}

	opts.Go = false
	if opts.Any() {
		t.Fatal("empty options should not count as non-empty")
	}
}

func TestParseGoReleaseValid(t *testing.T) {
	jsonData := `[
		{
			"version": "go1.22.0",
			"files": [
				{"os": "linux", "arch": "amd64", "kind": "archive", "sha256": "abc123deadbeef"},
				{"os": "linux", "arch": "arm64", "kind": "archive", "sha256": "def456"},
				{"os": "darwin", "arch": "amd64", "kind": "archive", "sha256": "ghi789"}
			]
		}
	]`

	version, sha256, err := parseGoRelease(jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "go1.22.0" {
		t.Errorf("expected version go1.22.0, got: %q", version)
	}
	if sha256 != "abc123deadbeef" {
		t.Errorf("expected sha256 abc123deadbeef, got: %q", sha256)
	}
}

func TestParseGoReleaseMissingLinuxAmd64(t *testing.T) {
	jsonData := `[
		{
			"version": "go1.22.0",
			"files": [
				{"os": "linux", "arch": "arm64", "kind": "archive", "sha256": "abc123"},
				{"os": "darwin", "arch": "amd64", "kind": "archive", "sha256": "def456"}
			]
		}
	]`

	_, _, err := parseGoRelease(jsonData)
	if err == nil {
		t.Fatal("expected error for missing linux/amd64 archive")
	}
	if !strings.Contains(err.Error(), "linux/amd64") {
		t.Errorf("expected error mentioning linux/amd64, got: %v", err)
	}
}

func TestParseGoReleaseEmptyArray(t *testing.T) {
	jsonData := "[]"

	_, _, err := parseGoRelease(jsonData)
	if err == nil {
		t.Fatal("expected error for empty releases array")
	}
	if !strings.Contains(err.Error(), "no Go releases") {
		t.Errorf("expected error 'no Go releases', got: %v", err)
	}
}

func TestParseGoReleaseWithBacktickWrappers(t *testing.T) {
	jsonData := "```\n[{\"version\":\"go1.22.0\",\"files\":[{\"os\":\"linux\",\"arch\":\"amd64\",\"kind\":\"archive\",\"sha256\":\"abc123deadbeef\"}]}]\n```"

	version, sha256, err := parseGoRelease(jsonData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "go1.22.0" {
		t.Errorf("expected version go1.22.0, got: %q", version)
	}
	if sha256 != "abc123deadbeef" {
		t.Errorf("expected sha256 abc123deadbeef, got: %q", sha256)
	}
}

func TestDetectShellBash(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "user:x:1000:1000:User:/home/user:/bin/bash", nil
		},
	}

	shell := detectShell(runner, "user")
	if shell != "bash" {
		t.Errorf("expected bash, got: %s", shell)
	}
}

func TestDetectShellZsh(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "user:x:1000:1000:User:/home/user:/bin/zsh", nil
		},
	}

	shell := detectShell(runner, "user")
	if shell != "zsh" {
		t.Errorf("expected zsh, got: %s", shell)
	}
}

func TestDetectShellGetentError(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "", errors.New("user not found")
		},
	}

	shell := detectShell(runner, "nonexistent")
	if shell != "bash" {
		t.Errorf("expected bash fallback, got: %s", shell)
	}
}

func TestDetectShellUnknown(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "user:x:1000:1000:User:/home/user:/bin/fish", nil
		},
	}

	shell := detectShell(runner, "user")
	if shell != "bash" {
		t.Errorf("expected bash fallback for unknown shell, got: %s", shell)
	}
}

func TestDetectShellCustomCheck(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "custom:x:1001:1001::/home/custom:/some/odd/path/zsh", nil
		},
	}

	shell := detectShell(runner, "custom")
	if shell != "zsh" {
		t.Errorf("expected zsh, got: %s", shell)
	}
}

func TestValidateTargetUserRejectsRoot(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "root:x:0:0:root:/root:/bin/bash", nil
		},
	}
	_, err := validateTargetUser(runner, "root")
	if err == nil {
		t.Fatal("expected root validation error")
	}
}

func TestValidateTargetUserUsesPasswdHome(t *testing.T) {
	runner := &mockRunner{
		outputFunc: func(name string, args ...string) (string, error) {
			return "dev:x:1000:1000:Dev:/srv/dev:/bin/bash", nil
		},
	}
	got, err := validateTargetUser(runner, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got.home != "/srv/dev" {
		t.Fatalf("expected /srv/dev, got %q", got.home)
	}
}
