package devtools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	setupuser "github.com/sqamsqam/setup/internal/user"
)

// Pinned Go version and SHA256 for deterministic, secure installation.
// When set to non-empty values, these take precedence over the go.dev API.
// Update these constants when a new Go version is desired.
var (
	pinnedGoVersion = ""
	pinnedGoSHA256  = ""
)

const (
	fnmVersion = "v1.39.0"
	fnmSHA256  = "7807664f39d39fc518da1c35ba0181e4b3267603c4b1dedeb4b5fc6ae440a224"
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
	if err := verifySHA256File(runner, tmpTarball, sha256); err != nil {
		return fmt.Errorf("go checksum verification failed: %w", err)
	}

	if err := runner.RemoveAll("/usr/local/go"); err != nil {
		return err
	}
	if err := runner.Run("tar", "-C", "/usr/local", "-xzf", tmpTarball); err != nil {
		return fmt.Errorf("extract Go: %w", err)
	}

	profileContent := "# Managed by setup — do not edit\n"
	profileContent += "export PATH=\"/usr/local/go/bin:$PATH\"\n"
	tmpProfile, err := runner.CreateTemp("/etc/profile.d", ".setup-go-profile-*.sh")
	if err != nil {
		return fmt.Errorf("create temp profile: %w", err)
	}
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
	if _, err := validateTargetUser(runner, username); err != nil {
		return err
	}

	setupexec.PrintStep(fmt.Sprintf("Installing Node.js toolchain for %s", username))

	shellName := detectShell(runner, username)
	if err := runner.Run("apt", "install", "-y", "curl", "unzip", "ca-certificates"); err != nil {
		return err
	}

	script := strings.TrimSpace(fmt.Sprintf(`
set -euo pipefail
export FNM_DIR="$HOME/.local/share/fnm"
export PATH="$FNM_DIR:$PATH"

mkdir -p "$FNM_DIR"

if [[ ! -x "$FNM_DIR/fnm" ]]; then
  tmp_zip="$(mktemp)"
  trap 'rm -f "$tmp_zip"' EXIT
  curl -fsSL "https://github.com/Schniz/fnm/releases/download/%s/fnm-linux.zip" -o "$tmp_zip"
  echo "%s  $tmp_zip" | sha256sum -c --status
  unzip -oq "$tmp_zip" -d "$FNM_DIR"
  chmod 0755 "$FNM_DIR/fnm"
fi

if [[ ! -x "$FNM_DIR/fnm" ]]; then
  echo "ERROR: fnm binary not found at $FNM_DIR/fnm" >&2
  exit 1
fi

eval "$("$FNM_DIR/fnm" env --shell %s)"

profile_file="$HOME/.bashrc"
if [[ "%s" == "zsh" ]]; then
  profile_file="$HOME/.zshrc"
fi
if ! grep -Fq "# Managed by setup - fnm" "$profile_file" 2>/dev/null; then
  cat >>"$profile_file" <<'EOF'

# Managed by setup - fnm
export FNM_DIR="$HOME/.local/share/fnm"
export PATH="$FNM_DIR:$PATH"
eval "$("$FNM_DIR/fnm" env --shell %s)"
EOF
fi

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
`, fnmVersion, fnmSHA256, shellName, shellName, shellName))

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

type InstallOptions struct {
	Go   bool
	Node bool
}

func AllInstallOptions() InstallOptions {
	return InstallOptions{Go: true, Node: true}
}

func (o InstallOptions) Any() bool {
	return o.Go || o.Node
}

func InstallAllDevTools(runner setupexec.CmdRunner, username string) error {
	return InstallSelected(runner, username, AllInstallOptions())
}

func InstallSelected(runner setupexec.CmdRunner, username string, opts InstallOptions) error {
	if !opts.Any() {
		setupexec.PrintStep("No development tools selected")
		return nil
	}
	if opts.Go {
		if err := InstallGo(runner); err != nil {
			return err
		}
	}
	if opts.Node {
		if err := InstallNode(runner, username); err != nil {
			return err
		}
	}
	return nil
}

type targetUser struct {
	uid  int
	gid  int
	home string
}

func validateTargetUser(runner setupexec.CmdRunner, username string) (targetUser, error) {
	if err := setupuser.ValidateUsername(username); err != nil {
		return targetUser{}, err
	}
	out, err := runner.Output("getent", "passwd", username)
	if err != nil {
		return targetUser{}, fmt.Errorf("lookup passwd entry for %s: %w", username, err)
	}
	parts := strings.Split(out, ":")
	if len(parts) < 7 || parts[0] != username {
		return targetUser{}, fmt.Errorf("invalid passwd entry for %s", username)
	}
	uid, err := strconv.Atoi(parts[2])
	if err != nil {
		return targetUser{}, fmt.Errorf("parse uid for %s: %w", username, err)
	}
	gid, err := strconv.Atoi(parts[3])
	if err != nil {
		return targetUser{}, fmt.Errorf("parse gid for %s: %w", username, err)
	}
	if uid < 1000 {
		return targetUser{}, fmt.Errorf("refusing to install dev tools for %s: uid %d is below 1000", username, uid)
	}
	home := strings.TrimSpace(parts[5])
	if home == "" || !filepath.IsAbs(home) {
		return targetUser{}, fmt.Errorf("invalid home directory for %s: %q", username, home)
	}
	return targetUser{uid: uid, gid: gid, home: home}, nil
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
