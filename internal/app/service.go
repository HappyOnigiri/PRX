package app

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	store    *store.Store
	provider githubprovider.Provider
}

func New(database *store.Store, provider githubprovider.Provider) *Service {
	return &Service{store: database, provider: provider}
}

func (s *Service) CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	title = strings.TrimSpace(title)
	if !slugPattern.MatchString(slug) {
		return domain.Feature{}, domain.NewError("invalid_slug", "slug must contain lowercase letters, numbers, and single hyphens")
	}
	if title == "" {
		return domain.Feature{}, domain.NewError("invalid_title", "feature title is required")
	}
	return s.store.CreateFeature(ctx, slug, title, strings.TrimSpace(description))
}

// UpdateFeature applies every field the caller supplied. A nil pointer means the
// field was omitted; an empty string is a request to clear it.
func (s *Service) UpdateFeature(ctx context.Context, id string, slug, title, description, status *string, archived *bool) (domain.Feature, error) {
	feature, err := s.ResolveFeature(ctx, id)
	if err != nil {
		return domain.Feature{}, err
	}
	if slug != nil {
		feature.Slug = strings.TrimSpace(strings.ToLower(*slug))
	}
	if title != nil {
		feature.Title = strings.TrimSpace(*title)
	}
	if description != nil {
		feature.Description = *description
	}
	if status != nil && *status != "" {
		feature.Status = *status
	}
	if archived != nil {
		feature.Archived = *archived
	}
	if !slugPattern.MatchString(feature.Slug) {
		return domain.Feature{}, domain.NewError("invalid_slug", "invalid feature slug")
	}
	if !oneOf(feature.Status, "active", "paused", "completed", "cancelled") {
		return domain.Feature{}, domain.NewError("invalid_status", "invalid feature status")
	}
	return s.store.UpdateFeature(ctx, feature)
}

func (s *Service) ResolveFeature(ctx context.Context, idOrSlug string) (domain.Feature, error) {
	if feature, err := s.store.GetFeature(ctx, idOrSlug); err == nil {
		return feature, nil
	}
	if feature, err := s.store.GetFeatureBySlug(ctx, idOrSlug); err == nil {
		return feature, nil
	}
	return domain.Feature{}, domain.NewError("not_found", "feature %q was not found", idOrSlug)
}

func (s *Service) DeleteFeature(ctx context.Context, id string, cascade bool) error {
	feature, err := s.ResolveFeature(ctx, id)
	if err != nil {
		return err
	}
	return s.store.DeleteFeature(ctx, feature.ID, cascade)
}

func (s *Service) CreateTask(ctx context.Context, featureID, title, scope, kind, assignee string) (domain.Task, error) {
	feature, err := s.ResolveFeature(ctx, featureID)
	if err != nil {
		return domain.Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, domain.NewError("invalid_title", "task title is required")
	}
	if kind == "" {
		kind = domain.TaskKindPR
	}
	if !oneOf(kind, domain.TaskKindPR, domain.TaskKindManual) {
		return domain.Task{}, domain.NewError("invalid_kind", "task kind must be pr or manual")
	}
	return s.store.CreateTask(ctx, feature.ID, title, strings.TrimSpace(scope), kind, strings.TrimSpace(assignee))
}

// UpdateTask applies every field the caller supplied. A nil pointer means the
// field was omitted; an empty string is a request to clear it.
func (s *Service) UpdateTask(ctx context.Context, id string, title, scope, status, assignee *string) (domain.Task, error) {
	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	if title != nil {
		task.Title = strings.TrimSpace(*title)
	}
	if scope != nil {
		task.Scope = *scope
	}
	if status != nil && *status != "" {
		task.Status = *status
	}
	if assignee != nil {
		task.Assignee = *assignee
	}
	if task.Title == "" {
		return domain.Task{}, domain.NewError("invalid_title", "task title is required")
	}
	if !oneOf(task.Status, domain.TaskPlanned, domain.TaskInProgress, domain.TaskCompleted, domain.TaskCancelled) {
		return domain.Task{}, domain.NewError("invalid_status", "invalid task status")
	}
	// PR tasks derive completion from a merged PR. Accepting completed here would
	// drop the task out of the ready queue while its dependents stay blocked,
	// since dependency satisfaction still requires a fresh merged PR.
	if task.Kind == domain.TaskKindPR && task.Status == domain.TaskCompleted {
		return domain.Task{}, domain.NewError("pr_task_completes_on_merge", "a PR task completes when its pull request is merged")
	}
	return s.store.UpdateTask(ctx, task)
}

