GO ?= go
PNPM ?= corepack pnpm
INSTALL_DIR ?= $(HOME)/.local/bin
VERSION := $(shell node -p "require('./package.json').version")
GO_COVERAGE_MIN ?= 68.8
GO_COVERAGE_PACKAGES := ./internal/domain ./internal/github ./internal/rpc ./internal/store
GO_COVERAGE_ZERO_PACKAGES := ./internal/app $(GO_COVERAGE_PACKAGES)
GOLANGCI_LINT_VERSION := $(shell awk '$$1 == "golangci-lint" { print $$2 }' .tool-versions)
GOLANGCI_LINT := bin/golangci-lint
# `make ci` runs its checks in parallel with one job per CPU. Override with `make ci CI_JOBS=4`.
CI_JOBS ?= $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
# GNU make 4 prints each parallel job's output as one block; GNU make 3.81 (macOS) interleaves it.
CI_MAKEFLAGS := -j$(CI_JOBS) --keep-going $(if $(filter output-sync,$(.FEATURES)),--output-sync=target)

.PHONY: generate generated-check mod-tidy-check fmt lint go-lint go-deadcode markdown-lint web-lint check-web-quality \
    test go-test web-test go-coverage-check go-coverage-zero-check test-race test-race-coverage test-cli \
    web-install web-build dev e2e build version-check install ci ci-checks clean $(GOLANGCI_LINT)

generate: web-install
	$(GO) tool sqlc generate
	$(GO) tool buf format -w proto
	$(GO) tool buf lint
	$(GO) tool buf generate
	$(GO) run ./cmd/prxdoc docs/cli

# Regenerates into a temporary directory and compares it with the tracked files, so this
# never rewrites the working tree and can run alongside the other checks.
generated-check: web-install
	$(GO) tool buf format -d --exit-code proto
	$(GO) tool buf lint
	$(GO) tool sqlc diff
	@out="$$(mktemp -d)" || exit $$?; \
	trap 'rm -rf "$$out"' EXIT; \
	$(GO) tool buf generate -o "$$out" || exit $$?; \
	diff -ru gen "$$out/gen" || exit $$?; \
	diff -ru web/src/gen "$$out/web/src/gen" || exit $$?; \
	$(GO) run ./cmd/prxdoc "$$out/docs/cli" || exit $$?; \
	diff -ru docs/cli "$$out/docs/cli"

mod-tidy-check:
	$(GO) mod tidy -diff

fmt: web-install $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) fmt ./...
	$(PNPM) --dir web format

lint: go-lint go-deadcode markdown-lint web-lint

# golangci-lint runs govet, so a separate `go vet` is redundant.
go-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

go-deadcode:
	@output="$$($(GO) tool deadcode -test ./...)"; \
	if [ -n "$$output" ]; then printf '%s\n' "$$output"; echo "deadcode: unreachable functions found"; exit 1; fi

markdown-lint:
	$(GO) run ./tools/checkmarkdownlines

web-lint: web-install
	$(PNPM) --dir web lint

check-web-quality: web-install
	$(GO) run ./tools/checkweblines
	$(PNPM) --dir web check:duplicates

$(GOLANGCI_LINT):
	@if [ ! -x "$@" ] || ! "$@" version 2>/dev/null | grep -Fq "$(GOLANGCI_LINT_VERSION)"; then \
		mkdir -p "$(dir $@)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b bin v$(GOLANGCI_LINT_VERSION); \
	fi

test: go-test web-test

go-test:
	$(GO) test ./...

web-test: web-install
	$(PNPM) --dir web test

go-coverage-check:
	@profile="$$(mktemp)" || exit $$?; \
	trap 'rm -f "$$profile"' EXIT; \
	$(GO) test -coverprofile="$$profile" $(GO_COVERAGE_PACKAGES) || exit $$?; \
	GO="$(GO)" scripts/check-go-coverage.sh "$$profile" "$(GO_COVERAGE_MIN)" $(GO_COVERAGE_PACKAGES)

go-coverage-zero-check:
	$(GO) run ./tools/checkgozerocoverage $(GO_COVERAGE_ZERO_PACKAGES)

test-race:
	$(GO) test -race ./...

# One race-enabled run gives `make ci` the results of go-test, test-race, and go-coverage-check.
test-race-coverage:
	@profile="$$(mktemp)" || exit $$?; \
	trap 'rm -f "$$profile"' EXIT; \
	$(GO) test -race -coverprofile="$$profile" ./... || exit $$?; \
	GO="$(GO)" scripts/check-go-coverage.sh "$$profile" "$(GO_COVERAGE_MIN)" $(GO_COVERAGE_PACKAGES)

test-cli:
	$(GO) test ./internal/cli -run TestBlackBox -count=1

web-install:
	$(PNPM) install --frozen-lockfile

# Type checking belongs to web-lint; the production bundle only needs Vite.
web-build: web-install
	$(PNPM) --dir web build

dev: web-install
	$(PNPM) --dir web dev:full

# Extra Playwright flags, e.g. `make e2e E2E_FLAGS=--shard=1/3` to run one shard of the suite.
E2E_FLAGS ?=

# scripts/run-e2e-server.sh also writes bin/prx, so e2e waits for build instead of running beside it.
e2e: build
	$(PNPM) --dir web e2e $(E2E_FLAGS)

build: web-build
	mkdir -p bin
	$(GO) build -trimpath -o bin/prx ./cmd/prx

version-check: build
	@test "$$($(CURDIR)/bin/prx --version)" = "prx version $(VERSION)-dev"

install: build
	install -d "$(INSTALL_DIR)"
	install -m 0755 bin/prx "$(INSTALL_DIR)/prx"

ci:
	$(MAKE) $(CI_MAKEFLAGS) ci-checks

# Every check is read-only or writes only to its own output (coverage/, test-results/, bin/prx,
# internal/webui/dist), so they are safe to run concurrently. Dependencies serialize the writers.
# The longest chain (web-build -> build -> e2e) comes first so make schedules it before the rest.
ci-checks: e2e version-check build lint test-race-coverage go-coverage-zero-check web-test check-web-quality \
    generated-check mod-tidy-check

clean:
	$(GO) clean -testcache
	$(PNPM) --dir web clean
