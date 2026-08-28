.PHONY: generate generated-check fmt lint check-web-quality test test-race test-cli web-install web-test web-build e2e build install ci clean

GO ?= go
PNPM ?= corepack pnpm@11.24.0
INSTALL_DIR ?= $(HOME)/.local/bin

generate:
	$(GO) tool sqlc generate
	$(GO) tool buf lint
	$(GO) tool buf generate

generated-check: generate
	git diff --exit-code -- internal/db gen web/src/gen

fmt:
	gofmt -w $$(find cmd internal gen -name '*.go' -type f)
	$(PNPM) --dir web format

lint:
	$(GO) vet ./...
	golangci-lint run ./...
	$(PNPM) --dir web lint

check-web-quality:
	python3 scripts/check_web_file_lines.py
	$(PNPM) --dir web check:duplicates

test:
	$(GO) test ./...
	$(PNPM) --dir web test

test-race:
	$(GO) test -race ./...

test-cli:
	$(GO) test ./internal/cli -run TestBlackBox -count=1

web-install:
	$(PNPM) install --frozen-lockfile

web-test:
	$(PNPM) --dir web test

web-build:
	$(PNPM) --dir web build

e2e: web-build
	$(PNPM) --dir web e2e

build: web-build
	mkdir -p bin
	$(GO) build -trimpath -o bin/prx ./cmd/prx

install: build
	install -d "$(INSTALL_DIR)"
	install -m 0755 bin/prx "$(INSTALL_DIR)/prx"

ci: generated-check lint check-web-quality test test-race build e2e

clean:
	$(GO) clean -testcache
	$(PNPM) --dir web clean
