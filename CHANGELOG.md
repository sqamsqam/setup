# Changelog

## [Unreleased]

## [0.11.1] - 2026-06-07

### Added
- Apache License 2.0 project licensing and README license reference.

## [0.11.0] - 2026-06-07

### Added
- `setup fresh --firewall` can enable the UFW baseline after SSH user access is configured, with `--ssh-port` for explicit SSH-port allow rules.
- `setup network deny`, `setup network limit`, `setup network reload`, and `setup network disable` cover common UFW operations without shelling out.

### Changed
- `setup network enable --allow-ssh` now fails if SSH port detection fails unless `--ssh-port` is provided.
- Diagnostics now call out missing UFW clearly and warn when Docker and UFW are both active.

## [0.10.0] - 2026-06-07

### Changed
- Managed service creation now validates generated systemd units, uses an explicit user runtime directory, and restarts existing services when their unit file changes.
- Managed service listing now supports `--state` to show load, active, sub, and enabled state for setup-managed units.
- `setup containers log-rotation` now requires `--yes` because it can restart Docker when daemon configuration changes.

### Security
- Docker log rotation now validates the candidate daemon configuration with `dockerd --validate` before writing and restarting Docker.

## [0.9.0] - 2026-06-06

### Added
- Top-level group administration commands for create, delete, list, add-user, and remove-user workflows.
- TUI admin-console home menu with focused areas for fresh setup, users, groups, services, instance maintenance, tools, dev tools, and diagnostics.
- Modular user-management commands for selective login-user creation, SSH keys, SSH access, passwordless sudo, linger, existing group membership, disable, delete, and setup-owned service users.
- TUI user-management selections for the default login-user workflow and setup-owned service-user creation.
- Managed service lifecycle commands and TUI actions for create, status, logs, restart, list, disable, and remove.
- TUI coverage for CLI-only instance administration helpers including UFW status/list/delete/reset, fail2ban status/unban, Docker disk/prune, update status actions, reboot, and configurable firewall/fail2ban/Docker options.

### Changed
- The TUI now starts with no selected actions and requires choosing a focused admin area before reviewing a plan.
- Setup-managed SSH `AllowUsers` now tracks an explicit requested-user list instead of regenerating from every UID >= 1000 account.
- Passwordless sudo disable now removes only setup-marked sudoers files and refuses unmanaged sudoers files.
- Firewall rule deletion, firewall reset, and Docker prune now require `--yes`.
- Managed service disable and remove now require confirmation and only operate on setup-marked unit files.

### Security
- Setup-owned admin files now refuse to replace unmanaged existing files for SSH hardening, unattended upgrades, fail2ban, and managed user-service units.
- Managed service status, logs, and restart operations now require the target unit file to be setup-managed.
- Fail2ban jail options now reject multiline values and validate the candidate configuration before restart.

## [0.8.8] - 2026-06-05

### Fixed
- Managed config writers now fail fast on unreadable existing files instead of replacing from ambiguous state.
- SSH `authorized_keys` and Go profile writes now explicitly preserve their intended file modes during atomic replacement.
- yq and GitHub release `.deb` installers now clean up temporary download files when downloads fail.

## [0.8.7] - 2026-06-05

### Fixed
- Managed sudoers, SSH AllowUsers, and apt source list writes now preserve their intended file modes when atomically replacing files.

## [0.8.6] - 2026-06-01

### Changed
- GitHub Actions workflows now use the latest Node 24-compatible action versions.

## [0.8.5] - 2026-06-01

### Changed
- TUI run-step statuses now use compact glyph styling with a clear success tick.
- TUI step output is hidden by default and can be expanded per step with keyboard or mouse input.
- Demo success recording now shows expanded step output in the generated visual assets.

## [0.8.4] - 2026-06-01

### Changed
- Demo recordings now hide color-environment setup while preserving colorized terminal GIFs and use slower section transitions.

## [0.8.3] - 2026-06-01

### Changed
- Refined the TUI visual system with bolder semantic colors, sharper panels, clearer focused states, and denser provisioning-plan copy.

## [0.8.2] - 2026-06-01

