#!/usr/bin/env bash
set -euo pipefail

assets=(
	"docs/assets/golden-demo.gif"
	"docs/assets/gifs/navigation.gif"
	"docs/assets/gifs/success.gif"
	"docs/assets/gifs/error.gif"
	"docs/assets/screenshots/home.png"
	"docs/assets/screenshots/empty-state.png"
	"docs/assets/screenshots/compact.png"
	"docs/assets/screenshots/large.png"
	"docs/assets/screenshots/success.png"
	"docs/assets/screenshots/error.png"
)

for asset in "${assets[@]}"; do
	if [[ ! -s "$asset" ]]; then
		echo "missing or empty visual asset: $asset" >&2
		exit 1
	fi
done

echo "Validated ${#assets[@]} visual assets."
