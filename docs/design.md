# PRX design policies

## Product direction

- Serve engineers coordinating 5–100 pull requests across repositories.
- Make the next safe task and its blockers understandable without opening every pull request.
- Keep the dependency graph durable, local-first, and useful to both people and coding agents.
- Treat graph causality as the primary product concern and generic project metrics as secondary.

## Sources of truth

This document records durable policy, rationale, and compatibility or safety constraints.
It does not mirror implementation details that should evolve with feature work.

| Concern | Source of current behavior |
|---|---|
| CLI commands and flags | Cobra definitions and the generated reference under `docs/cli/` |
| Domain values and derivation details | `internal/domain`, application code, and their tests |
| RPC fields | Protocol Buffer definitions under `proto/` |
| Persistence structure | Migrations and query definitions |
| WebUI components and interactions | `web/src/` and browser tests |

Update this document when a change alters a policy, rationale, public contract, or trust boundary.
Feature additions that follow existing policy should update their owning implementation and generated references instead.

## Architectural policy

Business rules have one application-level implementation shared by the CLI and RPC handlers.
Adapters translate that implementation instead of introducing alternate validation or status semantics.

Dependencies point inward toward domain policy.
Persistence owns storage and transactions, but it does not define competing business rules.
External providers remain replaceable behind application-facing interfaces.

The browser uses RPC and never opens SQLite or invokes the CLI.
The server remains authoritative for validation and derived business state.
The browser may translate and arrange structured state for presentation.
Known states and expected failure reasons cross RPC boundaries as enums or structured details.
Only unexpected diagnostics may remain unstructured English.

## Public contract policy

Behavior documented as a CLI, JSON, state, or dependency contract is public.
Changes to those contracts require coordinated implementation, tests, generated references, and policy updates when the policy itself changes.

Machine-readable CLI output must be deterministic and versioned, and its success data must stay free of presentation text.
Errors use stderr and leave stdout empty so automation cannot confuse a failed command with data.

| Condition | Output policy |
|---|---|
| No output flag | Concise text regardless of stdout |
| `--json` | JSON regardless of stdout |

Concise text is the default for routine inspection by people and coding agents.
Use JSON when a caller needs fields omitted from the text presentation or a stable schema for programmatic parsing.

Successful JSON commands emit their data object directly without a `schema_version`, `ok`, or `data` envelope.
Empty collections are `[]`, never `null`.
Failed JSON commands emit the versioned error object to stderr and leave stdout empty.
The current CLI response schema version is `2`.
Its error object contains `code`, `message`, and the failed command's complete help in `hint`, the only machine-readable field that carries presentation text.
Failures return a non-zero exit status.
Text output does not vary with terminal width or ambient environment.
The CLI implementation and black-box tests own current field names and presentation details.

Text failures print the error followed by the same complete command help used by normal help.
An explicit `--json` makes successful help a JSON object with the complete help in `hint`.
Help succeeds without opening configuration or storage resources.

Resource commands use their shallow form for routine reads.
Feature and task commands list without an identifier and show details with one identifier.
`show` resolves a feature public ID, feature slug, or task public ID when a feature slug conflicts with a mutation command name.
Dependency, pull-request, and document commands list when invoked without a mutation subcommand.
Implementation plans use `plan TASK_ID`, and configuration reads use `config`, `config host`, or `config auth`.
Mutation operations retain explicit verbs so state-changing intent remains visible.

Mutations remain non-interactive so people and coding agents use the same surface.
A missing mutation target fails instead of reporting a successful no-op.
Destructive traversal of referenced data requires an explicit cascade request.

Public feature and task identifiers remain distinct from storage identifiers.
Storage UUIDs must not cross the CLI, RPC, or WebUI boundary.

A bulk operation may report item-level failures without discarding successful items.
Command-level failure is reserved for failure of the operation itself.

## Domain policy

The server derives display state from stored state and external facts.
Clients must not recreate that derivation independently.

Manual task-state overrides take precedence over automatic derivation.
Dependency satisfaction uses raw completion semantics rather than display labels.
Presentation flags such as review, conflict, or staleness do not silently redefine completion.

Stale synchronization data preserves the last known state and remains visibly marked as stale.
An external failure must not rewrite known state as unknown.

Dependencies point from blocker to blocked.
Dependency mutations preserve feature ownership and DAG integrity.
Cycle rejection includes enough context for callers to explain the failure.

Current state values, display precedence, and readiness conditions belong to the domain implementation and its tests.

## Persistence and configuration policy

PRX uses SQLite to keep installation, inspection, and backup local.
The design accepts a single-writer constraint in exchange for a single local database.

