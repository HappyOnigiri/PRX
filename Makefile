.PHONY: generate generated-check fmt lint test test-race test-cli web-install web-test web-build e2e build ci clean

GO ?= go
PNPM ?= corepack pnpm@11.24.0

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

ci: generated-check lint test test-race build e2e

clean:
	$(GO) clean -testcache
	$(PNPM) --dir web clean
