-- The stored feature status gains an automatic mode, but SQLite cannot extend
-- the status CHECK constraint in place and rebuilding features would drag in
-- tasks, documents, and their own children. Keep the automatic flag in its own
-- column instead: status_auto = 1 means the feature derives its presented
-- status from its tasks, and the status column is normalized to 'active' and
-- carries no meaning until an explicit status replaces it.
ALTER TABLE features ADD COLUMN status_auto INTEGER NOT NULL DEFAULT 1 CHECK(status_auto IN (0,1));

-- Existing paused, completed, and cancelled features were chosen by a person,
-- so they keep their status as a manual override. Active features move to the
-- automatic mode. ADD COLUMN does not evaluate the CHECK against existing rows,
-- so this correction is about meaning rather than constraint satisfaction.
UPDATE features SET status_auto = 0 WHERE status <> 'active';
