-- task の status は常に手動で設定するものになる。自動を表す値はなくなり、その役割は
-- 導出側に移った。未完了の status は紐づく pull request に譲るため、格納された
-- 'not_started' は以前の 'auto' とまったく同じ結果を導く。自動だった行はすべて
-- 'not_started' に書き換える。
-- SQLite は CHECK 制約をその場で変更できないので、テーブルを作り直す。tasks の
-- 作り直しは dependencies・pull_requests・documents を巻き込む。外部キーが有効な状態の
-- DROP TABLE は暗黙の DELETE FROM を伴い、参照する行が残っている間は
-- ON DELETE RESTRICT がそれを拒むためである。以下のうち tasks 以外のテーブルは、
-- 元とまったく同じ定義で作り直す。
CREATE TEMP TABLE task_migration AS
SELECT
  id, feature_id, title, scope, status, assignee, created_at, updated_at,
  public_id
FROM tasks;

CREATE TEMP TABLE dependency_migration AS SELECT * FROM dependencies;
CREATE TEMP TABLE pull_request_migration AS SELECT * FROM pull_requests;
CREATE TEMP TABLE document_migration AS SELECT * FROM documents;

DROP TABLE documents;
DROP TABLE dependencies;
DROP TABLE pull_requests;
DROP TABLE tasks;

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE RESTRICT,
  title TEXT NOT NULL CHECK(length(trim(title)) > 0),
  scope TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'not_started' CHECK(status IN ('not_started','in_progress','completed','closed')),
  assignee TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  public_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX tasks_feature_idx ON tasks(feature_id, created_at, id);
CREATE INDEX tasks_status_idx ON tasks(status);
CREATE UNIQUE INDEX tasks_public_id_idx ON tasks(public_id);

INSERT INTO tasks (
  id, feature_id, title, scope, status, assignee, created_at, updated_at,
  public_id
)
SELECT
  id, feature_id, title, scope,
  CASE WHEN status = 'auto' THEN 'not_started' ELSE status END,
  assignee, created_at, updated_at, public_id
FROM task_migration;

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
  project_id TEXT REFERENCES projects(id) ON DELETE RESTRICT,
  feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
  task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
  kind TEXT NOT NULL CHECK(kind IN ('url','local_file','markdown')),
  title TEXT NOT NULL DEFAULT '',
  locator TEXT,
  content TEXT,
  is_implementation_plan INTEGER NOT NULL DEFAULT 0 CHECK(is_implementation_plan IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(
    (project_id IS NOT NULL AND feature_id IS NULL AND task_id IS NULL) OR
    (project_id IS NULL AND feature_id IS NOT NULL AND task_id IS NULL) OR
    (project_id IS NULL AND feature_id IS NULL AND task_id IS NOT NULL)
  ),
  CHECK(is_implementation_plan = 0 OR task_id IS NOT NULL),
  CHECK(
    (kind IN ('url','local_file') AND locator IS NOT NULL AND length(trim(locator)) > 0 AND content IS NULL) OR
    (kind = 'markdown' AND locator IS NULL AND content IS NOT NULL AND length(trim(content)) > 0)
  )
);

CREATE INDEX documents_project_idx ON documents(project_id, created_at, id);
CREATE INDEX documents_feature_idx ON documents(feature_id, created_at, id);
CREATE INDEX documents_task_idx ON documents(task_id, is_implementation_plan DESC, created_at, id);

-- 実装計画の upsert はこの部分インデックスを名前と述語で指定するため、
-- どちらも作り直しの前後で変えてはならない。
CREATE UNIQUE INDEX documents_one_plan_per_task_idx
ON documents(task_id) WHERE is_implementation_plan = 1;

INSERT INTO documents (
  id, project_id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
)
SELECT
  id, project_id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
FROM document_migration;

DROP TABLE task_migration;
DROP TABLE dependency_migration;
DROP TABLE pull_request_migration;
DROP TABLE document_migration;
