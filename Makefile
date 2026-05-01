SHELL := /bin/bash

BIN_DIR := bin
BIN := $(BIN_DIR)/wp2emdash
GOCACHE ?= /tmp/wp2emdash-go-build
GOENV := env GOCACHE=$(GOCACHE)
GOLANGCI_LINT_CACHE ?= /tmp/wp2emdash-golangci-lint
TEST_RUN ?=
E2E_RUN ?=
GO_FILES := $(shell find . -type f -name '*.go' -not -path './bin/*' | sort)
GOLANGCI_LINT ?= golangci-lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/sibukixxx/wp2emdash/internal/cli.Version=$(VERSION)

.PHONY: help build test test-e2e test-all vet lint golangci golangci-fix fmt fix run clean install dist

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?##"}{printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary into bin/
	@mkdir -p $(BIN_DIR)
	$(GOENV) go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/wp2emdash
	@echo "built: $(BIN) ($(VERSION))"

test: ## Run unit tests (TEST_RUN=TestName to filter)
	$(GOENV) go test -race -count=1 $(if $(TEST_RUN),-run $(TEST_RUN)) ./...

test-e2e: ## Run CLI end-to-end tests (E2E_RUN=TestName to filter)
	WP2EMDASH_E2E_TEST_ENABLED=true $(GOENV) go test -count=1 -v $(if $(E2E_RUN),-run $(E2E_RUN)) ./test/e2e/tests/...

test-all: test test-e2e ## Run unit tests and E2E tests

vet: ## Run go vet
	$(GOENV) go vet ./...

golangci: ## Run golangci-lint
	$(GOENV) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run -c .golangci.yml ./...

golangci-fix: ## Run golangci-lint with autofix
	$(GOENV) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run -c .golangci.yml --fix ./...

lint: vet golangci ## Run static analysis

fmt: ## Format Go files with gofmt
	@if [ -n "$(GO_FILES)" ]; then gofmt -w $(GO_FILES); fi

fix: fmt golangci-fix ## Format code and apply autofixable lint rules

run: build ## Build and run with --help
	$(BIN) --help

install: ## Install into $$(go env GOBIN) or $$GOPATH/bin
	$(GOENV) go install -ldflags="$(LDFLAGS)" ./cmd/wp2emdash

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
	      $(GOENV) go build -ldflags="$(LDFLAGS)" -o $$out ./cmd/wp2emdash; \
	  done; \
	done
	@ls -lh dist/
