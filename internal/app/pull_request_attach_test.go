package app_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

// Attaching records work that is current by definition, so the pull request
// must not wait for the next refresh to stop being presented as stale.
func TestAttachPullRequestRefreshesTheAttachedPullRequest(t *testing.T) {
	ctx := context.Background()
	service, database := newAutoSyncTestService(t)
	defer func() { _ = database.Close() }()
	task := createTaskForAttach(t, service, "Attach refresh")

	attached, err := service.AttachPullRequest(ctx, task.ID, "https://github.com/acme/api/pull/1")
	if err != nil {
		t.Fatal(err)
	}
	if attached.Stale || attached.SyncError != "" || attached.LastSyncedAt == nil ||
		attached.State == domain.PullRequestStateUnknown {
		t.Fatalf("attached pull request=%+v", attached)
	}

	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.PullRequests) != 1 || snapshot.PullRequests[0].State != attached.State ||
		snapshot.PullRequests[0].Stale {
		t.Fatalf("stored pull requests=%+v", snapshot.PullRequests)
	}
	if len(snapshot.StaleTasks) != 0 {
		t.Fatalf("stale tasks after attaching=%+v", snapshot.StaleTasks)
	}

	// The refresh covers one task, so the shared interval and the counts of the
	// last refresh that covered every pull request stay as they were.
	status, err := service.SyncStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastAttemptAt != nil || status.LastUpdatedAt != nil {
		t.Fatalf("run status after attaching=%+v", status)
	}
}

// A refresh that cannot reach GitHub is not a reason to reject the attachment:
// the pull request is recorded, and holds the staleness and the failure that
// say why it carries no state yet.
func TestAttachPullRequestKeepsAFailedRefreshVisible(t *testing.T) {
	ctx := context.Background()
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "attach-failure.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	service := app.NewWithConfig(database, failingProvider{}, configStore)
	task := createTaskForAttach(t, service, "Attach failure")

	attached, err := service.AttachPullRequest(ctx, task.ID, "https://github.com/acme/api/pull/1")
	if err != nil {
		t.Fatal(err)
	}
	if !attached.Stale || attached.SyncError != "GitHub unavailable" ||
		attached.State != domain.PullRequestStateUnknown {
		t.Fatalf("attached pull request=%+v", attached)
	}
}

func createTaskForAttach(t *testing.T, service *app.Service, title string) domain.Task {
	t.Helper()
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, title, "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, title, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return task
}

// failingProvider stands for a GitHub host that answers no request.
type failingProvider struct{}

func (failingProvider) Fetch(
	_ context.Context,
	current domain.PullRequest,
) (domain.PullRequest, error) {
	return current, errors.New("GitHub unavailable")
}
