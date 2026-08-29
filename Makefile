.PHONY: generate generated-check mod-tidy-check fmt lint check-web-quality test go-coverage-check go-coverage-update test-race test-cli web-install web-test web-build dev e2e build install ci clean

GO ?= go
PNPM ?= corepack pnpm@11.24.0
INSTALL_DIR ?= $(HOME)/.local/bin

generate: web-install
	$(GO) tool sqlc generate
	$(GO) tool buf lint
	$(GO) tool buf generate

generated-check: generate
	git diff --exit-code -- internal/db gen web/src/gen

mod-tidy-check:
	$(GO) mod tidy -diff

fmt: web-install
	gofmt -w $$(find cmd internal gen -name '*.go' -type f)
	$(PNPM) --dir web format

lint: web-install
	$(GO) vet ./...
	golangci-lint run ./...
	$(PNPM) --dir web lint

check-web-quality: web-install
	python3 scripts/check_web_file_lines.py
	$(PNPM) --dir web check:duplicates

test: web-install
	$(GO) test ./...
	$(PNPM) --dir web test

go-coverage-check:
	$(GO) run ./tools/covercheck

go-coverage-update:
	$(GO) run ./tools/covercheck -update

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
