# Development

## Local Workflow

```bash
make prep     # build bin/setup-linux-amd64
make test     # go test ./internal/...
make vet      # go vet ./internal/... ./cmd/...
make lint     # golangci-lint run ./...
make taste    # vet, test, lint
make plate    # regenerate and validate visual assets
make bake     # taste, plate, then local GoReleaser snapshot
make clean    # remove bin/
```

Run the CLI from source with:

```bash
make run-cli ARGS="version"
make run-cli ARGS="--dry-run base --timezone UTC"
```

Requirements:

- Go 1.26+
- `golangci-lint` for `make lint`
- GoReleaser for `make bake`, or network access so Make can run it through `go run`
- `sudo` and `apt-get` on Ubuntu-like systems when installing visual runtime dependencies

## Release Flow

`make bake` is the local release-quality check. It runs Go checks, regenerates visual assets, validates them, and creates a GoReleaser snapshot without publishing.

Published releases are tag-driven. Pushing a semver tag matching `v[0-9]+.[0-9]+.[0-9]+` runs the release workflow, builds one Linux amd64 binary, and uploads:

```text
setup-linux-amd64
checksums.txt
```

## Changelog

`CHANGELOG.md` follows Keep a Changelog.

Before tagging a release:

1. Move the `(Unreleased)` content under a new version heading.
2. Ensure user-visible changes are documented under Added, Changed, Fixed, or Security.
3. Open a PR for the changelog update, then tag after merge.

Every pull request with user-visible changes should add an entry under `[Unreleased]`.

## CI

GitHub Actions runs vet, tests, lint, and a separate visual job. The visual job runs `make plate` and uploads `docs/assets` so reviewers can inspect screenshots and GIFs.
