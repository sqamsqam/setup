package tools

import (
	"fmt"
	"os"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/github"
)

func InstallAll(runner setupexec.CmdRunner) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Installing ripgrep", func() error { return installGitHubDeb(runner, "BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`) }},
		{"Installing fd", func() error { return installGitHubDeb(runner, "sharkdp/fd", `fd_.*_amd64\.deb$`) }},
		{"Installing bat", func() error { return installGitHubDeb(runner, "sharkdp/bat", `bat_.*_amd64\.deb$`) }},
		{"Installing yq", func() error { return installYq(runner) }},
		{"Installing glow", func() error { return installGlow(runner) }},
	}

	if err := ensureDeps(runner); err != nil {
		return err
	}

	for _, step := range steps {
		setupexec.PrintStep(step.name)
		if err := step.fn(); err != nil {
			setupexec.PrintError(step.name)
			return fmt.Errorf("%s: %w", step.name, err)
		}
		setupexec.PrintDone(step.name)
	}

	setupexec.PrintDone("CLI tools installed")
	return nil
}

func ensureDeps(runner setupexec.CmdRunner) error {
	setupexec.PrintStep("Updating package lists")
	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "curl", "wget", "jq", "gpg", "ca-certificates")
}

func installGitHubDeb(runner setupexec.CmdRunner, repo, pattern string) error {
	debURL, err := github.LatestReleaseAsset(repo, pattern)
	if err != nil {
		return fmt.Errorf("find release asset: %w", err)
	}

	tmpFile := "/tmp/" + repoToFilename(repo) + ".deb"

	if err := runner.Run("wget", "-q", debURL, "-O", tmpFile); err != nil {
		return fmt.Errorf("download %s: %w", repo, err)
	}
	defer os.Remove(tmpFile)

	if err := runner.Run("apt", "install", "-y", tmpFile); err != nil {
		return fmt.Errorf("install %s deb: %w", repo, err)
	}
	return nil
}

func repoToFilename(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}

type dryRunner interface {
	IsDryRun() bool
}

func isDryRun(runner setupexec.CmdRunner) bool {
	if dr, ok := runner.(dryRunner); ok {
		return dr.IsDryRun()
	}
	return false
}

func installYq(runner setupexec.CmdRunner) error {
	yqPath := "/usr/local/bin/yq"
	yqURL := "https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64"

	if err := runner.Run("wget", "-q", yqURL, "-O", yqPath); err != nil {
		return fmt.Errorf("download yq: %w", err)
	}

	if isDryRun(runner) {
		return nil
	}

	fi, err := os.Stat(yqPath)
	if err != nil {
		return fmt.Errorf("stat yq: %w", err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("downloaded yq file is empty")
	}
	return runner.Run("chmod", "+x", yqPath)
}

func installGlow(runner setupexec.CmdRunner) error {
	keyringPath := "/etc/apt/keyrings/charm.gpg"
	listPath := "/etc/apt/sources.list.d/charm.list"

	if err := runner.Run("mkdir", "-p", "/etc/apt/keyrings"); err != nil {
		return err
	}

	keyScript := fmt.Sprintf(
		"curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o %s",
		keyringPath,
	)
	if err := runner.Shell(keyScript); err != nil {
		return fmt.Errorf("download charm gpg key: %w", err)
	}

	listContent := fmt.Sprintf("deb [signed-by=%s] https://repo.charm.sh/apt/ * *\n", keyringPath)
	tmpList := "/tmp/charm.list"
	if err := os.WriteFile(tmpList, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("write temp charm.list: %w", err)
	}
	if err := runner.Run("mv", tmpList, listPath); err != nil {
		return err
	}

	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "glow")
}
