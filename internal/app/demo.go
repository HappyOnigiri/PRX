package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
)

type demoTask struct {
	title    string
	scope    string
	status   domain.TaskStatus
	assignee string
	plan     string
	pr       *domain.PullRequest
}

func demoPullRequestURL(pr domain.PullRequest) string {
	return fmt.Sprintf("https://%s/%s/%s/pull/%d", pr.Host, pr.Owner, pr.Repository, pr.Number)
}

func demoPullRequestAuthor(index int) string {
	return []string{"octocat", "hubot", "monalisa"}[index%3]
}

// WriteDemoFixture records the demo pull-request states as a GitHub fixture
// file. Serving the demo through this fixture instead of the generated preset
// keeps a synchronization from replacing the states the demo exists to show.
func WriteDemoFixture(path string) error {
	fixtures := map[string]githubprovider.Fixture{}
	for _, tasks := range [][]demoTask{showcaseDemoTasks(), completedDemoTasks()} {
		for index, value := range tasks {
			if value.pr == nil {
				continue
			}
			fixtures[demoPullRequestURL(*value.pr)] = githubprovider.Fixture{
				State:        value.pr.State,
				Draft:        value.pr.Draft,
				ReviewState:  value.pr.ReviewState,
				Mergeability: value.pr.Mergeability,
				Author:       demoPullRequestAuthor(index),
				Error:        value.pr.SyncError,
			}
		}
	}
	body, err := json.Marshal(fixtures)
	if err != nil {
		return fmt.Errorf("encode demo GitHub fixture: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write demo GitHub fixture: %w", err)
	}
	return nil
}

// InitializeDemo populates a new, empty repository with the built-in demo.
func (s *Service) InitializeDemo(ctx context.Context, markdownPath string) error {
	if err := os.WriteFile(markdownPath, []byte(demoMarkdown), 0o600); err != nil {
		return fmt.Errorf("write demo Markdown: %w", err)
	}
	platform, err := s.CreateProject(
		ctx,
		"Delivery platform",
		"Two related features and the brief they share.",
	)
	if err != nil {
		return err
	}
	if _, err := s.AddDocument(
		ctx,
		domain.DocumentParent{ProjectID: platform.ID},
		domain.DocumentKindURL,
		"Platform charter",
		"https://example.com/prx-demo/platform-charter",
		"",
		false,
	); err != nil {
		return err
	}
	sunset, err := s.CreateProject(
		ctx,
		"Sunset initiative",
		"An archived project: everything inside it is read-only.",
	)
	if err != nil {
		return err
	}
	for _, initialize := range []func(context.Context) error{
		func(ctx context.Context) error { return s.createShowcaseDemo(ctx, markdownPath, platform.ID) },
		func(ctx context.Context) error { return s.createPausedDemo(ctx, platform.ID) },
		s.createCompletedDemo,
		func(ctx context.Context) error { return s.createCancelledDemo(ctx, sunset.ID) },
		func(ctx context.Context) error { return s.createSunsetPostmortemDemo(ctx, sunset.ID) },
	} {
		if err := initialize(ctx); err != nil {
			return err
		}
	}
	// The archive is applied last: an archived project refuses the writes that
	// build the work inside it.
	archived := true
	if _, err := s.UpdateProject(ctx, sunset.ID, domain.ProjectUpdate{Archived: &archived}); err != nil {
		return err
	}
	if issues := s.Validate(ctx); len(issues) > 0 {
		return fmt.Errorf("validate demo data: %v", issues)
	}
	return nil
}

func (s *Service) createShowcaseDemo(ctx context.Context, markdownPath, projectID string) error {
	showcase, err := s.CreateFeature(
		ctx,
		"Delivery control showcase",
		"Every task and pull-request state in one cross-repository graph.",
		projectID,
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
		domain.DocumentParent{FeatureID: showcase.ID},
		domain.DocumentKindURL,
		"Product brief",
		"https://example.com/prx-demo/product-brief",
		"",
		false,
	); err != nil {
		return err
	}
	if _, err := s.AddDocument(
		ctx,
		domain.DocumentParent{FeatureID: showcase.ID},
		domain.DocumentKindLocalFile,
		"Demo walkthrough",
		markdownPath,
		"",
		false,
	); err != nil {
		return err
	}
	return nil
}

