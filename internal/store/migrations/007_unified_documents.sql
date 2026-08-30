CREATE TABLE documents_new (
  id TEXT PRIMARY KEY,
  feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
  task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
  kind TEXT NOT NULL CHECK(kind IN ('url','local_file','markdown')),
  title TEXT NOT NULL DEFAULT '',
  locator TEXT,
  content TEXT,
  is_implementation_plan INTEGER NOT NULL DEFAULT 0 CHECK(is_implementation_plan IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK((feature_id IS NOT NULL AND task_id IS NULL) OR (feature_id IS NULL AND task_id IS NOT NULL)),
  CHECK(is_implementation_plan = 0 OR task_id IS NOT NULL),
  CHECK(
    (kind IN ('url','local_file') AND locator IS NOT NULL AND length(trim(locator)) > 0 AND content IS NULL) OR
    (kind = 'markdown' AND locator IS NULL AND content IS NOT NULL AND length(trim(content)) > 0)
  )
);

INSERT INTO documents_new (
  id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
)
SELECT
  id, feature_id, task_id,
  CASE kind WHEN 'markdown_path' THEN 'local_file' ELSE kind END,
  title, value, NULL, 0, created_at, created_at
FROM documents;

INSERT INTO documents_new (
  id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
)
SELECT
  lower(hex(randomblob(16))), NULL, task_id, 'markdown', '', NULL, content,
  1, created_at, updated_at
FROM implementation_plans;

DROP TABLE documents;
DROP TABLE implementation_plans;
ALTER TABLE documents_new RENAME TO documents;

CREATE INDEX documents_feature_idx ON documents(feature_id, created_at, id);
CREATE INDEX documents_task_idx ON documents(task_id, is_implementation_plan DESC, created_at, id);
CREATE UNIQUE INDEX documents_one_plan_per_task_idx
ON documents(task_id) WHERE is_implementation_plan = 1;
