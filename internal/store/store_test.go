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

	"github.com/HappyOnigiri/PRX/internal/app"
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
	if err := database.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil || migrations != 1 {
		t.Fatalf("migration count=%d err=%v", migrations, err)
	}
	var foreignKeys, journalMode int
	var journal string
	if err := database.DB().QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
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

func TestCycleDuplicateAndSafeDeletion(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "delivery", "Delivery", "")
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
	if _, err := service.AddDependency(ctx, a.ID, b.ID); domain.ErrorCode(err) != "duplicate_dependency" {
		t.Fatalf("duplicate code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.AddDependency(ctx, c.ID, a.ID); domain.ErrorCode(err) != "cycle" {
		t.Fatalf("cycle code=%s err=%v", domain.ErrorCode(err), err)
	}
	if err := service.DeleteTask(ctx, b.ID, false); domain.ErrorCode(err) != "references_exist" {
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
	feature, _ := service.CreateFeature(ctx, "concurrency", "Concurrency", "")
	a, _ := service.CreateTask(ctx, feature.ID, "A", "", domain.TaskKindPR, "")
	b, _ := service.CreateTask(ctx, feature.ID, "B", "", domain.TaskKindPR, "")
	if _, err := service.AttachPullRequest(ctx, a.ID, "https://github.com/acme/api/pull/42"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, b.ID, "https://github.com/acme/api/pull/42"); domain.ErrorCode(err) != "duplicate_pull_request" {
		t.Fatalf("duplicate PR code=%s err=%v", domain.ErrorCode(err), err)
	}
	const writers = 24
	var wait sync.WaitGroup
	errorsCh := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, err := service.CreateTask(ctx, feature.ID, fmt.Sprintf("task-%02d", index), "", domain.TaskKindManual, "")
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
	feature, err := service.CreateFeature(ctx, "corruption", "Corruption", "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		if _, err := service.CreateTask(ctx, feature.ID, fmt.Sprintf("task-%03d", i), strings.Repeat("x", 256), domain.TaskKindManual, ""); err != nil {
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
	feature, err := service.CreateFeature(ctx, "clearing", "Clearing", "initial description")
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

	updatedFeature, err := service.UpdateFeature(ctx, feature.ID, nil, nil, &empty, nil, nil)
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
	feature, err := service.CreateFeature(ctx, "archivable", "Archivable", "")
	if err != nil {
		t.Fatal(err)
	}
	archived, unarchived := true, false
	updated, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, &archived)
	if err != nil || !updated.Archived {
		t.Fatalf("archive: archived=%v err=%v", updated.Archived, err)
	}
	updated, err = service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, &unarchived)
	if err != nil || updated.Archived {
		t.Fatalf("unarchive: archived=%v err=%v", updated.Archived, err)
	}
}
