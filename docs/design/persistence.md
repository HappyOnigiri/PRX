# Persistence and configuration policy

PRX uses SQLite to keep installation, inspection, and backup local.
The design accepts a single-writer constraint in exchange for a single local database.
A normalized schema keeps a future PostgreSQL migration practical.

CLI and server settings share a configuration file; presentation-only settings follow the Local Storage rule in the repository `AGENTS.md`.

Persistent mutations and migrations are atomic.
Configuration writes preserve local secret-file protections and use atomic replacement.

Where the recorded migration versions and the actual schema disagree, the schema decides.
Opening a database repairs the record before applying anything: a version whose schema is absent loses its record, and a version whose schema is already in place regains one.
This keeps a database openable after an older build, which recognizes a schema only by the shape it knew, removes a record that a later migration has since reshaped.
A migration is never replayed against a schema that has already moved past it.

A configuration file using a supported version still loads when it contains unknown fields.
Those fields are reported as warnings instead of failing the command or the server start, and the next configuration write drops them.
Unsupported versions and every other decoding failure keep the configuration from loading, so a malformed or ambiguous file is never accepted silently.

Template and synchronization configuration policies live in [agent-prompts.md](agent-prompts.md) and [github-sync.md](github-sync.md).

`prx serve --demo` creates a new temporary database, configuration, and Markdown document set for each server process.
It never reads or writes the normal database and configuration paths, including paths supplied through environment variables.
The temporary environment is removed after a normal shutdown and is never reused after an abnormal shutdown.
All demo mutations remain available until that process exits so the WebUI behaves like the normal application.

## Documents and implementation plans

Document entries in snapshots carry only metadata needed for derived state.

Documents use one model for project, feature, and task references, and each document belongs to exactly one of the three.
Each document stores exactly one source: an HTTP or HTTPS URL, a registered local file path, or inline Markdown.
Inline Markdown is limited to 1 MiB and is loaded only by a detailed read.

A task may designate at most one document as its implementation plan.
That designation stays exclusive to tasks: a project or feature document can never be a plan.
The designation moves a not-started or designing task without a pull request to the designed display state, and leaves readiness, dependency satisfaction, and completion untouched.
Non-plan documents, feature documents, and project documents have no application-level count limit.
