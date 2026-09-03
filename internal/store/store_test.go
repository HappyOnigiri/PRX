package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func openTestService(t *testing.T) (*store.Store, *app.Service) {
	t.Helper()
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, _ := githubprovider.NewFixtureProvider("demo")
	return database, app.New(database, provider)
}

func TestMigrationConstraintsAndRollback(t *testing.T) {
	database, _ := openTestService(t)
	ctx := context.Background()
	var migrations int
	if err := database.DB().
		QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).
		Scan(&migrations); err != nil ||
		migrations != 9 {
		t.Fatalf("migration count=%d err=%v", migrations, err)
	}
	var foreignKeys, journalMode int
	var journal string
	if err := database.DB().
		QueryRowContext(ctx, `PRAGMA foreign_keys`).
		Scan(&foreignKeys); err != nil ||
		foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d err=%v", foreignKeys, err)
	}
	if err := database.DB().QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("journal_mode=%q err=%v", journal, err)
	}
	_ = journalMode
	tx, err := database.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE should_rollback(id INTEGER);`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INVALID SQL`); err == nil {
		t.Fatal("expected invalid migration SQL")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var name string
	err = database.DB().QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE name='should_rollback'`).Scan(&name)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed migration left a table: name=%q err=%v", name, err)
	}
}

func TestGitHubAutoSyncClaimIsAtomicAcrossConnections(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "auto-sync.db")
	first, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	stores := []*store.Store{first, second}
	start := make(chan struct{})
	results := make(chan bool, len(stores))
	errorsFound := make(chan error, len(stores))
	var group sync.WaitGroup
	for index, database := range stores {
		group.Add(1)
		go func(index int, database *store.Store) {
			defer group.Done()
			<-start
			acquired, acquireErr := database.AcquireGitHubAutoSync(
				ctx, fmt.Sprintf("run-%d", index), now, now.Add(-600*time.Second).Unix(),
			)
			results <- acquired
			errorsFound <- acquireErr
		}(index, database)
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsFound)
	claimed := 0
	for acquired := range results {
		if acquired {
			claimed++
		}
	}
	for acquireErr := range errorsFound {
		if acquireErr != nil {
			t.Fatal(acquireErr)
		}
	}
	if claimed != 1 {
		t.Fatalf("claimed runs=%d, want 1", claimed)
	}
	state, err := first.GitHubSyncState(ctx)
	if err != nil || state.LastAttemptAt == nil || !state.LastAttemptAt.Equal(now) {
		t.Fatalf("sync state=%+v err=%v", state, err)
	}
}

// A concurrent refresh replaces the run identifier, and the run it displaced
// must learn that the state it would read back is no longer its own.
func TestCompletingADisplacedGitHubSyncRunReportsThatItNoLongerOwnsTheState(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "complete.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	acquired, err := database.AcquireGitHubAutoSync(ctx, "automatic", now, now.Unix())
	if err != nil || !acquired {
		t.Fatalf("acquired=%v err=%v", acquired, err)
	}
	if err := database.StartGitHubSync(ctx, "manual", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	recorded, err := database.CompleteGitHubSync(ctx, "automatic", now.Add(2*time.Second), 3, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if recorded {
		t.Fatal("the displaced run reported that it recorded its outcome")
	}
	recorded, err = database.CompleteGitHubSync(ctx, "manual", now.Add(3*time.Second), 1, 0, "")
	if err != nil || !recorded {
		t.Fatalf("current run recorded=%v err=%v", recorded, err)
	}
	state, err := database.GitHubSyncState(ctx)
	if err != nil || state.Succeeded != 1 {
		t.Fatalf("sync state=%+v err=%v", state, err)
	}
}

func TestPublicIDsAreTypedAndStorageIDsStayInternal(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "public-ids", "Public IDs", "", "")
	if err != nil {
		t.Fatal(err)
	}
	secondFeature, err := service.CreateFeature(ctx, "public-ids-2", "Public IDs 2", "", "")
	if err != nil {
		t.Fatal(err)
	}
	firstTask, err := service.CreateTask(ctx, feature.ID, "First", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	secondTask, err := service.CreateTask(ctx, feature.ID, "Second", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	if feature.ID != "F-1" || secondFeature.ID != "F-2" {
		t.Fatalf("feature IDs=%q,%q, want F-1,F-2", feature.ID, secondFeature.ID)
	}
	bySlug, err := database.GetFeatureBySlug(ctx, "public-ids")
	if err != nil || bySlug.ID != feature.ID {
		t.Fatalf("feature by slug=%+v, err=%v", bySlug, err)
	}
	if firstTask.ID != "T-1" || secondTask.ID != "T-2" {
		t.Fatalf("task IDs=%q,%q, want T-1,T-2", firstTask.ID, secondTask.ID)
	}
	if feature.StorageID == "" || feature.StorageID == feature.ID ||
		firstTask.StorageID == "" || firstTask.StorageID == firstTask.ID {
		t.Fatalf("storage IDs were not kept separate: feature=%+v task=%+v", feature, firstTask)
	}
	if firstTask.FeatureID != feature.ID || firstTask.StorageFeatureID != feature.StorageID {
		t.Fatalf(
			"task parent IDs=%q,%q, want public=%q storage=%q",
			firstTask.FeatureID,
			firstTask.StorageFeatureID,
			feature.ID,
			feature.StorageID,
		)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Features) != 2 || len(snapshot.Tasks) != 2 {
		t.Fatalf("snapshot counts features=%d tasks=%d", len(snapshot.Features), len(snapshot.Tasks))
	}
	for _, task := range snapshot.Tasks {
		if task.ID != firstTask.ID && task.ID != secondTask.ID {
			t.Fatalf("snapshot exposed unexpected task ID %q", task.ID)
		}
		if task.FeatureID != feature.ID {
			t.Fatalf("snapshot exposed unexpected feature ID %q", task.FeatureID)
		}
	}
}

func TestLegacyTaskStatusMigrationPreservesRelatedRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	legacy.SetMaxOpenConns(1)
	legacySchema := []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE features (
      id TEXT PRIMARY KEY,
      slug TEXT NOT NULL UNIQUE CHECK(length(trim(slug)) > 0),
      title TEXT NOT NULL CHECK(length(trim(title)) > 0),
      description TEXT NOT NULL DEFAULT '',
      status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','completed','cancelled')),
      archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)),
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )`,
		`CREATE TABLE tasks (
      id TEXT PRIMARY KEY,
      feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE RESTRICT,
      title TEXT NOT NULL CHECK(length(trim(title)) > 0),
      scope TEXT NOT NULL DEFAULT '',
      kind TEXT NOT NULL CHECK(kind IN ('pr','manual')),
      status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','in_progress','completed','cancelled')),
      assignee TEXT NOT NULL DEFAULT '',
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )`,
		`CREATE TABLE dependencies (
      blocker_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
      blocked_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
      created_at TEXT NOT NULL,
      PRIMARY KEY(blocker_task_id, blocked_task_id),
      CHECK(blocker_task_id <> blocked_task_id)
    )`,
		`CREATE TABLE pull_requests (
      task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
      owner TEXT NOT NULL COLLATE NOCASE,
      repository TEXT NOT NULL COLLATE NOCASE,
      number INTEGER NOT NULL CHECK(number > 0),
      url TEXT NOT NULL,
      node_id TEXT NOT NULL DEFAULT '',
      author TEXT NOT NULL DEFAULT '',
      assignees_json TEXT NOT NULL DEFAULT '[]',
      state TEXT NOT NULL DEFAULT 'unknown' CHECK(state IN ('open','closed','merged','unknown')),
      draft INTEGER NOT NULL DEFAULT 0 CHECK(draft IN (0,1)),
      review_state TEXT NOT NULL DEFAULT 'unknown' CHECK(
        review_state IN ('none','required','approved','changes_requested','unknown')),
      mergeability TEXT NOT NULL DEFAULT 'unknown' CHECK(mergeability IN ('mergeable','conflicting','unknown')),
      github_updated_at TEXT,
      last_synced_at TEXT,
      sync_error TEXT NOT NULL DEFAULT '',
      stale INTEGER NOT NULL DEFAULT 1 CHECK(stale IN (0,1)),
      UNIQUE(owner, repository, number)
    )`,
		`CREATE TABLE documents (
      id TEXT PRIMARY KEY,
      feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
      task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
      kind TEXT NOT NULL CHECK(kind IN ('url','markdown_path')),
      title TEXT NOT NULL DEFAULT '',
      value TEXT NOT NULL CHECK(length(trim(value)) > 0),
      created_at TEXT NOT NULL,
      CHECK((feature_id IS NOT NULL AND task_id IS NULL) OR (feature_id IS NULL AND task_id IS NOT NULL))
    )`,
	}
	for _, statement := range legacySchema {
		if _, err := legacy.ExecContext(ctx, statement); err != nil {
			_ = legacy.Close()
			t.Fatal(err)
		}
	}
	const timestamp = "2026-08-30T00:00:00Z"
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES(1, ?)`,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO features(id, slug, title, created_at, updated_at) VALUES('feature', 'legacy', 'Legacy', ?, ?)`,
		timestamp,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	// A person chose each of these statuses, so the automatic mode must not
	// claim them while the default active feature above moves onto derivation.
	// They are created later so the migrated public IDs keep their order.
	const later = "2026-08-31T00:00:00Z"
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO features(id, slug, title, status, created_at, updated_at)
      VALUES('paused-feature', 'legacy-paused', 'Legacy paused', 'paused', ?, ?),
            ('completed-feature', 'legacy-completed', 'Legacy completed', 'completed', ?, ?),
            ('cancelled-feature', 'legacy-cancelled', 'Legacy cancelled', 'cancelled', ?, ?)`,
		later, later, later, later, later, later,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	legacyStatuses := []struct {
		id, kind, status string
	}{
		{id: "manual-planned", kind: "manual", status: "planned"},
		{id: "manual-progress", kind: "manual", status: "in_progress"},
		{id: "manual-completed", kind: "manual", status: "completed"},
		{id: "manual-cancelled", kind: "manual", status: "cancelled"},
		{id: "pr-planned", kind: "pr", status: "planned"},
		{id: "pr-progress", kind: "pr", status: "in_progress"},
		{id: "pr-completed", kind: "pr", status: "completed"},
		{id: "pr-cancelled", kind: "pr", status: "cancelled"},
	}
	for _, legacyTask := range legacyStatuses {
		if _, err := legacy.ExecContext(
			ctx,
			`INSERT INTO tasks(id, feature_id, title, kind, status, created_at, updated_at) VALUES(?, 'feature', ?, ?, ?, ?, ?)`,
			legacyTask.id,
			legacyTask.id,
			legacyTask.kind,
			legacyTask.status,
			timestamp,
			timestamp,
		); err != nil {
			_ = legacy.Close()
			t.Fatal(err)
		}
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO dependencies(blocker_task_id, blocked_task_id, created_at) VALUES('manual-planned', 'pr-planned', ?)`,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO pull_requests(task_id, owner, repository, number, url)
      VALUES('pr-planned', 'acme', 'api', 7, 'https://github.com/acme/api/pull/7')`,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO documents(id, task_id, kind, value, created_at)
      VALUES('document', 'pr-planned', 'url', 'https://example.com', ?)`,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	rows, err := database.DB().QueryContext(ctx, `SELECT id, status FROM tasks ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	})
	wantStatuses := map[string]string{
		"manual-cancelled": "closed", "manual-completed": "completed", "manual-planned": "auto",
		"manual-progress": "in_progress", "pr-cancelled": "closed", "pr-completed": "completed",
		"pr-planned": "auto", "pr-progress": "auto",
	}
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			t.Fatal(err)
		}
		if status != wantStatuses[id] {
			t.Fatalf("task %s status=%q, want %q", id, status, wantStatuses[id])
		}
		delete(wantStatuses, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(wantStatuses) != 0 {
		t.Fatalf("migration omitted tasks: %v", wantStatuses)
	}
	var dependencies, pullRequests, documents int
	if err := database.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM dependencies`).Scan(&dependencies); err != nil {
		t.Fatal(err)
	}
	if err := database.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests`).Scan(&pullRequests); err != nil {
		t.Fatal(err)
	}
	if err := database.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&documents); err != nil {
		t.Fatal(err)
	}
	if dependencies != 1 || pullRequests != 1 || documents != 1 {
		t.Fatalf("related rows=(%d, %d, %d), want (1, 1, 1)", dependencies, pullRequests, documents)
	}
	if errorsFound := database.Validate(ctx); len(errorsFound) > 0 {
		t.Fatalf("migrated database validation errors: %v", errorsFound)
	}
}

func TestMigrationRepairsConflictingBranchVersions(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "legacy-branch.db")
	legacy, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	legacy.SetMaxOpenConns(1)
	legacySchema := []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE features (
      id TEXT PRIMARY KEY,
      slug TEXT NOT NULL UNIQUE CHECK(length(trim(slug)) > 0),
      title TEXT NOT NULL CHECK(length(trim(title)) > 0),
      description TEXT NOT NULL DEFAULT '',
      status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','paused','completed','cancelled')),
      archived INTEGER NOT NULL DEFAULT 0 CHECK(archived IN (0,1)),
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )`,
		`CREATE TABLE tasks (
      id TEXT PRIMARY KEY,
      feature_id TEXT NOT NULL REFERENCES features(id) ON DELETE RESTRICT,
      title TEXT NOT NULL CHECK(length(trim(title)) > 0),
      scope TEXT NOT NULL DEFAULT '',
      kind TEXT NOT NULL CHECK(kind IN ('pr','manual')),
      status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','in_progress','completed','cancelled')),
      assignee TEXT NOT NULL DEFAULT '',
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )`,
		`CREATE TABLE dependencies (
      blocker_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
      blocked_task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
      created_at TEXT NOT NULL,
      PRIMARY KEY(blocker_task_id, blocked_task_id),
      CHECK(blocker_task_id <> blocked_task_id)
    )`,
		`CREATE TABLE pull_requests (
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
      review_state TEXT NOT NULL DEFAULT 'unknown' CHECK(
        review_state IN ('none','required','approved','changes_requested','unknown')),
      mergeability TEXT NOT NULL DEFAULT 'unknown' CHECK(mergeability IN ('mergeable','conflicting','unknown')),
      github_updated_at TEXT,
      last_synced_at TEXT,
      sync_error TEXT NOT NULL DEFAULT '',
      stale INTEGER NOT NULL DEFAULT 1 CHECK(stale IN (0,1)),
      UNIQUE(host, owner, repository, number)
    )`,
		`CREATE TABLE documents (
      id TEXT PRIMARY KEY,
      feature_id TEXT REFERENCES features(id) ON DELETE RESTRICT,
      task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
      kind TEXT NOT NULL CHECK(kind IN ('url','markdown_path')),
      title TEXT NOT NULL DEFAULT '',
      value TEXT NOT NULL CHECK(length(trim(value)) > 0),
      created_at TEXT NOT NULL,
      CHECK((feature_id IS NOT NULL AND task_id IS NULL) OR (feature_id IS NULL AND task_id IS NOT NULL))
    )`,
		`CREATE TABLE github_repository_auth_cache (
      host TEXT NOT NULL COLLATE NOCASE,
      owner TEXT NOT NULL COLLATE NOCASE,
      repository TEXT NOT NULL COLLATE NOCASE,
      auth_method_id TEXT NOT NULL,
      last_succeeded_at TEXT NOT NULL,
      PRIMARY KEY(host, owner, repository)
    )`,
		`CREATE TABLE implementation_plans (
      task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
      content TEXT NOT NULL CHECK(length(trim(content)) > 0),
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL
    )`,
	}
	for _, statement := range legacySchema {
		if _, err := legacy.ExecContext(ctx, statement); err != nil {
			_ = legacy.Close()
			t.Fatal(err)
		}
	}
	const timestamp = "2026-08-30T00:00:00Z"
	for _, version := range []int{1, 2, 3} {
		if _, err := legacy.ExecContext(
			ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			version,
			timestamp,
		); err != nil {
			_ = legacy.Close()
			t.Fatal(err)
		}
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO features(id, slug, title, created_at, updated_at) VALUES('feature', 'legacy', 'Legacy', ?, ?)`,
		timestamp,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	// A person chose each of these statuses, so the automatic mode must not
	// claim them while the default active feature above moves onto derivation.
	// They are created later so the migrated public IDs keep their order.
	const later = "2026-08-31T00:00:00Z"
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO features(id, slug, title, status, created_at, updated_at)
      VALUES('paused-feature', 'legacy-paused', 'Legacy paused', 'paused', ?, ?),
            ('completed-feature', 'legacy-completed', 'Legacy completed', 'completed', ?, ?),
            ('cancelled-feature', 'legacy-cancelled', 'Legacy cancelled', 'cancelled', ?, ?)`,
		later, later, later, later, later, later,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO tasks(id, feature_id, title, kind, status, created_at, updated_at)
      VALUES('task', 'feature', 'Legacy task', 'pr', 'planned', ?, ?)`,
		timestamp,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO pull_requests(task_id, host, owner, repository, number, url)
      VALUES('task', 'ghe.example.com', 'acme', 'api', 7, 'https://ghe.example.com/acme/api/pull/7')`,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO implementation_plans(task_id, content, created_at, updated_at)
      VALUES('task', '# Keep this plan', ?, ?)`,
		timestamp,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(
		ctx,
		`INSERT INTO documents(id, feature_id, task_id, kind, title, value, created_at)
      VALUES('feature-document', 'feature', NULL, 'markdown_path', 'Feature notes', 'docs/feature.md', ?),
            ('task-document', NULL, 'task', 'url', 'Task runbook', 'https://example.com/runbook', ?)`,
		timestamp,
		timestamp,
	); err != nil {
		_ = legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := store.Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	var migrationCount int
	if err := database.DB().
		QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).
		Scan(&migrationCount); err != nil || migrationCount != 9 {
		t.Fatalf("migration count=%d err=%v", migrationCount, err)
	}
	var status string
	if err := database.DB().
		QueryRowContext(ctx, `SELECT status FROM tasks WHERE id='task'`).
		Scan(&status); err != nil || status != "auto" {
		t.Fatalf("task status=%q err=%v, want auto", status, err)
	}
	for slug, want := range map[string]domain.FeatureStatus{
		"legacy":           domain.FeatureStatusAuto,
		"legacy-paused":    domain.FeatureStatusPaused,
		"legacy-completed": domain.FeatureStatusCompleted,
		"legacy-cancelled": domain.FeatureStatusCancelled,
	} {
		feature, err := database.GetFeatureBySlug(ctx, slug)
		if err != nil || feature.Status != want {
			t.Fatalf("migrated feature %q status=%q err=%v, want %q", slug, feature.Status, err, want)
		}
	}
	plan, err := database.GetImplementationPlan(ctx, "T-1")
	if err != nil || plan.Content != "# Keep this plan" {
		t.Fatalf("implementation plan=%+v err=%v", plan, err)
	}
	pullRequest, err := database.GetPullRequest(ctx, "T-1")
	if err != nil || pullRequest.Host != "ghe.example.com" {
		t.Fatalf("pull request=%+v err=%v, want host preserved", pullRequest, err)
	}
	assertMigratedDocuments(t, ctx, database, timestamp)
}

func assertMigratedDocuments(
	t *testing.T,
	ctx context.Context,
	database *store.Store,
	timestamp string,
) {
	t.Helper()
	created, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := database.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Documents) != 3 {
		t.Fatalf("documents=%+v, want the two legacy documents and the migrated plan", snapshot.Documents)
	}
	byID := make(map[string]domain.Document, len(snapshot.Documents))
	plans := make([]domain.Document, 0, 1)
	for _, document := range snapshot.Documents {
		byID[document.ID] = document
		if document.IsImplementationPlan {
			plans = append(plans, document)
		}
	}
	featureDocument, ok := byID["feature-document"]
	if !ok {
		t.Fatalf("documents=%+v, want the legacy feature document to keep its identifier", snapshot.Documents)
	}
	if featureDocument.Kind != domain.DocumentKindLocalFile ||
		featureDocument.Locator != "docs/feature.md" ||
		featureDocument.Title != "Feature notes" ||
		featureDocument.FeatureID != "F-1" ||
		featureDocument.TaskID != "" ||
		featureDocument.IsImplementationPlan ||
		!featureDocument.CreatedAt.Equal(created) {
		t.Fatalf("feature document=%+v, want a local file document owned by F-1", featureDocument)
	}
	taskDocument, ok := byID["task-document"]
	if !ok {
		t.Fatalf("documents=%+v, want the legacy task document to keep its identifier", snapshot.Documents)
	}
	if taskDocument.Kind != domain.DocumentKindURL ||
		taskDocument.Locator != "https://example.com/runbook" ||
		taskDocument.TaskID != "T-1" ||
		taskDocument.FeatureID != "" ||
		taskDocument.IsImplementationPlan ||
		!taskDocument.CreatedAt.Equal(created) {
		t.Fatalf("task document=%+v, want a URL document owned by T-1", taskDocument)
	}
	if len(plans) != 1 {
		t.Fatalf("implementation plan documents=%+v, want exactly one", plans)
	}
	if plans[0].Kind != domain.DocumentKindMarkdown ||
		plans[0].TaskID != "T-1" ||
		plans[0].Locator != "" {
		t.Fatalf("implementation plan document=%+v, want a Markdown document owned by T-1", plans[0])
	}
}

func TestMigrationAddsGitHubHostAndHostScopedUniqueness(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "legacy.db")
	database, err := store.Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	feature, err := database.CreateFeature(ctx, "legacy", "Legacy", "", "")
	if err != nil {
		t.Fatal(err)
	}
	firstTask, err := database.CreateTask(ctx, feature.ID, "GitHub.com PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	secondTask, err := database.CreateTask(ctx, feature.ID, "GHE PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	legacyTable := `CREATE TABLE pull_requests (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE RESTRICT,
  owner TEXT NOT NULL COLLATE NOCASE,
  repository TEXT NOT NULL COLLATE NOCASE,
  number INTEGER NOT NULL CHECK(number > 0),
  url TEXT NOT NULL,
  node_id TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT '',
  assignees_json TEXT NOT NULL DEFAULT '[]',
  state TEXT NOT NULL DEFAULT 'unknown' CHECK(state IN ('open','closed','merged','unknown')),
  draft INTEGER NOT NULL DEFAULT 0 CHECK(draft IN (0,1)),
	 review_state TEXT NOT NULL DEFAULT 'unknown' CHECK(review_state IN (
	   'none','required','approved','changes_requested','unknown')),
  mergeability TEXT NOT NULL DEFAULT 'unknown' CHECK(mergeability IN ('mergeable','conflicting','unknown')),
  github_updated_at TEXT,
  last_synced_at TEXT,
  sync_error TEXT NOT NULL DEFAULT '',
  stale INTEGER NOT NULL DEFAULT 1 CHECK(stale IN (0,1)),
  UNIQUE(owner, repository, number)
)`
	if _, err := database.DB().ExecContext(ctx, `DROP TABLE github_repository_auth_cache`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(ctx, `DROP TABLE pull_requests`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(ctx, legacyTable); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(
		ctx,
		`CREATE INDEX pull_requests_state_idx ON pull_requests(state, review_state, mergeability, stale)`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(
		ctx,
		`INSERT INTO pull_requests (task_id, owner, repository, number, url,
			state, review_state, mergeability, stale)
			VALUES (?, ?, ?, ?, ?, 'open', 'none', 'unknown', 1)`,
		firstTask.StorageID,
		"Acme",
		"API",
		42,
		"https://github.com/Acme/API/pull/42",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(ctx, `DELETE FROM schema_migrations WHERE version=2`); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = store.Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	legacy, err := database.GetPullRequest(ctx, firstTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Host != "github.com" {
		t.Fatalf("migrated host=%q, want github.com", legacy.Host)
	}
	if _, err := database.UpsertPullRequest(ctx, domain.PullRequest{
		TaskID: secondTask.ID, Host: "ghe.example.com", Owner: "Acme", Repository: "API", Number: 42,
		URL: "https://ghe.example.com/Acme/API/pull/42", State: domain.PullRequestStateUnknown,
		ReviewState: domain.ReviewStateUnknown, Mergeability: domain.MergeabilityUnknown, Stale: true,
	}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.DB().
		QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests WHERE owner='Acme' AND repository='API' AND number=42`).
		Scan(&count); err != nil ||
		count != 2 {
		t.Fatalf("host-scoped PR count=%d err=%v", count, err)
	}
}

