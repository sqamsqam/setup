# Changelog

## [Unreleased]

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

[0.4.1]: https://github.com/sqamsqam/setup/releases/tag/v0.4.1
[0.4.0]: https://github.com/sqamsqam/setup/releases/tag/v0.4.0
[0.3.0]: https://github.com/sqamsqam/setup/releases/tag/v0.3.0
[0.2.0]: https://github.com/sqamsqam/setup/releases/tag/v0.2.0
[0.1.0]: https://github.com/sqamsqam/setup/releases/tag/v0.1.0