Settings follow ownership rather than convenience:

| Setting | Storage policy |
|---|---|
| Affects CLI or server behavior | A configuration file accessible to the CLI |
| Affects only WebUI presentation | Browser Local Storage |

Persistent mutations and migrations are atomic.
Configuration writes preserve local secret-file protections and use atomic replacement.

Pull-request identity includes the normalized host.
Repositories with the same owner and name on different GitHub hosts must remain distinct.

Caches may remember which credential method succeeded, but they must never contain credential material or token-derived secrets.
Removing a credential method invalidates its cached selection without requiring manual database repair.

Synchronization fails safely:

- Preserve each field's last successful value when that field cannot be refreshed.
- Record staleness and the failure without replacing known state with guesses.
- Isolate item-level failures so one inaccessible repository does not discard unrelated successes.

GitHub synchronization is opportunistic rather than daemon-driven.
Commands that open the database and a visible WebUI check whether the shared interval has expired.
No refresh occurs while both the CLI and WebUI are idle.

Pull requests are fetched through GraphQL in batches that group the repositories of one host into a single request.
A host whose GraphQL endpoint answers with an HTTP error falls back to fetching each pull request over REST.

The YAML configuration owns the shared interval and each host's GraphQL endpoint.
The interval defaults to 3600 seconds and cannot be lower than 600 seconds.
SQLite records the latest attempt and completion, and atomically grants one caller the right to run an expired refresh.

Automatic refreshes include pull requests from active features only.
Merged pull requests are terminal and are not fetched again.
Closed pull requests remain eligible so a reopened pull request can be detected.
Manual refreshes ignore the interval and may target an archived feature, but they also skip merged pull requests.
Only a refresh that covers every eligible pull request records a run and resets the interval; one narrowed to a feature or task leaves the recorded run status untouched.

Automatic failures are best effort and never fail the command or page load that noticed the expired interval.
An automatic refresh is bounded by a deadline so an unreachable host cannot block the command that noticed the expired interval; exceeding it is recorded as an automatic failure.
They remain visible in the persisted run status and the stale state of affected pull requests.
Manual refreshes continue to return operation-level failures while preserving successful item updates.

Large Markdown bodies stay outside snapshots.
Snapshots carry only the metadata needed for derived state.

## GitHub credential policy

Credential methods are scoped to one normalized host and evaluated in explicit order.
An omitted method list may use documented compatibility defaults.
An explicitly empty method list disables implicit credentials.

An explicitly selected account must not inherit ambient credentials from another source.
Authorization-bearing requests must not follow redirects to another origin.

Fallback is appropriate for authentication and permission failures that another credential may resolve.
Rate limits, transport failures, and server failures do not trigger credential rotation.
The provider implementation owns the current error classification and disambiguation probes.

Public CLI, RPC, WebUI, log, error, and cache reads remain secret-free.
Inline credentials are an explicit local-trust trade-off, not permission to expose stored values.

## Local trust boundary

`prx serve` is a local tool for one trusted user, not an authenticated multi-user service.
Its threat model treats the network and browser as untrusted.

The server binds to loopback by default.
Non-loopback exposure requires an explicit listen address.
Requests must be bound to the configured origin and RPC protocol so another origin cannot drive the local database.

Markdown preview is limited to explicitly registered document paths.
It must not become a general filesystem reader.
Markdown preview reads remain bounded.

Production responses use restrictive browser security headers.
The server implementation owns the current header set and request-validation mechanics.

## WebUI policy

The dependency canvas was selected because causal relationships are the product's defining information.
Navigation and inspection should preserve that context instead of replacing it with a generic dashboard workflow.

Business state and credentials stay on the server.
Language, theme, zoom, and similar presentation-only preferences may remain browser-local.

State colors are reserved for state communication rather than decoration.
Identifiers and counts may use monospace, while normal content prioritizes readability in English and Japanese.
Nonessential motion respects the reduced-motion preference.

Icon-only controls require an accessible name and tooltip.
Controls keep a visible label when an icon cannot communicate the target, result, or danger scope.
Pointer interactions retain a keyboard-accessible alternative.

Current screens, components, gestures, and control placement belong to the WebUI implementation and its tests.

## Trade-offs

- Synchronization runs opportunistically from CLI commands and a visible WebUI without introducing a background worker.
- A normalized schema keeps a future PostgreSQL migration practical.
- Inline credentials favor local automation while accepting the configuration file's trust boundary.
- Track prospective features in their owning plans or pull requests instead of maintaining a feature backlog here.
