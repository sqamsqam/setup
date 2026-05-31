# setup

Provisioning scripts for fresh Ubuntu LXC containers. Designed to be run in
numeric order on a newly-created container.

## Scripts

| Script | Purpose | Run as |
|--------|---------|--------|
| `00-root-bootstrap.sh` | Locale, base packages, SSH hardening, Docker, automatic updates | root |
| `10-add-user.sh <user> '<pubkey>'` | Create sudo user with SSH key auth | root |
| `20-install-cli-tools.sh` | ripgrep, fd, bat, yq, glow | root |
| `30-install-node-toolchain.sh <user>` | Node.js, TypeScript, tsx (via fnm) | root |
| `40-install-go-toolchain.sh` | Latest Go for amd64 | root |

## Usage

```bash
# On a fresh container, run in order:
sudo ./scripts/00-root-bootstrap.sh
sudo ./scripts/10-add-user.sh myuser 'ssh-ed25519 AAAA...'
sudo ./scripts/20-install-cli-tools.sh
sudo ./scripts/30-install-node-toolchain.sh myuser
sudo ./scripts/40-install-go-toolchain.sh
```

## Customisation

- `TIMEZONE` env var (default `Australia/Sydney`) — set before running
  `00-root-bootstrap.sh`.