func (s *Service) DeleteTask(ctx context.Context, id string, cascade bool) error {
	return s.store.DeleteTask(ctx, id, cascade)
}
func (s *Service) AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error) {
	return s.store.AddDependency(ctx, blocker, blocked)
}
func (s *Service) RemoveDependency(ctx context.Context, blocker, blocked string) error {
	return s.store.RemoveDependency(ctx, blocker, blocked)
}

func (s *Service) AttachPullRequest(ctx context.Context, taskID, rawURL string) (domain.PullRequest, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if task.Kind != domain.TaskKindPR {
		return domain.PullRequest{}, domain.NewError("pull_request_on_manual_task", "manual tasks cannot have pull requests")
	}
	owner, repo, number, canonical, err := githubprovider.ParsePullRequestURL(rawURL)
	if err != nil {
		return domain.PullRequest{}, domain.NewError("invalid_pull_request_url", "%s", err)
	}
	return s.store.UpsertPullRequest(ctx, domain.PullRequest{TaskID: taskID, Owner: owner, Repository: repo, Number: number, URL: canonical, State: "unknown", ReviewState: "unknown", Mergeability: "unknown", Stale: true})
}

func (s *Service) DetachPullRequest(ctx context.Context, taskID string) error {
	return s.store.DeletePullRequest(ctx, taskID)
}

func (s *Service) AddDocument(ctx context.Context, featureID, taskID, kind, title, value string) (domain.Document, error) {
	if (featureID == "") == (taskID == "") {
		return domain.Document{}, domain.NewError("invalid_parent", "set exactly one of feature_id or task_id")
	}
	if !oneOf(kind, "url", "markdown_path") {
		return domain.Document{}, domain.NewError("invalid_document_kind", "document kind must be url or markdown_path")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.Document{}, domain.NewError("invalid_document", "document value is required")
	}
	if kind == "url" {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return domain.Document{}, domain.NewError("invalid_document_url", "document URL must use http or https")
		}
	}
	if featureID != "" {
		feature, err := s.ResolveFeature(ctx, featureID)
		if err != nil {
			return domain.Document{}, err
		}
		featureID = feature.ID
	}
	if taskID != "" {
		if _, err := s.store.GetTask(ctx, taskID); err != nil {
			return domain.Document{}, err
		}
	}
	return s.store.CreateDocument(ctx, featureID, taskID, kind, strings.TrimSpace(title), value)
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.store.DeleteDocument(ctx, id)
}

func (s *Service) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	snapshot, err := s.store.Snapshot(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snapshot.Tasks = domain.Derive(snapshot.Tasks, snapshot.Dependencies, snapshot.PullRequests)
	taskIndex := map[string]int{}
	for i, task := range snapshot.Tasks {
		taskIndex[task.ID] = i
		if task.Ready {
			snapshot.ReadyTasks = append(snapshot.ReadyTasks, task)
		}
		if task.DisplayState == "review_waiting" {
			snapshot.ReviewWaitingTasks = append(snapshot.ReviewWaitingTasks, task)
		}
		if task.DisplayState == "conflict" {
			snapshot.ConflictTasks = append(snapshot.ConflictTasks, task)
		}
	}
	for _, pr := range snapshot.PullRequests {
		if pr.Stale || pr.SyncError != "" {
			if i, ok := taskIndex[pr.TaskID]; ok {
				snapshot.StaleTasks = append(snapshot.StaleTasks, snapshot.Tasks[i])
			}
		}
	}
	featureIndex := map[string]int{}
	for i := range snapshot.Features {
		featureIndex[snapshot.Features[i].ID] = i
	}
	for _, task := range snapshot.Tasks {
		// A task whose feature is missing means the database has lost referential
		// integrity; skip it rather than crediting feature zero or indexing past
		// the end of an empty slice.
		i, ok := featureIndex[task.FeatureID]
		if !ok {
			continue
		}
		feature := &snapshot.Features[i]
		feature.TaskCount++
		if task.Ready {
			feature.ReadyCount++
		}
		if task.DisplayState == "review_waiting" {
			feature.ReviewWaitingCount++
		}
		if task.DisplayState == "conflict" {
			feature.ConflictCount++
		}
		if task.DisplayState == "merged" {
			feature.MergedCount++
		}
	}
	return snapshot, nil
}

