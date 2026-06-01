# Usage

`setup` has two modes: a guided terminal UI when run without a command, and a scriptable CLI when a command is provided.

## Guided TUI

```bash
sudo setup
```

The TUI lets you pick the work to run before anything changes:

- Base system setup: locale, package updates, timezone, unattended upgrades, SSH hardening, and Docker.
- User setup: sudo user, SSH key, linger, and `AllowUsers`.
- Instance care: UFW rules, fail2ban, Docker log rotation, diagnostics, and update checks.
- CLI tools: ripgrep, fd, bat, yq, glow, and gh.
- Development tools: Go, Node.js, Rust, golangci-lint, GoReleaser, govulncheck, and pnpm.

Use arrow keys to move, Space to toggle, `/` to filter, Enter to continue, and Esc to go back.

## CLI Reference

Fresh instance setup:

```bash
sudo setup fresh --user dev --key-file ~/.ssh/id_ed25519.pub --timezone UTC
```

Focused commands:

```bash
sudo setup base --timezone UTC
sudo setup user --user dev --key-file ~/.ssh/id_ed25519.pub
sudo setup tools
sudo setup dev --user dev --all
sudo setup check
setup version
```

Instance helpers:

```bash
sudo setup network status
sudo setup network list
sudo setup network enable --allow-ssh
sudo setup network allow --port 443 --proto tcp
sudo setup network delete --number 2
sudo setup network reset

sudo setup guard install
sudo setup guard status
sudo setup guard unban --ip 203.0.113.10

sudo setup containers log-rotation
sudo setup containers disk
sudo setup containers prune --containers --images --build-cache

sudo setup updates check
sudo setup updates upgrade
sudo setup updates reboot-needed
sudo setup updates unattended
sudo setup updates failed-units
sudo setup updates reboot --yes

sudo setup service create --user dev --name app --workdir /home/dev/app --cmd "npm start"
sudo setup service status --user dev --name app
sudo setup service logs --user dev --name app
sudo setup service restart --user dev --name app
```

Use `setup --help` or `setup <command> --help` for generated help.

## Preview Modes

Use `--dry-run` to print intended commands without changing the host:

```bash
sudo setup --dry-run fresh \
  --user dev \
  --key-file ~/.ssh/id_ed25519.pub \
  --timezone UTC
```

Use `--demo` for the same non-mutating path without dry-run labels:

```bash
setup --demo
setup --demo fresh --user dev --key-file ~/.ssh/id_ed25519.pub --timezone UTC
```

Visual demos must use `--demo` so they stay deterministic and safe.

## Safety Notes

- SSH password login and root SSH login are disabled by the managed base setup.
- SSH config is validated with `sshd -t` before restart.
- Managed files are clearly marked and compared before replacement where practical.
- CLI mode does not prompt interactively.
- Existing users, SSH keys, group memberships, managed config files, and installed tool versions are handled carefully where practical.
