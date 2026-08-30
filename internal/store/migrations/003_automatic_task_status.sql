-- SQLite cannot add a CHECK constraint to an existing table in place. Keep
-- every dependent row while rebuilding the task table and its foreign-key
-- children around the new stored status values.
CREATE TEMP TABLE task_status_migration AS
SELECT id, feature_id, title, scope, kind, status, assignee, created_at, updated_at
FROM tasks;

CREATE TEMP TABLE dependency_migration AS SELECT * FROM dependencies;
CREATE TEMP TABLE pull_request_migration AS SELECT * FROM pull_requests;
CREATE TEMP TABLE document_migration AS SELECT * FROM documents;

CREATE TABLE tasks_new (
  id TEXT PRIMARY KEY,
  feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE RESTRICT,
  title TEXT NOT NULL CHECK(length(trim(title)) > 0),
  scope TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL CHECK(kind IN ('pr','manual')),
  status TEXT NOT NULL DEFAULT 'auto' CHECK(status IN ('auto','not_started','in_progress','completed','closed')),
  assignee TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO tasks_new (id, feature_id, title, scope, kind, status, assignee, created_at, updated_at)
SELECT
  id,
  feature_id,
  title,
  scope,
  kind,
  CASE
    WHEN status = 'in_progress' AND kind = 'manual' THEN 'in_progress'
    WHEN status = 'completed' THEN 'completed'
    WHEN status = 'cancelled' THEN 'closed'
    ELSE 'auto'
  END,
  assignee,
  created_at,
  updated_at
FROM task_status_migration;

DROP TABLE dependencies;
DROP TABLE pull_requests;
DROP TABLE documents;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX tasks_feature_idx ON tasks(feature_id, created_at, id);
CREATE INDEX tasks_status_idx ON tasks(status, kind);

CREATE TABLE dependencies (
  blocker_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
  blocked_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
  created_at TEXT NOT NULL,
  PRIMARY KEY(blocker_task_id, blocked_task_id),
  CHECK(blocker_task_id <> blocked_task_id)
);

CREATE INDEX dependencies_blocked_idx ON dependencies(blocked_task_id);
INSERT INTO dependencies (blocker_task_id, blocked_task_id, created_at)
SELECT blocker_task_id, blocked_task_id, created_at FROM dependency_migration;

CREATE TABLE pull_requests (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
  host TEXT NOT NULL DEFAULT 'github.com' COLLATE NOCASE,
  owner TEXT NOT NULL COLLATE NOCASE,
  repository TEXT NOT NULL COLLATE NOCASE,
  number INTEGER NOT NULL CHECK(number > 0),
  url TEXT NOT NULL,
  node_id TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT '',
  assignees_json TEXT NOT NULL DEFAULT '[]',
  state TEXT NOT NULL DEFAULT 'unknown' CHECK(state IN ('open','closed','merged','unknown')),
  draft INTEGER NOT NULL DEFAULT 0 CHECK(draft IN (0,1)),
  review_state TEXT NOT NULL DEFAULT 'unknown' CHECK(review_state IN ('none','required','approved','changes_requested','unknown')),
  mergeability TEXT NOT NULL DEFAULT 'unknown' CHECK(mergeability IN ('mergeable','conflicting','unknown')),
  github_updated_at TEXT,
  last_synced_at TEXT,
  sync_error TEXT NOT NULL DEFAULT '',
  stale INTEGER NOT NULL DEFAULT 1 CHECK(stale IN (0,1)),
  UNIQUE(host, owner, repository, number)
);

CREATE INDEX pull_requests_state_idx ON pull_requests(state, review_state, mergeability, stale);
CREATE INDEX pull_requests_repository_idx ON pull_requests(host, owner, repository, number);
INSERT INTO pull_requests (
  task_id, host, owner, repository, number, url, node_id, author, assignees_json,
  state, draft, review_state, mergeability, github_updated_at, last_synced_at,
  sync_error, stale
)
SELECT
  task_id, host, owner, repository, number, url, node_id, author, assignees_json,
  state, draft, review_state, mergeability, github_updated_at, last_synced_at,
  sync_error, stale
FROM pull_request_migration;

CREATE TABLE documents (
  id TEXT PRIMARY KEY,
  feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
  task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
  kind TEXT NOT NULL CHECK(kind IN ('url','markdown_path')),
  title TEXT NOT NULL DEFAULT '',
  value TEXT NOT NULL CHECK(length(trim(value)) > 0),
  created_at TEXT NOT NULL,
  CHECK((feature_id IS NOT NULL AND task_id IS NULL) OR (feature_id IS NULL AND task_id IS NOT NULL))
);

CREATE INDEX documents_feature_idx ON documents(feature_id);
CREATE INDEX documents_task_idx ON documents(task_id);
INSERT INTO documents (id, feature_id, task_id, kind, title, value, created_at)
SELECT id, feature_id, task_id, kind, title, value, created_at FROM document_migration;

DROP TABLE task_status_migration;
DROP TABLE dependency_migration;
DROP TABLE pull_request_migration;
DROP TABLE document_migration;
