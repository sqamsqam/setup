#!/usr/bin/env bash
set -euo pipefail

missing=0
for tool in go vhs ffmpeg ttyd; do
	if ! command -v "$tool" >/dev/null 2>&1; then
		echo "missing required visual tool: $tool" >&2
		missing=1
	fi
done

if [[ "$missing" -ne 0 ]]; then
	echo "run: make install-visual-tools" >&2
	exit 1
fi
