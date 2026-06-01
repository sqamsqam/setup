#!/usr/bin/env bash
set -euo pipefail

repo="sqamsqam/setup"
asset="setup-linux-amd64"
install_path="/usr/local/bin/setup"
url="https://github.com/${repo}/releases/latest/download/${asset}"

if [[ "$(id -u)" -ne 0 ]]; then
	echo "Please run this installer with sudo or as root." >&2
	exit 1
fi

if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
	echo "setup currently ships a Linux amd64 release binary." >&2
	exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required to install setup." >&2
	exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

echo "Downloading setup from the latest release..."
curl -fsSL -o "$tmp" "$url"
install -m 0755 "$tmp" "$install_path"

echo "Installed setup to ${install_path}."
if [[ "${SETUP_SKIP_LAUNCH:-}" != "1" ]]; then
	echo "Opening setup..."
	exec "$install_path"
fi
