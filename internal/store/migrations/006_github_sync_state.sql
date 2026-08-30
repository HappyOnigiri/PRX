CREATE TABLE github_sync_state (
  singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
  run_id TEXT NOT NULL DEFAULT '',
  last_attempt_unix INTEGER,
  last_completed_unix INTEGER,
  succeeded INTEGER NOT NULL DEFAULT 0 CHECK(succeeded >= 0),
  failed INTEGER NOT NULL DEFAULT 0 CHECK(failed >= 0),
  run_error TEXT NOT NULL DEFAULT ''
);

INSERT INTO github_sync_state(singleton) VALUES(1);