### Changed
- Installer reruns now verify the latest release checksum and atomically replace the existing `setup` binary.

## [0.8.1] - 2026-06-01

### Changed
- Dry-run and demo Ubuntu codename simulation now matches Ubuntu 26.04 `resolute`.

### Fixed
- GitHub CLI apt source generation now trims the detected architecture before writing the source list.

### Tests
- Added apt key and source coverage for Docker, Glow, and GitHub CLI installers.
- Added coverage for managed file writes and update-management helpers.

## [0.8.0] - 2026-06-01

### Added
- Top-level `install.sh` for a shorter latest-release install-and-launch one-liner.
- Focused usage, development, and visual workflow docs under `docs/`.

### Changed
- Top-level CLI commands now use a warmer public vocabulary: `base`, `user`, `tools`, `dev`, `check`, `network`, `guard`, `containers`, and `fresh`.
- Awkward nested command names were renamed to `network list`, `containers log-rotation`, `updates reboot-needed`, and `updates unattended`.
- Public make targets now use `prep`, `taste`, `plate`, and `bake`, with `make bake` running checks, visual generation, and a local GoReleaser snapshot.
- README is now a shorter landing page with detailed usage and development material moved into docs.

### Removed
- Old CLI command names and old public make target names are no longer kept as aliases.

## [0.7.0] - 2026-06-01

### Added
- VHS-based visual review workflow with deterministic demo-mode tapes, generated screenshots, supporting GIFs, and a golden demo GIF.
- Demo mode (`--demo`) for non-mutating public demos without dry-run banners or `[DRY-RUN]` log prefixes.
- `make bake` alias for regenerating the happy-path golden demo animation.

### Changed
- TUI dry-run mode no longer shows a root warning, and running/done panels reserve enough space for their borders and help text.
- VHS visual tapes now use demo mode while preserving dry-run safety.

## [0.6.0] - 2026-06-01

### Added
- UFW firewall management commands and TUI actions for safe baseline setup, rule status, allow rules, deletion, and reset.
- fail2ban SSH jail installation, status, and unban commands.
- Docker log rotation configuration plus disk usage and safe prune helpers.
- Read-only doctor diagnostics for LXC/VM/system state, apt locks, reboot state, SSH, UFW, and Docker.
- Update management commands for package checks, upgrades, reboot-required state, unattended-upgrades status, failed units, and confirmed reboot.
- Setup-managed per-user systemd service creation, status, logs, and restart helpers.
- Rust toolchain installation and optional Go/Node ecosystem tools: golangci-lint, GoReleaser, govulncheck, and pnpm.

### Changed
- System bootstrap now configures Docker json-file log rotation after installing Docker.
- The TUI setup plan now includes instance management and expanded toolchain actions alongside the original bootstrap flow.

## [0.5.4] - 2026-06-01

### Changed
- TUI provisioning logs now use a fixed-height output pane with a header, scroll position status, and clearer step separators.

### Fixed
- TUI provisioning logs no longer expand the run screen when long output lines would wrap.

## [0.5.3] - 2026-06-01

### Fixed
- TUI plan editing now toggles selected steps when Space is pressed.

## [0.5.2] - 2026-06-01

### Changed
- TUI now uses the full terminal screen and gives the provisioning log more room, including a wider run layout on capable terminals.
- TUI provisioning logs now visually distinguish commands, step starts, and completion/error messages.

### Fixed
- TUI confirmation screen can now scroll when the full plan does not fit.
- TUI running and done screens now keep the step list and log inside the terminal height on narrower terminals.
- TUI timezone input now bounds visible fuzzy matches so validation errors do not push the screen past the terminal height.

## [0.5.1] - 2026-06-01

### Added
- TUI timezone input now supports fuzzy searching and match selection.

## [0.5.0] - 2026-06-01

### Added
- TUI provisioning plans are now configurable before execution, including per-tool toggles for ripgrep, fd, bat, yq, glow, gh, Go, and Node.js.
- The TUI now uses Bubble Tea component models from Bubbles for lists, text input, SSH key editing, spinner, progress, help text, and scrollable output.

