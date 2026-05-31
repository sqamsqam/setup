#!/usr/bin/env bash
# Initial system bootstrap for a fresh Ubuntu LXC container.
# - Sets locale to en_US.UTF-8
# - Upgrades all packages
# - Installs core utilities (ssh, git, curl, vim, htop, etc.)
# - Hardens SSH (pubkey-only, no root login)
# - Enables unattended security updates
# - Installs Docker
# Override TIMEZONE env var to change the default (Australia/Sydney).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_common.sh"

require_root

TIMEZONE="${TIMEZONE:-Australia/Sydney}"

# ---- Locale ----
# Uncomment en_US.UTF-8 in locale.gen (idempotent: no-op if already uncommented).
sed -i 's/^# *en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen
locale-gen
update-locale LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8

# ---- Base system update ----
apt update
apt full-upgrade -y

# ---- Core packages ----
apt install -y \
  sudo openssh-server curl wget git zip unzip htop jq mosh tmux gpg vim \
  ca-certificates unattended-upgrades systemd

# ---- Automatic security updates ----
cat >/etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
EOF

timedatectl set-timezone "$TIMEZONE"

# ---- SSH hardening ----
mkdir -p /etc/ssh/sshd_config.d

cat >/etc/ssh/sshd_config.d/99-hardening.conf <<'EOF'
PermitRootLogin no
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
MaxAuthTries 3
LoginGraceTime 30
EOF

passwd -l root || true

# ---- Docker ----
# Uses Docker's official convenience script (curl | sh).
# Review https://get.docker.com before running if untrusted.
curl -fsSL https://get.docker.com | sh

# ---- Start SSH ----
systemctl enable --now ssh

# Validate SSH config before restarting (avoids lockout on misconfiguration).
if sshd -t 2>/dev/null; then
  systemctl restart ssh
else
  echo "WARNING: sshd configuration test failed — SSH will not be restarted." >&2
fi

echo "Root bootstrap complete."
