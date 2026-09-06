# PRX design policies

Each document in this directory is a separate reading unit.
`AGENTS.md` at the repository root indexes which of them a change has to read.

## Product direction

- Serve engineers coordinating 5–100 pull requests across repositories.
- Make the next safe task and its blockers understandable without opening every pull request.
- Keep the dependency graph durable, local-first, and useful to both people and coding agents.
- Treat graph causality as the primary product concern and generic project metrics as secondary.

## Sources of truth

These documents record durable policy, rationale, and compatibility or safety constraints.
They do not mirror implementation details that should evolve with feature work.

| Concern | Source of current behavior |
|---|---|
| CLI commands and flags | Cobra definitions and the generated reference under `docs/cli/` |
| Domain values and derivation details | `internal/domain`, application code, and their tests |
| RPC fields | Protocol Buffer definitions under `proto/` |
| Persistence structure | Migrations and query definitions |
| WebUI components and interactions | `web/src/` and browser tests |

Update these documents when a change alters a policy, rationale, public contract, or trust boundary.
Feature additions that follow existing policy should update their owning implementation and generated references instead.
Track prospective features in their owning plans or pull requests instead of maintaining a feature backlog here.
