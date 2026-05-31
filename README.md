# setup

Provisioning tool for fresh Ubuntu 26.04 LXC containers. Provides both an
interactive terminal UI and a traditional CLI interface.

## What it does

- Bootstraps a fresh Ubuntu 26.04 LXC container (locale, base packages, SSH
  hardening, unattended upgrades, Docker)
- Creates users with sudo access, SSH key authentication, and linger support
- Installs modern CLI tools: ripgrep, fd, bat, yq, glow, gh
- Installs Go (system-wide, latest stable)
- Installs Node.js toolchain per user (fnm, corepack, TypeScript, tsx)
- Supports dry-run mode for safe previews
- Idempotent — safe to re-run

## Target environment

- Ubuntu 26.04 LXC container
- Linux amd64
- Run as root (or with sudo)

## Installation

Download the latest binary from the [releases page][releases] and place it on
your target system:

```bash
sudo curl -fsSL -o /usr/local/bin/setup \
  https://github.com/sqamsqam/setup/releases/latest/download/setup-linux-amd64
sudo chmod +x /usr/local/bin/setup
```

Or build from source:

```bash
go install github.com/sqamsqam/setup/cmd/setup@latest
```

## Usage

### Interactive mode (TUI)

Run without arguments to launch the interactive provisioning interface:

```bash
sudo setup
```

Navigate with arrow keys, toggle steps with space, and follow the prompts.

### CLI mode (scripting / automation)

```bash
# Root bootstrap (locale, packages, SSH, Docker, automatic updates)
sudo setup bootstrap [--timezone Australia/Sydney]

# Create a sudo user with SSH key authentication
sudo setup add-user --user <username> --key "<ssh-public-key>"

# Install CLI tools (ripgrep, fd, bat, yq, glow, gh)
sudo setup install-tools

# Install development tools (Go system-wide, Node.js per-user)
sudo setup devtools --user <username> [--all] [--go] [--node]

# Run the full provisioning flow in one command
sudo setup full --user <username> --key "<ssh-public-key>" [--timezone Australia/Sydney]

# Show version
setup version
```

### Dry run

Add `--dry-run` before any CLI command to preview what would be executed
without making changes:

```bash
sudo setup --dry-run full --user dev --key "ssh-ed25519 AAAA..."
```

## Commands reference

| Command | Description |
|---|---|
| `bootstrap` | Locale, system update, base packages, SSH hardening, unattended upgrades, Docker |
| `add-user` | Create sudo user, install SSH key, enable linger, update AllowUsers |
| `install-tools` | ripgrep, fd, bat (GitHub releases), yq (binary), glow (charm.sh apt repo), gh (GitHub CLI apt repo) |
| `devtools` | Go (system-wide from go.dev), Node.js (per-user via fnm) |
| `full` | Runs bootstrap → add-user → install-tools → devtools |
| `version` | Prints version and build info |

Global flags: `--dry-run`

## Fresh LXC bootstrap flow

On a freshly created Ubuntu 26.04 container:

```bash
# Interactive
sudo setup

# Or non-interactive
sudo setup full \
  --user dev \
  --key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI..." \
  --timezone Australia/Sydney
```

This will:
1. Generate locales, upgrade packages, install core utilities
2. Harden SSH (pubkey-only, no root login, no passwords)
3. Enable unattended security updates
4. Install Docker
5. Create the specified user with passwordless sudo
6. Install the user's SSH public key
7. Install CLI tools (ripgrep, fd, bat, yq, glow, gh)
8. Install Go (system-wide) and Node.js (per-user)

## Release process

1. Tag a commit: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions builds the binary, runs tests, and creates a release
3. The binary is attached to the release as `setup-linux-amd64`

## Development

```bash
# Run tests
make test

# Run vet
make vet

# Build locally
make build

# Run in CLI mode
make run-cli ARGS="version"
```

Requirements: Go 1.26+

### Project structure

```
cmd/setup/          Entry point
internal/
  cli/              CLI mode (flag parsing, subcommand routing)
  tui/              Bubble Tea interactive terminal UI
  exec/             Command runner (real + dry-run)
  system/           Root bootstrap logic
  user/             User management + input validation
  tools/            CLI tool installation
  devtools/         Go + Node.js toolchains
  github/           GitHub release asset lookup
```

## Safety notes

- Password authentication is disabled for SSH after bootstrap
- Root login over SSH is disabled
- SSH is only restarted after config validation (`sshd -t` passes)
- Dry-run mode logs all shell commands without executing them
- No interactive prompts in CLI mode
- `DEBIAN_FRONTEND=noninteractive` is set for all apt operations
- The Docker install script (`get.docker.com`) and fnm install script
  (`fnm.vercel.app`) are piped from their official sources — review them
  if you're running in an untrusted environment

## License

MIT

[releases]: https://github.com/sqamsqam/setup/releases
