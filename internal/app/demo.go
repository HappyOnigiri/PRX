package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

type demoTask struct {
	title    string
	scope    string
	kind     domain.TaskKind
	status   domain.TaskStatus
	assignee string
	plan     string
	pr       *domain.PullRequest
}

// InitializeDemo populates a new, empty repository with the built-in demo.
func (s *Service) InitializeDemo(ctx context.Context, markdownPath string) error {
	if err := os.WriteFile(markdownPath, []byte(demoMarkdown), 0o600); err != nil {
		return fmt.Errorf("write demo Markdown: %w", err)
	}
	for _, initialize := range []func(context.Context) error{
		func(ctx context.Context) error { return s.createShowcaseDemo(ctx, markdownPath) },
		s.createPausedDemo,
		s.createCompletedDemo,
		s.createCancelledDemo,
	} {
		if err := initialize(ctx); err != nil {
			return err
		}
	}
	if issues := s.Validate(ctx); len(issues) > 0 {
		return fmt.Errorf("validate demo data: %v", issues)
	}
	return nil
}

func (s *Service) createShowcaseDemo(ctx context.Context, markdownPath string) error {
	showcase, err := s.CreateFeature(
		ctx,
		"delivery-control",
		"Delivery control showcase",
		"Every task and pull-request state in one cross-repository graph.",
	)
	if err != nil {
		return err
	}
	showcaseTasks, err := s.createDemoTasks(ctx, showcase, showcaseDemoTasks())
	if err != nil {
		return err
	}
	for _, edge := range [][2]int{{0, 1}, {3, 6}, {6, 10}, {6, 11}, {10, 12}} {
		if _, err := s.AddDependency(ctx, showcaseTasks[edge[0]].ID, showcaseTasks[edge[1]].ID); err != nil {
			return err
		}
	}
	if _, err := s.AddDocument(
		ctx,
		showcase.ID,
		"",
		domain.DocumentKindURL,
		"Product brief",
		"https://github.com/HappyOnigiri/PRX",
	); err != nil {
		return err
	}
	if _, err := s.AddDocument(
		ctx,
		showcase.ID,
		"",
		domain.DocumentKindMarkdownPath,
		"Demo walkthrough",
		markdownPath,
	); err != nil {
		return err
	}
	return nil
}

