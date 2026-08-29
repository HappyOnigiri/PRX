ALTER TABLE features ADD COLUMN public_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN public_id TEXT NOT NULL DEFAULT '';

WITH numbered AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS number
  FROM features
)
UPDATE features
SET public_id = 'F-' || (SELECT number FROM numbered WHERE numbered.id = features.id);

WITH numbered AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS number
  FROM tasks
)
UPDATE tasks
SET public_id = 'T-' || (SELECT number FROM numbered WHERE numbered.id = tasks.id);

CREATE UNIQUE INDEX features_public_id_idx ON features(public_id);
CREATE UNIQUE INDEX tasks_public_id_idx ON tasks(public_id);

CREATE TABLE id_sequences (
  entity TEXT PRIMARY KEY CHECK(entity IN ('feature', 'task')),
  next_value INTEGER NOT NULL CHECK(next_value > 0)
);

INSERT INTO id_sequences (entity, next_value)
SELECT 'feature', COALESCE(MAX(CAST(substr(public_id, 3) AS INTEGER)), 0) + 1
FROM features;

INSERT INTO id_sequences (entity, next_value)
SELECT 'task', COALESCE(MAX(CAST(substr(public_id, 3) AS INTEGER)), 0) + 1
FROM tasks;
