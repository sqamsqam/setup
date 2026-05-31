# Changelog

## v0.1.0 (Unreleased)

Initial release of the Ubuntu LXC provisioning tool.

### Added
- Interactive TUI mode with 7-step guided workflow
- CLI mode for scripting and automation
- Root bootstrap (locale, apt, base packages, SSH hardening,
  unattended upgrades, timezone, Docker)
- User creation with passwordless sudo and SSH key authentication
- SSH AllowUsers management and hardening
- Dry-run mode for safe preview
- CLI tool installation (ripgrep, fd, bat, yq, glow, gh)
- Go system-wide installation with SHA256 verification
- Node.js per-user installation via fnm
- Idempotent provisioning — safe to re-run
