# AGENTS.md

## Project

This repository contains a Go CLI application for provisioning fresh Ubuntu 26.04 LXC containers. It provides both an interactive terminal UI (default) and a non-interactive CLI (for scripting/automation).

Module path: `github.com/sqamsqam/setup`
Minimum Go version: 1.26

Primary target:

- Ubuntu 26.04 LXC
- Linux amd64
- Root execution for system provisioning
- Per-user execution for user development tooling

This is a practical provisioning helper, not a general-purpose configuration-management system.

## Modes

The tool has two operating modes.

### TUI mode (default when run with no subcommand)

A 7-screen interactive terminal UI built with Bubble Tea v2 + Lipgloss v2:

1. Welcome
2. Step selection (4 toggleable provisioning steps)
3. Username input
4. SSH public key input
5. Timezone input
6. Confirmation
7. Running / Done

Code lives in `internal/tui/` (5 files: model, view, update, run, steps).

### CLI mode (when a recognized subcommand is given)

Non-interactive, suitable for scripting. See the Command Reference in the README.

## Priorities

When working in this repo, prioritise:

1. Safety
2. Predictability
3. Idempotency where practical
4. Clear errors
5. Boring, maintainable Go
6. Compatibility with fresh Ubuntu LXC containers

Avoid clever abstractions unless they clearly reduce risk or repetition.

## Behavioural Baseline

The provisioning flow includes:

- locale generation
- apt update and upgrade
- base package installation
- unattended security upgrades
- timezone configuration
- SSH hardening (PermitRootLogin no, PasswordAuthentication no, PubkeyAuthentication yes)
- AllowUsers configuration (non-system users, UID >= 1000)
- sshd -t validation before restarting SSH
- root password locking
- user creation with passwordless sudo
- linger support (loginctl enable-linger)
- authorized SSH key installation
- Docker installation via get.docker.com
- Docker group membership
- CLI tool installation (ripgrep, fd, bat via GitHub releases; yq binary; glow via charm.sh apt repo; gh via GitHub CLI apt repo)
- Go installation (system-wide, from go.dev, SHA256 verified)
- Node.js toolchain (fnm, Node, corepack, TypeScript, tsx)

Improve correctness and safety where needed, but document meaningful behaviour changes.

## Internal Packages

Code lives under `internal/`. Each package has a clear, narrow purpose:

```
cmd/setup/          Entry point — mode detection (TUI vs CLI), version injection
internal/
  cli/              CLI mode: subcommand routing with urfave/cli/v3, dry-run setup
  tui/              Bubble Tea v2 interactive terminal UI (model, view, update, run, steps)
  exec/             CmdRunner interface, RealRunner, DryRunner, helper formatters
  system/           Root bootstrap: locale, packages, SSH, unattended upgrades, Docker
  user/             User creation, sudo, SSH key, linger, AllowUsers, input validation
  tools/            CLI tool installers (ripgrep, fd, bat, yq, glow, gh)
  devtools/         Dev toolchain installers (Go system-wide, Node.js per-user via fnm)
  github/           GitHub Releases API client for asset discovery
```

Keep packages small and obvious. Do not over-engineer this into a framework.

## Command Execution

All command execution goes through the `CmdRunner` interface in `internal/exec`:

```go
type CmdRunner interface {
    Run(name string, args ...string) error
    Output(name string, args ...string) (string, error)
    RunAsUser(user, name string, args ...string) error
    Shell(script string) error
}
```

Two implementations:

- `RealRunner` — executes real OS commands via `os/exec`. Sets `DEBIAN_FRONTEND=noninteractive` in CLI mode.
- `DryRunner` — logs `[DRY-RUN]` to stderr without executing anything.

`RealRunner.RunAsUser` shells out via `sudo -iu <user> -- <cmd>`. `RealRunner.Shell` runs a script through `bash -c` (used for Docker get.docker.com and fnm install scripts).

Do not scatter direct `os/exec` calls throughout task packages unless there is a strong reason.

Most provisioning should still shell out to standard system tools. This is expected and preferred for: apt, dpkg, systemctl, loginctl, usermod, adduser, passwd, curl, wget, npm, fnm, corepack.

## Dependencies

