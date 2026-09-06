# Persistence and configuration policy

PRX uses SQLite to keep installation, inspection, and backup local.
The design accepts a single-writer constraint in exchange for a single local database.
A normalized schema keeps a future PostgreSQL migration practical.

Settings follow ownership rather than convenience:

| Setting | Storage policy |
|---|---|
| Affects CLI or server behavior | A configuration file accessible to the CLI |
| Affects only WebUI presentation | Browser Local Storage |

Persistent mutations and migrations are atomic.
Configuration writes preserve local secret-file protections and use atomic replacement.

A configuration file using a supported version still loads when it contains unknown fields.
Those fields are reported as warnings instead of failing the command or the server start, and the next configuration write drops them.
Unsupported versions and every other decoding failure keep the configuration from loading, so a malformed or ambiguous file is never accepted silently.

The YAML configuration also owns the agent prompt templates, which the CLI and the WebUI both render from.
It owns the shared synchronization interval and each host's GraphQL endpoint as well, under the policy recorded in [github-sync.md](github-sync.md).

`prx serve --demo` creates a new temporary database, configuration, and Markdown document set for each server process.
It never reads or writes the normal database and configuration paths, including paths supplied through environment variables.
The temporary environment is removed after a normal shutdown and is never reused after an abnormal shutdown.
All demo mutations remain available until that process exits so the WebUI behaves like the normal application.

## Documents and implementation plans

Large Markdown bodies stay outside snapshots.
Snapshots carry only the metadata needed for derived state.

Documents use one model for project, feature, and task references, and each document belongs to exactly one of the three.
Each document stores exactly one source: an HTTP or HTTPS URL, a registered local file path, or inline Markdown.
Inline Markdown is limited to 1 MiB and is loaded only by a detailed read.
Snapshots and list operations never include inline bodies.

A task may designate at most one document as its implementation plan.
That designation stays exclusive to tasks: a project or feature document can never be a plan.
The designation moves a not-started task without a pull request from the not-started display state to the designed one, and leaves readiness, dependency satisfaction, and completion untouched.
Non-plan documents, feature documents, and project documents have no application-level count limit.
