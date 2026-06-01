.PHONY: prep test vet lint taste clean run-cli install-visual-tools visual-tools-check vhs-validate visual-screenshots visual-gifs visual-golden visual-test plate visual-clean bake

NAME    := setup
BIN_DIR := bin
SRC     := ./cmd/setup
VHS_VERSION ?= v0.11.0
GORELEASER_VERSION ?= latest
GORELEASER ?= $(shell command -v goreleaser 2>/dev/null || printf '%s' 'go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)')
export PATH := $(HOME)/go/bin:$(PATH)

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.buildDate=$(DATE)'

prep:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(NAME)-linux-amd64 $(SRC)

test:
	go test ./internal/...

vet:
	go vet ./internal/... ./cmd/...

lint:
	golangci-lint run ./...

taste: vet test lint

clean:
	rm -rf $(BIN_DIR)

run-cli:
	go run $(SRC) $(ARGS)

install-visual-tools:
	VHS_VERSION=$(VHS_VERSION) scripts/install-visual-tools.sh

visual-tools-check:
	scripts/check-visual-tools.sh

vhs-validate: visual-tools-check
	vhs validate "demo/**/*.tape" "demo/*.tape"

visual-screenshots: prep visual-tools-check
	mkdir -p docs/assets/screenshots
	set -e; for tape in demo/screenshots/*.tape; do vhs "$$tape"; done

visual-gifs: prep visual-tools-check
	mkdir -p docs/assets/gifs
	set -e; for tape in demo/navigation.tape demo/success.tape demo/error.tape; do vhs "$$tape"; done

visual-golden: prep visual-tools-check
	mkdir -p docs/assets
	vhs demo/golden.tape

visual-test: vhs-validate
	scripts/validate-visual-assets.sh

plate: install-visual-tools visual-clean prep vhs-validate visual-screenshots visual-gifs visual-golden visual-test

bake: taste plate
	$(GORELEASER) release --snapshot --clean

visual-clean:
	rm -rf docs/assets
