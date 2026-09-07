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

func TestAutomaticSyncClaimsOnceAndFiltersArchivedButRefreshesMergedPullRequests(t *testing.T) {
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

	active := newFeature(t, ctx, service, "Active sync")
	activeTask, err := service.CreateTask(ctx, active.ID, "Active PR", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, activeTask.ID, "https://github.com/acme/api/pull/1"); err != nil {
		t.Fatal(err)
	}
	archivedTask, err := archivedFeatureWithPullRequest(
		t, service, "archived-sync", "https://github.com/acme/web/pull/2",
	)
	if err != nil {
		t.Fatal(err)
	}
	// project がアーカイブ済みであることだけを理由に読み取り専用になった feature も、
	// 自身がアーカイブ済みの feature と同じ refresh から外れる。
	projectTask, err := featureInArchivedProjectWithPullRequest(
		t, service, "sunset", "project-sync", "https://github.com/acme/mobile/pull/4",
	)
	if err != nil {
		t.Fatal(err)
	}
	mergedTask, err := service.CreateTask(ctx, active.ID, "Merged PR", "", "")
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

	clearSyncMarkers(t, database)

	before, err := service.SyncStatus(ctx)
	if err != nil || before.LastUpdatedAt != nil || before.IntervalSeconds != 3600 {
		t.Fatalf("initial status=%+v err=%v", before, err)
	}
	ran, status, err := service.SyncIfDue(ctx)
	if err != nil || !ran || status.Succeeded != 2 || status.Failed != 0 || status.LastUpdatedAt == nil {
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
		byTask[projectTask.ID].LastSyncedAt != nil ||
		byTask[mergedTask.ID].LastSyncedAt == nil || byTask[mergedTask.ID].State != domain.PullRequestStateMerged {
		t.Fatalf("filtered pull requests=%+v", byTask)
	}
	succeeded, failed, err := service.Sync(ctx, "", "")
	if err != nil || succeeded != 2 || failed != 0 {
		t.Fatalf("manual full sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	snapshot, err = service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, pullRequest := range snapshot.PullRequests {
		readOnly := pullRequest.TaskID == archivedTask.ID || pullRequest.TaskID == projectTask.ID
		if readOnly && pullRequest.LastSyncedAt != nil {
			t.Fatalf("unscoped manual sync included read-only pull request=%+v", pullRequest)
		}
	}
	for _, taskID := range []string{archivedTask.ID, projectTask.ID} {
		succeeded, failed, err = service.Sync(ctx, "", taskID)
		if err != nil || succeeded != 1 || failed != 0 {
			t.Fatalf("manual scoped sync of %s succeeded=%d failed=%d err=%v", taskID, succeeded, failed, err)
		}
	}
}

// archivedFeatureWithPullRequest は先に中身を作ってから archive する。
// アーカイブ済み feature は、それを作る書き込みを拒むため。
func archivedFeatureWithPullRequest(
	t *testing.T,
	service *app.Service,
	slug, pullRequestURL string,
) (domain.Task, error) {
	t.Helper()
	ctx := context.Background()
	feature := newFeature(t, ctx, service, slug)
	task, err := service.CreateTask(ctx, feature.ID, "Archived PR", "", "")
	if err != nil {
		return domain.Task{}, err
	}
	if _, err := service.AttachPullRequest(ctx, task.ID, pullRequestURL); err != nil {
		return domain.Task{}, err
	}
	archived := true
	if _, err := service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{Archived: &archived}); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// featureInArchivedProjectWithPullRequest は feature 自体を active のままにし、
// 読み取り専用の原因を、所属する project だけに限定する。
func featureInArchivedProjectWithPullRequest(
	t *testing.T,
	service *app.Service,
	projectSlug, featureSlug, pullRequestURL string,
) (domain.Task, error) {
	t.Helper()
	ctx := context.Background()
	project, err := service.CreateProject(ctx, projectSlug, "")
	if err != nil {
		return domain.Task{}, err
	}
	feature, err := service.CreateFeature(ctx, featureSlug, "", project.ID)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := service.CreateTask(ctx, feature.ID, "Project PR", "", "")
	if err != nil {
		return domain.Task{}, err
	}
	if _, err := service.AttachPullRequest(ctx, task.ID, pullRequestURL); err != nil {
		return domain.Task{}, err
	}
	archived := true
	if _, err := service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{Archived: &archived}); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// 単一 task に絞った refresh は、飛ばした pull request について何も語らない。
// よって共有の interval を消費したり、その件数を公開したりしてはならない。
func TestTargetedManualSyncLeavesTheAutomaticIntervalAndStatusUntouched(t *testing.T) {
	ctx := context.Background()
	service, database := newAutoSyncTestService(t)
	defer func() { _ = database.Close() }()

	feature := newFeature(t, ctx, service, "Targeted")
	task, err := service.CreateTask(ctx, feature.ID, "Targeted PR", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, task.ID, "https://github.com/acme/api/pull/1"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := service.Sync(ctx, "", task.ID); err != nil {
		t.Fatal(err)
	}
	status, err := service.SyncStatus(ctx)
	if err != nil || status.LastAttemptAt != nil || status.LastUpdatedAt != nil {
		t.Fatalf("targeted sync recorded a run: status=%+v err=%v", status, err)
	}
	ran, status, err := service.SyncIfDue(ctx)
	if err != nil || !ran || status.LastUpdatedAt == nil {
		t.Fatalf("automatic sync after a targeted sync ran=%v status=%+v err=%v", ran, status, err)
	}
}

// 呼び出し側が消えたことで refresh が終わることがある。結果の記録はそれを生き延びる
// 必要がある。さもないと取得済みの試行が、何に止められたかの説明もないまま
// interval を握ってしまう。
func TestAutomaticSyncRecordsTheRunAfterTheCallerCancels(t *testing.T) {
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "cancelled.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	// attach 後の refresh が、このテストの狙いである自動実行向けのキャンセルを
	// 使い切らないよう、provider は pull request が揃ってから
	// キャンセルを始める。
	var cancelAutomaticSync context.CancelFunc
	service := app.NewWithConfig(
		database,
		cancellingProvider{cancel: func() {
			if cancelAutomaticSync != nil {
				cancelAutomaticSync()
			}
		}},
		configStore,
	)

	feature := newFeature(t, context.Background(), service, "Cancelled")
	task, err := service.CreateTask(context.Background(), feature.ID, "PR", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(
		context.Background(), task.ID, "https://github.com/acme/api/pull/1",
	); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelAutomaticSync = cancel

	ran, _, err := service.SyncIfDue(ctx)
	if err != nil || !ran {
		t.Fatalf("cancelled automatic sync ran=%v err=%v", ran, err)
	}
	status, err := service.SyncStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.LastUpdatedAt == nil || status.Error == "" {
		t.Fatalf("cancelled run was not recorded: status=%+v", status)
	}
}

// cancellingProvider は refresh の実行中に、切断された RPC や中断されたコマンドと
// 同じように呼び出し側の context を終了させる。
type cancellingProvider struct{ cancel context.CancelFunc }

func (p cancellingProvider) Fetch(
	_ context.Context,
	current domain.PullRequest,
) (domain.PullRequest, error) {
	p.cancel()
	return current, context.Canceled
}

func newAutoSyncTestService(t *testing.T) (*app.Service, *store.Store) {
	t.Helper()
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "auto-sync.db"))
	if err != nil {
		t.Fatal(err)
	}
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
	return app.NewWithConfig(database, provider, configStore), database
}
