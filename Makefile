# Tideland Go Actor - Makefile
#
# Copyright (C) 2021-2025 Frank Mueller / Tideland / Germany
#
# All rights reserved. Use of this source code is governed
# by the new BSD license.

# Variables
GO := go
GOLANGCI_LINT := golangci-lint
GOLANGCI_LINT_VERSION := v2.7.2
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# Default target
.DEFAULT_GOAL := all

# Phony targets
.PHONY: all help tidy lint build test bench fuzz coverage clean install-tools check-tools

## all: Run complete build process (tidy, lint, build, test)
all: tidy lint build test
	@echo "$(COLOR_GREEN)$(COLOR_BOLD)✓ All tasks completed successfully$(COLOR_RESET)"

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)Tideland Go Actor - Available Targets:$(COLOR_RESET)"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "$(COLOR_BLUE)Usage: make [target]$(COLOR_RESET)"
	@echo ""

## tidy: Update go.mod and go.sum files
tidy:
	@echo "$(COLOR_YELLOW)→ Tidying Go modules...$(COLOR_RESET)"
	@$(GO) mod tidy
	@$(GO) mod verify
	@echo "$(COLOR_GREEN)✓ Module dependencies updated$(COLOR_RESET)"

## lint: Run golangci-lint on source code
lint:
	@echo "$(COLOR_YELLOW)→ Running golangci-lint...$(COLOR_RESET)"
	@$(GOLANGCI_LINT) run --timeout=5m
	@echo "$(COLOR_GREEN)✓ Linting completed$(COLOR_RESET)"

## build: Build the package (verify compilation)
build:
	@echo "$(COLOR_YELLOW)→ Building package...$(COLOR_RESET)"
	@$(GO) build -v ./...
	@echo "$(COLOR_GREEN)✓ Build successful$(COLOR_RESET)"

## test: Run all tests
test:
	@echo "$(COLOR_YELLOW)→ Running tests...$(COLOR_RESET)"
	@$(GO) test -v -race ./...
	@echo "$(COLOR_GREEN)✓ Tests passed$(COLOR_RESET)"

## bench: Run benchmarks
bench:
	@echo "$(COLOR_YELLOW)→ Running benchmarks...$(COLOR_RESET)"
	@$(GO) test -bench=. -benchmem -run=^$$ ./...
	@echo "$(COLOR_GREEN)✓ Benchmarks completed$(COLOR_RESET)"

## fuzz: Run fuzz tests (requires Go 1.18+)
fuzz:
	@echo "$(COLOR_YELLOW)→ Running fuzz tests...$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Note: Fuzz tests run for 30 seconds each$(COLOR_RESET)"
	@$(GO) test -fuzz=FuzzAction -fuzztime=30s -run=^$$ ./...
	@echo "$(COLOR_GREEN)✓ Fuzz tests completed$(COLOR_RESET)"

## coverage: Generate test coverage report
coverage:
	@echo "$(COLOR_YELLOW)→ Generating coverage report...$(COLOR_RESET)"
	@$(GO) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Coverage: " $$3}'
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(COVERAGE_HTML)$(COLOR_RESET)"

## clean: Remove build artifacts and coverage files
clean:
	@echo "$(COLOR_YELLOW)→ Cleaning build artifacts...$(COLOR_RESET)"
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@$(GO) clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)✓ Clean completed$(COLOR_RESET)"

## install-tools: Install required development tools
install-tools:
	@echo "$(COLOR_YELLOW)→ Installing development tools...$(COLOR_RESET)"
	@which $(GOLANGCI_LINT) > /dev/null 2>&1 || \
		(echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION))
	@$(GOLANGCI_LINT) version | grep -q "has version $(shell echo $(GOLANGCI_LINT_VERSION) | sed 's/v//')" || \
		(echo "Updating golangci-lint to $(GOLANGCI_LINT_VERSION)..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION))
	@echo "$(COLOR_GREEN)✓ Tools installed$(COLOR_RESET)"

## check-tools: Check installed tool versions and compatibility
check-tools:
	@echo "$(COLOR_YELLOW)→ Checking tool versions...$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Go version:$(COLOR_RESET)"
	@$(GO) version
	@echo "$(COLOR_BLUE)golangci-lint version:$(COLOR_RESET)"
	@$(GOLANGCI_LINT) version || echo "$(COLOR_YELLOW)golangci-lint not found - run 'make install-tools'$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)✓ Tool version check completed$(COLOR_RESET)"

## ci: Run CI pipeline (used by GitHub Actions)
ci: tidy lint build test
	@echo "$(COLOR_GREEN)$(COLOR_BOLD)✓ CI pipeline completed$(COLOR_RESET)"
