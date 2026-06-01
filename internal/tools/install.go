package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/github"
)

func InstallAll(runner setupexec.CmdRunner) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Installing ripgrep", func() error {
			return installGitHubDeb(runner, "ripgrep", "ripgrep", "BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`)
		}},
		{"Installing fd", func() error { return installGitHubDeb(runner, "fd", "fd-find", "sharkdp/fd", `fd_.*_amd64\.deb$`) }},
		{"Installing bat", func() error { return installGitHubDeb(runner, "bat", "bat", "sharkdp/bat", `bat_.*_amd64\.deb$`) }},
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

func installGitHubDeb(runner setupexec.CmdRunner, commandName, aptPackage, repo, pattern string) error {
	if setupexec.IsDryRun(runner) {
		if err := runner.Run("apt", "install", "-y", aptPackage); err != nil {
			return err
		}
		return setupDebianAliases(runner, commandName)
	}

	if version, err := runner.Output("dpkg-query", "-W", "-f=${Version}", aptPackage); err == nil && strings.TrimSpace(version) != "" {
		setupexec.PrintStep(fmt.Sprintf("%s already installed (%s), skipping GitHub .deb", aptPackage, version))
		return setupDebianAliases(runner, commandName)
	}

	debURL, err := github.LatestReleaseAsset(repo, pattern)
	if err != nil {
		return fmt.Errorf("find release asset: %w", err)
	}

	// Extract original filename from download URL for checksum matching
	debName := debURL[strings.LastIndex(debURL, "/")+1:]
	tmpF, err := os.CreateTemp("", "setup-"+repoToFilename(repo)+"-*.deb")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	debPath := tmpF.Name()
	_ = tmpF.Close()

	if err := runner.Run("wget", "-q", debURL, "-O", debPath); err != nil {
		return fmt.Errorf("download %s: %w", repo, err)
	}
	defer func() { _ = runner.Remove(debPath) }()

	if err := verifyDebChecksum(runner, repo, debPath, debName); err != nil {
		setupexec.PrintStep(fmt.Sprintf("%s .deb checksum unavailable or invalid (%v); using signed Ubuntu apt package", repo, err))
		if err := runner.Run("apt", "install", "-y", aptPackage); err != nil {
			return fmt.Errorf("install %s from apt: %w", aptPackage, err)
		}
		return setupDebianAliases(runner, commandName)
	}

	if err := runner.Run("apt", "install", "-y", debPath); err != nil {
		return fmt.Errorf("install %s deb: %w", repo, err)
	}
	return nil
}

func repoToFilename(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}

func verifyDebChecksum(runner setupexec.CmdRunner, repo, debPath, debName string) error {
	checksumPatterns := []string{regexp.QuoteMeta(debName) + `\.sha256$`, `SHA256SUMS$`, `sha256sums\.txt$`}
	var checksumURL string
	var err error
	for _, cp := range checksumPatterns {
		checksumURL, err = github.LatestReleaseAsset(repo, cp)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("no checksum file found for %s; refusing unverified install", repo)
	}

	tmpChecksum := debPath + ".sha256"
	if err := runner.Run("wget", "-q", checksumURL, "-O", tmpChecksum); err != nil {
		return fmt.Errorf("download checksum file for %s: %w", repo, err)
	}
	defer func() { _ = runner.Remove(tmpChecksum) }()

	checksumContent, err := runner.ReadFile(tmpChecksum)
	if err != nil {
		return fmt.Errorf("read checksum file for %s: %w", repo, err)
	}

	var expectedSHA string
	for _, line := range strings.Split(string(checksumContent), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "  "+debName) || strings.HasSuffix(line, " *"+debName) {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				expectedSHA = parts[0]
			}
			break
		}
		parts := strings.Fields(line)
		if len(parts) == 1 {
			expectedSHA = parts[0]
			break
		}
	}
	if expectedSHA == "" {
		return fmt.Errorf("no checksum entry found for %s", debName)
	}

	if err := verifySHA256File(runner, debPath, expectedSHA); err != nil {
		return fmt.Errorf("checksum verification failed for %s: %w", repo, err)
	}
	return nil
}