### Changed
- The default TUI plan replaces the fixed Full Setup chain while still selecting the full bootstrap flow by default.
- TUI provisioning output is captured inside the running/done views instead of writing directly over the terminal UI.

## [0.4.2] - 2026-06-01

### Security
- Root command execution now uses a fixed safe PATH instead of the caller's inherited PATH.
- Charm and GitHub CLI apt keys are verified from temporary keyring files before replacing trusted keyring paths.

## [0.4.1] - 2026-06-01

### Fixed
- Release binary asset name is now `setup-linux-amd64` instead of including version, OS, and architecture suffixes.

## [0.4.0] - 2026-06-01

### Added
- Searchable timezone selection and validation in the TUI
- TUI dry-run transcripts for provisioning steps

### Changed
- Docker installation now uses the official apt repository with GPG fingerprint verification instead of `get.docker.com`
- fnm installation now uses a pinned release zip with SHA256 verification instead of piping a remote install script
- TUI Full Setup now blocks continuation after failed steps and offers retry
- CLI examples and docs now prefer `--key-file` for SSH public keys
- Release workflow now runs lint and emits the documented `setup-linux-amd64` binary name

### Fixed
- SSH public key paste handling in the TUI
- CLI tool installation no longer attempts GitHub `.deb` downgrades when the distro package is already installed
- SSH drop-in validation now checks effective sshd config and rolls back failed candidates
- Existing users are validated as non-system accounts and their passwd home directory is used for SSH keys
- Unknown CLI commands now fail instead of opening the TUI

### Security
- Downloaded binaries are verified before replacing live executables
- Checksum verification fails closed for GitHub `.deb` installs
- Shell-interpolated checksum verification was replaced with Go SHA256 checks

## [0.3.0] - 2026-05-31

### Added
- Menu-driven TUI wizard replaces linear 7-screen flow: users choose an action first (Full Setup, System Bootstrap, Add User, Install CLI Tools, Install Dev Tools), then follow a guided wizard tailored to that action
- Re-runnable individual actions — after completing one task the user returns to the main menu, enabling repeated use without restarting the tool
- Full Setup chains four guided wizards sequentially (Bootstrap → Add User → CLI Tools → Dev Tools), each with its own confirm/run/done cycle and chain progress tracking
- Spinner animation during provisioning steps using `tea.Tick` + `tea.Batch` for live feedback
- Native terminal progress bar via `tea.ProgressBar` (OSC 9;4) for chain execution in supported terminals
- In-terminal progress bar drawn with lipgloss box-drawing characters for chain execution
- Flow-based navigation with `esc` stepping back through wizard screens and returning to the main menu

### Changed
- TUI dry-run now uses `DryRunner` (same execution path as CLI mode) instead of skipping provisioning calls — `[DRY-RUN]` messages are silenced in TUI mode while the TUI displays its own `(dry run)` status
- Step-selection toggle screen (checkbox list) replaced by a cursor-driven main menu with descriptions
- Welcome screen replaced by main menu with persistent root-privilege warning
- Running view shows chain progress (completed/pending steps) for Full Setup and a simple spinner for standalone actions
- Done view shows chain continuation prompts (`Next: Add User — enter continue · esc back to menu · q quit`) for Full Setup

### Removed
- `screenWelcome` and `screenStepSelect` from the TUI screen enum
- `stepFlags`, `cursor`, `selectedSteps()`, `needsUserInput()`, `needsKeyInput()`, `needsTimezoneInput()`, `hasSelections()` from the TUI model
- `tuiRunner` wrapper — replaced by `newWizardRunner` which uses `DryRunner` for dry-run parity

## [0.2.0] - 2026-05-31

### Changed
- Replaced hand-rolled CLI arg parsing with `urfave/cli/v3` for structured subcommand routing, automatic help generation, and flag suggestions
- Added short-flag aliases for all CLI subcommands (`-u`, `-k`, `-t`, `-b`, `-a`, `-i`, `-d`, `-f`, `-v`)
- Wrapped root warning logic into `provisioningAction` helper
- Replaced global `testRunner` with `RunnerFactory` parameter for thread safety

