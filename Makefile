.PHONY: generate generated-check mod-tidy-check fmt lint check-web-quality test go-coverage-check test-race test-cli web-install web-test web-build dev e2e build install ci clean

GO ?= go
PNPM ?= corepack pnpm@11.24.0
INSTALL_DIR ?= $(HOME)/.local/bin
GO_COVERAGE_MIN ?= 68.8
GO_COVERAGE_PACKAGES := ./internal/domain ./internal/github ./internal/rpc ./internal/store

generate: web-install
	$(GO) tool sqlc generate
	$(GO) tool buf format -w proto
	$(GO) tool buf lint
	$(GO) tool buf generate
	$(GO) run ./cmd/prxdoc docs/cli

generated-check: generate
	git diff --exit-code -- internal/db gen web/src/gen docs/cli

mod-tidy-check:
	$(GO) mod tidy -diff

fmt: web-install
	golangci-lint fmt ./...
	$(PNPM) --dir web format

lint: web-install
	$(GO) tool buf format -d --exit-code proto
	$(GO) vet ./...
	golangci-lint run ./...
	@output="$$($(GO) tool deadcode -test ./...)"; \
	if [ -n "$$output" ]; then printf '%s\n' "$$output"; echo "deadcode: unreachable functions found"; exit 1; fi
	$(PNPM) --dir web lint

check-web-quality: web-install
	python3 scripts/check_web_file_lines.py
	$(PNPM) --dir web check:duplicates

test: web-install
	$(GO) test ./...
	$(PNPM) --dir web test

go-coverage-check:
	@profile="$$(mktemp)" || exit $$?; \
	trap 'rm -f "$$profile"' EXIT; \
	$(GO) test -coverprofile="$$profile" $(GO_COVERAGE_PACKAGES) || exit $$?; \
	coverage="$$( $(GO) tool cover -func="$$profile" | awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }' )" || exit $$?; \
	printf 'Go coverage: %s%% (minimum: %s%%)\n' "$$coverage" "$(GO_COVERAGE_MIN)"; \
	awk -v actual="$$coverage" -v minimum="$(GO_COVERAGE_MIN)" 'BEGIN { \
		if (actual !~ /^[0-9]+([.][0-9]+)?$$/ || minimum !~ /^[0-9]+([.][0-9]+)?$$/) exit 2; \
		if (actual + 0 < minimum + 0) exit 1; \
	}'

test-race:
	$(GO) test -race ./...

test-cli:
	$(GO) test ./internal/cli -run TestBlackBox -count=1

web-install:
	$(PNPM) install --frozen-lockfile

web-test: web-install
	$(PNPM) --dir web test

web-build: web-install
	$(PNPM) --dir web build

dev: web-install
	$(PNPM) --dir web dev:full

e2e: web-build
	$(PNPM) --dir web e2e

build: web-build
	mkdir -p bin
	$(GO) build -trimpath -o bin/prx ./cmd/prx

install: build
	install -d "$(INSTALL_DIR)"
	install -m 0755 bin/prx "$(INSTALL_DIR)/prx"

ci: generated-check mod-tidy-check lint check-web-quality test go-coverage-check test-race build e2e

clean:
	$(GO) clean -testcache
	$(PNPM) --dir web clean
