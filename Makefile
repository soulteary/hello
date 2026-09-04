SHELL          := /usr/bin/env bash
BINARY         := hello
PKG            := ./...
FUZZPKG        := ./internal/animation
GOFILES        := $(shell find . -type f -name '*.go' -not -path './.git/*')
VERSION        ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS        := -s -w -X main.version=$(VERSION)
DOCKER_IMAGE   ?= soulteary/hello
DOCKER_TAG     ?= dev
COVERAGE_MIN   ?= 100.0

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary into ./$(BINARY).
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/hello

.PHONY: install
install: ## Install the binary into $$GOBIN.
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/hello

.PHONY: run
run: ## Run with default animation.
	go run ./cmd/hello $(ARGS)

.PHONY: test
test: ## Run tests with race detector.
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests, produce coverage.out and enforce COVERAGE_MIN.
	go test -race -count=1 -covermode=atomic -coverprofile=coverage.out $(PKG)
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
		echo "total coverage: $${total}% (minimum: $(COVERAGE_MIN)%)"; \
		awk -v total="$${total}" -v minimum="$(COVERAGE_MIN)" 'BEGIN { exit !(total + 0 >= minimum + 0) }' || \
			{ echo "coverage is below $(COVERAGE_MIN)%"; exit 1; }

.PHONY: cover-html
cover-html: cover ## Open coverage report in the browser.
	go tool cover -html=coverage.out

.PHONY: vet
vet: ## go vet the codebase.
	go vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (required).
	@command -v golangci-lint >/dev/null 2>&1 || \
		{ echo "golangci-lint is required: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run

.PHONY: vuln
vuln: ## Scan reachable Go code with govulncheck (required).
	@command -v govulncheck >/dev/null 2>&1 || \
		{ echo "govulncheck is required: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	govulncheck $(PKG)

.PHONY: fuzz
fuzz: ## Fuzz the animation parser for 30s.
	go test -run '^$$' -fuzz=FuzzLoadFromBytes -fuzztime=30s $(FUZZPKG)

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

.PHONY: tidy-check
tidy-check: ## Fail if go.mod or go.sum needs tidying.
	go mod tidy -diff

.PHONY: check
check: tidy-check fmt-check vet lint vuln cover ## Run all required quality gates.

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
