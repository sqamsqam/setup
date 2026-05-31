#!/usr/bin/env bash
# Shared utilities sourced by setup scripts. Not meant to be run directly.

export DEBIAN_FRONTEND=noninteractive

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    echo "ERROR: This script must be run as root."
    exit 1
  fi
}
