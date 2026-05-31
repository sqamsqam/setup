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
	if !strings.Contains(output, "[DRY-RUN]") {
		t.Error("expected dry-run log output")
	}
	if !strings.Contains(output, "wget") {
		t.Error("expected wget command in output")
	}
}

func TestInstallGitHubDebDryRunShortCircuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"assets":[{"name":"ripgrep_14.1.0_amd64.deb","browser_download_url":"https://example.com/rg.deb"}]}`))
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

	err := installGitHubDeb(runner, "BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`)
	if err != nil {
		t.Fatalf("installGitHubDeb with dry runner returned error: %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("expected empty output for dry-run short-circuit, got: %q", output)
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
