// Package domain defines PRX entities, status derivation, DAG validation, and
// ready-task calculation. It is a leaf package and may depend only on the
// standard library, never on application, storage, GitHub, CLI, or RPC layers.
// Dependency satisfaction fails closed: stale, unknown, or incomplete data does
// not satisfy a blocker; display priority is merged, closed without merge,
// draft, conflict, changes requested, approved, review waiting, open, then
// unknown; and Ready is derived rather than stored. Read the Domain decisions
// and Packages sections of docs/design.md before changing domain semantics.
package domain
