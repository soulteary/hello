SHELL          := /usr/bin/env bash
BINARY         := hello
PKG            := ./...
GOFILES        := $(shell find . -type f -name '*.go' -not -path './.git/*')
VERSION        ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS        := -s -w -X main.version=$(VERSION)
DOCKER_IMAGE   ?= soulteary/hello
DOCKER_TAG     ?= dev

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary into ./$(BINARY).
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

.PHONY: install
install: ## Install the binary into $$GOBIN.
	go install -trimpath -ldflags "$(LDFLAGS)" .

.PHONY: run
run: ## Run with default animation.
	go run . $(ARGS)

.PHONY: test
test: ## Run tests with race detector.
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests and produce coverage.out.
	go test -race -count=1 -covermode=atomic -coverprofile=coverage.out $(PKG)
	@go tool cover -func=coverage.out | tail -1

.PHONY: cover-html
cover-html: cover ## Open coverage report in the browser.
	go tool cover -html=coverage.out

.PHONY: vet
vet: ## go vet the codebase.
	go vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (skipped with a warning if not installed).
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; skipping. Install: https://golangci-lint.run/welcome/install/"; \
	fi

.PHONY: fuzz
fuzz: ## Fuzz the animation parser for 30s.
	go test -run '^$$' -fuzz=FuzzLoadFromBytes -fuzztime=30s $(PKG)

.PHONY: bench
bench: ## Run benchmarks.
	go test -run '^$$' -bench=. -benchmem $(PKG)

.PHONY: fmt
fmt: ## Format the codebase with gofmt.
	gofmt -w $(GOFILES)

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-clean.
	@unformatted=$$(gofmt -l $(GOFILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: tidy
tidy: ## Run go mod tidy.
	go mod tidy

.PHONY: check
check: fmt-check vet lint test ## Run fmt-check, vet, lint and tests (CI-equivalent).

.PHONY: docker
docker: ## Build a local Docker image for the host platform.
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg REVISION=$$(git rev-parse HEAD 2>/dev/null || echo unknown) \
		--build-arg CREATED=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run: docker ## Build and run the local Docker image.
	docker run --rm $(DOCKER_IMAGE):$(DOCKER_TAG) $(ARGS)

.PHONY: clean
clean: ## Remove build artifacts.
	rm -f $(BINARY) $(BINARY).exe coverage.out