Direct dependencies (fully vendored):

- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/lipgloss/v2` — terminal styling
- `github.com/urfave/cli/v3` — CLI framework (subcommand routing, flag parsing)

## CLI Parsing

CLI arg parsing in `internal/cli/cli.go` uses urfave/cli/v3. Flags use `--key=value` or `--key value` syntax. Global flag `--dry-run` can appear anywhere in the arg list and is filtered before the framework processes the command.

## Safety Rules

Do not:

- hardcode a personal username
- hardcode a home directory
- silently ignore serious command failures
- weaken SSH hardening
- enable password SSH login
- enable root SSH login
- delete existing user data
- overwrite unmanaged user files without care
- introduce interactive prompts in CLI mode
- assume the current shell environment is available

Always:

- validate usernames (regex: `^[a-z_][a-z0-9_-]*$`, max 32 chars)
- validate SSH public keys (must start with a recognized key type prefix)
- resolve user home directories properly
- use explicit paths
- mark generated managed files clearly (`# Managed by setup — do not edit`)
- preserve existing users where practical
- set `DEBIAN_FRONTEND=noninteractive` for apt operations (CLI mode)
- validate SSH config with `sshd -t` before restarting the daemon
- compare old config before overwriting to avoid unnecessary restarts
- print what is happening
- fail fast on unsafe or ambiguous states

## Idempotency

Provisioning commands should be safe to run more than once where practical.

Before changing system state, prefer checking whether the desired state already exists.

Examples implemented:

- do not fail if a user already exists and is valid (checked via `id <user>`)
- do not duplicate group memberships
- do not append duplicate SSH keys
- do not recreate files unnecessarily
- compare SSH/sudoers config before overwriting
- do not reinstall tools if the current version matches

Idempotency is important, but not at the cost of hiding real errors.

## Testing

Tests live alongside their packages under `internal/`. Run with:

```
make test          # go test ./internal/...
```

Good test targets:

- username validation
- SSH key validation
- GitHub release asset selection
- managed file content (SSH config, unattended upgrades config, Go profile script)
- dry-run behaviour (command logging, no execution)

Do not unit test real provisioning side effects. Do not write tests that require mutating the host system.

## GitHub Actions / CI

CI workflow (`.github/workflows/ci.yml`):

- Trigger: push to `main`, any pull request
- Runs `go test ./internal/...`
- Runs `go vet ./internal/... ./cmd/...`
- Runs golangci-lint

## GitHub Actions / Releases

Release workflow (`.github/workflows/release.yml`):

- Trigger: push of tag matching semver `v[0-9]+.[0-9]+.[0-9]+`
- Runs the same checks as CI
- GoReleaser builds a single `linux/amd64` binary, attaches it to the release as `setup-linux-amd64`
- A `checksums.txt` file is generated alongside the binary

Configuration: `.goreleaser.yml`

## Changelog

`CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/) format.

Before tagging a release:
1. Move the `(Unreleased)` heading under a new version heading
2. Ensure all user-visible changes are documented under Added / Changed / Fixed / Security
3. Open a PR to update the changelog, then tag after merge

Every pull request with user-visible changes should add a corresponding entry under the Unreleased section.

## Makefile

Available targets:

```
make build            # Build bin/setup-linux-amd64 (with ldflags: version, commit, date)
make test             # go test ./internal/...
make vet              # go vet ./internal/... ./cmd/...
make lint             # golangci-lint run ./...
make check            # Runs vet → test → lint in sequence
make clean            # Remove bin/
make run-cli ARGS="..."  # go run ./cmd/setup with given args
```

## Documentation

Keep the README accurate. When changing behaviour, update the README in the same change.

The README should cover: purpose, target environment, installation, interactive/TUI mode, CLI command reference, dry-run usage, fresh LXC bootstrap flow, release process, development workflow, safety notes.

## Style

Be practical. Prefer small, reviewable changes. Use clear names. Avoid speculative features.

Do not add: configuration systems, plugins, daemons, background services, or remote orchestration unless explicitly requested.

## Before Finishing

Before considering work complete, run:

```
make vet && make test
```

Summarise:

- what changed
- how it was tested
- any risks or follow-up work
