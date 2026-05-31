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
		{"Installing gh", func() error { return installGh(runner) }},
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
	if setupexec.IsDryRun(runner) {
		return nil
	}

	debURL, err := github.LatestReleaseAsset(repo, pattern)
	if err != nil {
		return fmt.Errorf("find release asset: %w", err)
	}

	// Extract original filename from download URL for checksum matching
	debName := debURL[strings.LastIndex(debURL, "/")+1:]
	tmpFile := "/tmp/" + repoToFilename(repo) + ".deb"

	if err := runner.Run("wget", "-q", debURL, "-O", tmpFile); err != nil {
		return fmt.Errorf("download %s: %w", repo, err)
	}
	defer os.Remove(tmpFile)

	if err := verifyDebChecksum(runner, repo, tmpFile, debName); err != nil {
		return err
	}

	if err := runner.Run("apt", "install", "-y", tmpFile); err != nil {
		return fmt.Errorf("install %s deb: %w", repo, err)
	}
	return nil
}

func repoToFilename(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}

func verifyDebChecksum(runner setupexec.CmdRunner, repo, debPath, debName string) error {
	checksumPatterns := []string{`SHA256SUMS$`, `sha256sums\.txt$`}
	var checksumURL string
	var err error
	for _, cp := range checksumPatterns {
		checksumURL, err = github.LatestReleaseAsset(repo, cp)
		if err == nil {
			break
		}
	}
	if err != nil {
		setupexec.PrintStep("Warning: no checksum file found, skipping verification for " + repo)
		return nil
	}

	tmpChecksum := debPath + ".sha256"
	if err := runner.Run("wget", "-q", checksumURL, "-O", tmpChecksum); err != nil {
		setupexec.PrintStep("Warning: could not download checksum file, skipping verification for " + repo)
		return nil
	}
	defer os.Remove(tmpChecksum)

	checksumContent, err := os.ReadFile(tmpChecksum)
	if err != nil {
		setupexec.PrintStep("Warning: could not read checksum file, skipping verification for " + repo)
		return nil
	}

	var expectedSHA string
	for _, line := range strings.Split(string(checksumContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "  "+debName) || strings.HasSuffix(line, " *"+debName) {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				expectedSHA = parts[0]
			}
			break
		}
	}
	if expectedSHA == "" {
		setupexec.PrintStep("Warning: no checksum entry found for " + debName + ", skipping verification")
		return nil
	}

	if err := runner.Shell(fmt.Sprintf("echo '%s  %s' | sha256sum -c --status", expectedSHA, debPath)); err != nil {
		return fmt.Errorf("checksum verification failed for %s: %w", repo, err)
	}
	return nil
}

func installYq(runner setupexec.CmdRunner) error {
	yqPath := "/usr/local/bin/yq"
	yqURL := "https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64"

	if err := runner.Run("wget", "-q", yqURL, "-O", yqPath); err != nil {
		return fmt.Errorf("download yq: %w", err)
	}

	if setupexec.IsDryRun(runner) {
		return nil
	}

	shaURL := yqURL + ".sha256"
	shaPath := yqPath + ".sha256"
	if err := runner.Run("wget", "-q", shaURL, "-O", shaPath); err != nil {
		return fmt.Errorf("download yq checksum: %w", err)
	}
	defer os.Remove(shaPath)

	shaContent, err := os.ReadFile(shaPath)
	if err != nil {
		return fmt.Errorf("read yq checksum: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(shaContent)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("empty yq checksum file")
	}
	parts := strings.Fields(lines[0])
	if len(parts) < 2 {
		return fmt.Errorf("invalid yq checksum format")
	}
	if err := runner.Shell(fmt.Sprintf("echo '%s  %s' | sha256sum -c --status", parts[0], yqPath)); err != nil {
		return fmt.Errorf("yq checksum verification failed")
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

	if !setupexec.IsDryRun(runner) {
		const charmFingerprint = "F506 F2D6 02D1 C400 A1E4  5D96 7E2E 87C7 1D5E 9D67"
		out, err := runner.Output("gpg", "--show-keys", "--with-fingerprint", "--with-colons", keyringPath)
		if err != nil {
			return fmt.Errorf("verify charm gpg key: %w", err)
		}
		normalizedExpected := strings.ReplaceAll(charmFingerprint, " ", "")
		found := false
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "fpr:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 10 && parts[9] == normalizedExpected {
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("charm gpg key fingerprint mismatch")
		}
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

func installGh(runner setupexec.CmdRunner) error {
	keyringPath := "/etc/apt/keyrings/githubcli-archive-keyring.gpg"
	listPath := "/etc/apt/sources.list.d/github-cli.list"

	if err := runner.Run("mkdir", "-p", "/etc/apt/keyrings"); err != nil {
		return err
	}

	if err := runner.Run("wget", "-nv", "-O", keyringPath, "https://cli.github.com/packages/githubcli-archive-keyring.gpg"); err != nil {
		return fmt.Errorf("download gh gpg key: %w", err)
	}

	if err := runner.Run("chmod", "go+r", keyringPath); err != nil {
		return err
	}

	arch, err := runner.Output("dpkg", "--print-architecture")
	if err != nil {
		return fmt.Errorf("get architecture: %w", err)
	}

	if err := runner.Run("mkdir", "-p", "/etc/apt/sources.list.d"); err != nil {
		return err
	}

	listContent := fmt.Sprintf("deb [arch=%s signed-by=%s] https://cli.github.com/packages stable main\n", arch, keyringPath)
	tmpList := "/tmp/github-cli.list"
	if err := os.WriteFile(tmpList, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("write temp github-cli.list: %w", err)
	}
	if err := runner.Run("mv", tmpList, listPath); err != nil {
		return err
	}

	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "gh")
}
