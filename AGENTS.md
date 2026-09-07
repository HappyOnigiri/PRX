# Repository instructions

PRX is a local-first tool for coordinating dependency graphs of tasks and GitHub pull requests. It is delivered as a Go CLI/server with an embedded React UI and SQLite storage.

- Treat the CLI, JSON, and state behavior documented under `docs/` as public contracts. Update implementation, tests, and documentation together when they change.
- Keep `README.md` minimal so that parallel branches rarely touch it. It holds only what someone needs to build, start, and develop PRX, plus links to the documentation.
- Put durable design policy and rationale in `docs/design/`, non-obvious verification policy in `docs/development.md`, and the generated CLI surface in `docs/cli/`.
- Edit `README.md` only when a change makes something it already states wrong.
- RPC schemas and behavior may change without backward compatibility.
- Apply each RPC change to the Protocol Buffer source, server, in-repository clients, and tests.
- Update documentation in the same change when the recorded public contract or policy changes.
- Read the design documents a change touches, not the whole directory.
- Edit Protocol Buffer, migration, and SQL query sources, then run `make generate`.
- Do not hand-edit generated files under `gen/`, `internal/db/`, or `web/src/gen/`.
- `internal/webui/dist/` is build output. Keep only `.gitkeep` tracked and produce assets through `make build` or `make web-build`.
- Run `make ci` before handing off implementation changes.

## Comments

A comment is at most 3 lines and each line at most 200 display columns; both limits are enforced for Go and for everything ESLint reads under `web/`.

- Adjacent comment lines count as one comment. A blank source line, or moving a paragraph next to the code it explains, splits them.
- Tooling directives (`//go:*`, `//nolint`, `eslint-disable*`, `@ts-*`, …) count towards neither limit.
- Move rationale that outgrows 3 lines into the matching `docs/design/` document and leave a pointer to it.
- To keep a longer comment, put `commentlint:allow-long -- <reason>` on its own line in the same comment. It waives the line limit only, and one comment accepts one marker.

## Design documents

| Document | Read it before changing |
|---|---|
| `docs/design/README.md` | Product direction, and which source owns a detail this directory does not record |
| `docs/design/architecture.md` | Layering, adapter responsibilities, or what crosses the RPC boundary |
| `docs/design/cli-contract.md` | CLI command shape, output modes, JSON schema, identifiers, or mutation rules |
| `docs/design/diagnostics.md` | `prx debug` and its RPC |
| `docs/design/agent-prompts.md` | Agent prompt templates, their vocabulary, or template selection |
| `docs/design/domain.md` | Display state derivation, status semantics, dependencies, or project membership |
| `docs/design/archive.md` | Archived projects or features and the writes they forbid |
| `docs/design/persistence.md` | Storage, configuration files, demo mode, documents, or implementation plans |
| `docs/design/github-sync.md` | Pull-request identity, synchronization scope, scheduling, or failure handling |
| `docs/design/github-credentials.md` | Credential resolution, fallback, or secret handling |
| `docs/design/security.md` | The local trust boundary, server exposure, or local file access |
| `docs/design/webui.md` | WebUI structure, presentation state, or accessibility rules |

## Git workflow

- User approval is not required before committing or pushing changes.
- When updating an existing pull request, commit and push the changes.

## Settings storage

- Store settings that affect CLI behavior in a config file accessible to the CLI. This config does not exist yet. Introduce it when such settings are implemented.
- Store settings that affect only WebUI presentation in the browser's Local Storage.
