# Usage

`setup` has two modes: a guided terminal UI when run without a command, and a scriptable CLI when a command is provided.

## Guided TUI

```bash
sudo setup
```

The TUI opens as an admin console with no selected actions. Choose an area, select the work for that area, then review the full plan before anything changes:

- Fresh setup: locale, package updates, timezone, unattended upgrades, SSH hardening, and Docker.
- Users: login-user creation, SSH keys, SSH access, passwordless sudo, linger, Docker group membership, access lock/delete, and setup-owned service users.
- Groups: create, delete, list, add users to groups, and remove users from groups.
- Services: create, inspect, restart, list, disable, and remove setup-owned per-user systemd services.
- Instance: UFW rules/status, fail2ban, Docker log rotation/disk/prune, updates, and reboot actions.
- Tools: ripgrep, fd, bat, yq, glow, and gh.
- Dev tools: Go, Node.js, Rust, golangci-lint, GoReleaser, govulncheck, and pnpm.
- Diagnostics: read-only instance checks.

Use arrow keys to move, Enter to open an area, Space to toggle actions, `/` to filter, Enter to continue from an area, and Esc to go back.

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

User management:

```bash
# Compatibility shortcut: preserves the original full login-user setup.
sudo setup user --user dev --key-file ~/.ssh/id_ed25519.pub

# Selective login-user actions.
sudo setup user create --user dev --key-file ~/.ssh/id_ed25519.pub --allow-ssh --sudo --linger --group docker
sudo setup user ssh key add --user dev --key-file ~/.ssh/id_ed25519.pub
sudo setup user ssh allow --user dev
sudo setup user ssh deny --user dev
sudo setup user sudo enable --user dev
sudo setup user sudo disable --user dev
sudo setup user linger enable --user dev
sudo setup user linger disable --user dev
sudo setup user group add --user dev --group docker
sudo setup user group remove --user dev --group docker

# Lock access without deleting user data.
sudo setup user disable --user dev

# Delete preserves the home directory unless explicitly told otherwise.
sudo setup user delete --user dev
sudo setup user delete --user dev --remove-home

# Setup-owned service users are system no-login accounts under /var/lib/<user>.
sudo setup user service create --user app --group www-data
```

Group management:

```bash
sudo setup group create --group app
sudo setup group list
sudo setup group user add --user dev --group app
sudo setup group user remove --user dev --group app
sudo setup group delete --group app --yes
```

Instance helpers:

```bash
sudo setup network status
sudo setup network list
sudo setup network enable --allow-ssh
sudo setup network allow --port 443 --proto tcp
sudo setup network delete --number 2 --yes
sudo setup network reset --yes

sudo setup guard install
sudo setup guard status
sudo setup guard unban --ip 203.0.113.10

sudo setup containers log-rotation --yes
sudo setup containers disk
sudo setup containers prune --containers --images --build-cache --yes

sudo setup updates check
sudo setup updates upgrade
sudo setup updates reboot-needed
sudo setup updates unattended
sudo setup updates failed-units
sudo setup updates reboot --yes

sudo setup service create --user dev --name app --workdir /home/dev/app --cmd "npm start"
sudo setup service list --user dev
sudo setup service list --user dev --state
sudo setup service status --user dev --name app
sudo setup service logs --user dev --name app
sudo setup service restart --user dev --name app
sudo setup service disable --user dev --name app --yes
sudo setup service remove --user dev --name app --yes
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
- `setup user ssh allow` and `setup user ssh deny` manage only the setup-owned `AllowUsers` list instead of scanning all UID >= 1000 users.
- Passwordless sudo is managed only through setup-owned `/etc/sudoers.d/<user>` files. Disable refuses to remove unmanaged sudoers files.
- Setup-owned admin files, including SSH hardening, unattended-upgrades, fail2ban, and managed user-service units, refuse to replace unmanaged existing files.
- Destructive or service-stopping admin commands such as firewall rule deletion, firewall reset, Docker prune, service disable, and service remove require `--yes`.
- User group-membership commands require the group to already exist; they do not create groups implicitly.
- Top-level group deletion requires `--yes` and refuses to delete a group that is a primary group for an existing user.
- Service users are setup-owned no-login system accounts with homes under `/var/lib/<user>`. They are not modifications to distro-owned accounts such as `root`, `www-data`, `sshd`, or `nobody`.
- `setup user disable` locks access and removes setup-managed SSH, linger, and sudo access without deleting data. `setup user delete --remove-home` is required for irreversible home removal.
- Managed files are clearly marked and compared before replacement where practical.
- CLI mode does not prompt interactively.
- Existing users, SSH keys, group memberships, managed config files, and installed tool versions are handled carefully where practical.
