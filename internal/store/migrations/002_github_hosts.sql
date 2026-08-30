CREATE TABLE pull_requests_new (
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

INSERT INTO pull_requests_new (
  task_id, host, owner, repository, number, url, node_id, author, assignees_json,
  state, draft, review_state, mergeability, github_updated_at, last_synced_at,
  sync_error, stale
)
SELECT
  task_id, 'github.com', owner, repository, number, url, node_id, author, assignees_json,
  state, draft, review_state, mergeability, github_updated_at, last_synced_at,
  sync_error, stale
FROM pull_requests;

DROP TABLE pull_requests;
ALTER TABLE pull_requests_new RENAME TO pull_requests;

CREATE INDEX pull_requests_state_idx ON pull_requests(state, review_state, mergeability, stale);
CREATE INDEX pull_requests_repository_idx ON pull_requests(host, owner, repository, number);

CREATE TABLE github_repository_auth_cache (
  host TEXT NOT NULL COLLATE NOCASE,
  owner TEXT NOT NULL COLLATE NOCASE,
  repository TEXT NOT NULL COLLATE NOCASE,
  auth_method_id TEXT NOT NULL,
  last_succeeded_at TEXT NOT NULL,
  PRIMARY KEY(host, owner, repository)
);