func (s *Service) createPausedDemo(ctx context.Context, projectID string) error {
	paused, err := s.CreateFeature(
		ctx,
		"Paused rollout",
		"A small branch-and-merge graph with ready and blocked manual work.",
		projectID,
	)
	if err != nil {
		return err
	}
	pausedStatus := domain.FeatureStatusPaused
	paused, err = s.UpdateFeature(ctx, paused.ID, domain.FeatureUpdate{Status: &pausedStatus})
	if err != nil {
		return err
	}
	pausedTasks, err := s.createDemoTasks(ctx, paused, []demoTask{
		{
			title:    "Foundation complete",
			scope:    "Shared rollout foundation",
			status:   domain.TaskStatusCompleted,
			assignee: "Bob",
		},
		{
			title:  "Prepare operator guide",
			scope:  "Document the rollout",
			status: domain.TaskStatusNotStarted,
		},
		{
			title:    "Run limited rollout",
			scope:    "Exercise the first cohort",
			status:   domain.TaskStatusInProgress,
			assignee: "Carol",
		},
		{
			title:    "Complete general rollout",
			scope:    "Merge both rollout branches",
			status:   domain.TaskStatusNotStarted,
			assignee: "Alice",
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
		"Completed 100-task program",
		"A large balanced graph for checking layout and navigation at scale.",
		"",
	)
	if err != nil {
		return err
	}
	largeTasks, err := s.createDemoTasks(ctx, completed, completedDemoTasks())
	if err != nil {
		return err
	}
	// Every task carries a merged pull request, so the feature keeps its
	// automatic status and demonstrates the derived completion itself.
	for index := 1; index < len(largeTasks); index++ {
		if _, err := s.AddDependency(ctx, largeTasks[(index-1)/2].ID, largeTasks[index].ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createCancelledDemo(ctx context.Context, projectID string) error {
	cancelled, err := s.CreateFeature(
		ctx,
		"Cancelled experiment",
		"Archived work remains available for historical inspection.",
		projectID,
	)
	if err != nil {
		return err
	}
	cancelledTasks, err := s.createDemoTasks(ctx, cancelled, []demoTask{
		{
			title:    "Record experiment",
			scope:    "Preserve the original hypothesis",
			status:   domain.TaskStatusCompleted,
			assignee: "Alice",
		},
		{
			title:    "Close implementation",
			scope:    "Stop implementation work",
			status:   domain.TaskStatusClosed,
			assignee: "Bob",
		},
		{
			title:  "Capture decision",
			scope:  "Document the cancellation",
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
	if _, err := s.UpdateFeature(
		ctx,
		cancelled.ID,
		domain.FeatureUpdate{Status: &cancelledStatus, Archived: &archived},
	); err != nil {
		return err
	}
	return nil
}

// createSunsetPostmortemDemo puts a feature that is not archived itself inside
// the project the demo archives, which is the state the walkthrough points at:
// the read-only presentation comes from the project alone.
func (s *Service) createSunsetPostmortemDemo(ctx context.Context, projectID string) error {
	postmortem, err := s.CreateFeature(
		ctx,
		"Sunset postmortem",
		"Not archived itself: the archived project around it makes it read-only.",
		projectID,
	)
	if err != nil {
		return err
	}
	_, err = s.createDemoTasks(ctx, postmortem, []demoTask{
		{
			title:    "Collect lessons",
			scope:    "Summarize what the experiment showed",
			status:   domain.TaskStatusCompleted,
			assignee: "Alice",
		},
		{
			title:    "Share the summary",
			scope:    "Hand the summary to the teams that asked for it",
			status:   domain.TaskStatusCompleted,
			assignee: "Bob",
		},
	})
	return err
}

func (s *Service) createDemoTasks(
	ctx context.Context,
	feature domain.Feature,
	values []demoTask,
) ([]domain.Task, error) {
	tasks := make([]domain.Task, len(values))
	for index, value := range values {
		task, err := s.CreateTask(ctx, feature.ID, value.title, value.scope, value.assignee)
		if err != nil {
			return nil, err
		}
		if value.status != domain.TaskStatusNotStarted {
			status := value.status
			task, err = s.UpdateTask(ctx, task.ID, nil, nil, &status, nil)
			if err != nil {
				return nil, err
			}
		}
		if value.plan != "" {
			plan := domain.Document{Kind: domain.DocumentKindMarkdown, Content: value.plan}
			if _, err := s.UpsertImplementationPlan(ctx, task.ID, plan); err != nil {
				return nil, err
			}
		}
		if value.pr != nil {
			attached, err := s.AttachPullRequest(ctx, task.ID, demoPullRequestURL(*value.pr))
			if err != nil {
				return nil, err
			}
			pr := *value.pr
			pr.TaskID = task.ID
			pr.URL = attached.URL
			pr.NodeID = fmt.Sprintf("demo:%d", pr.Number)
			pr.Author = demoPullRequestAuthor(index)
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

func completedDemoTasks() []demoTask {
	tasks := make([]demoTask, 100)
	for index := range tasks {
		tasks[index] = demoTask{
			title:    fmt.Sprintf("Completed delivery slice %03d", index+1),
			scope:    "A completed repository delivery boundary",
			status:   domain.TaskStatusNotStarted,
			assignee: []string{"Alice", "Bob", "Carol"}[index%3],
			pr: &domain.PullRequest{
				Host: "github.com", Owner: "prx-demo", Repository: "prx-demo-scale",
				Number: int64(1001 + index), State: domain.PullRequestStateMerged,
				ReviewState: domain.ReviewStateApproved, Mergeability: domain.MergeabilityMergeable,
			},
		}
	}
	return tasks
}

func showcaseManualTasks() []demoTask {
	return []demoTask{
		{
			title:  "Define rollout scope",
			scope:  "A ready manual task",
			status: domain.TaskStatusNotStarted,
		},
		{
			title:    "Design dependency policy",
			scope:    "A planned task waiting on scope",
			status:   domain.TaskStatusNotStarted,
			assignee: "Bob",
			plan:     "1. Confirm dependency direction.\n2. Document readiness rules.\n3. Verify the graph.",
		},
		{
			title:    "Implement command surface",
			scope:    "Explicitly in-progress work",
			status:   domain.TaskStatusInProgress,
			assignee: "Carol",
		},
		{
			title:    "Verify storage boundary",
			scope:    "Explicitly completed work",
			status:   domain.TaskStatusCompleted,
			assignee: "Alice",
		},
		{
			title:  "Retire legacy path",
			scope:  "Explicitly closed work",
			status: domain.TaskStatusClosed,
		},
	}
}

func showcasePullRequestTasks() []demoTask {
	values := []demoTask{
		demoPullRequestTask("Merged server support", "Merged pull request", "prx-server", 101,
			domain.PullRequestStateMerged, false, domain.ReviewStateApproved, domain.MergeabilityMergeable, "Alice"),
		demoPullRequestTask("Draft WebUI shell", "Draft pull request", "prx-web", 102,
			domain.PullRequestStateOpen, true, domain.ReviewStateNone, domain.MergeabilityMergeable, "Bob"),
		demoPullRequestTask("Resolve graph conflict", "Conflicting pull request", "prx-graph", 103,
			domain.PullRequestStateOpen, false, domain.ReviewStateNone, domain.MergeabilityConflicting, "Carol"),
		demoPullRequestTask("Address review feedback", "Changes requested", "prx-cli", 104,
			domain.PullRequestStateOpen, false, domain.ReviewStateChangesRequested, domain.MergeabilityMergeable, ""),
		demoPullRequestTask("Approved configuration", "Approved pull request", "prx-config", 105,
			domain.PullRequestStateOpen, false, domain.ReviewStateApproved, domain.MergeabilityMergeable, "Alice"),
		demoPullRequestTask("Await fixture review", "Review waiting", "prx-fixtures", 106,
			domain.PullRequestStateOpen, false, domain.ReviewStateRequired, domain.MergeabilityMergeable, "Bob"),
		demoPullRequestTask("Open integration work", "Open pull request", "prx-integrations", 107,
			domain.PullRequestStateOpen, false, domain.ReviewStateNone, domain.MergeabilityMergeable, ""),
		demoPullRequestTask("Stale external state", "A failed sync preserving its last result", "prx-external", 108,
			domain.PullRequestStateUnknown, false, domain.ReviewStateUnknown, domain.MergeabilityUnknown, "Carol"),
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
		title: title, scope: scope,
		status: domain.TaskStatusNotStarted, assignee: assignee,
		pr: &domain.PullRequest{
			Host: "github.com", Owner: "prx-demo", Repository: repository,
			Number: number, State: state, Draft: draft,
			ReviewState: review, Mergeability: mergeability,
		},
	}
}

const demoMarkdown = `# PRX demo walkthrough

This file lives beside the temporary demo database and configuration.

- Inspect every display state in Delivery control showcase.
- Open Projects to see how Delivery platform groups two features.
- Open Sunset initiative to see an archived project make its features read-only.
- Open Completed 100-task program to exercise the large graph.
- Edit any feature or task, then reload to see the change persist.
- Restart the server to restore the original demo data.
`
