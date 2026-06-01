# setup

Provisioning tool for fresh Ubuntu 26.04 LXC containers. Provides both an
interactive terminal UI and a traditional CLI interface.

## What it does

- Bootstraps a fresh Ubuntu 26.04 LXC container (locale, base packages, SSH
  hardening, unattended upgrades, Docker with log rotation)
- Manages existing dev instances with UFW, fail2ban, diagnostics, update checks,
  Docker disk maintenance, and per-user systemd services
- Creates users with sudo access, SSH key authentication, and linger support
- Installs modern CLI tools: ripgrep, fd, bat, yq, glow, gh
- Installs Go (system-wide, latest stable) plus optional golangci-lint,
  GoReleaser, and govulncheck
- Installs Node.js toolchain per user (pinned fnm, corepack, TypeScript, tsx,
  optional pnpm)
- Installs Rust per user via rustup with rustfmt, clippy, rust-analyzer, and
  rust-src
- Supports dry-run mode for safe previews
- Idempotent where practical — safe to re-run after reviewing partial failures

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

The TUI presents a configurable **provisioning plan**. The default plan matches
the full bootstrap flow, but you can toggle individual items before anything
runs:

- **System Bootstrap** — Configure locale, SSH, Docker, unattended upgrades
- **Add User** — Create a user with passwordless sudo and SSH key
- **Instance Management** — Toggle UFW baseline, common UFW rules (HTTP, HTTPS,
  Mosh), fail2ban, Docker log rotation, doctor diagnostics, and update checks
- **CLI Tools** — Toggle ripgrep, fd, bat, yq, glow, and gh individually
- **Development Tools** — Toggle Go, Node.js, Rust, golangci-lint, GoReleaser,
  govulncheck, and pnpm individually

Navigate with arrow keys, press space to toggle items, use `/` to filter the
plan, and press enter to continue. The wizard only asks for inputs required by
the selected plan. Timezone input supports fuzzy search with Tab to accept the
highlighted match and Up/Down to choose between matches. Running output is
captured inside the TUI, and failed steps can be retried without rebuilding the
plan.

Add `--dry-run` to preview what would happen without making changes:

```bash
sudo setup --dry-run
```

### CLI mode (scripting / automation)

```bash
# Root bootstrap (locale, packages, SSH, Docker, automatic updates)
sudo setup bootstrap [--timezone Australia/Sydney]

# Create a sudo user with SSH key authentication
sudo setup add-user --user <username> --key-file ~/.ssh/id_ed25519.pub

# Install CLI tools (ripgrep, fd, bat, yq, glow, gh)
sudo setup install-tools

# Install development tools (Go system-wide, Node.js per-user)
sudo setup devtools --user <username> [--all] [--go] [--node]

# Run diagnostics for an existing instance
sudo setup doctor

# Configure UFW safely
sudo setup firewall enable --allow-ssh
sudo setup firewall allow --port 443 --proto tcp
sudo setup firewall numbered
sudo setup firewall delete --number 3

# Configure fail2ban SSH protection
sudo setup fail2ban install
sudo setup fail2ban status

# Configure Docker log rotation and inspect/prune Docker disk usage
sudo setup docker logs-config
sudo setup docker disk
sudo setup docker prune --containers --images --build-cache

# Check updates and reboot state
sudo setup updates check
sudo setup updates reboot-required

# Create a per-user systemd service
sudo setup service create \
  --user dev \
  --name app \
  --workdir /home/dev/app \
  --cmd "npm start" \
  --env-file /home/dev/app/.env

# Run the full provisioning flow in one command
sudo setup full --user <username> --key-file ~/.ssh/id_ed25519.pub [--timezone Australia/Sydney]

# Show version
setup version
```

### Help

Print usage information and per-command help:

```bash
setup --help
setup bootstrap --help
setup add-user --help
setup devtools --help
setup full --help
```

### Dry run

Add `--dry-run` before any CLI command to preview what would be executed
without making changes:

```bash
sudo setup --dry-run full --user dev --key-file ~/.ssh/id_ed25519.pub
```

Prefer `--key-file` for SSH keys. Inline `--key` is supported, but command
arguments can be visible to other local processes.

## Commands reference

