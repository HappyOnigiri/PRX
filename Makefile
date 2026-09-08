GO ?= go
PNPM ?= corepack pnpm
INSTALL_DIR ?= $(HOME)/.local/bin
RELEASE_DIR ?= artifacts/release
VERSION := $(shell node -p "require('./package.json').version")
GO_COVERAGE_MIN ?= 68.8
GO_COVERAGE_PACKAGES := ./internal/domain ./internal/github ./internal/rpc ./internal/store
GO_COVERAGE_ZERO_PACKAGES := ./internal/app $(GO_COVERAGE_PACKAGES)
GOLANGCI_LINT_VERSION := $(shell awk '$$1 == "golangci-lint" { print $$2 }' .tool-versions)
GOLANGCI_LINT := bin/golangci-lint
# `make ci` は CPU 数だけジョブを並列に走らせる。`make ci CI_JOBS=4` で上書きできる。
CI_JOBS ?= $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
# GNU make 4 は並列ジョブの出力をまとめて表示するが、GNU make 3.81 (macOS) は混ざる。
CI_MAKEFLAGS := -j$(CI_JOBS) --keep-going $(if $(filter output-sync,$(.FEATURES)),--output-sync=target)

.PHONY: generate generated-check mod-tidy-check fmt lint go-lint go-deadcode markdown-lint web-lint check-web-quality \
    go-comment-lint test go-test web-test go-coverage-check go-coverage-zero-check test-race test-race-coverage test-cli \
    web-install web-build dev demo e2e build version-check install release release-check ci ci-checks clean \
    $(GOLANGCI_LINT)

generate: web-install
	$(GO) tool sqlc generate
	$(GO) tool buf format -w proto
	$(GO) tool buf lint
	$(GO) tool buf generate
	$(GO) run ./cmd/prxdoc docs/cli

# 一時ディレクトリに再生成して追跡中のファイルと比較する。作業ツリーを書き換えないので、
# 他のチェックと並行して実行できる。
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

lint: go-lint go-deadcode go-comment-lint markdown-lint web-lint

# golangci-lint が govet を実行するので、別に `go vet` を走らせる必要はない。
go-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

go-deadcode:
	@output="$$($(GO) tool deadcode -test ./...)"; \
	if [ -n "$$output" ]; then printf '%s\n' "$$output"; echo "deadcode: unreachable functions found"; exit 1; fi

# TypeScript と JavaScript に対する同じチェックは ESLint ルールなので web-lint で走る。
go-comment-lint:
	$(GO) run ./tools/checkcomments

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

# race 有効の 1 回の実行で、`make ci` は go-test・test-race・go-coverage-check の結果を得る。
test-race-coverage:
	@profile="$$(mktemp)" || exit $$?; \
	trap 'rm -f "$$profile"' EXIT; \
	$(GO) test -race -coverprofile="$$profile" ./... || exit $$?; \
	GO="$(GO)" scripts/check-go-coverage.sh "$$profile" "$(GO_COVERAGE_MIN)" $(GO_COVERAGE_PACKAGES)

test-cli:
	$(GO) test ./internal/cli -run TestBlackBox -count=1

web-install:
	$(PNPM) install --frozen-lockfile

# 型チェックは web-lint の担当で、本番バンドルの生成には Vite だけあればよい。
web-build: web-install
	$(PNPM) --dir web build

# 開発用ミドルウェアは web-build が出力したライセンスレポートを配信する。
dev: web-build
	$(PNPM) --dir web dev:full

# demo データを見ながら開発する。demo の一時環境はサーバプロセスごとなので、Go の変更で再ビルドが
# 走ると demo データは初期状態に戻る。WebUI の変更は再起動なしで反映される。
demo: web-build
	$(PNPM) --dir web dev:demo

# Playwright への追加フラグ。例えば `make e2e E2E_FLAGS=--shard=1/3` で 1 シャードだけ走る。
E2E_FLAGS ?=

# scripts/run-e2e-server.sh も bin/prx を書くので、e2e は build と並行せず終わるのを待つ。
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

# 配布物は明示したタグでだけ作り、開発用の build / install が付ける -dev をそのまま残す。
# 配布バイナリだけで WebUI も使えるよう、Go のビルドより先に web-build を済ませる。
release: web-build
	GO="$(GO)" RELEASE_VERSION="$(RELEASE_VERSION)" RELEASE_DIR="$(RELEASE_DIR)" scripts/build-release.sh

# 対象 OS/arch とチェックサム、インストーラーへの版番号の差し込みを検査する。
# Go 1.27 の `go version -m` は -ldflags を表示しないので、macOS arm64 でだけ実行して版番号を確かめる。
release-check: web-build
	@directory="$$(mktemp -d)" || exit $$?; \
	trap 'rm -rf "$$directory"' EXIT; \
	GO="$(GO)" RELEASE_VERSION=v0.0.0 RELEASE_DIR="$$directory" scripts/build-release.sh || exit $$?; \
	$(GO) version -m "$$directory/prx-darwin-arm64" > "$$directory/build-info" || exit $$?; \
	grep -Fq 'CGO_ENABLED=0' "$$directory/build-info" || exit $$?; \
	grep -Fq 'GOOS=darwin' "$$directory/build-info" || exit $$?; \
	grep -Fq 'GOARCH=arm64' "$$directory/build-info" || exit $$?; \
	grep -Fq "release_version='v0.0.0'" "$$directory/install.sh" || exit $$?; \
	(cd "$$directory" && shasum -a 256 -c checksums.txt) || exit $$?; \
	if [ "$$(uname -sm)" = 'Darwin arm64' ]; then \
	  test "$$("$$directory/prx-darwin-arm64" --version)" = 'prx version 0.0.0'; \
	fi

ci:
	$(MAKE) $(CI_MAKEFLAGS) ci-checks

# どのチェックも読み取り専用か、自分の出力先 (coverage/、test-results/、bin/prx、
# internal/webui/dist) にしか書かないので、並行実行しても安全。書き込み側は依存関係で直列化する。
# 最長の連鎖 (web-build -> build -> e2e) を先頭に置き、make が他より先に着手するようにしている。
ci-checks: e2e version-check build release-check lint test-race-coverage go-coverage-zero-check web-test \
    check-web-quality generated-check mod-tidy-check

clean:
	$(GO) clean -testcache
	$(PNPM) --dir web clean
