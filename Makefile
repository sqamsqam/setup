.PHONY: build test vet lint check clean run-cli

NAME    := setup
BIN_DIR := bin
SRC     := ./cmd/setup

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
