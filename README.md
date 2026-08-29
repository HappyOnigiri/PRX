# PRX

PRX is a local-first dependency control room for initiatives that span many GitHub pull requests. It stores a normalized DAG in SQLite, derives the next safe tasks, refreshes pull-request state directly from GitHub, and exposes the same application rules through a non-interactive CLI and a ConnectRPC-powered React workspace.

## Requirements

- Go 1.26 or newer
- The Node.js version is defined in `.tool-versions`, and the pnpm version is defined in the root `package.json` `packageManager`
- `golangci-lint` is installed into `bin/` at the version in `.tool-versions` by `make lint`
- Python is not required
- A Chromium browser for Playwright development checks
- `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI for live GitHub synchronization

`sqlc`, Buf, and Protocol Buffer generators are pinned as Go tool dependencies. They do not need global installation. The shipped binary does not require Node.js, pnpm, or `gh` when a token is supplied.

## Build and start

```sh
make install
prx seed --github-fixture demo
prx serve
```

`make install` installs the locked web dependencies, builds the production binary, and installs it to `~/.local/bin/prx`. Set `INSTALL_DIR` to install elsewhere. Ensure the installation directory is on `PATH`, then open <http://127.0.0.1:7331>. The production web build is embedded in the binary; no separate frontend process is needed.

The default database is stored under the operating system's user configuration directory. Use `--db /path/to/prx.db` or `PRX_DB` to select another database. The server binds only to `127.0.0.1:7331` unless `--addr` is explicitly supplied.

For development, start the Go API reloader and the Vite development server together:

```sh
make dev
```

Open <http://127.0.0.1:7331>, the same URL used by the production server. Vite
applies WebUI changes with hot module replacement and proxies RPC requests to
the development Go server on port 7332. Air rebuilds and restarts the Go server
when Go source files change. Stop any existing `prx serve` process that is using
port 7331 before starting the development servers.

## CLI

The complete command reference, including options and examples, is available in [docs/cli/prx.md](docs/cli/prx.md) and is generated from the Cobra command definitions by `make generate`.
Every mutation is non-interactive. Add `--json` for a versioned envelope whose stdout contains JSON only; warnings and server logs go to stderr.

Representative commands:

```sh
prx feature create --slug checkout --title "Checkout rollout" --json
prx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID --json
prx ready --json
```

`prx feature archive SLUG` hides a feature from the sidebar and `prx feature unarchive SLUG` brings it back; `prx feature update SLUG --archived=false` does the same.

`--github-fixture` supplies a deterministic provider for the command. Without a fixture, only `prx serve` attempts live GitHub authentication; other commands, including `sync`, run without a provider and report `GitHub provider is not configured` when synchronization is requested.

Removing a dependency, detaching a pull request, or deleting a document that does not exist fails with `not_found` rather than reporting success.

Feature and Task deletion refuses to remove referenced data unless `--cascade` is supplied. Adding a dependency performs same-feature validation and cycle detection in the write transaction; cycle errors include the discovered path.

`prx seed --github-fixture demo --features 100 --tasks 50` creates deterministic performance data without network access. A fixture JSON file can map canonical PR URLs to GitHub states:

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

## State rules

Display priority is merged, closed without merge, draft, conflict, changes requested, approved, review waiting, open, then unknown. Review and mergeability remain separate stored fields.

A PR task satisfies dependencies only when its PR is freshly known to be merged. A manual task satisfies them only when manually completed. Cancelled, closed-without-merge, stale, unknown, and incomplete data fail closed. Ready is derived when a planned, incomplete task has only satisfied blockers; it is never stored as a manual flag.

Failed GitHub refreshes preserve the last successful fields and mark the record stale with a sync error. Bulk refreshes persist successes and failures independently so one inaccessible repository does not discard other results.

## WebUI preferences

The WebUI supports English and Japanese. It uses a saved display-language preference first, then the browser's preferred languages, and falls back to English. The selector in the navigation rail stores the preference in browser Local Storage; it does not alter CLI behavior or server data.

The WebUI theme follows the operating system or browser color-scheme preference by default. If the browser does not report a preference, the light theme is used. The navigation rail can explicitly select the light or dark theme; that choice is stored in browser Local Storage and takes precedence until the theme is set back to System.

The dependency canvas also stores the user's zoom level in browser Local Storage. Every feature graph opens at that same zoom level and is centered without automatically fitting its size to the viewport.

ConnectRPC represents fixed domain states, blocked reasons, and known errors with enums and structured details. The WebUI turns those values into localized text. Messages from unexpected server or GitHub failures are shown in their original form.

## Data and security

- Migrations are embedded and applied transactionally with a schema version table.
- SQLite enables foreign keys, WAL, a five-second busy timeout, and bounded connection pooling.
- The browser only receives typed ConnectRPC responses and cannot invoke the CLI. Markdown preview can read only files whose paths were explicitly registered as `markdown_path` documents; each preview is limited to 1 MiB.
- `prx serve` rejects requests whose `Host` or `Origin` header does not match the listen address, and the RPC handler requires the Connect protocol header, so a rebound DNS name cannot drive the local database.
- Markdown documents are stored as path references. The WebUI reads registered paths relative to the server's working directory (or as absolute paths), renders them in a read-only preview, and never includes their contents in the general snapshot response.
- Tokens are never written to SQLite or application logs.
- Production responses set a restrictive content security policy and related browser headers.

See [docs/design.md](docs/design.md) for package boundaries, status decisions, the remote-CLI seam, UI alternatives, and trade-offs.

## Verification

```sh
make generate               # sqlc, Buf format/lint, Go/TypeScript protobuf, and CLI reference generation
make generated-check        # regeneration into a temporary directory must match the tracked files
make mod-tidy-check         # go.mod and go.sum must be tidy
make lint                   # auto-installed golangci-lint (govet / gofumpt / gci / golines), deadcode, Prettier, ESLint, strict TypeScript, knip, stylelint
make test                   # Go, Vitest, and component coverage
make go-coverage-check      # handwritten Go packages must stay at or above the coverage baseline
make go-coverage-zero-check # every function in the core Go packages must be executed by tests
make test-race              # Go race detector
make test-race-coverage     # one race-enabled Go test run that also enforces the coverage baseline
make e2e                    # real Go server, SQLite, ConnectRPC, Chromium
make ci                     # install web dependencies, run all required checks in parallel, and build production assets
```

`make ci` runs the checks concurrently with one job per CPU (`CI_JOBS` overrides the count) and keeps going after a failure so one run reports every problem. Each check reads the tree or writes only its own output, and `make ci` uses `test-race-coverage` in place of the three separate Go test runs. GNU make 4 groups each job's output; the GNU make 3.81 shipped with macOS interleaves it.

`GO_COVERAGE_MIN` records the current coverage baseline for the handwritten Go packages. Raise it when their coverage improves so later changes cannot reduce it.

`make go-coverage-zero-check` runs the Go tests with cross-package coverage for `internal/app`, `internal/domain`, `internal/github`, `internal/rpc`, and `internal/store`, then fails if any target function has exactly 0.0% coverage. It excludes the WebUI, CLI black-box process tests, and generated code because those paths are verified by their own checks and are not part of the core package function contract.

Playwright covers browser CRUD, language selection persistence, cycle rejection, pull-request and Markdown attachment, deterministic sync, dependency removal, persistence after reload, console/network failures, 320-pixel reflow, and 8/50/100-node layouts. Graph screenshots are written to `test-results/screenshots/` and uploaded by GitHub Actions.
