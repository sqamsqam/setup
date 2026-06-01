#!/usr/bin/env bash
set -euo pipefail

vhs_version="${VHS_VERSION:-v0.11.0}"
export PATH="${HOME}/go/bin:${PATH}"

if ! command -v go >/dev/null 2>&1; then
	echo "go is required before VHS can be installed" >&2
	exit 1
fi

if ! command -v vhs >/dev/null 2>&1; then
	echo "Installing VHS ${vhs_version}"
	go install "github.com/charmbracelet/vhs@${vhs_version}"
fi

apt_packages=()
if ! command -v ffmpeg >/dev/null 2>&1; then
	apt_packages+=(ffmpeg)
fi
if ! command -v ttyd >/dev/null 2>&1; then
	apt_packages+=(ttyd)
fi
for pkg in libnss3 libnspr4 libxkbcommon0; do
	if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
		apt_packages+=("$pkg")
	fi
done

if [[ "${#apt_packages[@]}" -gt 0 ]]; then
	if ! command -v apt-get >/dev/null 2>&1; then
		echo "apt-get is required to install: ${apt_packages[*]}" >&2
		exit 1
	fi
	if [[ "$(id -u)" -eq 0 ]]; then
		apt-get update
		DEBIAN_FRONTEND=noninteractive apt-get install -y "${apt_packages[@]}"
	else
		sudo apt-get update
		sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "${apt_packages[@]}"
	fi
fi

scripts/check-visual-tools.sh
