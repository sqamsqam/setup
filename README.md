# setup

`setup` gets a fresh Ubuntu 26.04 LXC container ready for day-to-day development: hardened SSH, a sudo user, useful CLI tools, Docker, and language toolchains.

![Golden demo](docs/assets/golden-demo.gif)

## Quick Start

Install the latest release and open the guided TUI:

```bash
curl -fsSL https://raw.githubusercontent.com/sqamsqam/setup/main/install.sh | sudo bash
```

Prefer to install without launching?

```bash
curl -fsSL https://raw.githubusercontent.com/sqamsqam/setup/main/install.sh | sudo SETUP_SKIP_LAUNCH=1 bash
```

## Common Commands

```bash
sudo setup              # open the guided TUI
sudo setup fresh --user dev --key-file ~/.ssh/id_ed25519.pub --timezone UTC
sudo setup base --timezone UTC
sudo setup user --user dev --key-file ~/.ssh/id_ed25519.pub
sudo setup tools
sudo setup dev --user dev --all
sudo setup check
setup version
```

Instance helpers:

```bash
sudo setup network enable --allow-ssh
sudo setup network allow --port 443 --proto tcp
sudo setup guard install
sudo setup containers log-rotation
sudo setup updates check
sudo setup service create --user dev --name app --workdir /home/dev/app --cmd "npm start"
```

Use `--dry-run` to preview changes and `--demo` for clean non-mutating demos.

## Build Locally

```bash
make prep    # build bin/setup-linux-amd64
make taste   # vet, test, lint
make plate   # regenerate visual assets
make bake    # taste, plate, then local GoReleaser snapshot
```

## Docs

- [Usage](docs/usage.md): TUI flow, CLI reference, preview modes, and safety notes.
- [Development](docs/development.md): local workflow, releases, CI, and changelog guidance.
- [Visuals](docs/visuals.md): VHS tapes, screenshots, GIFs, and review expectations.

## Safety

`setup` is built for fresh Ubuntu LXC containers. It keeps provisioning explicit, validates SSH config before restart, avoids interactive prompts in CLI mode, and aims for idempotent re-runs where practical.
