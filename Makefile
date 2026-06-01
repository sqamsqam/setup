.PHONY: build test vet lint check clean run-cli install-visual-tools visual-tools-check vhs-validate screenshots demo demo-gif golden-demo bake visual-test review-ui visual-clean

NAME    := setup
BIN_DIR := bin
SRC     := ./cmd/setup
VHS_VERSION ?= v0.11.0
export PATH := $(HOME)/go/bin:$(PATH)

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.buildDate=$(DATE)'

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(NAME)-linux-amd64 $(SRC)

test:
	go test ./internal/...

vet:
	go vet ./internal/... ./cmd/...

lint:
	golangci-lint run ./...

check: vet test lint

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

screenshots: build visual-tools-check
	mkdir -p docs/assets/screenshots
	set -e; for tape in demo/screenshots/*.tape; do vhs "$$tape"; done

demo: demo-gif

demo-gif: build visual-tools-check
	mkdir -p docs/assets/gifs
	set -e; for tape in demo/navigation.tape demo/success.tape demo/error.tape; do vhs "$$tape"; done

golden-demo: build visual-tools-check
	mkdir -p docs/assets
	vhs demo/golden.tape

bake: golden-demo

visual-test: vhs-validate
	scripts/validate-visual-assets.sh

review-ui: install-visual-tools visual-clean build vhs-validate screenshots demo-gif golden-demo visual-test

visual-clean:
	rm -rf docs/assets
