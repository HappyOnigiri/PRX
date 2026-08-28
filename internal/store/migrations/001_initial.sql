CREATE TABLE features (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE CHECK(length(trim(slug)) > 0),
  title TEXT NOT NULL CHECK(length(trim(title)) > 0),
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','completed','cancelled')),
  archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE RESTRICT,
  title TEXT NOT NULL CHECK(length(trim(title)) > 0),
  scope TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL CHECK(kind IN ('pr','manual')),
  status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','in_progress','completed','cancelled')),
  assignee TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

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

CREATE TABLE pull_requests (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
  -- GitHub treats owner and repository names case-insensitively, so the unique
  -- constraint has to as well; otherwise the same PR can be attached twice.
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
  UNIQUE(owner, repository, number)
);

CREATE INDEX pull_requests_state_idx ON pull_requests(state, review_state, mergeability, stale);

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
