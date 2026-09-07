# Domain policy

The server derives display state from stored state and external facts.
Clients must not recreate that derivation independently.

A task has no kind: any task may hold a pull request, and one without a pull request reaches completion through its stored status.
Deciding that in advance changed nothing the server derives, so it is not asked for.

A task status is always chosen by hand; there is no automatic member to select.
The unfinished statuses yield to an attached pull request, so linking one presents the pull request's state without a second edit.
The designing status yields once more, to a registered implementation plan, so a task marked as being designed is presented as designed the moment its plan is registered.
The finished statuses outrank a pull request, so a task settled by hand stays settled while its pull request is still open.
A task left in progress without a pull request satisfies nothing and clears no dependent, because the work it names has not landed anywhere.
Designing describes deciding how the work will be built rather than building it, so it keeps the readiness question a not-started task asks.
Dependency satisfaction uses raw completion semantics rather than display labels.
Presentation flags such as review, conflict, or staleness do not silently redefine completion.

A feature is presented through the same two layers of stored status and derived status, but it keeps an automatic member that a task no longer has.
Its stored status defaults to automatic, and an automatic feature is presented as completed once it owns at least one task and every one of them is finished.
A stored status other than automatic is a manual decision and is presented unchanged, so a feature returned to active work stays active while its tasks remain finished.
A feature has no separate derived vocabulary: the derived value is a stored status without the automatic member.

A project is the unit above a feature: it groups features and holds the documents they share.
Membership is required, a feature belongs to exactly one project, and dependencies stay inside one feature regardless of project.
A feature is created in a project and can only move to another one, so there is no state in which work sits outside every project.
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
