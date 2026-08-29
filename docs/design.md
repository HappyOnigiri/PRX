# PRX initial design

## Product job

- Audience: engineers coordinating 5–100 pull requests across repositories.
- Primary job: identify the next safe task and understand why another task is blocked.
- Artifact: a durable feature graph backed by SQLite and refreshed from GitHub.
- Frequency: checked throughout the workday by people and coding agents.
- First action: open the ready queue or select a graph node.
- Success evidence: dependency, review, conflict, merge, and stale states are visible without opening each pull request.

## Architecture

`cmd/prx` constructs one application service used directly by Cobra commands and by thin ConnectRPC handlers. The service owns validation and derived state. The SQLite repository owns persistence and transactions. A GitHub provider interface isolates network synchronization and makes fixtures deterministic.

The browser uses generated Protocol Buffer descriptors through ConnectRPC. It never opens SQLite or invokes the CLI. Fixed states and known reasons cross the RPC boundary as enums or structured details, while unexpected server and GitHub error messages may remain English. The production build is emitted into `internal/webui/dist` and embedded in the Go binary.

Package dependencies follow the same direction: `internal/domain` and `internal/github` are leaves; `internal/store` may import only `internal/domain` and `internal/db`; `internal/app` may import `internal/domain` and `internal/github` and depends on its `Repository` interface; `internal/rpc` and `internal/cli` use a `Service` interface; and `cmd/prx` assembles all concrete pieces.

## Packages

- `internal/domain`: entities, status derivation, DAG validation, ready calculation.
- `internal/store`: embedded migrations, sqlc queries, and transactional repository.
- `internal/app`: use cases shared by CLI and RPC.
- `internal/github`: direct REST provider and deterministic fixtures.
- `internal/cli`: non-interactive Cobra surface and stable JSON envelopes.
- `internal/rpc`: ConnectRPC translation only.
- `internal/webui`: embedded Vite production assets.
- `web`: React application, generated API types, tests, and Playwright scenarios.

## CLI contract

Every mutation is non-interactive so that people and coding agents drive the same surface. `--json` emits a versioned envelope; stdout then carries JSON only, and warnings and server logs go to stderr, so output can be piped without filtering. Operating on data that does not exist — removing a dependency, detaching a pull request, deleting a document, or deleting an implementation plan — fails with `not_found` instead of reporting success, so a caller cannot mistake a typo for a completed change. Feature and task deletion refuses to remove referenced data unless `--cascade` is supplied. Implementation plans are managed by `prx implementation-plan get|set|delete`; `set` reads exactly one of a file or stdin and returns the stored plan.

The root `package.json` is the single source of the product version. Every current build and install path appends `-dev` so it identifies the release on which development is based without claiming to be that release. A future distribution pipeline will stamp the stable version only into its release artifacts. `prx --version` and the WebUI read the same value from the running binary; the server injects it into the embedded index rather than maintaining a separate frontend version.

## Domain decisions

Feature states are `active`, `paused`, `completed`, and `cancelled`. Task storage states are `auto`, `not_started`, `in_progress`, `completed`, and `closed`; `auto` is the default and the other four values are manual overrides. With `auto`, a linked pull-request task uses the PR display priority, while a task without a PR is `designed` when it has an implementation plan and otherwise `not_started`. Manual overrides take precedence over both the PR and plan.

Dependencies use raw completion semantics rather than display labels. A manual completed or closed override satisfies a blocker. An automatic PR task satisfies a blocker when its last known raw PR state is open, closed, or merged, regardless of draft, review, conflict, or stale flags; an unknown state or missing PR remains blocking. Stale data stays visible as a warning and does not erase the last known state. Ready tasks are automatic or manual-not-started tasks displayed as not started or designed whose blockers are satisfied.

GitHub display priority is merged, closed without merge, draft, conflict, changes requested, approved, review waiting, open, then unknown. Conflict and review decision remain separate stored fields. Review waiting means an open, non-draft PR whose current review decision is review-required or that has requested reviewers.

Dependencies are directed from blocker to blocked. Edge insertion loads the feature graph inside the write transaction and returns the discovered cycle path. Database checks cover self edges, duplicates, ownership, and foreign keys; application validation covers reachability.