func (s *Service) Sync(ctx context.Context, featureID, taskID string) (succeeded, failed int, err error) {
	if s.provider == nil {
		return 0, 0, domain.NewError("github_auth", "GitHub provider is not configured")
	}
	snapshot, err := s.store.Snapshot(ctx)
	if err != nil {
		return 0, 0, err
	}
	featureResolved := ""
	if featureID != "" {
		feature, err := s.ResolveFeature(ctx, featureID)
		if err != nil {
			return 0, 0, err
		}
		featureResolved = feature.ID
	}
	taskFeature := map[string]string{}
	for _, task := range snapshot.Tasks {
		taskFeature[task.ID] = task.FeatureID
	}
	for _, pr := range snapshot.PullRequests {
		if taskID != "" && pr.TaskID != taskID {
			continue
		}
		if featureResolved != "" && taskFeature[pr.TaskID] != featureResolved {
			continue
		}
		updated, fetchErr := s.provider.Fetch(ctx, pr)
		if fetchErr != nil {
			now := time.Now().UTC()
			pr.LastSyncedAt = &now
			pr.SyncError = fetchErr.Error()
			pr.Stale = true
			if _, persistErr := s.store.UpsertPullRequest(ctx, pr); persistErr != nil {
				return succeeded, failed, persistErr
			}
			failed++
			continue
		}
		updated.TaskID = pr.TaskID
		if _, err := s.store.UpsertPullRequest(ctx, updated); err != nil {
			return succeeded, failed, err
		}
		succeeded++
	}
	return succeeded, failed, nil
}

func (s *Service) Validate(ctx context.Context) []string { return s.store.Validate(ctx) }

func (s *Service) SeedDemo(ctx context.Context, slug string, count int) error {
	if slug == "" {
		slug = "demo-roadmap"
	}
	if count < 1 {
		count = 8
	}
	// Seeding is idempotent: rerunning it reuses whatever already exists so the
	// command in the README can be repeated without leaving partial data behind.
	feature, err := s.store.GetFeatureBySlug(ctx, slug)
	if err != nil {
		feature, err = s.CreateFeature(ctx, slug, fmt.Sprintf("Cross-repository launch · %d nodes", count), "A representative branching and merging delivery graph.")
		if err != nil {
			return err
		}
	}
	snapshot, err := s.store.Snapshot(ctx)
	if err != nil {
		return err
	}
	existingTasks := map[string]domain.Task{}
	for _, task := range snapshot.Tasks {
		if task.FeatureID == feature.ID {
			existingTasks[task.Title] = task
		}
	}
	linkedTasks := map[string]bool{}
	for _, pr := range snapshot.PullRequests {
		linkedTasks[pr.TaskID] = true
	}
	existingDeps := map[string]bool{}
	for _, dep := range snapshot.Dependencies {
		existingDeps[dep.BlockerTaskID+"→"+dep.BlockedTaskID] = true
	}
	tasks := make([]domain.Task, count)
	for i := 0; i < count; i++ {
		title := fmt.Sprintf("Delivery slice %02d", i+1)
		task, ok := existingTasks[title]
		if !ok {
			task, err = s.CreateTask(ctx, feature.ID, title, "Implement and verify repository boundary", domain.TaskKindPR, []string{"Ari", "Mika", "Ren"}[i%3])
			if err != nil {
				return err
			}
		}
		tasks[i] = task
		if !linkedTasks[task.ID] {
			if _, err := s.AttachPullRequest(ctx, task.ID, fmt.Sprintf("https://github.com/HappyOnigiri/%s/pull/%d", slug, i+1)); err != nil {
				return err
			}
		}
	}
	for i := 1; i < count; i++ {
		blocker := (i - 1) / 2
		if existingDeps[tasks[blocker].ID+"→"+tasks[i].ID] {
			continue
		}
		if _, err := s.AddDependency(ctx, tasks[blocker].ID, tasks[i].ID); err != nil {
			return err
		}
	}
	_, _, err = s.Sync(ctx, feature.ID, "")
	return err
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func SortTasks(tasks []domain.Task) {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Title < tasks[j].Title })
}
