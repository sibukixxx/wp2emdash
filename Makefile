SHELL := /bin/bash

BIN_DIR := bin
BIN := $(BIN_DIR)/wp2emdash

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/rokubunnoni-inc/wp2emdash/internal/cli.Version=$(VERSION)

.PHONY: help build test vet lint run clean install dist

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?##"}{printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary into bin/
	@mkdir -p $(BIN_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/wp2emdash
	@echo "built: $(BIN) ($(VERSION))"

test: ## Run unit tests
	go test ./...

vet: ## Run go vet
	go vet ./...

lint: vet ## Alias for vet (add staticcheck/golangci-lint here when ready)

run: build ## Build and run with --help
	$(BIN) --help

install: ## Install into $$(go env GOBIN) or $$GOPATH/bin
	go install -ldflags="$(LDFLAGS)" ./cmd/wp2emdash

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) wp2emdash-output coverage.out

# ---------- Cross-compile (release) ----------

dist: ## Cross-compile static binaries for darwin/linux x amd64/arm64
	@mkdir -p dist
	@for os in darwin linux; do \
	  for arch in amd64 arm64; do \
	    out=dist/wp2emdash-$$os-$$arch; \
	    echo "  -> $$out"; \
	    GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	      go build -ldflags="$(LDFLAGS)" -o $$out ./cmd/wp2emdash; \
	  done; \
	done
	@ls -lh dist/
