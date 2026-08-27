# PRX initial design

## Product job

- Audience: engineers coordinating 5–100 pull requests across repositories.
- Primary job: identify the next safe task and understand why another task is blocked.
- Artifact: a durable feature graph backed by SQLite and refreshed from GitHub.
- Frequency: checked throughout the workday by people and coding agents.
- First action: open the ready queue or select a graph node.
- Success evidence: dependency, review, conflict, merge, and stale states are visible without opening each pull request.

## Architecture

`cmd/prmap` constructs one application service used directly by Cobra commands and by thin ConnectRPC handlers. The service owns validation and derived state. The SQLite repository owns persistence and transactions. A GitHub provider interface isolates network synchronization and makes fixtures deterministic.

The browser uses generated Protocol Buffer descriptors through ConnectRPC. It never opens SQLite or invokes the CLI. The production build is emitted into `internal/webui/dist` and embedded in the Go binary.

## Packages

- `internal/domain`: entities, status derivation, DAG validation, ready calculation.
- `internal/store`: embedded migrations, sqlc queries, and transactional repository.
- `internal/app`: use cases shared by CLI and RPC.
- `internal/github`: direct REST provider and deterministic fixtures.
- `internal/cli`: non-interactive Cobra surface and stable JSON envelopes.
- `internal/rpc`: ConnectRPC translation only.
- `internal/webui`: embedded Vite production assets.
- `web`: React application, generated API types, tests, and Playwright scenarios.

## Domain decisions

Feature states are `active`, `paused`, `completed`, and `cancelled`. Task states are `planned`, `in_progress`, `completed`, and `cancelled`; PR tasks derive completion from a merged PR, while manual tasks use `completed`. A stale or incomplete blocker fails closed even when its last successful display state is retained.

GitHub display priority is merged, closed without merge, draft, conflict, changes requested, approved, review waiting, open, then unknown. Conflict and review decision remain separate stored fields. Review waiting means an open, non-draft PR whose current review decision is review-required or that has requested reviewers.

Dependencies are directed from blocker to blocked. Edge insertion loads the feature graph inside the write transaction and returns the discovered cycle path. Database checks cover self edges, duplicates, ownership, and foreign keys; application validation covers reachability.

## Storage and operational boundaries

SQLite uses WAL, foreign keys, a 5-second busy timeout, and explicit transactions. Migrations are embedded and each version is committed atomically. The default database lives in the user data directory and is replaceable with `--db` or `PRMAP_DB`.

The CLI calls a `Service` interface. Its local implementation is used now; a future remote implementation can forward the same operations over Connect without changing command parsing.

GitHub calls are direct HTTP requests. Authentication checks `GITHUB_TOKEN`, then `GH_TOKEN`, then invokes `gh auth token` only to obtain a credential; tokens are never persisted or logged.

## UI structure decision

Three structures were considered: metric cards leading to tables, a queue/table split with a mini graph, and a full dependency canvas with navigation and an inspector. The full canvas is provisionally selected because graph causality—not generic project metrics—is the product's defining work.

The palette is Blueprint `#101b2d`, Grid `#263650`, Fog `#e8edf2`, Link `#58a6ff`, Ready `#50d1c0`, Conflict `#ff756d`, and Merged `#92d06d`. DIN-style condensed headings evoke engineering drawings; system sans supports dense reading; monospace labels identify repositories and state. The signature is a directional dependency spine that remains legible from 8 to 100 nodes. Motion is limited to state transitions and disabled under reduced-motion preferences.

## Trade-offs

SQLite keeps installation local and a single binary possible, while WAL and retries mitigate but do not remove its single-writer constraint. Manual sync avoids worker lifecycle complexity. A normalized schema makes future PostgreSQL migration practical. Layout persistence, webhooks, remote mode, authentication, and collaboration remain outside the initial implementation.
