#!/usr/bin/env bash
# Install the latest Go toolchain from golang.org.
# Downloads the latest stable linux-amd64 tarball, verifies SHA256,
# extracts to /usr/local/go, and sets up the system-wide PATH via
# /etc/profile.d/go.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_common.sh"

require_root

# ---- Ensure jq is available (needed for JSON parsing) ----
if ! command -v jq >/dev/null 2>&1; then
  apt update && apt install -y jq
fi

# ---- Download latest Go ----
GO_JSON="$(curl -fsSL 'https://go.dev/dl/?mode=json')"
GO_VERSION="$(echo "$GO_JSON" | jq -r '.[0].version')"
GO_TARBALL="${GO_VERSION}.linux-amd64.tar.gz"
GO_DOWNLOAD_URL="https://go.dev/dl/${GO_TARBALL}"
GO_SHA256="$(echo "$GO_JSON" | jq -r '.[0].files[] | select(.os == "linux" and .arch == "amd64" and .kind == "archive") | .sha256')"

echo "Latest Go version: ${GO_VERSION}"

# Download and verify checksum.
curl -fsSL "$GO_DOWNLOAD_URL" -o "/tmp/${GO_TARBALL}"
echo "${GO_SHA256}  /tmp/${GO_TARBALL}" | sha256sum -c --status

# Extract, removing any prior Go installation.
rm -rf /usr/local/go
tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
rm -f "/tmp/${GO_TARBALL}"

# ---- System-wide PATH ----
cat >/etc/profile.d/go.sh <<'EOF'
export PATH="/usr/local/go/bin:$PATH"
EOF
chmod 644 /etc/profile.d/go.sh

# Make Go available for the final version check below.
export PATH="/usr/local/go/bin:$PATH"

echo "Go toolchain installed: $(go version)"
