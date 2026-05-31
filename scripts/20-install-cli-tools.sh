#!/usr/bin/env bash
# Install modern CLI utilities from GitHub releases and APT repos.
# - ripgrep (rg): recursive grep
# - fd: fast find alternative
# - bat: cat with syntax highlighting
# - yq: YAML/JSON CLI processor
# - glow: terminal markdown renderer (via charm.sh APT repo)
#
# Set GITHUB_TOKEN env var to avoid API rate limits (60 req/hr unauthenticated).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_common.sh"

require_root

apt update
apt install -y curl wget jq gpg ca-certificates

# Query GitHub Releases API for the latest .deb asset matching a regex pattern.
# Uses GITHUB_TOKEN if set, otherwise falls back to unauthenticated requests.
install_latest_github_deb() {
  local repo="$1"
  local pattern="$2"

  local api_url="https://api.github.com/repos/${repo}/releases/latest"

  local curl_opts=(-fsSL)
  [[ -n "${GITHUB_TOKEN:-}" ]] && curl_opts+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")

  local deb_url
  deb_url="$(
    curl "${curl_opts[@]}" "$api_url" |
      jq -r --arg pattern "$pattern" '
        .assets[]
        | select(.name | test($pattern))
        | .browser_download_url
      ' |
      head -n1
  )"

  if [[ -z "$deb_url" || "$deb_url" == "null" ]]; then
    echo "ERROR: Could not find matching deb for $repo using pattern: $pattern"
    exit 1
  fi

  local tmp
  tmp="$(mktemp --suffix=.deb)"

  echo "Downloading $repo..."
  wget -q "$deb_url" -O "$tmp"

  apt install -y "$tmp"
  rm -f "$tmp"
}

install_latest_github_deb "BurntSushi/ripgrep" 'ripgrep_.*_amd64\.deb$'
install_latest_github_deb "sharkdp/fd" 'fd_.*_amd64\.deb$'
install_latest_github_deb "sharkdp/bat" 'bat_.*_amd64\.deb$'

# yq (mikefarah/go-yq) — statically-linked binary, no external deps
wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq
if [[ ! -s /usr/local/bin/yq ]]; then
  echo "ERROR: Failed to download yq."
  exit 1
fi
chmod +x /usr/local/bin/yq

# glow (charm.sh apt repo with signed keyring)
mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" >/etc/apt/sources.list.d/charm.list

apt update
apt install -y glow

echo "CLI tools installed."
