# fiber-helpers — developer tasks.
# Most contributors use mise (see mise.toml); these targets wrap the common
# go commands so `make <thing>` works the same locally and in CI.

.DEFAULT_GOAL := help

GO ?= go

.PHONY: help
help: ## Show this help.
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile all packages.
	$(GO) build ./...

.PHONY: test
test: ## Run unit tests with the race detector.
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests and write a coverage profile.
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -func=coverage.txt | tail -1

.PHONY: e2e
e2e: ## Run the create-fiber-app end-to-end generator tests.
	RUN_E2E=1 $(GO) test ./cmd/create-fiber-app/

.PHONY: lint
lint: ## Run golangci-lint.
	golangci-lint run

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: vuln
vuln: ## Scan dependencies for known vulnerabilities.
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum.
	$(GO) mod tidy

.PHONY: fmt
fmt: ## Format the codebase.
	$(GO) fmt ./...

.PHONY: check
check: vet lint test ## Run vet, lint, and tests (pre-push gate).
