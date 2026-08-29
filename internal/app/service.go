package app

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const maxMarkdownPreviewBytes = 1 << 20

// Repository is the persistence boundary used by the application service.
// Keeping this interface in the app package lets the service be tested without
// opening SQLite and leaves alternative persistence implementations free to
// satisfy the same use cases.
type Repository interface {
	CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error)
	UpdateFeature(ctx context.Context, feature domain.Feature) (domain.Feature, error)
	GetFeature(ctx context.Context, id string) (domain.Feature, error)
	GetFeatureBySlug(ctx context.Context, slug string) (domain.Feature, error)
	DeleteFeature(ctx context.Context, id string, cascade bool) error

	CreateTask(
		ctx context.Context,
		featureID, title, scope string,
		kind domain.TaskKind,
		assignee string,
	) (domain.Task, error)
	GetTask(ctx context.Context, id string) (domain.Task, error)
	UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	DeleteTask(ctx context.Context, id string, cascade bool) error

	AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error)
	RemoveDependency(ctx context.Context, blocker, blocked string) error

	UpsertPullRequest(ctx context.Context, value domain.PullRequest) (domain.PullRequest, error)
	DeletePullRequest(ctx context.Context, taskID string) error

	CreateDocument(
		ctx context.Context,
		featureID, taskID string,
		kind domain.DocumentKind,
		title, value string,
	) (domain.Document, error)
	GetDocument(ctx context.Context, id string) (domain.Document, error)
	DeleteDocument(ctx context.Context, id string) error

	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Validate(ctx context.Context) []string
}

type Service struct {
	repository Repository
	provider   githubprovider.Provider
}

func New(repository Repository, provider githubprovider.Provider) *Service {
	return &Service{repository: repository, provider: provider}
}

func (s *Service) CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	title = strings.TrimSpace(title)
	if !slugPattern.MatchString(slug) {
		return domain.Feature{}, domain.NewError(
			domain.DomainErrorCodeInvalidSlug,
			"slug must contain lowercase letters, numbers, and single hyphens",
		)
	}
	if title == "" {
		return domain.Feature{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "feature title is required")
	}
	return s.repository.CreateFeature(ctx, slug, title, strings.TrimSpace(description))
}

// UpdateFeature applies every field the caller supplied. A nil pointer means the
// field was omitted; an empty string is a request to clear it.
func (s *Service) UpdateFeature(
	ctx context.Context,
	id string,
	slug, title, description *string,
	status *domain.FeatureStatus,
	archived *bool,
) (domain.Feature, error) {
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
		return domain.Feature{}, domain.NewError(domain.DomainErrorCodeInvalidSlug, "invalid feature slug")
	}
	if !oneOf(
		feature.Status,
		domain.FeatureStatusActive,
		domain.FeatureStatusPaused,
		domain.FeatureStatusCompleted,
		domain.FeatureStatusCancelled,
	) {
		return domain.Feature{}, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid feature status")
	}
	return s.repository.UpdateFeature(ctx, feature)
}

func (s *Service) ResolveFeature(ctx context.Context, idOrSlug string) (domain.Feature, error) {
	if feature, err := s.repository.GetFeature(ctx, idOrSlug); err == nil {
		return feature, nil
	}
	if feature, err := s.repository.GetFeatureBySlug(ctx, idOrSlug); err == nil {
		return feature, nil
	}
	return domain.Feature{}, domain.NewError(domain.DomainErrorCodeNotFound, "feature %q was not found", idOrSlug)
}

func (s *Service) DeleteFeature(ctx context.Context, id string, cascade bool) error {
	feature, err := s.ResolveFeature(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteFeature(ctx, feature.ID, cascade)
}

func (s *Service) CreateTask(
	ctx context.Context,
	featureID, title, scope string,
	kind domain.TaskKind,
	assignee string,
) (domain.Task, error) {
	feature, err := s.ResolveFeature(ctx, featureID)
	if err != nil {
		return domain.Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "task title is required")
	}
	if kind == "" {
		kind = domain.TaskKindPR
	}
	if !oneOf(kind, domain.TaskKindPR, domain.TaskKindManual) {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidKind, "task kind must be pr or manual")
	}
	return s.repository.CreateTask(ctx, feature.ID, title, strings.TrimSpace(scope), kind, strings.TrimSpace(assignee))
}

