# PRX

PRX is a local-first dependency control room for initiatives that span many GitHub pull requests. It stores a normalized DAG in SQLite, derives the next safe tasks, refreshes pull-request state directly from GitHub, and exposes the same application rules through a non-interactive CLI and a ConnectRPC-powered React workspace.

## Requirements

- Go 1.26 or newer
- Node.js 24 and pnpm 11.24
- A Chromium browser for Playwright development checks
- `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI for live GitHub synchronization

`sqlc`, Buf, and Protocol Buffer generators are pinned as Go tool dependencies. They do not need global installation. The shipped binary does not require Node.js, pnpm, or `gh` when a token is supplied.

## Build and start

```sh
pnpm install --frozen-lockfile
make install
prx seed --github-fixture demo
prx serve
```

`make install` builds the production binary and installs it to `~/.local/bin/prx`. Set `INSTALL_DIR` to install elsewhere. Ensure the installation directory is on `PATH`, then open <http://127.0.0.1:7331>. The production web build is embedded in the binary; no separate frontend process is needed.

The default database is stored under the operating system's user configuration directory. Use `--db /path/to/prx.db` or `PRX_DB` to select another database. The server binds only to `127.0.0.1:7331` unless `--addr` is explicitly supplied.

For frontend development, run the API and Vite separately:

```sh
go run ./cmd/prx --github-fixture demo serve
pnpm --dir web dev
```

## CLI

Every mutation is non-interactive. Add `--json` for a versioned envelope whose stdout contains JSON only; warnings and server logs go to stderr.

```sh
prx feature create --slug checkout --title "Checkout rollout" --json
prx task create --feature checkout --title "Add payment intent API" --assignee Mika --json
prx dependency add BLOCKER_TASK_ID BLOCKED_TASK_ID --json
prx pr attach --task TASK_ID --url https://github.com/acme/payments/pull/42 --json
prx document add --task TASK_ID --kind markdown_path --value docs/checkout.md --json
prx sync --feature checkout --json
prx graph checkout --json
prx ready --json
prx reviews --json
prx conflicts --json
prx stale --json
prx snapshot --json
prx validate --json
prx serve
```

`prx feature archive SLUG` hides a feature from the sidebar and `prx feature unarchive SLUG` brings it back; `prx feature update SLUG --archived=false` does the same.

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

## State rules

Display priority is merged, closed without merge, draft, conflict, changes requested, approved, review waiting, open, then unknown. Review and mergeability remain separate stored fields.

A PR task satisfies dependencies only when its PR is freshly known to be merged. A manual task satisfies them only when manually completed. Cancelled, closed-without-merge, stale, unknown, and incomplete data fail closed. Ready is derived when a planned, incomplete task has only satisfied blockers; it is never stored as a manual flag.

Failed GitHub refreshes preserve the last successful fields and mark the record stale with a sync error. Bulk refreshes persist successes and failures independently so one inaccessible repository does not discard other results.

## Data and security

- Migrations are embedded and applied transactionally with a schema version table.
- SQLite enables foreign keys, WAL, a five-second busy timeout, and bounded connection pooling.
- The browser only receives typed ConnectRPC responses and cannot invoke the CLI or read arbitrary local files.
- `prx serve` rejects requests whose `Host` or `Origin` header does not match the listen address, and the RPC handler requires the Connect protocol header, so a rebound DNS name cannot drive the local database.
- Markdown documents are stored as path references; their contents are not served.
- Tokens are never written to SQLite or application logs.
- Production responses set a restrictive content security policy and related browser headers.

See [docs/design.md](docs/design.md) for package boundaries, status decisions, the remote-CLI seam, UI alternatives, and trade-offs.

## Verification

```sh
make generate          # sqlc, Buf lint, Go/TypeScript protobuf generation
make generated-check  # regeneration must produce no diff
make lint              # go vet, golangci-lint, ESLint, strict TypeScript
make test              # Go, Vitest, and component coverage
make test-race         # Go race detector
make e2e               # real Go server, SQLite, ConnectRPC, Chromium
make ci                # all required checks and production build
```

Playwright covers browser CRUD, cycle rejection, pull-request and Markdown attachment, deterministic sync, dependency removal, persistence after reload, console/network failures, 320-pixel reflow, and 8/50/100-node layouts. Graph screenshots are written to `test-results/screenshots/` and uploaded by GitHub Actions.
