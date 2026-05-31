#!/usr/bin/env bash
# Create a new user with sudo access and SSH key authentication.
# Usage: sudo ./10-add-user.sh <username> '<ssh-public-key>'
# - Adds user to sudo and docker groups
# - Grants passwordless sudo
# - Configures SSH authorized_keys with proper permissions
# - Updates AllowUsers in sshd_config to include all non-system users
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_common.sh"

require_root

USER_TO_ADD="${1:-}"
USER_PUB_KEY="${2:-}"

if [[ -z "$USER_TO_ADD" || -z "$USER_PUB_KEY" ]]; then
  echo "Usage: sudo $0 <username> '<ssh-public-key>'"
  exit 1
fi

# Basic username validation (POSIX: lowercase start, letters/digits/hyphens/underscores).
if [[ ! "$USER_TO_ADD" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
  echo "ERROR: Invalid username: $USER_TO_ADD"
  exit 1
fi

# Basic SSH public key sanity check (must start with a known key type prefix).
if [[ ! "$USER_PUB_KEY" =~ ^(ssh-(rsa|ed25519|dss)|ecdsa-sha2-nistp(256|384|521)|sk-ssh-ed25519|sk-ecdsa-sha2-nistp256) ]]; then
  echo "ERROR: Provided key does not look like a valid SSH public key."
  exit 1
fi

# ---- Create user (if not already present) ----
if ! id "$USER_TO_ADD" >/dev/null 2>&1; then
  adduser --disabled-password --gecos "" "$USER_TO_ADD"
fi

# Enable lingering so user services survive logout (harmless if systemd-logind unavailable).
loginctl enable-linger "$USER_TO_ADD" || true

# ---- Group membership ----
usermod -aG sudo "$USER_TO_ADD"
usermod -aG docker "$USER_TO_ADD" || true  # docker group may not exist yet

# ---- Passwordless sudo ----
printf '%s ALL=(ALL) NOPASSWD:ALL\n' "$USER_TO_ADD" >"/etc/sudoers.d/$USER_TO_ADD"
chmod 440 "/etc/sudoers.d/$USER_TO_ADD"

# ---- SSH key ----
install -d -m 700 -o "$USER_TO_ADD" -g "$USER_TO_ADD" "/home/$USER_TO_ADD/.ssh"

printf '%s\n' "$USER_PUB_KEY" >"/home/$USER_TO_ADD/.ssh/authorized_keys"
chown "$USER_TO_ADD:$USER_TO_ADD" "/home/$USER_TO_ADD/.ssh/authorized_keys"
chmod 600 "/home/$USER_TO_ADD/.ssh/authorized_keys"

# ---- SSH AllowUsers (dynamic, rebuilt from /etc/passwd) ----
# Only restart SSH if the AllowUsers list actually changed.
ALLOW_FILE="/etc/ssh/sshd_config.d/98-allow-users.conf"
ALLOW_TMP="$(mktemp)"
{
  printf 'AllowUsers'
  awk -F: '$3 >= 1000 && $1 != "nobody" { printf " " $1 }' /etc/passwd
  printf '\n'
} > "$ALLOW_TMP"

if ! cmp -s "$ALLOW_TMP" "$ALLOW_FILE" 2>/dev/null; then
  mv "$ALLOW_TMP" "$ALLOW_FILE"
  systemctl restart ssh
else
  rm -f "$ALLOW_TMP"
fi

echo "User provisioned: $USER_TO_ADD"