// UpdateTask applies every field the caller supplied. A nil pointer means the
// field was omitted; an empty string is a request to clear it.
func (s *Service) UpdateTask(
	ctx context.Context,
	id string,
	title, scope *string,
	status *domain.TaskStatus,
	assignee *string,
) (domain.Task, error) {
	task, err := s.repository.GetTask(ctx, id)
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
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "task title is required")
	}
	if !oneOf(
		task.Status,
		domain.TaskStatusPlanned,
		domain.TaskStatusInProgress,
		domain.TaskStatusCompleted,
		domain.TaskStatusCancelled,
	) {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	}
	// PR tasks derive completion from a merged PR. Accepting completed here would
	// drop the task out of the ready queue while its dependents stay blocked,
	// since dependency satisfaction still requires a fresh merged PR.
	if task.Kind == domain.TaskKindPR && task.Status == domain.TaskStatusCompleted {
		return domain.Task{}, domain.NewError(
			domain.DomainErrorCodePRTaskCompletesOnMerge,
			"a PR task completes when its pull request is merged",
		)
	}
	return s.repository.UpdateTask(ctx, task)
}

func (s *Service) DeleteTask(ctx context.Context, id string, cascade bool) error {
	return s.repository.DeleteTask(ctx, id, cascade)
}

func (s *Service) AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error) {
	return s.repository.AddDependency(ctx, blocker, blocked)
}

func (s *Service) RemoveDependency(ctx context.Context, blocker, blocked string) error {
	return s.repository.RemoveDependency(ctx, blocker, blocked)
}

func (s *Service) AttachPullRequest(ctx context.Context, taskID, rawURL string) (domain.PullRequest, error) {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if task.Kind != domain.TaskKindPR {
		return domain.PullRequest{}, domain.NewError(
			domain.DomainErrorCodePullRequestOnManualTask,
			"manual tasks cannot have pull requests",
		)
	}
	owner, repo, number, canonical, err := githubprovider.ParsePullRequestURL(rawURL)
	if err != nil {
		return domain.PullRequest{}, domain.NewError(domain.DomainErrorCodeInvalidPullRequestURL, "%s", err)
	}
	return s.repository.UpsertPullRequest(
		ctx,
		domain.PullRequest{
			TaskID:       taskID,
			Owner:        owner,
			Repository:   repo,
			Number:       number,
			URL:          canonical,
			State:        domain.PullRequestStateUnknown,
			ReviewState:  domain.ReviewStateUnknown,
			Mergeability: domain.MergeabilityUnknown,
			Stale:        true,
		},
	)
}

func (s *Service) DetachPullRequest(ctx context.Context, taskID string) error {
	return s.repository.DeletePullRequest(ctx, taskID)
}

func (s *Service) AddDocument(
	ctx context.Context,
	featureID, taskID string,
	kind domain.DocumentKind,
	title, value string,
) (domain.Document, error) {
	if (featureID == "") == (taskID == "") {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"set exactly one of feature_id or task_id",
		)
	}
	if !oneOf(kind, domain.DocumentKindURL, domain.DocumentKindMarkdownPath) {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidDocumentKind,
			"document kind must be url or markdown_path",
		)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.Document{}, domain.NewError(domain.DomainErrorCodeInvalidDocument, "document value is required")
	}
	if kind == domain.DocumentKindURL {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return domain.Document{}, domain.NewError(
				domain.DomainErrorCodeInvalidDocumentURL,
				"document URL must use http or https",
			)
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
		if _, err := s.repository.GetTask(ctx, taskID); err != nil {
			return domain.Document{}, err
		}
	}
	return s.repository.CreateDocument(ctx, featureID, taskID, kind, strings.TrimSpace(title), value)
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.repository.DeleteDocument(ctx, id)
}

