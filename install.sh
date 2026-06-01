#!/usr/bin/env bash
set -euo pipefail

repo="sqamsqam/setup"
asset="setup-linux-amd64"
install_path="/usr/local/bin/setup"
release_url="https://github.com/${repo}/releases/latest/download"
asset_url="${release_url}/${asset}"
checksums_url="${release_url}/checksums.txt"

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

if ! command -v sha256sum >/dev/null 2>&1; then
	echo "sha256sum is required to verify setup." >&2
	exit 1
fi

install_dir="$(dirname "$install_path")"
tmp_dir="$(mktemp -d)"
tmp_asset="${tmp_dir}/${asset}"
tmp_checksums="${tmp_dir}/checksums.txt"
tmp_install="${install_dir}/.setup-install.$$"
trap 'rm -rf "$tmp_dir"; rm -f "$tmp_install"' EXIT

echo "Downloading setup from the latest release..."
curl -fsSL -o "$tmp_asset" "$asset_url"
curl -fsSL -o "$tmp_checksums" "$checksums_url"

echo "Verifying checksum..."
(
	cd "$tmp_dir"
	awk -v asset="$asset" '$2 == asset { print; found = 1 } END { exit found ? 0 : 1 }' checksums.txt | sha256sum -c -
)

install -m 0755 "$tmp_asset" "$tmp_install"
mv -f "$tmp_install" "$install_path"

if command -v "$install_path" >/dev/null 2>&1; then
	echo "Installed setup to ${install_path} ($("$install_path" version))."
else
	echo "Installed setup to ${install_path}."
fi
if [[ "${SETUP_SKIP_LAUNCH:-}" != "1" ]]; then
	echo "Opening setup..."
	exec "$install_path"
fi