### Fixed
- Resolved all golangci-lint `errcheck` and `staticcheck` issues

### CI
- Bumped `golangci-lint-action` from v6 to v9; reverted version pin to track latest

## [0.1.0] - 2026-05-31

Initial release of the Ubuntu LXC provisioning tool.

### Added
- All file operations use Go native methods (WriteFile, ReadFile, Rename, Chmod, Chown, MkdirAll) via expanded CmdRunner interface — safer, no shell injection risk through file paths
- SSH authorized_keys appends (rather than replaces) on re-run — preserves existing keys, skips duplicates
- Idempotency check for SSH keys — re-running with the same key is a no-op
- SSH hardening: ClientAliveInterval, ClientAliveCountMax, MaxSessions, MaxStartups
- Hardening config validated with `sshd -t -f <tmpfile>` before installation (was: after)
- AllowUsers config validated with `sshd -t -f <tmpfile>` before installation (was: validatating old config)
- `--key-file <path>` CLI flag for safer SSH key passing (avoids /proc exposure)
- Pinned Go version support (fallback to API fetch when pin is empty)
- GitHub CLI GPG key fingerprint verification
- `--dry-run` in TUI now shows step progression instead of silently skipping
- Inline input validation in TUI (username, SSH key validated on enter, error shown before proceeding)
- TUI `q`/`ctrl+c` now works during provisioning (Running screen)
- Confirmation screen shows managed files list
- Adaptive colors for light/dark terminal support (softened red for accessibility)

### Changed
- Default timezone from Australia/Sydney to UTC
- All temp files use `os.CreateTemp` with random suffixes instead of hardcoded `/tmp/` paths
- All file ownership changes use Go `os.Chown` via `runner.Chown()` with `os/user.Lookup` resolution
- `installYq` dry-run guard moved before download (was: downloading in dry-run mode)
- `chmod +x` → `chmod 0755` for explicit permissions
- Status icons use wrapped unicode with ASCII-friendly characters

### Removed
- `CombinedOutput` from CmdRunner interface (unused)
- Unused `centerText` function

[0.11.0]: https://github.com/sqamsqam/setup/releases/tag/v0.11.0
[0.10.0]: https://github.com/sqamsqam/setup/releases/tag/v0.10.0
[0.9.0]: https://github.com/sqamsqam/setup/releases/tag/v0.9.0
[0.8.4]: https://github.com/sqamsqam/setup/releases/tag/v0.8.4
[0.8.3]: https://github.com/sqamsqam/setup/releases/tag/v0.8.3
[0.8.2]: https://github.com/sqamsqam/setup/releases/tag/v0.8.2
[0.8.1]: https://github.com/sqamsqam/setup/releases/tag/v0.8.1
[0.8.0]: https://github.com/sqamsqam/setup/releases/tag/v0.8.0
[0.7.0]: https://github.com/sqamsqam/setup/releases/tag/v0.7.0
[0.6.0]: https://github.com/sqamsqam/setup/releases/tag/v0.6.0
[0.5.4]: https://github.com/sqamsqam/setup/releases/tag/v0.5.4
[0.5.3]: https://github.com/sqamsqam/setup/releases/tag/v0.5.3
[0.5.2]: https://github.com/sqamsqam/setup/releases/tag/v0.5.2
[0.5.1]: https://github.com/sqamsqam/setup/releases/tag/v0.5.1
[0.5.0]: https://github.com/sqamsqam/setup/releases/tag/v0.5.0
[0.4.2]: https://github.com/sqamsqam/setup/releases/tag/v0.4.2
[0.4.1]: https://github.com/sqamsqam/setup/releases/tag/v0.4.1
[0.4.0]: https://github.com/sqamsqam/setup/releases/tag/v0.4.0
[0.3.0]: https://github.com/sqamsqam/setup/releases/tag/v0.3.0
[0.2.0]: https://github.com/sqamsqam/setup/releases/tag/v0.2.0
[0.1.0]: https://github.com/sqamsqam/setup/releases/tag/v0.1.0
