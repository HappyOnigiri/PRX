# Repository instructions

PRX is a local-first tool for coordinating dependency graphs of tasks and GitHub pull requests, delivered as a Go CLI/server with an embedded React UI and SQLite storage.

- Treat the CLI, JSON, and state behavior documented in `README.md` as public contracts; update implementation, tests, and documentation together when they change.
- Read `docs/design.md` before changing dependency or status semantics, persistence, GitHub synchronization, RPC boundaries, or the UI structure.
- Edit Protocol Buffer, migration, and SQL query sources, then run `make generate`; do not hand-edit generated files under `gen/`, `internal/db/`, or `web/src/gen/`.
- `internal/webui/dist/` is build output. Keep only `.gitkeep` tracked and produce assets through `make build` or `make web-build`.
- Run `make ci` before handing off implementation changes.
