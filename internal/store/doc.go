// Package store owns SQLite persistence, embedded migrations, sqlc queries, and
// transactional repository operations. Among PRX packages it may import only
// internal/db and internal/domain, keeping application rules outside storage.
// The database must enable WAL, foreign keys, and a five-second busy_timeout;
// _txlock=immediate takes the write lock when a transaction starts because a
// deferred read-modify-write transaction can fail with SQLITE_BUSY_SNAPSHOT,
// which busy_timeout does not retry; and embedded migrations apply and record
// each version in one transaction. Read the Packages, Domain decisions, and
// Storage and operational boundaries sections of docs/design.md before changing
// persistence behavior.
package store