## Storage and operational boundaries

SQLite uses WAL, foreign keys, a 5-second busy timeout, and explicit transactions. Migrations are embedded and each version is committed atomically. The default database lives in the user data directory and is replaceable with `--db` or `PRX_DB`.

The CLI calls a `Service` interface. Its local implementation is used now; a future remote implementation can forward the same operations over Connect without changing command parsing.

GitHub calls are direct HTTP requests. Authentication checks `GITHUB_TOKEN`, then `GH_TOKEN`, then invokes `gh auth token` only to obtain a credential; tokens are never persisted or logged.

Synchronization fails safe rather than destructively. A failed refresh keeps the last successful fields and marks the record stale with a sync error, so a temporary outage never rewrites known state as unknown. If the PR body request succeeds but review metadata fails, the newly received core PR fields are persisted together with the previous review fields and the error. A bulk refresh persists successes and failures independently, so one inaccessible repository does not discard the results for the others.

Implementation plans live in a one-to-one `implementation_plans` table keyed by task. Their Markdown body is fetched only by the plan RPC or CLI command; snapshots carry only `has_implementation_plan`, allowing the server to derive `designed` without loading every body. Task and feature deletion treats plans as references unless cascading was requested.

## Local trust boundary

`prx serve` is a local tool rather than an authenticated service, so its defenses assume a single trusted user and an untrusted network and browser. It binds to `127.0.0.1:7331` unless `--addr` is supplied explicitly, rejects requests whose `Host` or `Origin` header does not match the listen address, and requires the Connect protocol header on RPC calls; a page from another origin or a rebound DNS name therefore cannot drive the local database. Production responses set a restrictive content security policy and related browser headers.

Markdown documents are stored as path references, resolved relative to the server's working directory or as absolute paths. Only paths explicitly registered as `markdown_path` documents can be read, which keeps the preview from turning the server into a general file reader.

## UI structure decision

Three structures were considered: metric cards leading to tables, a queue/table split with a mini graph, and a full dependency canvas with navigation and an inspector. The full canvas is provisionally selected because graph causality—not generic project metrics—is the product's defining work.

The interface uses matched neutral light and dark palettes with role-based tokens for the application background, rail, surfaces, borders, text, actions, and task states. A single blue accent identifies primary actions, links, selection, and focus; ready, review, conflict, and merged colors appear only on state indicators. System sans supports daily reading in English and Japanese, while monospace is reserved for identifiers and counts. The signature is a directional dependency spine with restrained state stripes that remains legible from 8 to 100 nodes. Motion is limited to state feedback and disabled under reduced-motion preferences.
CSS class names use kebab-case, and state tokens use `state-<kebab>`.

WebUI copy uses semantic i18next keys with bundled English and Japanese resources. The initial language is selected from a saved Local Storage preference, then the browser's preferred languages, with English as the fallback. Changing the display language updates Local Storage, the document language, and the page title without changing CLI or server configuration. The theme follows the browser color-scheme preference by default and falls back to light when no preference is reported. An explicit light or dark selection is stored in Local Storage and takes precedence until the user selects System again. The dependency canvas stores one user-selected zoom level in Local Storage and applies it to every feature graph while centering each graph independently. The server remains responsible for deriving business state; the browser only translates and assembles display text from structured RPC values.

Task cards expose pull requests, document references, and a short red GitHub sync-error marker as explicit rows. External references open in a new browser tab, while a registered Markdown path is read on demand through a document-ID RPC and rendered in a read-only modal. Markdown contents remain outside the snapshot, and preview reads are limited to 1 MiB. The task inspector opens only from the card's edit button so reference activation and editing are distinct actions. Its implementation-plan section loads the plan on demand, edits Markdown in a textarea, and renders a safe preview with the existing ReactMarkdown component; save and delete invalidate the snapshot so `not_started` and `designed` update immediately. Stale PR state and the last display state remain separate in the inspector.

## Trade-offs

SQLite keeps installation local and a single binary possible, while WAL and retries mitigate but do not remove its single-writer constraint. Manual sync avoids worker lifecycle complexity. A normalized schema makes future PostgreSQL migration practical. Layout persistence, webhooks, remote mode, authentication, and collaboration remain outside the initial implementation.
