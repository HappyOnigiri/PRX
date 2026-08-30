# Repository instructions

PRX is a local-first tool for coordinating dependency graphs of tasks and GitHub pull requests. It is delivered as a Go CLI/server with an embedded React UI and SQLite storage.

- Treat the CLI, JSON, and state behavior documented under `docs/` as public contracts. Update implementation, tests, and documentation together when they change.
- Keep `README.md` minimal so that parallel branches rarely touch it. It holds only what someone needs to build, start, and develop PRX, plus links to the documentation.
- Put durable design policy and rationale in `docs/design.md`, non-obvious verification policy in `docs/development.md`, and the generated CLI surface in `docs/cli/`.
- Edit `README.md` only when a change makes something it already states wrong.
- RPC schemas and behavior may change without backward compatibility.
- Apply each RPC change to the Protocol Buffer source, server, in-repository clients, and tests.
- Update documentation in the same change when the recorded public contract or policy changes.
- Read `docs/design.md` before changing dependency or status semantics, persistence, GitHub synchronization, RPC boundaries, or the UI structure.
- Edit Protocol Buffer, migration, and SQL query sources, then run `make generate`.
- Do not hand-edit generated files under `gen/`, `internal/db/`, or `web/src/gen/`.
- `internal/webui/dist/` is build output. Keep only `.gitkeep` tracked and produce assets through `make build` or `make web-build`.
- Run `make ci` before handing off implementation changes.

## Git workflow

- User approval is not required before committing or pushing changes.
- When updating an existing pull request, commit and push the changes.

## Settings storage

- Store settings that affect CLI behavior in a config file accessible to the CLI. This config does not exist yet. Introduce it when such settings are implemented.
- Store settings that affect only WebUI presentation in the browser's Local Storage.
