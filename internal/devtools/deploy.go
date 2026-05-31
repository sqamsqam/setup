package devtools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
)

// Pinned Go version and SHA256 for deterministic, secure installation.
// When set to non-empty values, these take precedence over the go.dev API.
// Update these constants when a new Go version is desired.
var (
	pinnedGoVersion = ""
	pinnedGoSHA256  = ""
)

func InstallGo(runner setupexec.CmdRunner) error {
	if setupexec.IsDryRun(runner) {
		setupexec.PrintStep("Would download and install latest Go")
		setupexec.PrintDone("Go installation skipped (dry-run)")
		return nil
	}

	setupexec.PrintStep("Looking up latest Go release")

	if !setupexec.CheckCommand("curl") {
		if err := runner.Run("apt", "update"); err != nil {
			return err
		}
		if err := runner.Run("apt", "install", "-y", "curl"); err != nil {
			return err
		}
	}

	var version, sha256 string
	if pinnedGoVersion != "" && pinnedGoSHA256 != "" {
		version = pinnedGoVersion
		sha256 = pinnedGoSHA256
		setupexec.PrintStep(fmt.Sprintf("Using pinned Go version %s", version))
	} else {
		goJSON, err := runner.Output("curl", "-fsSL", "https://go.dev/dl/?mode=json")
		if err != nil {
			return fmt.Errorf("fetch Go releases: %w", err)
		}

		version, sha256, err = parseGoRelease(goJSON)
		if err != nil {
			return err
		}
	}

	// Check if already installed and up to date
	if out, err := runner.Output("/usr/local/go/bin/go", "version"); err == nil {
		parts := strings.Fields(out)
		if len(parts) >= 3 {
			installedVersion := parts[2]
			if installedVersion == version {
				setupexec.PrintStep(fmt.Sprintf("Go %s already installed, skipping", version))
				setupexec.PrintDone("Go installation skipped (up to date)")
				return nil
			}
		}
	}

	tarball := version + ".linux-amd64.tar.gz"
	downloadURL := "https://go.dev/dl/" + tarball
	tmpFile, err := os.CreateTemp("", "setup-go-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpTarball := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = runner.Remove(tmpTarball) }()

	setupexec.PrintStep(fmt.Sprintf("Downloading Go %s", version))

	if err := runner.Run("curl", "-fsSL", downloadURL, "-o", tmpTarball); err != nil {
		return fmt.Errorf("download Go: %w", err)
	}

	setupexec.PrintStep("Verifying checksum")
	if err := runner.Shell(fmt.Sprintf("echo '%s  %s' | sha256sum -c --status", sha256, tmpTarball)); err != nil {
		return fmt.Errorf("go checksum verification failed")
	}

	if err := runner.RemoveAll("/usr/local/go"); err != nil {
		return err
	}
	if err := runner.Run("tar", "-C", "/usr/local", "-xzf", tmpTarball); err != nil {
		return fmt.Errorf("extract Go: %w", err)
	}

	profileContent := "# Managed by setup — do not edit\n"
	profileContent += "export PATH=\"/usr/local/go/bin:$PATH\"\n"
	tmpFile2, err := os.CreateTemp("", "setup-go-profile-*.sh")
	if err != nil {
		return fmt.Errorf("create temp profile: %w", err)
	}
	tmpProfile := tmpFile2.Name()
	_ = tmpFile2.Close()
	defer func() { _ = runner.Remove(tmpProfile) }()

	if err := runner.WriteFile(tmpProfile, []byte(profileContent), 0644); err != nil {
		return fmt.Errorf("write temp go profile: %w", err)
	}
	if err := runner.Rename(tmpProfile, "/etc/profile.d/go.sh"); err != nil {
		return err
	}

	verifyRunner := setupexec.NewRealRunner()
	verifyRunner.Env = append(verifyRunner.Env, "PATH=/usr/local/go/bin:"+os.Getenv("PATH"))
	out, err := verifyRunner.Output("/usr/local/go/bin/go", "version")
	if err != nil {
		return fmt.Errorf("verify Go installation: %w", err)
	}
	setupexec.PrintDone(out)
	return nil
}

func parseGoRelease(jsonData string) (version string, sha256 string, err error) {
	type goFile struct {
		OS     string `json:"os"`
		Arch   string `json:"arch"`
		Kind   string `json:"kind"`
		SHA256 string `json:"sha256"`
	}
	type goRelease struct {
		Version string   `json:"version"`
		Files   []goFile `json:"files"`
	}

	var releases []goRelease

	jsonData = strings.TrimPrefix(jsonData, "```")
	jsonData = strings.TrimSuffix(jsonData, "```")

	if err := json.Unmarshal([]byte(jsonData), &releases); err != nil {
		return "", "", fmt.Errorf("parse Go release JSON: %w", err)
	}

	if len(releases) == 0 {
		return "", "", fmt.Errorf("no Go releases found")
	}

	latest := releases[0]
	for _, f := range latest.Files {
		if f.OS == "linux" && f.Arch == "amd64" && f.Kind == "archive" {
			return latest.Version, f.SHA256, nil
		}
	}

	return "", "", fmt.Errorf("could not find linux/amd64 archive for Go %s", latest.Version)
}

func InstallNode(runner setupexec.CmdRunner, username string) error {
	if setupexec.IsDryRun(runner) {
		setupexec.PrintStep(fmt.Sprintf("Would install Node.js toolchain for %s", username))
		setupexec.PrintDone("Node.js installation skipped (dry-run)")
		return nil
	}

	setupexec.PrintStep(fmt.Sprintf("Installing Node.js toolchain for %s", username))

	shellName := detectShell(runner, username)

	script := strings.TrimSpace(fmt.Sprintf(`
set -euo pipefail
export PATH="$HOME/.local/share/fnm:$PATH"

if [[ ! -d "$HOME/.local/share/fnm" ]]; then
  curl -fsSL https://fnm.vercel.app/install | %s
fi

export FNM_DIR="$HOME/.local/share/fnm"

if [[ ! -x "$FNM_DIR/fnm" ]]; then
  echo "ERROR: fnm binary not found at $FNM_DIR/fnm" >&2
  exit 1
fi

eval "$("$FNM_DIR/fnm" env --shell %s)"

fnm install --latest
fnm use latest
fnm default "$(fnm current)"

if ! command -v npm >/dev/null 2>&1; then
  echo "ERROR: npm not found after Node.js installation." >&2
  exit 1
fi

npm install -g corepack
corepack enable
npm install -g typescript tsx

echo "Node.js toolchain installed for $(whoami)."
`, shellName, shellName))

	sudoArgs := []string{"-iu", username, "--", shellName, "-c", script}
	setupexec.PrintStep("Installing fnm and Node.js (this may take a few minutes)")
	if err := runner.Run("sudo", sudoArgs...); err != nil {
		return fmt.Errorf("install Node toolchain: %w", err)
	}

	setupexec.PrintDone(fmt.Sprintf("Node.js toolchain installed for %s", username))
	return nil
}

func detectShell(runner setupexec.CmdRunner, username string) string {
	out, err := runner.Output("getent", "passwd", username)
	if err != nil {
		return "bash"
	}
	parts := strings.Split(out, ":")
	if len(parts) < 7 {
		return "bash"
	}
	shell := strings.TrimSpace(parts[6])
	if strings.HasSuffix(shell, "bash") {
		return "bash"
	}
	if strings.HasSuffix(shell, "zsh") {
		return "zsh"
	}
	return "bash"
}

func InstallAllDevTools(runner setupexec.CmdRunner, username string) error {
	if err := InstallGo(runner); err != nil {
		return err
	}
	if err := InstallNode(runner, username); err != nil {
		return err
	}
	return nil
}

