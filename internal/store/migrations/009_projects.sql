-- Projects group features. The order below matters: projects has to exist
-- before features can reference it, and documents is rebuilt last so its new
-- foreign key finds every parent table already in place.
CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  public_id TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE CHECK(length(trim(slug)) > 0),
  title TEXT NOT NULL CHECK(length(trim(title)) > 0),
  description TEXT NOT NULL DEFAULT '',
  archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX projects_public_id_idx ON projects(public_id);

-- Migrations run inside a transaction, which cannot disable foreign keys, so
-- this column must stay nullable: SQLite only allows ADD COLUMN with a
-- reference when the default is NULL. A feature without a project is also the
-- normal case, because membership is optional.
ALTER TABLE features ADD COLUMN project_id TEXT REFERENCES projects(id) ON DELETE RESTRICT;

CREATE INDEX features_project_idx ON features(project_id, updated_at DESC, slug);

-- The entity CHECK has to be widened, and SQLite cannot alter it in place.
-- id_sequences has no children, so rebuilding it drags nothing along.
CREATE TABLE id_sequences_new (
  entity TEXT PRIMARY KEY CHECK(entity IN ('feature', 'task', 'project')),
  next_value INTEGER NOT NULL CHECK(next_value > 0)
);

INSERT INTO id_sequences_new (entity, next_value)
SELECT entity, next_value FROM id_sequences;

DROP TABLE id_sequences;
ALTER TABLE id_sequences_new RENAME TO id_sequences;

-- nextPublicID returns next_value - 1 after incrementing, so 1 hands out P-1.
INSERT INTO id_sequences (entity, next_value) VALUES ('project', 1);

-- documents gains a third parent. The exclusive CHECK becomes a three-way
-- choice, and the plan designation stays restricted to tasks.
CREATE TABLE documents_new (
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

INSERT INTO documents_new (
  id, project_id, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
)
SELECT
  id, NULL, feature_id, task_id, kind, title, locator, content,
  is_implementation_plan, created_at, updated_at
FROM documents;

DROP TABLE documents;
ALTER TABLE documents_new RENAME TO documents;

CREATE INDEX documents_project_idx ON documents(project_id, created_at, id);
CREATE INDEX documents_feature_idx ON documents(feature_id, created_at, id);
CREATE INDEX documents_task_idx ON documents(task_id, is_implementation_plan DESC, created_at, id);

-- The upsert of an implementation plan targets this partial index by name and
-- predicate, so both have to survive the rebuild unchanged.
CREATE UNIQUE INDEX documents_one_plan_per_task_idx
ON documents(task_id) WHERE is_implementation_plan = 1;
