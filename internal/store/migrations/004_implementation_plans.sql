CREATE TABLE implementation_plans (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
  content TEXT NOT NULL CHECK(length(trim(content)) > 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
