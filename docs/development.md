# Development

## Verification

```sh
make generate               # sqlc, Buf format/lint, Go/TypeScript protobuf, and CLI reference generation
make generated-check        # regeneration into a temporary directory must match the tracked files
make mod-tidy-check         # go.mod and go.sum must be tidy
make lint                   # auto-installed golangci-lint (govet / gofumpt / gci / golines), deadcode, Prettier, ESLint, strict TypeScript, knip, and stylelint
make check-web-quality      # WebUI function size limits and duplicate detection
make test                   # Go, Vitest, and component coverage
make go-coverage-check      # handwritten Go packages must stay at or above the coverage baseline
make go-coverage-zero-check # every function in the core Go packages must be executed by tests
make test-race              # Go race detector
make test-race-coverage     # one race-enabled Go test run that also enforces the coverage baseline
make test-cli               # CLI black-box process tests only
make e2e                    # real Go server, SQLite, ConnectRPC, Chromium
make ci                     # install web dependencies, run all required checks in parallel, and build production assets
```

`make ci` runs the checks concurrently with one job per CPU (`CI_JOBS` overrides the count) and keeps going after a failure so one run reports every problem. Each check reads the tree or writes only its own output, and `make ci` uses `test-race-coverage` in place of the three separate Go test runs. GNU make 4 groups each job's output; the GNU make 3.81 shipped with macOS interleaves it.

## Versions and releases

The root `package.json` owns the PRX version. Every current build path, including `make build`, `make install`, `go build`, `go run`, Air, and the Vite development setup, shows `0.1.0-dev` in both `prx --version` and the WebUI. This preserves the base release in diagnostic output without identifying locally built code as an official release. A future distribution pipeline will be the only path that stamps the stable version into release artifacts.

Run the `Release` workflow manually with the next stable `X.Y.Z` version. It creates a `release/vX.Y.Z` branch, updates `package.json`, and opens a release pull request with the generated changelog. Merging that pull request creates the `vX.Y.Z` tag and GitHub Release, then removes the release branch. The workflow accepts stable SemVer versions only; prerelease identifiers such as `-rc.1` are not supported.

The repository must provide a `GH_TOKEN` Actions secret with permission to write contents and pull requests. The release reminder uses the workflow `GITHUB_TOKEN` to report the pull requests merged since the latest release.

## Coverage contracts

`GO_COVERAGE_MIN` records the current coverage baseline for the handwritten Go packages. Raise it when their coverage improves so later changes cannot reduce it.

`make go-coverage-zero-check` runs the Go tests with cross-package coverage for `internal/app`, `internal/domain`, `internal/github`, `internal/rpc`, and `internal/store`, then fails if any target function has exactly 0.0% coverage. It excludes the WebUI, CLI black-box process tests, and generated code because those paths are verified by their own checks and are not part of the core package function contract.

`pnpm --dir web test` includes `web/src/**/*.{ts,tsx}` in the Vitest function-coverage check and fails if any included function has an execution count of exactly zero. Generated `src/gen/**` files and declaration files are excluded; coverage-rate thresholds remain separate from this zero-function check.

## End-to-end tests

Playwright covers browser CRUD, language selection persistence, cycle rejection, pull-request and Markdown attachment, deterministic sync, dependency removal, persistence after reload, console/network failures, 320-pixel reflow, and 8/50/100-node layouts. Graph screenshots are written to `test-results/screenshots/` and uploaded by GitHub Actions. `E2E_FLAGS` passes extra flags to Playwright, for example `make e2e E2E_FLAGS=--shard=1/3`.

GitHub Actions runs the same checks as `make ci`, but as one job per Makefile target plus three Playwright shards, so a run takes as long as its slowest check. Each Go job restores its own module and build cache from the newest cache saved on `main`; pull request runs only save a cache when nothing could be restored.

## GitHub fixtures

`--github-fixture` supplies a deterministic provider for the command, so tests and demos run without network access. It takes precedence over the YAML resolver and the SQLite authentication cache. Without a fixture, both `prx serve` and `prx sync` use the configured live resolver; credentials are read lazily per host and repository, so a server can start with no available credential and report stale synchronization failures later.

`prx seed --github-fixture demo --features 100 --tasks 50` creates deterministic performance data. Instead of `demo`, a fixture JSON file can map canonical PR URLs to GitHub states:

```json
{
  "https://github.com/acme/api/pull/42": {
    "state": "open",
    "review_state": "approved",
    "mergeability": "mergeable",
    "author": "octocat",
    "assignees": ["mona"]
  }
}
```

`state` must be `open`, `closed`, `merged`, or `unknown`; `review_state` must be `none`, `required`, `approved`, `changes_requested`, or `unknown`; `mergeability` must be `mergeable`, `conflicting`, or `unknown`. A fixture with any other value is rejected when the file is read. An entry may instead carry `"error"` to simulate a fetch failure.

Task fixtures and integration tests use `auto` as the stored task status. A plan is supplied through the implementation-plan CLI/RPC rather than embedded in a snapshot; tests that need the `designed` state should create a plan and then request a fresh snapshot. Synchronization tests should distinguish a first-time `unknown` PR from a stale PR with a previously known open, closed, or merged state: only the former blocks dependent tasks.

## GitHub configuration

The default configuration path is `os.UserConfigDir()/prx/config.yaml`. `--config` takes precedence over `PRX_CONFIG`; when neither is set, the default path is used. `prx config show`, `prx config validate`, and the `prx config host` / `prx config auth` subcommands manage this file without opening SQLite or contacting GitHub.

The file supports multiple GitHub.com or GitHub Enterprise Server hosts. Authentication methods are scoped to one normalized host and tried in their YAML order. Supported sources are `keychain`, `environment`, `inline`, and `gh_cli`:

```yaml
version: 1
github:
  hosts:
    - host: github.com
      web_url: https://github.com
      api_url: https://api.github.com/
      upload_url: https://uploads.github.com/
    - host: ghe.example.com
      web_url: https://ghe.example.com
      api_url: https://ghe.example.com/api/v3/
      upload_url: https://ghe.example.com/api/uploads/
  auth_methods:
    - id: ghe-environment
      host: ghe.example.com
      type: environment
      variable: GH_ENTERPRISE_TOKEN
    - id: github-cli
      host: github.com
      type: gh_cli
      user: HappyOnigiri
```

When `auth_methods` is omitted, GitHub.com keeps the compatibility candidates `GITHUB_TOKEN`, `GH_TOKEN`, and `gh auth token`. An explicit empty list disables those implicit candidates. Inline tokens are accepted through `prx config auth add --token-stdin`; they are stored in YAML but omitted from CLI, RPC, and WebUI reads. The SQLite cache contains only the successful method ID for a host/repository and can be ignored safely when a method is removed.

Configuration tests should use a temporary `--config` path and never depend on the real Keychain, ambient token variables, `gh` accounts, or GitHub. The live resolver tests use fake HTTPS servers and assert host isolation, cache fallback, `404` disambiguation, rate-limit handling, and secret-free outputs. Run the focused checks with `go test ./internal/config ./internal/github ./internal/app ./internal/rpc ./internal/cli` and `corepack pnpm --dir web test`.
