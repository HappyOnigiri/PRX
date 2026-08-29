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
		migrations != 2 {
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

func TestPublicIDsAreTypedAndStorageIDsStayInternal(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "public-ids", "Public IDs", "")
	if err != nil {
		t.Fatal(err)
	}
	secondFeature, err := service.CreateFeature(ctx, "public-ids-2", "Public IDs 2", "")
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
	feature, _ := service.CreateFeature(ctx, "concurrency", "Concurrency", "")
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
	feature, err := service.CreateFeature(ctx, "corruption", "Corruption", "")
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

func TestConcurrentDependencyWritesDoNotLock(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "contention", "Contention", "")
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

func TestSeedDemoIsIdempotent(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	if err := service.SeedDemo(ctx, "demo-roadmap", 6); err != nil {
		t.Fatal(err)
	}
	first, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SeedDemo(ctx, "demo-roadmap", 6); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	second, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Features) != len(first.Features) ||
		len(second.Tasks) != len(first.Tasks) ||
		len(second.Dependencies) != len(first.Dependencies) ||
		len(second.PullRequests) != len(first.PullRequests) {
		t.Fatalf("seed changed counts: features %d→%d tasks %d→%d deps %d→%d prs %d→%d",
			len(first.Features), len(second.Features),
			len(first.Tasks), len(second.Tasks),
			len(first.Dependencies), len(second.Dependencies),
			len(first.PullRequests), len(second.PullRequests))
	}
}

func TestSnapshotSurvivesOrphanedTask(t *testing.T) {
	database, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "orphans", "Orphans", "")
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

func TestPRTaskCannotBeCompletedManually(t *testing.T) {
	_, service := openTestService(t)
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, "completion", "Completion", "")
	if err != nil {
		t.Fatal(err)
	}
	prTask, err := service.CreateTask(ctx, feature.ID, "Ship API", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	completed := domain.TaskCompleted
	if _, err := service.UpdateTask(
		ctx,
		prTask.ID,
		nil,
		nil,
		&completed,
		nil,
	); domain.ErrorCode(
		err,
	) != domain.DomainErrorCodePRTaskCompletesOnMerge {
		t.Fatalf("PR task completion code=%s err=%v", domain.ErrorCode(err), err)
	}
	manual, err := service.CreateTask(ctx, feature.ID, "Sign off", "", domain.TaskKindManual, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateTask(ctx, manual.ID, nil, nil, &completed, nil); err != nil {
		t.Fatalf("manual task completion: %v", err)
	}

	// A PR task completes only through a merged pull request.
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
	feature, err := service.CreateFeature(ctx, "targeted-sync", "Targeted sync", "")
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

func TestInMemoryDatabaseIsSharedAcrossConcurrentCallers(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, _ := githubprovider.NewFixtureProvider("demo")
	service := app.New(database, provider)
	feature, err := service.CreateFeature(ctx, "in-memory", "In memory", "")
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
	feature, err := service.CreateFeature(ctx, "casing", "Casing", "")
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
	feature, err := service.CreateFeature(ctx, "deletions", "Deletions", "")
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
	feature, err := service.CreateFeature(ctx, "pull-request-read", "Pull request read", "")
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
	if got.TaskID != want.TaskID || got.Owner != want.Owner || got.Repository != want.Repository ||
		got.Number != want.Number || got.URL != want.URL || got.NodeID != want.NodeID || got.Author != want.Author ||
		got.State != want.State || got.ReviewState != want.ReviewState || got.Mergeability != want.Mergeability ||
		got.Stale != want.Stale || got.DisplayState != domain.PullRequestDisplayStateMerged {
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
