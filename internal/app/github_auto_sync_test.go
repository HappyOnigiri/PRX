package app_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestAutomaticSyncClaimsOnceAndFiltersArchivedAndMergedPullRequests(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "auto-sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	provider, err := githubprovider.NewFixtureProvider("demo")
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, provider, configStore)

	active, err := service.CreateFeature(ctx, "active-sync", "Active sync", "")
	if err != nil {
		t.Fatal(err)
	}
	activeTask, err := service.CreateTask(ctx, active.ID, "Active PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, activeTask.ID, "https://github.com/acme/api/pull/1"); err != nil {
		t.Fatal(err)
	}
	archived, err := service.CreateFeature(ctx, "archived-sync", "Archived sync", "")
	if err != nil {
		t.Fatal(err)
	}
	archivedValue := true
	archived, err = service.UpdateFeature(ctx, archived.ID, nil, nil, nil, nil, &archivedValue)
	if err != nil {
		t.Fatal(err)
	}
	archivedTask, err := service.CreateTask(ctx, archived.ID, "Archived PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, archivedTask.ID, "https://github.com/acme/web/pull/2"); err != nil {
		t.Fatal(err)
	}
	mergedTask, err := service.CreateTask(ctx, active.ID, "Merged PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	merged, err := service.AttachPullRequest(ctx, mergedTask.ID, "https://github.com/acme/api/pull/3")
	if err != nil {
		t.Fatal(err)
	}
	merged.State = domain.PullRequestStateMerged
	if _, err := database.UpsertPullRequest(ctx, merged); err != nil {
		t.Fatal(err)
	}

	before, err := service.SyncStatus(ctx)
	if err != nil || before.LastUpdatedAt != nil || before.IntervalSeconds != 3600 {
		t.Fatalf("initial status=%+v err=%v", before, err)
	}
	ran, status, err := service.SyncIfDue(ctx)
	if err != nil || !ran || status.Succeeded != 1 || status.Failed != 0 || status.LastUpdatedAt == nil {
		t.Fatalf("automatic sync ran=%v status=%+v err=%v", ran, status, err)
	}
	ran, _, err = service.SyncIfDue(ctx)
	if err != nil || ran {
		t.Fatalf("second automatic sync ran=%v err=%v", ran, err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byTask := map[string]domain.PullRequest{}
	for _, pullRequest := range snapshot.PullRequests {
		byTask[pullRequest.TaskID] = pullRequest
	}
	if byTask[activeTask.ID].LastSyncedAt == nil || byTask[archivedTask.ID].LastSyncedAt != nil ||
		byTask[mergedTask.ID].LastSyncedAt != nil {
		t.Fatalf("filtered pull requests=%+v", byTask)
	}
	succeeded, failed, err := service.Sync(ctx, "", archivedTask.ID)
	if err != nil || succeeded != 1 || failed != 0 {
		t.Fatalf("manual archived sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
}
