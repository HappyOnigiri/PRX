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

// attach が記録するのは定義上その時点で最新の情報なので、pull request が stale 表示を
// やめるのに次の refresh を待たせてはならない。
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

	// この refresh は 1 つの task だけを対象とするので、共有の interval と、
	// 全 pull request を対象にした直近の refresh の件数はそのまま残る。
	status, err := service.SyncStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastAttemptAt != nil || status.LastUpdatedAt != nil {
		t.Fatalf("run status after attaching=%+v", status)
	}
}

// GitHub に到達できない refresh は attach を拒む理由にならない。pull request は記録され、
// なぜまだ状態を持たないかを示す stale フラグと失敗内容を保持する。
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
	feature := newFeature(t, ctx, service, title)
	task, err := service.CreateTask(ctx, feature.ID, title, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return task
}

// failingProvider は、どのリクエストにも応答しない GitHub host を表す。
type failingProvider struct{}

func (failingProvider) Fetch(
	_ context.Context,
	current domain.PullRequest,
) (domain.PullRequest, error) {
	return current, errors.New("GitHub unavailable")
}