func (s *Service) createPausedDemo(ctx context.Context) error {
	paused, err := s.CreateFeature(
		ctx,
		"paused-rollout",
		"Paused rollout",
		"A small branch-and-merge graph with ready and blocked manual work.",
	)
	if err != nil {
		return err
	}
	pausedStatus := domain.FeatureStatusPaused
	paused, err = s.UpdateFeature(ctx, paused.ID, nil, nil, nil, &pausedStatus, nil)
	if err != nil {
		return err
	}
	pausedTasks, err := s.createDemoTasks(ctx, paused, []demoTask{
		{
			title:    "Foundation complete",
			scope:    "Shared rollout foundation",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusCompleted,
			assignee: "Mika",
		},
		{
			title:  "Prepare operator guide",
			scope:  "Document the rollout",
			kind:   domain.TaskKindManual,
			status: domain.TaskStatusNotStarted,
		},
		{
			title:    "Run limited rollout",
			scope:    "Exercise the first cohort",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusInProgress,
			assignee: "Ren",
		},
		{
			title:    "Complete general rollout",
			scope:    "Merge both rollout branches",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusAuto,
			assignee: "Ari",
		},
	})
	if err != nil {
		return err
	}
	for _, edge := range [][2]int{{0, 1}, {0, 2}, {1, 3}, {2, 3}} {
		if _, err := s.AddDependency(ctx, pausedTasks[edge[0]].ID, pausedTasks[edge[1]].ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createCompletedDemo(ctx context.Context) error {
	completed, err := s.CreateFeature(
		ctx,
		"completed-program",
		"Completed 100-task program",
		"A large balanced graph for checking layout and navigation at scale.",
	)
	if err != nil {
		return err
	}
	completedTasks := make([]demoTask, 100)
	for index := range completedTasks {
		completedTasks[index] = demoTask{
			title:    fmt.Sprintf("Completed delivery slice %03d", index+1),
			scope:    "A completed repository delivery boundary",
			kind:     domain.TaskKindPR,
			status:   domain.TaskStatusAuto,
			assignee: []string{"Ari", "Mika", "Ren"}[index%3],
			pr: &domain.PullRequest{
				Host: "github.com", Owner: "HappyOnigiri", Repository: "prx-demo-scale",
				Number: int64(1001 + index), State: domain.PullRequestStateMerged,
				ReviewState: domain.ReviewStateApproved, Mergeability: domain.MergeabilityMergeable,
			},
		}
	}
	largeTasks, err := s.createDemoTasks(ctx, completed, completedTasks)
	if err != nil {
		return err
	}
	for index := 1; index < len(largeTasks); index++ {
		if _, err := s.AddDependency(ctx, largeTasks[(index-1)/2].ID, largeTasks[index].ID); err != nil {
			return err
		}
	}
	completedStatus := domain.FeatureStatusCompleted
	if _, err := s.UpdateFeature(ctx, completed.ID, nil, nil, nil, &completedStatus, nil); err != nil {
		return err
	}
	return nil
}

func (s *Service) createCancelledDemo(ctx context.Context) error {
	cancelled, err := s.CreateFeature(
		ctx,
		"cancelled-experiment",
		"Cancelled experiment",
		"Archived work remains available for historical inspection.",
	)
	if err != nil {
		return err
	}
	cancelledTasks, err := s.createDemoTasks(ctx, cancelled, []demoTask{
		{
			title:    "Record experiment",
			scope:    "Preserve the original hypothesis",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusCompleted,
			assignee: "Ari",
		},
		{
			title:    "Close implementation",
			scope:    "Stop implementation work",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusClosed,
			assignee: "Mika",
		},
		{
			title:  "Capture decision",
			scope:  "Document the cancellation",
			kind:   domain.TaskKindManual,
			status: domain.TaskStatusCompleted,
		},
	})
	if err != nil {
		return err
	}
	for index := 1; index < len(cancelledTasks); index++ {
		if _, err := s.AddDependency(ctx, cancelledTasks[index-1].ID, cancelledTasks[index].ID); err != nil {
			return err
		}
	}
	cancelledStatus := domain.FeatureStatusCancelled
	archived := true
	if _, err := s.UpdateFeature(ctx, cancelled.ID, nil, nil, nil, &cancelledStatus, &archived); err != nil {
		return err
	}
	return nil
}

func (s *Service) createDemoTasks(
	ctx context.Context,
	feature domain.Feature,
	values []demoTask,
) ([]domain.Task, error) {
	tasks := make([]domain.Task, len(values))
	for index, value := range values {
		task, err := s.CreateTask(ctx, feature.ID, value.title, value.scope, value.kind, value.assignee)
		if err != nil {
			return nil, err
		}
		if value.status != domain.TaskStatusAuto {
			status := value.status
			task, err = s.UpdateTask(ctx, task.ID, nil, nil, &status, nil)
			if err != nil {
				return nil, err
			}
		}
		if value.plan != "" {
			if _, err := s.UpsertImplementationPlan(ctx, task.ID, value.plan); err != nil {
				return nil, err
			}
		}
		if value.pr != nil {
			url := fmt.Sprintf(
				"https://%s/%s/%s/pull/%d",
				value.pr.Host,
				value.pr.Owner,
				value.pr.Repository,
				value.pr.Number,
			)
			attached, err := s.AttachPullRequest(ctx, task.ID, url)
			if err != nil {
				return nil, err
			}
			pr := *value.pr
			pr.TaskID = task.ID
			pr.URL = attached.URL
			pr.NodeID = fmt.Sprintf("demo:%d", pr.Number)
			pr.Author = []string{"octocat", "hubot", "monalisa"}[index%3]
			now := time.Date(2026, time.August, 30, 0, index, 0, 0, time.UTC)
			pr.GitHubUpdatedAt = &now
			pr.LastSyncedAt = &now
			if _, err := s.repository.UpsertPullRequest(ctx, pr); err != nil {
				return nil, err
			}
		}
		tasks[index] = task
	}
	return tasks, nil
}

func showcaseDemoTasks() []demoTask {
	return append(showcaseManualTasks(), showcasePullRequestTasks()...)
}

func showcaseManualTasks() []demoTask {
	return []demoTask{
		{
			title:  "Define rollout scope",
			scope:  "A ready manual task",
			kind:   domain.TaskKindManual,
			status: domain.TaskStatusNotStarted,
		},
		{
			title:    "Design dependency policy",
			scope:    "A planned task waiting on scope",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusAuto,
			assignee: "Mika",
			plan:     "1. Confirm dependency direction.\n2. Document readiness rules.\n3. Verify the graph.",
		},
		{
			title:    "Implement command surface",
			scope:    "Explicitly in-progress work",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusInProgress,
			assignee: "Ren",
		},
		{
			title:    "Verify storage boundary",
			scope:    "Explicitly completed work",
			kind:     domain.TaskKindManual,
			status:   domain.TaskStatusCompleted,
			assignee: "Ari",
		},
		{
			title:  "Retire legacy path",
			scope:  "Explicitly closed work",
			kind:   domain.TaskKindManual,
			status: domain.TaskStatusClosed,
		},
	}
}

func showcasePullRequestTasks() []demoTask {
	values := []demoTask{
		demoPullRequestTask("Merged server support", "Merged pull request", "prx-server", 101,
			domain.PullRequestStateMerged, false, domain.ReviewStateApproved, domain.MergeabilityMergeable, "Ari"),
		demoPullRequestTask("Draft WebUI shell", "Draft pull request", "prx-web", 102,
			domain.PullRequestStateOpen, true, domain.ReviewStateNone, domain.MergeabilityMergeable, "Mika"),
		demoPullRequestTask("Resolve graph conflict", "Conflicting pull request", "prx-graph", 103,
			domain.PullRequestStateOpen, false, domain.ReviewStateNone, domain.MergeabilityConflicting, "Ren"),
		demoPullRequestTask("Address review feedback", "Changes requested", "prx-cli", 104,
			domain.PullRequestStateOpen, false, domain.ReviewStateChangesRequested, domain.MergeabilityMergeable, ""),
		demoPullRequestTask("Approved configuration", "Approved pull request", "prx-config", 105,
			domain.PullRequestStateOpen, false, domain.ReviewStateApproved, domain.MergeabilityMergeable, "Ari"),
		demoPullRequestTask("Await fixture review", "Review waiting", "prx-fixtures", 106,
			domain.PullRequestStateOpen, false, domain.ReviewStateRequired, domain.MergeabilityMergeable, "Mika"),
		demoPullRequestTask("Open integration work", "Open pull request", "prx-integrations", 107,
			domain.PullRequestStateOpen, false, domain.ReviewStateNone, domain.MergeabilityMergeable, ""),
		demoPullRequestTask("Stale external state", "A failed sync preserving its last result", "prx-external", 108,
			domain.PullRequestStateUnknown, false, domain.ReviewStateUnknown, domain.MergeabilityUnknown, "Ren"),
	}
	values[7].pr.Stale = true
	values[7].pr.SyncError = "demo fixture: repository temporarily unavailable"
	return values
}

func demoPullRequestTask(
	title, scope, repository string,
	number int64,
	state domain.PullRequestState,
	draft bool,
	review domain.ReviewState,
	mergeability domain.Mergeability,
	assignee string,
) demoTask {
	return demoTask{
		title: title, scope: scope, kind: domain.TaskKindPR,
		status: domain.TaskStatusAuto, assignee: assignee,
		pr: &domain.PullRequest{
			Host: "github.com", Owner: "HappyOnigiri", Repository: repository,
			Number: number, State: state, Draft: draft,
			ReviewState: review, Mergeability: mergeability,
		},
	}
}

const demoMarkdown = `# PRX demo walkthrough

This file lives beside the temporary demo database and configuration.

- Inspect every display state in Delivery control showcase.
- Open Completed 100-task program to exercise the large graph.
- Edit any feature or task, then reload to see the change persist.
- Restart the server to restore the original demo data.
`
