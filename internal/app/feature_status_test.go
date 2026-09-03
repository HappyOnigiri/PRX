package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

// statusTask describes one task to create for a feature whose derived status is
// under test.
type statusTask struct {
	kind   domain.TaskKind
	status domain.TaskStatus
	pr     domain.PullRequestState
}

func TestFeatureCompletesOnlyWhenEveryTaskIsFinished(t *testing.T) {
	tests := []struct {
		name  string
		slug  string
		tasks []statusTask
		want  domain.FeatureStatus
	}{
		{
			name: "a feature without tasks has nothing to complete",
			slug: "no-tasks",
			want: domain.FeatureStatusActive,
		},
		{
			name:  "manual completions finish the work",
			slug:  "manual-completed",
			tasks: []statusTask{{kind: domain.TaskKindManual, status: domain.TaskStatusCompleted}},
			want:  domain.FeatureStatusCompleted,
		},
		{
			name:  "manual closures finish the work",
			slug:  "manual-closed",
			tasks: []statusTask{{kind: domain.TaskKindManual, status: domain.TaskStatusClosed}},
			want:  domain.FeatureStatusCompleted,
		},
		{
			name: "merged and closed pull requests finish the work",
			slug: "merged-and-closed",
			tasks: []statusTask{
				{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateMerged},
				{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateClosed},
			},
			want: domain.FeatureStatusCompleted,
		},
		{
			name: "manual and automatic finishes mix",
			slug: "mixed-finished",
			tasks: []statusTask{
				{kind: domain.TaskKindManual, status: domain.TaskStatusCompleted},
				{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateMerged},
			},
			want: domain.FeatureStatusCompleted,
		},
		{
			name: "one task not started keeps the feature active",
			slug: "one-not-started",
			tasks: []statusTask{
				{kind: domain.TaskKindManual, status: domain.TaskStatusCompleted},
				{kind: domain.TaskKindManual, status: domain.TaskStatusNotStarted},
			},
			want: domain.FeatureStatusActive,
		},
		{
			name: "one task in progress keeps the feature active",
			slug: "one-in-progress",
			tasks: []statusTask{
				{kind: domain.TaskKindManual, status: domain.TaskStatusCompleted},
				{kind: domain.TaskKindManual, status: domain.TaskStatusInProgress},
			},
			want: domain.FeatureStatusActive,
		},
		{
			name: "one open pull request keeps the feature active",
			slug: "one-open-pull-request",
			tasks: []statusTask{
				{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateMerged},
				{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateOpen},
			},
			want: domain.FeatureStatusActive,
		},
	}
	service, database := newAutoSyncTestService(t)
	defer func() { _ = database.Close() }()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			feature := createFeatureWithTasks(t, service, database, test.slug, test.tasks)
			if got := snapshotFeature(t, service, feature.ID); got.DisplayStatus != test.want {
				t.Fatalf("display status=%q, want %q (finished %d of %d)",
					got.DisplayStatus, test.want, got.FinishedCount, got.TaskCount)
			}
		})
	}
}

func TestManualFeatureStatusOverridesAutomaticCompletion(t *testing.T) {
	ctx := context.Background()
	service, database := newAutoSyncTestService(t)
	defer func() { _ = database.Close() }()
	feature := createFeatureWithTasks(t, service, database, "override", []statusTask{
		{kind: domain.TaskKindManual, status: domain.TaskStatusCompleted},
	})
	if got := snapshotFeature(t, service, feature.ID); got.DisplayStatus != domain.FeatureStatusCompleted {
		t.Fatalf("automatic display status=%q", got.DisplayStatus)
	}
	for _, test := range []struct {
		stored domain.FeatureStatus
		want   domain.FeatureStatus
	}{
		{domain.FeatureStatusActive, domain.FeatureStatusActive},
		{domain.FeatureStatusPaused, domain.FeatureStatusPaused},
		{domain.FeatureStatusCancelled, domain.FeatureStatusCancelled},
		{domain.FeatureStatusAuto, domain.FeatureStatusCompleted},
	} {
		t.Run(string(test.stored), func(t *testing.T) {
			stored := test.stored
			updated, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, &stored, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != test.stored {
				t.Fatalf("stored status=%q, want %q", updated.Status, test.stored)
			}
			if got := snapshotFeature(t, service, feature.ID); got.DisplayStatus != test.want {
				t.Fatalf("display status=%q, want %q", got.DisplayStatus, test.want)
			}
		})
	}
}

// A completed feature leaves the refreshes that maintain work in flight, so its
// pull requests stop changing until a caller asks for that feature by name.
func TestUnscopedSyncSkipsCompletedFeaturesAndExplicitScopeRefreshesThem(t *testing.T) {
	ctx := context.Background()
	service, database := newAutoSyncTestService(t)
	defer func() { _ = database.Close() }()
	feature := createFeatureWithTasks(t, service, database, "completed-sync", []statusTask{
		{kind: domain.TaskKindPR, status: domain.TaskStatusAuto, pr: domain.PullRequestStateMerged},
	})
	if got := snapshotFeature(t, service, feature.ID); got.DisplayStatus != domain.FeatureStatusCompleted {
		t.Fatalf("display status=%q", got.DisplayStatus)
	}

	succeeded, failed, err := service.Sync(ctx, "", "")
	if err != nil || succeeded != 0 || failed != 0 {
		t.Fatalf("unscoped sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, pullRequest := range snapshot.PullRequests {
		if pullRequest.LastSyncedAt != nil {
			t.Fatalf("unscoped sync refreshed a completed feature's pull request=%+v", pullRequest)
		}
	}

	succeeded, failed, err = service.Sync(ctx, feature.ID, "")
	if err != nil || succeeded != 1 || failed != 0 {
		t.Fatalf("feature sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
}

// demoPullRequestNumber keeps every attached pull request in these tests
// distinct, because storage rejects a repeated host, owner, repository, and
// number.
var demoPullRequestNumber = 0

func createFeatureWithTasks(
	t *testing.T,
	service *app.Service,
	database *store.Store,
	slug string,
	tasks []statusTask,
) domain.Feature {
	t.Helper()
	ctx := context.Background()
	feature, err := service.CreateFeature(ctx, slug, slug, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range tasks {
		task, err := service.CreateTask(ctx, feature.ID, fmt.Sprintf("Task %d", index+1), "", value.kind, "")
		if err != nil {
			t.Fatal(err)
		}
		if value.status != domain.TaskStatusAuto {
			status := value.status
			if _, err := service.UpdateTask(ctx, task.ID, nil, nil, &status, nil); err != nil {
				t.Fatal(err)
			}
		}
		if value.pr == "" {
			continue
		}
		demoPullRequestNumber++
		url := fmt.Sprintf("https://github.com/acme/api/pull/%d", demoPullRequestNumber)
		attached, err := service.AttachPullRequest(ctx, task.ID, url)
		if err != nil {
			t.Fatal(err)
		}
		attached.State = value.pr
		if _, err := database.UpsertPullRequest(ctx, attached); err != nil {
			t.Fatal(err)
		}
	}
	return feature
}

func snapshotFeature(t *testing.T, service *app.Service, id string) domain.Feature {
	t.Helper()
	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range snapshot.Features {
		if feature.ID == id {
			return feature
		}
	}
	t.Fatalf("feature %q is missing from the snapshot", id)
	return domain.Feature{}
}