| Command | Description |
|---|---|
| `bootstrap` | Locale, system update, base packages, SSH hardening, unattended upgrades, Docker |
| `add-user` | Create sudo user, install SSH key, enable linger, update AllowUsers |
| `install-tools` | ripgrep, fd, bat (GitHub releases when not already installed), yq (verified binary), glow (charm.sh apt repo), gh (GitHub CLI apt repo) |
| `devtools` | Go, Node.js, Rust, and ecosystem tools |
| `doctor` | Read-only LXC/VM/system diagnostics |
| `firewall` | UFW status, safe enable, allow rules, numbered rule deletion, reset |
| `fail2ban` | Install/status/unban for the managed SSH jail |
| `docker` | Docker log rotation config, disk usage, safe prune helpers |
| `updates` | Package update checks, upgrades, reboot-required checks, failed units |
| `service` | Create/status/logs/restart setup-managed per-user systemd services |
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
  --key-file ~/.ssh/id_ed25519.pub \
  --timezone Australia/Sydney
```

This will:
1. Generate locales
2. Upgrade packages
3. Install core utilities
4. Enable unattended security updates
5. Configure timezone
6. Harden SSH (pubkey-only, no root login, no passwords)
7. Lock root password
8. Install Docker from the official Docker apt repository
9. Enable and start SSH
10. Create the specified user with passwordless sudo
11. Install the user's SSH public key
12. Install CLI tools (ripgrep, fd, bat, yq, glow, gh)
13. Configure Docker json-file log rotation
14. Install Go (system-wide) and Node.js (per-user)

## Release process

1. Move the changelog `(Unreleased)` entries under the new version heading
2. Open and merge a PR for the changelog/release metadata update
3. Tag the merged commit: `git tag v0.1.0 && git push origin v0.1.0`
4. GitHub Actions runs vet, tests, lint, and GoReleaser
5. The binary is attached to the release as `setup-linux-amd64`

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
  firewall/         UFW management
  security/         fail2ban management
  docker/           Docker maintenance helpers
  diagnostics/      Read-only instance diagnostics
  updates/          apt/update/reboot helpers
  service/          Per-user systemd service helpers
  managed/          Shared managed-file helpers
  github/           GitHub release asset lookup
```

## Safety notes

- Password authentication is disabled for SSH after bootstrap
- Root login over SSH is disabled
- SSH drop-ins are rolled back if effective config validation (`sshd -t`) fails
- Dry-run mode logs all shell commands without executing them
- No interactive prompts in CLI mode
- `DEBIAN_FRONTEND=noninteractive` is set for all apt operations
- Commands run through a fixed safe PATH instead of trusting the caller's PATH
- Go downloads are verified against the official SHA256 checksum
- Docker is installed through the official apt repository after GPG fingerprint
  verification
- Docker log rotation is configured in `/etc/docker/daemon.json` without
  replacing unrelated daemon settings
- UFW baseline setup allows the detected SSH port before enabling the firewall
- fail2ban SSH protection is written as a setup-managed jail drop-in
- Diagnostics are read-only and intended for LXC, VM, or bare-metal instances
- Per-user services are created under the target user's systemd user directory
  and refuse system users
- Third-party apt keys are verified from temporary keyrings before replacing
  trusted keyring paths
- fnm is installed from a pinned release zip with SHA256 verification
- `AllowUsers` is managed from local non-system users (UID >= 1000), so review
  local account state before running on a reused container

## Troubleshooting

### apt lock contention

If another process (e.g. unattended-upgrades) is holding the apt lock,
the tool will fail with a lock error. Wait for the other process to finish
and re-run the command.

### Network timeouts during downloads

Some downloads (Go tarball, fnm, Docker apt packages, GitHub releases) may
timeout on slow connections. The tool will print the error and exit.
Re-running is safe after reviewing any partial failure.

### Docker install failure

Docker is installed from `download.docker.com` using the official apt
repository. If it fails, check network connectivity, apt repository access,
and the Ubuntu codename reported by `/etc/os-release`.

### Safe re-run after partial failure

Provisioning steps are designed to be idempotent where practical. If a step
fails midway, fix the underlying issue (e.g. network, disk space) and re-run
the same command. The tool skips many already-completed steps, but installers
that track upstream releases may still check the network.

### SSH lockout prevention

Before restarting sshd, the tool validates the SSH configuration with
`sshd -t`. If validation fails, the restart is skipped and you are
left with a working SSH session while the config error is reported.

## License

MIT

[releases]: https://github.com/sqamsqam/setup/releases