func TestGitHubRepositoryAuthCacheRoundTrip(t *testing.T) {
	database, _ := openTestService(t)
	ctx := context.Background()
	if _, found, err := database.GetGitHubRepositoryAuthCache(ctx, "github.com", "acme", "api"); err != nil || found {
		t.Fatalf("missing cache found=%v err=%v", found, err)
	}
	if err := database.UpsertGitHubRepositoryAuthCache(ctx, "github.com", "Acme", "API", "first"); err != nil {
		t.Fatal(err)
	}
	if err := database.UpsertGitHubRepositoryAuthCache(ctx, "github.com", "acme", "api", "second"); err != nil {
		t.Fatal(err)
	}
	if method, found, err := database.GetGitHubRepositoryAuthCache(
		ctx,
		"GITHUB.COM",
		"ACME",
		"api",
	); err != nil || !found ||
		method != "second" {
		t.Fatalf("cache method=%q found=%v err=%v", method, found, err)
	}
	if err := database.DeleteGitHubRepositoryAuthCache(ctx, "github.com", "acme", "api"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := database.GetGitHubRepositoryAuthCache(ctx, "github.com", "acme", "api"); err != nil || found {
		t.Fatalf("deleted cache found=%v err=%v", found, err)
	}
}

func TestCycleDuplicateAndSafeDeletion(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "delivery", "Delivery", "", "")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := service.CreateTask(ctx, feature.ID, "A", "", domain.TaskKindPR, "")
	b, _ := service.CreateTask(ctx, feature.ID, "B", "", domain.TaskKindPR, "")
	c, _ := service.CreateTask(ctx, feature.ID, "C", "", domain.TaskKindPR, "")
	if _, err := service.AddDependency(ctx, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(ctx, b.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(
		ctx,
		a.ID,
		b.ID,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeDuplicateDependency {
		t.Fatalf("duplicate code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.AddDependency(ctx, c.ID, a.ID); domain.ErrorCode(err) != domain.DomainErrorCodeCycle {
		t.Fatalf("cycle code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DeleteTask(ctx, b.ID, false); domain.ErrorCode(err) != domain.DomainErrorCodeReferencesExist {
		t.Fatalf("safe deletion code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DeleteTask(ctx, b.ID, true); err != nil {
		t.Fatal(err)
	}
	if errorsFound := service.Validate(ctx); len(errorsFound) > 0 {
		t.Fatalf("validation after cascade: %v", errorsFound)
	}
}

func TestDuplicatePullRequestAndConcurrentWriters(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, _ := service.CreateFeature(ctx, "concurrency", "Concurrency", "", "")
	a, _ := service.CreateTask(ctx, feature.ID, "A", "", domain.TaskKindPR, "")
	b, _ := service.CreateTask(ctx, feature.ID, "B", "", domain.TaskKindPR, "")
	if _, err := service.AttachPullRequest(ctx, a.ID, "https://github.com/acme/api/pull/42"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(
		ctx,
		b.ID,
		"https://github.com/acme/api/pull/42",
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeDuplicatePullRequest {
		t.Fatalf("duplicate PR code=%s err=%v", domain.ErrorCode(err), err)
	}
	const writers = 24
	var wait sync.WaitGroup
	errorsCh := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, err := service.CreateTask(
				ctx,
				feature.ID,
				fmt.Sprintf("task-%02d", index),
				"",
				domain.TaskKindManual,
				"",
			)
			errorsCh <- err
		}(i)
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent write: %v", err)
		}
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Tasks) != writers+2 {
		t.Fatalf("tasks=%d want=%d", len(snapshot.Tasks), writers+2)
	}
}

func TestValidateReportsCorruption(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "corrupt.db")
	database, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	provider, _ := githubprovider.NewFixtureProvider("demo")
	service := app.New(database, provider)
	feature, err := service.CreateFeature(ctx, "corruption", "Corruption", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		if _, err := service.CreateTask(
			ctx,
			feature.ID,
			fmt.Sprintf("task-%03d", i),
			strings.Repeat("x", 256),
			domain.TaskKindManual,
			"",
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Overwrite payload pages while leaving the header and the schema page
	// intact, so the database still opens but no longer passes integrity_check.
	const pageSize = 4096
	if len(body) < 4*pageSize {
		t.Fatalf("database is only %d bytes; expected several pages", len(body))
	}
	for i := 2 * pageSize; i < len(body); i++ {
		body[i] ^= 0xff
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	corrupted, err := store.Open(ctx, path)
	if err != nil {
		t.Skipf("corrupted database could not be opened: %v", err)
	}
	t.Cleanup(func() { _ = corrupted.Close() })
	errorsFound := corrupted.Validate(ctx)
	reported := false
	for _, line := range errorsFound {
		if strings.Contains(line, "in database main") {
			reported = true
			break
		}
	}
	if !reported {
		t.Fatalf("Validate did not report the integrity_check rows: %q", errorsFound)
	}
}

func TestUpdateClearsFieldsWhenExplicitlyEmpty(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "clearing", "Clearing", "initial description", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Task", "initial scope", domain.TaskKindManual, "mona")
	if err != nil {
		t.Fatal(err)
	}

	empty := ""
	updatedTask, err := service.UpdateTask(ctx, task.ID, nil, &empty, nil, &empty)
	if err != nil {
		t.Fatal(err)
	}
	if updatedTask.Scope != "" || updatedTask.Assignee != "" {
		t.Fatalf("scope=%q assignee=%q, want both cleared", updatedTask.Scope, updatedTask.Assignee)
	}
	if updatedTask.Title != "Task" {
		t.Fatalf("omitted title was changed to %q", updatedTask.Title)
	}

	updatedFeature, err := service.UpdateFeature(ctx, feature.ID, nil, nil, &empty, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updatedFeature.Description != "" {
		t.Fatalf("description=%q, want cleared", updatedFeature.Description)
	}
	if updatedFeature.Title != "Clearing" {
		t.Fatalf("omitted title was changed to %q", updatedFeature.Title)
	}
}

func TestArchiveRoundTrip(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "archivable", "Archivable", "", "")
	if err != nil {
		t.Fatal(err)
	}
	archived, unarchived := true, false
	updated, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, &archived, nil)
	if err != nil || !updated.Archived {
		t.Fatalf("archive: archived=%v err=%v", updated.Archived, err)
	}
	updated, err = service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, &unarchived, nil)
	if err != nil || updated.Archived {
		t.Fatalf("unarchive: archived=%v err=%v", updated.Archived, err)
	}
}

// The stored status lives in two columns, so every status has to survive a
// write and a read, and the automatic mode has to normalize the status column
// whose CHECK constraint does not know the automatic value.
func TestFeatureStatusRoundTripKeepsTheStoredColumnsConsistent(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "status-columns", "Status columns", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if feature.Status != domain.FeatureStatusAuto {
		t.Fatalf("created status=%q, want %q", feature.Status, domain.FeatureStatusAuto)
	}
	for _, test := range []struct {
		status     domain.FeatureStatus
		wantColumn string
		wantAuto   int64
	}{
		{domain.FeatureStatusPaused, "paused", 0},
		{domain.FeatureStatusCompleted, "completed", 0},
		{domain.FeatureStatusCancelled, "cancelled", 0},
		{domain.FeatureStatusActive, "active", 0},
		{domain.FeatureStatusAuto, "active", 1},
	} {
		t.Run(string(test.status), func(t *testing.T) {
			status := test.status
			updated, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, &status, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != test.status {
				t.Fatalf("updated status=%q, want %q", updated.Status, test.status)
			}
			reread, err := database.GetFeature(ctx, feature.ID)
			if err != nil || reread.Status != test.status {
				t.Fatalf("reread status=%q err=%v, want %q", reread.Status, err, test.status)
			}
			var column string
			var auto int64
			if err := database.DB().QueryRowContext(
				ctx,
				`SELECT status, status_auto FROM features WHERE public_id = ?`,
				feature.ID,
			).Scan(&column, &auto); err != nil {
				t.Fatal(err)
			}
			if column != test.wantColumn || auto != test.wantAuto {
				t.Fatalf("stored status=%q status_auto=%d, want %q and %d",
					column, auto, test.wantColumn, test.wantAuto)
			}
		})
	}
}

func TestConcurrentDependencyWritesDoNotLock(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "contention", "Contention", "", "")
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := service.CreateTask(ctx, feature.ID, "Blocker", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	const writers = 16
	blocked := make([]domain.Task, writers)
	for i := range blocked {
		task, err := service.CreateTask(ctx, feature.ID, fmt.Sprintf("blocked-%02d", i), "", domain.TaskKindManual, "")
		if err != nil {
			t.Fatal(err)
		}
		blocked[i] = task
	}
	var wait sync.WaitGroup
	errorsCh := make(chan error, writers)
	for _, task := range blocked {
		wait.Add(1)
		go func(blockedID string) {
			defer wait.Done()
			_, err := service.AddDependency(ctx, blocker.ID, blockedID)
			errorsCh <- err
		}(task.ID)
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		// Domain errors are acceptable outcomes; a raw lock error is not.
		if err != nil && domain.ErrorCode(err) == "" {
			t.Fatalf("concurrent dependency write: %v", err)
		}
		if err != nil && strings.Contains(err.Error(), "locked") {
			t.Fatalf("concurrent dependency write hit a lock error: %v", err)
		}
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Dependencies) != writers {
		t.Fatalf("dependencies=%d, want %d", len(snapshot.Dependencies), writers)
	}
}

func TestInitializeDemoCreatesCompleteShowcase(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	markdownPath := filepath.Join(t.TempDir(), "walkthrough.md")
	if err := service.InitializeDemo(ctx, markdownPath); err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Features) != 4 || len(snapshot.Tasks) != 120 {
		t.Fatalf("features=%d tasks=%d, want 4 and 120", len(snapshot.Features), len(snapshot.Tasks))
	}
	statuses := map[domain.FeatureStatus]bool{}
	displayStatuses := map[domain.FeatureStatus]bool{}
	featuresBySlug := map[string]domain.Feature{}
	for _, feature := range snapshot.Features {
		statuses[feature.Status] = true
		displayStatuses[feature.DisplayStatus] = true
		featuresBySlug[feature.Slug] = feature
	}
	if !statuses[domain.FeatureStatusAuto] {
		t.Error("no demo feature keeps the automatic status")
	}
	for _, status := range []domain.FeatureStatus{
		domain.FeatureStatusActive,
		domain.FeatureStatusPaused,
		domain.FeatureStatusCompleted,
		domain.FeatureStatusCancelled,
	} {
		if !displayStatuses[status] {
			t.Errorf("missing feature display status %q", status)
		}
	}
	// The 100-task program reaches completion from its merged pull requests
	// alone, so the demo exercises the derivation rather than a manual status.
	completedProgram := featuresBySlug["completed-program"]
	if completedProgram.Status != domain.FeatureStatusAuto ||
		completedProgram.DisplayStatus != domain.FeatureStatusCompleted ||
		completedProgram.FinishedCount != completedProgram.TaskCount {
		t.Errorf("completed program status=%+v", completedProgram)
	}
	displayStates := map[domain.TaskDisplayState]bool{}
	for _, task := range snapshot.Tasks {
		displayStates[task.DisplayState] = true
	}
	for _, state := range []domain.TaskDisplayState{
		domain.TaskDisplayStateNotStarted,
		domain.TaskDisplayStateInProgress,
		domain.TaskDisplayStateCompleted,
		domain.TaskDisplayStateClosed,
		domain.TaskDisplayStateMerged,
		domain.TaskDisplayStateDraft,
		domain.TaskDisplayStateConflict,
		domain.TaskDisplayStateChangesRequested,
		domain.TaskDisplayStateApproved,
		domain.TaskDisplayStateReviewWaiting,
		domain.TaskDisplayStateOpen,
		domain.TaskDisplayStateUnknown,
	} {
		if !displayStates[state] {
			t.Errorf("missing display state %q", state)
		}
	}
	if len(snapshot.ReadyTasks) == 0 || len(snapshot.ReviewWaitingTasks) == 0 ||
		len(snapshot.ConflictTasks) == 0 || len(snapshot.StaleTasks) == 0 {
		t.Errorf("demo queues are incomplete: ready=%d reviews=%d conflicts=%d stale=%d",
			len(snapshot.ReadyTasks), len(snapshot.ReviewWaitingTasks),
			len(snapshot.ConflictTasks), len(snapshot.StaleTasks))
	}
	references, plans := 0, 0
	for _, document := range snapshot.Documents {
		if document.IsImplementationPlan {
			plans++
			continue
		}
		references++
	}
	// Two feature documents plus the shared charter on the active project.
	if references != 3 || plans != 1 {
		t.Errorf("documents: references=%d plans=%d, want 3 and 1", references, plans)
	}
	if !featuresBySlug["cancelled-experiment"].Archived {
		t.Error("cancelled feature is not archived")
	}
	featureByTask := map[string]string{}
	var plannedTaskID string
	for _, task := range snapshot.Tasks {
		featureByTask[task.ID] = task.FeatureID
		if task.Title == "Design dependency policy" {
			plannedTaskID = task.ID
		}
	}
	plan, err := service.GetImplementationPlan(ctx, plannedTaskID)
	if err != nil || plan.Content == "" {
		t.Errorf("demo implementation plan = %+v, err=%v", plan, err)
	}
	completedPRs := 0
	for _, pr := range snapshot.PullRequests {
		if featureByTask[pr.TaskID] != featuresBySlug["completed-program"].ID {
			continue
		}
		completedPRs++
		if pr.State != domain.PullRequestStateMerged {
			t.Errorf("completed PR %s state=%q", pr.TaskID, pr.State)
		}
	}
	if completedPRs != 100 {
		t.Errorf("completed PRs=%d, want 100", completedPRs)
	}
	for _, document := range snapshot.Documents {
		if document.Kind != domain.DocumentKindLocalFile {
			continue
		}
		body, readErr := service.ReadDocumentContent(ctx, document.ID)
		if readErr != nil || !strings.Contains(body, "PRX demo walkthrough") {
			t.Errorf("Markdown preview=%q, err=%v", body, readErr)
		}
	}
	if issues := service.Validate(ctx); len(issues) != 0 {
		t.Errorf("validation issues: %v", issues)
	}
}

// The WebUI synchronizes on its first render, so the demo only shows what it
// promises when a synchronization reproduces the states rather than replacing
// them with the generated preset.
func TestDemoSurvivesSynchronization(t *testing.T) {
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	temporaryRoot := t.TempDir()
	fixturePath := filepath.Join(temporaryRoot, "github-fixture.json")
	if err := app.WriteDemoFixture(fixturePath); err != nil {
		t.Fatal(err)
	}
	provider, err := githubprovider.NewFixtureProvider(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	configStore, err := config.NewStore(filepath.Join(temporaryRoot, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, provider, configStore)
	ctx := context.Background()
	if err := service.InitializeDemo(ctx, filepath.Join(temporaryRoot, "walkthrough.md")); err != nil {
		t.Fatal(err)
	}
	before, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ran, _, err := service.SyncIfDue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("the first synchronization did not run, so this test proves nothing")
	}
	after, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	displayStates := func(snapshot domain.Snapshot) map[domain.TaskDisplayState]int {
		counts := map[domain.TaskDisplayState]int{}
		for _, task := range snapshot.Tasks {
			counts[task.DisplayState]++
		}
		return counts
	}
	beforeStates, afterStates := displayStates(before), displayStates(after)
	for state, count := range beforeStates {
		if afterStates[state] != count {
			t.Errorf("display state %q: %d tasks before the synchronization, %d after",
				state, count, afterStates[state])
		}
	}
	if len(afterStates) != len(beforeStates) {
		t.Errorf("display states=%d after the synchronization, want %d", len(afterStates), len(beforeStates))
	}
	if len(after.StaleTasks) != len(before.StaleTasks) {
		t.Errorf("stale tasks=%d after the synchronization, want %d",
			len(after.StaleTasks), len(before.StaleTasks))
	}
	if len(after.ConflictTasks) != len(before.ConflictTasks) ||
		len(after.ReviewWaitingTasks) != len(before.ReviewWaitingTasks) {
		t.Errorf("queues changed: conflicts=%d reviews=%d, want %d and %d",
			len(after.ConflictTasks), len(after.ReviewWaitingTasks),
			len(before.ConflictTasks), len(before.ReviewWaitingTasks))
	}
}

func TestSnapshotSurvivesOrphanedTask(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "orphans", "Orphans", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTask(ctx, feature.ID, "Kept", "", domain.TaskKindManual, ""); err != nil {
		t.Fatal(err)
	}
	orphan, err := service.CreateTask(ctx, feature.ID, "Orphan", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	// Break referential integrity the way a damaged database would: foreign key
	// enforcement is what normally prevents this, and Validate exists to find it.
	if _, err := database.DB().ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().
		ExecContext(ctx, `UPDATE tasks SET feature_id = 'missing-feature' WHERE id = ?`, orphan.StorageID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.DB().ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}

	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Features) != 1 {
		t.Fatalf("features=%d, want 1", len(snapshot.Features))
	}
	if snapshot.Features[0].TaskCount != 1 {
		t.Fatalf("task count=%d, want 1 (the orphan must not be credited)", snapshot.Features[0].TaskCount)
	}
}

func TestTaskStatusOverridesAndAutomaticPRState(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "completion", "Completion", "", "")
	if err != nil {
		t.Fatal(err)
	}
	prTask, err := service.CreateTask(ctx, feature.ID, "Ship API", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	completed := domain.TaskStatusCompleted
	updated, err := service.UpdateTask(
		ctx,
		prTask.ID,
		nil,
		nil,
		&completed,
		nil,
	)
	if err != nil || updated.Status != domain.TaskStatusCompleted {
		t.Fatalf("PR task completion override: task=%+v err=%v", updated, err)
	}
	manual, err := service.CreateTask(ctx, feature.ID, "Sign off", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateTask(ctx, manual.ID, nil, nil, &completed, nil); err != nil {
		t.Fatalf("manual task completion: %v", err)
	}

	auto := domain.TaskStatusAuto
	if _, err := service.UpdateTask(ctx, prTask.ID, nil, nil, &auto, nil); err != nil {
		t.Fatalf("clear PR task override: %v", err)
	}
	if _, err := service.AttachPullRequest(ctx, prTask.ID, "https://github.com/acme/api/pull/3"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Sync(ctx, feature.ID, ""); err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range snapshot.Tasks {
		if task.ID != prTask.ID {
			continue
		}
		if task.DisplayState != "merged" {
			t.Fatalf("PR task display state=%q, want merged", task.DisplayState)
		}
	}
}

func TestSyncByTaskID(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "targeted-sync", "Targeted sync", "", "")
	if err != nil {
		t.Fatal(err)
	}
	target, err := service.CreateTask(ctx, feature.ID, "Target", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.CreateTask(ctx, feature.ID, "Other", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	for index, task := range []domain.Task{target, other} {
		if _, err := service.AttachPullRequest(
			ctx,
			task.ID,
			fmt.Sprintf("https://github.com/acme/api/pull/%d", index+1),
		); err != nil {
			t.Fatal(err)
		}
	}
	succeeded, failed, err := service.Sync(ctx, "", target.ID)
	if err != nil || succeeded != 1 || failed != 0 {
		t.Fatalf("targeted sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, pullRequest := range snapshot.PullRequests {
		switch pullRequest.TaskID {
		case target.ID:
			if pullRequest.Stale || pullRequest.LastSyncedAt == nil {
				t.Fatalf("target pull request was not refreshed: %+v", pullRequest)
			}
		case other.ID:
			if !pullRequest.Stale || pullRequest.LastSyncedAt != nil {
				t.Fatalf("other pull request was unexpectedly refreshed: %+v", pullRequest)
			}
		}
	}
	if _, _, err := service.Sync(ctx, "", "missing-task"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing task sync code=%s err=%v", domain.ErrorCode(err), err)
	}
}

func TestImplementationPlanLifecycleAndCascade(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "implementation-plans", "Implementation plans", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Design API", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Tasks[0].HasImplementationPlan || snapshot.Tasks[0].DisplayState != domain.TaskDisplayStateNotStarted {
		t.Fatalf("initial task=%+v", snapshot.Tasks[0])
	}
	const content = "# API design\n\nDocument the boundary.\n"
	plan, err := service.UpsertImplementationPlan(
		ctx, task.ID, domain.Document{Kind: domain.DocumentKindMarkdown, Content: content},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.TaskID != task.ID || plan.Content != content || plan.CreatedAt.IsZero() || plan.UpdatedAt.IsZero() {
		t.Fatalf("stored plan=%+v", plan)
	}
	read, err := service.GetImplementationPlan(ctx, task.ID)
	if err != nil || read.Content != content || !read.CreatedAt.Equal(plan.CreatedAt) {
		t.Fatalf("read plan=%+v err=%v", read, err)
	}
	normal, err := service.AddDocument(
		ctx,
		domain.DocumentParent{TaskID: task.ID},
		domain.DocumentKindURL,
		"Reference",
		"https://example.com/reference",
		"",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	updatedTitle := "Updated reference"
	updated, err := service.UpdateDocument(ctx, normal.ID, &updatedTitle, nil, nil)
	if err != nil || updated.Title != updatedTitle {
		t.Fatalf("updated document=%+v err=%v", updated, err)
	}
	markAsPlan := true
	if _, err := service.UpdateDocument(
		ctx,
		normal.ID,
		nil,
		nil,
		&markAsPlan,
	); domain.ErrorCode(err) != domain.DomainErrorCodeDuplicateImplementationPlan {
		t.Fatalf("duplicate plan code=%s err=%v", domain.ErrorCode(err), err)
	}
	snapshot, err = service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Tasks[0].HasImplementationPlan ||
		snapshot.Tasks[0].DisplayState != domain.TaskDisplayStateNotStarted ||
		!snapshot.Tasks[0].Ready {
		t.Fatalf("planned task=%+v", snapshot.Tasks[0])
	}
	if err := service.DeleteTask(ctx, task.ID, false); domain.ErrorCode(err) != domain.DomainErrorCodeReferencesExist {
		t.Fatalf("delete with plan code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DeleteImplementationPlan(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, err = service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Tasks[0].HasImplementationPlan || snapshot.Tasks[0].DisplayState != domain.TaskDisplayStateNotStarted {
		t.Fatalf("task after plan deletion=%+v", snapshot.Tasks[0])
	}
	if err := service.DeleteImplementationPlan(ctx, task.ID); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("second plan deletion code=%s err=%v", domain.ErrorCode(err), err)
	}
	second, err := service.CreateTask(ctx, feature.ID, "Delete feature", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpsertImplementationPlan(
		ctx, second.ID, domain.Document{Kind: domain.DocumentKindMarkdown, Content: "# Keep this plan"},
	); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteFeature(ctx, feature.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetImplementationPlan(
		ctx,
		second.ID,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeNotFound {
		t.Fatalf("feature cascade plan code=%s err=%v", domain.ErrorCode(err), err)
	}
}

type scriptedProvider struct {
	value domain.PullRequest
	err   error
}

func (p *scriptedProvider) Fetch(context.Context, domain.PullRequest) (domain.PullRequest, error) {
	return p.value, p.err
}

func TestSyncKeepsPartialCoreStateAndUsesItForDependencies(t *testing.T) {
	database, _ := openTestService(t)
	ctx := context.Background()
	provider := &scriptedProvider{}
	service := app.New(database, provider)
	feature, err := service.CreateFeature(ctx, "partial-sync", "Partial sync", "", "")
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := service.CreateTask(ctx, feature.ID, "Open PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := service.CreateTask(ctx, feature.ID, "Dependent task", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(ctx, blocker.ID, blocked.ID); err != nil {
		t.Fatal(err)
	}
	attached, err := service.AttachPullRequest(ctx, blocker.ID, "https://github.com/acme/api/pull/7")
	if err != nil {
		t.Fatal(err)
	}
	partial := attached
	partial.State = domain.PullRequestStateOpen
	partial.NodeID = "core-node"
	partial.GitHubUpdatedAt = timePtr(time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC))
	provider.value = partial
	provider.err = errors.New("review endpoint unavailable")
	if succeeded, failed, err := service.Sync(ctx, feature.ID, ""); succeeded != 0 || failed != 1 || err != nil {
		t.Fatalf("partial sync result=(%d, %d, %v)", succeeded, failed, err)
	}
	stored, err := database.GetPullRequest(ctx, blocker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.PullRequestStateOpen || stored.NodeID != "core-node" ||
		stored.SyncError != "review endpoint unavailable" || !stored.Stale {
		t.Fatalf("partial pull request=%+v", stored)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range snapshot.Tasks {
		if task.ID == blocked.ID && !task.Ready {
			t.Fatalf("known open PR should satisfy dependency after partial sync: %+v", task)
		}
	}
	provider.value = stored
	provider.value.State = domain.PullRequestStateMerged
	provider.err = nil
	if succeeded, failed, err := service.Sync(ctx, feature.ID, ""); succeeded != 1 || failed != 0 || err != nil {
		t.Fatalf("successful sync result=(%d, %d, %v)", succeeded, failed, err)
	}
	stored, err = database.GetPullRequest(ctx, blocker.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.PullRequestStateMerged || stored.SyncError != "" || stored.Stale {
		t.Fatalf("recovered pull request=%+v", stored)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
func TestInMemoryDatabaseIsSharedAcrossConcurrentCallers(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, _ := githubprovider.NewFixtureProvider("demo")
	service := app.New(database, provider)
	feature, err := service.CreateFeature(ctx, "in-memory", "In memory", "", "")
	if err != nil {
		t.Fatal(err)
	}
	const readers = 12
	var wait sync.WaitGroup
	errorsCh := make(chan error, readers)
	for i := 0; i < readers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			snapshot, err := service.Snapshot(ctx)
			if err != nil {
				errorsCh <- err
				return
			}
			if len(snapshot.Features) != 1 || snapshot.Features[0].ID != feature.ID {
				errorsCh <- fmt.Errorf("concurrent reader saw %d features", len(snapshot.Features))
			}
		}()
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("in-memory database is not shared: %v", err)
	}
}

func TestPullRequestURLCasingIsNotADistinctPullRequest(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "casing", "Casing", "", "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.CreateTask(ctx, feature.ID, "First", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateTask(ctx, feature.ID, "Second", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, first.ID, "https://github.com/Acme/API/pull/42"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(
		ctx,
		second.ID,
		"https://github.com/acme/api/pull/42",
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeDuplicatePullRequest {
		t.Fatalf("case-only variant code=%s err=%v", domain.ErrorCode(err), err)
	}
}

func TestDeletingWhatIsNotThereReportsNotFound(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "deletions", "Deletions", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Task", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveDependency(
		ctx,
		task.ID,
		"missing-task",
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodeNotFound {
		t.Fatalf("remove dependency code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DetachPullRequest(ctx, task.ID); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("detach pull request code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DeleteDocument(ctx, "missing-document"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("delete document code=%s err=%v", domain.ErrorCode(err), err)
	}
}

func TestGetPullRequestRoundTripAndNotFound(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "pull-request-read", "Pull request read", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Read pull request", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	githubUpdatedAt := time.Date(2026, 8, 29, 1, 2, 3, 456000000, time.UTC)
	lastSyncedAt := time.Date(2026, 8, 29, 2, 3, 4, 567000000, time.UTC)
	want := domain.PullRequest{
		TaskID:          task.ID,
		Host:            "github.com",
		Owner:           "Acme",
		Repository:      "API",
		Number:          42,
		URL:             "https://github.com/Acme/API/pull/42",
		NodeID:          "node-42",
		Author:          "octocat",
		Assignees:       []string{"alice", "bob"},
		State:           domain.PullRequestStateMerged,
		ReviewState:     domain.ReviewStateApproved,
		Mergeability:    domain.MergeabilityMergeable,
		GitHubUpdatedAt: &githubUpdatedAt,
		LastSyncedAt:    &lastSyncedAt,
		Stale:           false,
	}
	if _, err := database.UpsertPullRequest(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := database.GetPullRequest(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskID != want.TaskID || got.Host != want.Host ||
		got.Owner != want.Owner || got.Repository != want.Repository ||
		got.Number != want.Number || got.URL != want.URL || got.NodeID != want.NodeID || got.Author != want.Author ||
		got.State != want.State || got.ReviewState != want.ReviewState ||
		got.Mergeability != want.Mergeability ||
		got.Stale != want.Stale ||
		got.DisplayState != domain.PullRequestDisplayStateMerged {
		t.Fatalf("pull request=%+v, want fields from %+v", got, want)
	}
	if len(got.Assignees) != 2 || got.Assignees[0] != "alice" || got.Assignees[1] != "bob" {
		t.Fatalf("assignees=%v, want [alice bob]", got.Assignees)
	}
	if got.GitHubUpdatedAt == nil || !got.GitHubUpdatedAt.Equal(githubUpdatedAt) {
		t.Fatalf("github updated at=%v, want %v", got.GitHubUpdatedAt, githubUpdatedAt)
	}
	if got.LastSyncedAt == nil || !got.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("last synced at=%v, want %v", got.LastSyncedAt, lastSyncedAt)
	}
	if _, err := database.GetPullRequest(ctx, "missing-task"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing pull request code=%s err=%v", domain.ErrorCode(err), err)
	}
}

func TestDefaultPathUsesUserConfigDirectory(t *testing.T) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configDir, "prx", "prx.db")
	if got != want {
		t.Fatalf("DefaultPath()=%q, want %q", got, want)
	}
}

func TestStoreUpdateDocumentWritesOnlyRequestedFields(t *testing.T) {
	ctx := context.Background()
	database, _ := openTestService(t)
	feature, err := database.CreateFeature(ctx, "checkout", "Checkout", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := database.CreateTask(ctx, feature.ID, "Ship it", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	document, err := database.CreateDocument(
		ctx,
		domain.DocumentParent{TaskID: task.ID},
		domain.DocumentKindMarkdown,
		"Original",
		"",
		"# Original",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}

	title := "Renamed"
	if _, err := database.UpdateDocument(ctx, document.ID, &title, nil, nil); err != nil {
		t.Fatal(err)
	}
	// A caller that read the document before the rename must not restore the old title.
	source := domain.Document{Kind: domain.DocumentKindURL, Locator: "https://example.com/runbook"}
	updated, err := database.UpdateDocument(ctx, document.ID, nil, &source, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != title ||
		updated.Kind != domain.DocumentKindURL ||
		updated.Locator != "https://example.com/runbook" ||
		updated.Content != "" {
		t.Fatalf("document=%+v, want the rename kept and the source replaced", updated)
	}

	plan := true
	withPlan, err := database.UpdateDocument(ctx, document.ID, nil, nil, &plan)
	if err != nil {
		t.Fatal(err)
	}
	if !withPlan.IsImplementationPlan ||
		withPlan.Title != title ||
		withPlan.Locator != "https://example.com/runbook" {
		t.Fatalf("document=%+v, want only the implementation plan flag changed", withPlan)
	}
}

func TestStoreUpsertImplementationPlanKeepsExistingTitle(t *testing.T) {
	ctx := context.Background()
	database, _ := openTestService(t)
	feature, err := database.CreateFeature(ctx, "checkout", "Checkout", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := database.CreateTask(ctx, feature.ID, "Ship it", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := database.UpsertImplementationPlan(ctx, task.ID, domain.Document{
		Kind:    domain.DocumentKindMarkdown,
		Content: "# First",
	})
	if err != nil {
		t.Fatal(err)
	}
	title := "Design plan"
	if _, err := database.UpdateDocument(ctx, plan.ID, &title, nil, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := database.UpsertImplementationPlan(ctx, task.ID, domain.Document{
		Kind:    domain.DocumentKindURL,
		Locator: "https://example.com/plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != title || updated.Locator != "https://example.com/plan" {
		t.Fatalf("plan=%+v, want the title kept and the source replaced", updated)
	}
}