func setupDebianAliases(runner setupexec.CmdRunner, commandName string) error {
	switch commandName {
	case "fd":
		if err := runner.MkdirAll("/usr/local/bin", 0755); err != nil {
			return err
		}
		return runner.Run("ln", "-sf", "/usr/bin/fdfind", "/usr/local/bin/fd")
	case "bat":
		if err := runner.MkdirAll("/usr/local/bin", 0755); err != nil {
			return err
		}
		if err := runner.Run("test", "-x", "/usr/bin/bat"); err == nil {
			return nil
		}
		return runner.Run("ln", "-sf", "/usr/bin/batcat", "/usr/local/bin/bat")
	default:
		return nil
	}
}

func installYq(runner setupexec.CmdRunner) error {
	yqPath := "/usr/local/bin/yq"
	yqURL := "https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64"

	if err := runner.MkdirAll(filepath.Dir(yqPath), 0755); err != nil {
		return err
	}

	tmpYq, err := runner.CreateTemp(filepath.Dir(yqPath), ".setup-yq-*")
	if err != nil {
		return fmt.Errorf("create temp yq file: %w", err)
	}
	defer func() { _ = runner.Remove(tmpYq) }()

	if err := runner.Run("wget", "-q", yqURL, "-O", tmpYq); err != nil {
		return fmt.Errorf("download yq: %w", err)
	}

	shaURL := "https://github.com/mikefarah/yq/releases/latest/download/checksums"
	shaPath, err := runner.CreateTemp(filepath.Dir(yqPath), ".setup-yq-*.sha256")
	if err != nil {
		return fmt.Errorf("create temp yq checksum file: %w", err)
	}
	if err := runner.Run("wget", "-q", shaURL, "-O", shaPath); err != nil {
		return fmt.Errorf("download yq checksum: %w", err)
	}
	defer func() { _ = runner.Remove(shaPath) }()

	orderURL := "https://github.com/mikefarah/yq/releases/latest/download/checksums_hashes_order"
	orderPath, err := runner.CreateTemp(filepath.Dir(yqPath), ".setup-yq-order-*")
	if err != nil {
		return fmt.Errorf("create temp yq checksum order file: %w", err)
	}
	if err := runner.Run("wget", "-q", orderURL, "-O", orderPath); err != nil {
		return fmt.Errorf("download yq checksum order: %w", err)
	}
	defer func() { _ = runner.Remove(orderPath) }()

	if setupexec.IsDryRun(runner) {
		return nil
	}

	shaContent, err := runner.ReadFile(shaPath)
	if err != nil {
		return fmt.Errorf("read yq checksum: %w", err)
	}
	orderContent, err := runner.ReadFile(orderPath)
	if err != nil {
		return fmt.Errorf("read yq checksum order: %w", err)
	}
	expectedSHA, err := checksumForAsset(string(shaContent), string(orderContent), "yq_linux_amd64")
	if err != nil {
		return fmt.Errorf("find yq checksum: %w", err)
	}
	if err := verifySHA256File(runner, tmpYq, expectedSHA); err != nil {
		return fmt.Errorf("yq checksum verification failed: %w", err)
	}

	fi, err := runner.Stat(tmpYq)
	if err != nil {
		return fmt.Errorf("stat yq: %w", err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("downloaded yq file is empty")
	}
	if err := runner.Chmod(tmpYq, 0755); err != nil {
		return err
	}
	return runner.Rename(tmpYq, yqPath)
}

func installGlow(runner setupexec.CmdRunner) error {
	keyringPath := "/etc/apt/keyrings/charm.gpg"
	listPath := "/etc/apt/sources.list.d/charm.list"

	if err := runner.MkdirAll("/etc/apt/keyrings", 0755); err != nil {
		return err
	}

	tmpKeyring, err := runner.CreateTemp(filepath.Dir(keyringPath), ".setup-charm-keyring-*.gpg")
	if err != nil {
		return fmt.Errorf("create temp charm gpg key: %w", err)
	}
	defer func() { _ = runner.Remove(tmpKeyring) }()

	keyScript := fmt.Sprintf(
		"curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o %s",
		shellQuote(tmpKeyring),
	)
	if err := runner.Shell(keyScript); err != nil {
		return fmt.Errorf("download charm gpg key: %w", err)
	}

	if !setupexec.IsDryRun(runner) {
		const charmFingerprint = "F506 F2D6 02D1 C400 A1E4  5D96 7E2E 87C7 1D5E 9D67"
		if err := verifyGPGFingerprint(runner, tmpKeyring, charmFingerprint); err != nil {
			return fmt.Errorf("verify charm gpg key: %w", err)
		}
	}
	if err := runner.Chmod(tmpKeyring, 0644); err != nil {
		return err
	}
	if err := runner.Rename(tmpKeyring, keyringPath); err != nil {
		return err
	}

	listContent := fmt.Sprintf("deb [signed-by=%s] https://repo.charm.sh/apt/ * *\n", keyringPath)
	tmpList, err := runner.CreateTemp(filepath.Dir(listPath), ".setup-charm-list-*")
	if err != nil {
		return fmt.Errorf("create temp charm.list: %w", err)
	}
	defer func() { _ = runner.Remove(tmpList) }()

	if err := runner.WriteFile(tmpList, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("write temp charm.list: %w", err)
	}
	if err := runner.Rename(tmpList, listPath); err != nil {
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

	if err := runner.MkdirAll("/etc/apt/keyrings", 0755); err != nil {
		return err
	}

	tmpKeyring, err := runner.CreateTemp(filepath.Dir(keyringPath), ".setup-github-cli-keyring-*.gpg")
	if err != nil {
		return fmt.Errorf("create temp gh gpg key: %w", err)
	}
	defer func() { _ = runner.Remove(tmpKeyring) }()

	if err := runner.Run("wget", "-nv", "-O", tmpKeyring, "https://cli.github.com/packages/githubcli-archive-keyring.gpg"); err != nil {
		return fmt.Errorf("download gh gpg key: %w", err)
	}

	// Verify GitHub CLI GPG key fingerprint
	if !setupexec.IsDryRun(runner) {
		const ghFingerprint = "23F3 D1D8 6577 3DE1 7D9D  8C30 A7B6 3A2B 8F85 411F"
		if err := verifyGPGFingerprint(runner, tmpKeyring, ghFingerprint); err != nil {
			return fmt.Errorf("verify gh gpg key: %w", err)
		}
	}
	if err := runner.Chmod(tmpKeyring, 0644); err != nil {
		return err
	}
	if err := runner.Rename(tmpKeyring, keyringPath); err != nil {
		return err
	}

	arch, err := runner.Output("dpkg", "--print-architecture")
	if err != nil {
		return fmt.Errorf("get architecture: %w", err)
	}

	if err := runner.MkdirAll("/etc/apt/sources.list.d", 0755); err != nil {
		return err
	}

	listContent := fmt.Sprintf("deb [arch=%s signed-by=%s] https://cli.github.com/packages stable main\n", arch, keyringPath)
	tmpList, err := runner.CreateTemp(filepath.Dir(listPath), ".setup-github-cli-list-*")
	if err != nil {
		return fmt.Errorf("create temp github-cli.list: %w", err)
	}
	defer func() { _ = runner.Remove(tmpList) }()

	if err := runner.WriteFile(tmpList, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("write temp github-cli.list: %w", err)
	}
	if err := runner.Rename(tmpList, listPath); err != nil {
		return err
	}

	if err := runner.Run("apt", "update"); err != nil {
		return err
	}
	return runner.Run("apt", "install", "-y", "gh")
}

func verifyGPGFingerprint(runner setupexec.CmdRunner, keyringPath, expectedFingerprint string) error {
	out, err := runner.Output("gpg", "--show-keys", "--with-fingerprint", "--with-colons", keyringPath)
	if err != nil {
		return err
	}
	normalizedExpected := strings.ReplaceAll(expectedFingerprint, " ", "")
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "fpr:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 10 && parts[9] == normalizedExpected {
				return nil
			}
		}
	}
	return fmt.Errorf("fingerprint mismatch")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func verifySHA256File(runner setupexec.CmdRunner, path, expected string) error {
	expected = strings.TrimSpace(expected)
	decoded, err := hex.DecodeString(expected)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("invalid SHA256 %q", expected)
	}
	data, err := runner.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), expected) {
		return fmt.Errorf("expected %s", expected)
	}
	return nil
}

func checksumForAsset(checksums, order, asset string) (string, error) {
	shaField := -1
	for i, line := range strings.Split(strings.TrimSpace(order), "\n") {
		if strings.TrimSpace(line) == "SHA-256" {
			shaField = i + 1
			break
		}
	}
	if shaField == -1 {
		return "", fmt.Errorf("SHA-256 missing from checksum order")
	}
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) > shaField && fields[0] == asset {
			return fields[shaField], nil
		}
	}
	return "", fmt.Errorf("asset %s missing from checksums", asset)
}
