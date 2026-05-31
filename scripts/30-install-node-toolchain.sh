#!/usr/bin/env bash
# Set up the Node.js / TypeScript development toolchain for a user.
# Installs: fnm (Node.js version manager), latest Node.js, corepack,
#           typescript, and tsx globally.
# Usage: sudo ./30-install-node-toolchain.sh <username>
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/_common.sh"

require_root

USER_TO_SETUP="${1:-}"

if [[ -z "$USER_TO_SETUP" ]]; then
  echo "Usage: sudo $0 <username>"
  exit 1
fi

if ! id "$USER_TO_SETUP" >/dev/null 2>&1; then
  echo "ERROR: User does not exist: $USER_TO_SETUP"
  exit 1
fi

# Run the toolchain install as the target user via a login shell.
# The heredoc is quoted ('EOF') to prevent variable expansion in root's shell,
# so the user's shell evaluates everything in its own environment.
sudo -iu "$USER_TO_SETUP" bash <<'EOF'
set -euo pipefail

# ---- fnm (Fast Node Manager) ----
# Uses fnm's official install script (curl | bash).
# Review https://fnm.vercel.app if untrusted.
export PATH="$HOME/.local/share/fnm:$PATH"

if [[ ! -d "$HOME/.local/share/fnm" ]]; then
  curl -fsSL https://fnm.vercel.app/install | bash
fi

export FNM_DIR="$HOME/.local/share/fnm"

if [[ ! -x "$FNM_DIR/fnm" ]]; then
  echo "ERROR: fnm binary not found at $FNM_DIR/fnm" >&2
  exit 1
fi

eval "$("$FNM_DIR/fnm" env --shell bash)"

fnm install --latest
fnm use latest
fnm default "$(fnm current)"

# ---- npm tooling ----
if ! command -v npm >/dev/null 2>&1; then
  echo "ERROR: npm not found after Node.js installation." >&2
  exit 1
fi

npm install -g corepack
corepack enable

npm install -g typescript tsx

echo "Node.js toolchain installed for $(whoami)."
EOF
