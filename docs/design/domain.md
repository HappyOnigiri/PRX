# Domain policy

The server derives display state from stored state and external facts.
Clients must not recreate that derivation independently.

Manual task-state overrides take precedence over automatic derivation.
Dependency satisfaction uses raw completion semantics rather than display labels.
Presentation flags such as review, conflict, or staleness do not silently redefine completion.

A feature's presented status follows the same two-layer rule as a task's.
Its stored status defaults to automatic, and an automatic feature is presented as completed once it owns at least one task and every one of them is finished.
A stored status other than automatic is a manual decision and is presented unchanged, so a feature returned to active work stays active while its tasks remain finished.
A feature has no separate derived vocabulary: the derived value is a stored status without the automatic member.

A project is the unit above a feature: it groups features and holds the documents they share.
Membership is optional, a feature belongs to at most one project, and dependencies stay inside one feature regardless of project.
A project's only state is whether it is archived; it is deliberately outside the two-layer status rule that features and tasks share.
Archiving reaches into a project: a feature inside an archived project is presented as read-only even when its own archived flag is false.
The server derives that as `Feature.ReadOnly` and clients read it directly instead of combining the feature's flag with its project's.
Every read that carries a feature derives it, including the ones that return a single feature, so a client never has to know which read produced the value.
A read-only feature is presented in the archived category and leaves the active feature lists, the overview, and task search, which is the same treatment an individually archived feature receives.
The writes archiving forbids are recorded in [archive.md](archive.md).

Dependencies point from blocker to blocked.
Dependency mutations preserve feature ownership and DAG integrity.
Cycle rejection includes enough context for callers to explain the failure.

Current state values, display precedence, and readiness conditions belong to the domain implementation and its tests.
