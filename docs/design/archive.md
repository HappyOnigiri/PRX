# Archive policy

The application layer refuses creates, updates, and deletes inside archived projects or features, including changes to the containers themselves.
Refusals report one stable error code, because the caller's remedy is the same whichever container is archived.

The barrier lifts for exactly three operations:

- Moving the archived flag, in either direction, so archiving an already archived record is a no-op instead of an error.
  A request that also changes another field stays refused, so unarchiving cannot smuggle an edit past the barrier.
- Deleting a project or a feature, which is how archived work is finally discarded.
- A GitHub refresh that names a feature or a task explicitly, because that imports external fact rather than changing recorded intent.

The refusal is an application-layer decision taken before the write, not a database constraint, and it does not share a transaction with the write it guards.
Two processes on one database therefore have a window: a write that passed the barrier can land immediately after another process archived its container.
The archive is a coordination rule between people and their agents, not a lock, so recovering from that window is a manual deletion rather than a guarantee the store enforces.

Moving a feature into or out of an archived project is a write and is therefore refused; activating the project comes first.
A project's cascade deletion is the exception: it deletes the features it holds, which is a deletion rather than a membership change.

How an archived container is presented, and how it reaches the features inside it, is recorded in [domain.md](domain.md).
