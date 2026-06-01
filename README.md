# setup

`setup` is a Go CLI for provisioning fresh Ubuntu 26.04 LXC containers. It has an interactive Bubble Tea terminal UI for guided setup and a non-interactive CLI for scripts.

![Golden demo](docs/assets/golden-demo.gif)

## Highlights

- Guided TUI for choosing system bootstrap, user creation, instance management, CLI tools, and development toolchains.
- Scriptable CLI subcommands for bootstrap, users, tools, diagnostics, firewall, fail2ban, Docker maintenance, updates, and services.
- Safe preview modes that log intended commands without changing the host.
- SSH hardening by default: root login off, password auth off, pubkey auth on, `sshd -t` validation before restart.
- Idempotent behavior where practical: existing users, SSH keys, managed config files, and installed tool versions are handled carefully.
- Visual demos and screenshots generated with Charm VHS for UI review and README assets.

## Target

- Ubuntu 26.04 LXC
- Linux amd64
- Root execution for provisioning
- Per-user execution for user-level toolchains

This is a practical provisioning helper, not a general-purpose configuration-management system.

## Installation

Download the release binary:

```bash
sudo curl -fsSL -o /usr/local/bin/setup \
  https://github.com/sqamsqam/setup/releases/latest/download/setup-linux-amd64
sudo chmod +x /usr/local/bin/setup
```

Or build from source:

```bash
git clone https://github.com/sqamsqam/setup.git
cd setup
make build
sudo install -m 0755 bin/setup-linux-amd64 /usr/local/bin/setup
```

## Interactive Usage

Run without a subcommand to open the TUI:

```bash
sudo setup
```

The TUI lets you toggle provisioning work before anything runs:

- System bootstrap: locale, packages, timezone, unattended upgrades, SSH hardening, Docker.
- Add user: sudo user, SSH key, linger, `AllowUsers`.
- Instance management: UFW, common ports, fail2ban, Docker log rotation, diagnostics, update checks.
- CLI tools: ripgrep, fd, bat, yq, glow, gh.
- Development tools: Go, Node.js, Rust, golangci-lint, GoReleaser, govulncheck, pnpm.

Use arrow keys to move, Space to toggle, `/` to filter, Enter to continue, and Esc to go back.

## CLI Usage

```bash
sudo setup bootstrap --timezone UTC
sudo setup add-user --user dev --key-file ~/.ssh/id_ed25519.pub
sudo setup install-tools
sudo setup devtools --user dev --all
sudo setup doctor
sudo setup full --user dev --key-file ~/.ssh/id_ed25519.pub --timezone UTC
setup version
```

Additional command groups:

```bash
sudo setup firewall enable --allow-ssh
sudo setup firewall allow --port 443 --proto tcp
sudo setup fail2ban install
sudo setup docker logs-config
sudo setup docker disk
sudo setup updates check
sudo setup service create --user dev --name app --workdir /home/dev/app --cmd "npm start"
```

Use `setup --help` or `setup <command> --help` for the full command reference.

## Preview Modes

Add `--dry-run` to preview work safely:

```bash
sudo setup --dry-run full \
  --user dev \
  --key-file ~/.ssh/id_ed25519.pub \
  --timezone UTC
```

Use `--demo` for the same non-mutating behavior without dry-run banners or `[DRY-RUN]` log prefixes:

```bash
setup --demo
setup --demo full --user dev --key-file ~/.ssh/id_ed25519.pub --timezone UTC
```

All visual demos use demo mode. VHS tapes must never depend on credentials, external services, or live host state.

## Visual Demos

Visual assets are generated with [Charm VHS](https://github.com/charmbracelet/vhs), a tool for scripting terminal recordings.

- Tapes live in `demo/`.
- The canonical happy-path tape is `demo/golden.tape`.
- Screenshots are generated from `demo/screenshots/*.tape`.
- Generated assets live in `docs/assets/`.

Key outputs:

```text
docs/assets/golden-demo.gif
docs/assets/gifs/navigation.gif
docs/assets/gifs/success.gif
docs/assets/gifs/error.gif
docs/assets/screenshots/*.png
```

Install visual tooling:

```bash
make install-visual-tools
```

Regenerate everything:

```bash
make review-ui
```

`make review-ui` installs or verifies VHS tooling, builds the binary, validates tapes, regenerates screenshots, regenerates supporting GIFs, regenerates the golden demo, and checks that expected outputs exist.

Focused commands:

```bash
make screenshots     # PNG screenshots
make demo-gif        # supporting GIFs
make golden-demo     # docs/assets/golden-demo.gif
make bake            # alias for the happy-path golden demo animation
make visual-test     # tape and asset validation
```

When a UI, UX, layout, workflow, navigation, or styling change is made, run `make review-ui`, review the regenerated assets, and commit updated visual files when they changed.

## Development

```bash
make build
make test
make vet
make lint
make check
make run-cli ARGS="version"
```

Requirements:

- Go 1.26+
- `golangci-lint` for `make lint`
- `sudo` and `apt-get` for automatic visual dependency installation on Ubuntu-like systems

## CI

GitHub Actions runs the normal Go checks and a separate visual job. The visual job runs `make review-ui` and uploads `docs/assets` as a CI artifact so reviewers can inspect generated screenshots and GIFs.

## Project Layout

```text
cmd/setup/          entry point and mode detection
internal/cli/       urfave/cli subcommands
internal/tui/       Bubble Tea / Bubbles / Lip Gloss TUI
internal/exec/      real, dry-run, and demo command runners
internal/system/    root bootstrap
internal/user/      user creation and SSH key validation
internal/tools/     CLI tool installers
internal/devtools/  Go, Node.js, Rust, and ecosystem tools
demo/               VHS tapes
docs/assets/        generated visual assets
scripts/            visual tooling helpers
```

## Safety Notes

- SSH password login and root SSH login are disabled by the managed bootstrap.
- SSH config is validated before restart.
- Managed files are marked and compared before replacement where practical.
- CLI mode must not introduce interactive prompts.
- Demo mode is the required path for visual demos; dry-run mode remains available for explicit safe previews.
