# Changelog

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

[0.1.0]: https://github.com/sqamsqam/setup/releases/tag/v0.1.0
