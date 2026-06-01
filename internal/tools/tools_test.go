package tools

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
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