// ReadMarkdownDocument only reads paths that were explicitly registered as a
// Markdown document. The size limit keeps a preview request from consuming an
// unbounded amount of memory in the server or browser.
func (s *Service) ReadMarkdownDocument(ctx context.Context, id string) (string, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return "", err
	}
	if document.Kind != domain.DocumentKindMarkdownPath {
		return "", domain.NewError(domain.DomainErrorCodeInvalidDocumentKind, "document is not a Markdown path")
	}
	file, err := os.Open(document.Value)
	if err != nil {
		return "", domain.NewError(domain.DomainErrorCodeDocumentReadFailed, "could not read the Markdown file")
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxMarkdownPreviewBytes+1))
	if err != nil {
		return "", domain.NewError(domain.DomainErrorCodeDocumentReadFailed, "could not read the Markdown file")
	}
	if len(content) > maxMarkdownPreviewBytes {
		return "", domain.NewError(domain.DomainErrorCodeDocumentTooLarge, "Markdown preview is limited to 1 MiB")
	}
	return string(content), nil
}

func (s *Service) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	snapshot, err := s.repository.Snapshot(ctx)
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
		if task.DisplayState == domain.TaskDisplayStateReviewWaiting {
			snapshot.ReviewWaitingTasks = append(snapshot.ReviewWaitingTasks, task)
		}
		if task.DisplayState == domain.TaskDisplayStateConflict {
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
		if task.DisplayState == domain.TaskDisplayStateReviewWaiting {
			feature.ReviewWaitingCount++
		}
		if task.DisplayState == domain.TaskDisplayStateConflict {
			feature.ConflictCount++
		}
		if task.DisplayState == domain.TaskDisplayStateMerged {
			feature.MergedCount++
		}
	}
	return snapshot, nil
}

func (s *Service) Sync(ctx context.Context, featureID, taskID string) (succeeded, failed int, err error) {
	if s.provider == nil {
		return 0, 0, domain.NewError(domain.DomainErrorCodeGitHubAuth, "GitHub provider is not configured")
	}
	snapshot, err := s.repository.Snapshot(ctx)
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
			if _, persistErr := s.repository.UpsertPullRequest(ctx, pr); persistErr != nil {
				return succeeded, failed, persistErr
			}
			failed++
			continue
		}
		updated.TaskID = pr.TaskID
		if _, err := s.repository.UpsertPullRequest(ctx, updated); err != nil {
			return succeeded, failed, err
		}
		succeeded++
	}
	return succeeded, failed, nil
}

func (s *Service) Validate(ctx context.Context) []string { return s.repository.Validate(ctx) }

func (s *Service) SeedDemo(ctx context.Context, slug string, count int) error {
	if slug == "" {
		slug = "demo-roadmap"
	}
	if count < 1 {
		count = 8
	}
	// Seeding is idempotent: rerunning it reuses whatever already exists so the
	// command in the README can be repeated without leaving partial data behind.
	feature, err := s.repository.GetFeatureBySlug(ctx, slug)
	if err != nil {
		feature, err = s.CreateFeature(
			ctx,
			slug,
			fmt.Sprintf("Cross-repository launch · %d nodes", count),
			"A representative branching and merging delivery graph.",
		)
		if err != nil {
			return err
		}
	}
	snapshot, err := s.repository.Snapshot(ctx)
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
			task, err = s.CreateTask(
				ctx,
				feature.ID,
				title,
				"Implement and verify repository boundary",
				domain.TaskKindPR,
				[]string{"Ari", "Mika", "Ren"}[i%3],
			)
			if err != nil {
				return err
			}
		}
		tasks[i] = task
		if !linkedTasks[task.ID] {
			if _, err := s.AttachPullRequest(
				ctx,
				task.ID,
				fmt.Sprintf("https://github.com/HappyOnigiri/%s/pull/%d", slug, i+1),
			); err != nil {
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

func oneOf[T comparable](value T, values ...T) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
