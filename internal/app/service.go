package app

import (
	"context"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const (
	maxDocumentContentBytes = 1 << 20
)

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

	GetImplementationPlan(ctx context.Context, taskID string) (domain.Document, error)
	UpsertImplementationPlan(ctx context.Context, taskID string, document domain.Document) (domain.Document, error)
	DeleteImplementationPlan(ctx context.Context, taskID string) error

	AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error)
	RemoveDependency(ctx context.Context, blocker, blocked string) error

	UpsertPullRequest(ctx context.Context, value domain.PullRequest) (domain.PullRequest, error)
	DeletePullRequest(ctx context.Context, taskID string) error

	CreateDocument(
		ctx context.Context,
		featureID, taskID string,
		kind domain.DocumentKind,
		title, locator, content string,
		isImplementationPlan bool,
	) (domain.Document, error)
	GetDocument(ctx context.Context, id string) (domain.Document, error)
	UpdateDocument(
		ctx context.Context,
		id string,
		title *string,
		source *domain.Document,
		isImplementationPlan *bool,
	) (domain.Document, error)
	DeleteDocument(ctx context.Context, id string) error

	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Validate(ctx context.Context) []string
}

// GitHubAuthCache is implemented by the SQLite repository when live GitHub
// synchronization is enabled. Keeping it optional preserves the small
// repository fakes used by application-level tests.
type GitHubAuthCache interface {
	GetGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository string) (string, bool, error)
	UpsertGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository, authMethodID string) error
	DeleteGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository string) error
}

type GitHubSyncStateRepository interface {
	GitHubSyncState(ctx context.Context) (domain.GitHubSyncState, error)
	AcquireGitHubAutoSync(ctx context.Context, runID string, attemptedAt time.Time, dueBeforeUnix int64) (bool, error)
	StartGitHubSync(ctx context.Context, runID string, attemptedAt time.Time) error
	// CompleteGitHubSync reports whether the run still owned the shared state.
	// A concurrent refresh overwrites the run identifier, and the caller must
	// not present the state it reads back afterwards as its own outcome.
	CompleteGitHubSync(
		ctx context.Context,
		runID string,
		completedAt time.Time,
		succeeded, failed int,
		runError string,
	) (bool, error)
}

type Service struct {
	repository  Repository
	provider    githubprovider.Provider
	configStore *config.Store
	now         func() time.Time
	// processInfo is written once while the service is wired and read only by
	// the diagnostic report.
	processInfo ProcessInfo
	// serveEndpoint is the only mutable field: the listen address is known after
	// the listener is bound, and HTTP handlers read it afterwards.
	serveEndpoint atomic.Pointer[serveEndpoint]
}

func New(repository Repository, provider githubprovider.Provider) *Service {
	return &Service{repository: repository, provider: provider, now: func() time.Time { return time.Now().UTC() }}
}

func NewWithConfig(repository Repository, provider githubprovider.Provider, configStore *config.Store) *Service {
	return &Service{
		repository: repository, provider: provider, configStore: configStore,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) ConfigStore() *config.Store { return s.configStore }

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
		domain.FeatureStatusAuto,
		domain.FeatureStatusActive,
		domain.FeatureStatusPaused,
		domain.FeatureStatusCompleted,
		domain.FeatureStatusCancelled,
	) {
		return domain.Feature{}, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid feature status")
	}
	return s.repository.UpdateFeature(ctx, feature)
}

// ResolveFeature only falls through to the next lookup when the previous one
// reported a missing row, so a storage failure such as a locked database keeps
// its own cause instead of being reported as a missing feature.
func (s *Service) ResolveFeature(ctx context.Context, idOrSlug string) (domain.Feature, error) {
	feature, err := s.repository.GetFeature(ctx, idOrSlug)
	if err == nil {
		return feature, nil
	}
	if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		return domain.Feature{}, err
	}
	feature, err = s.repository.GetFeatureBySlug(ctx, idOrSlug)
	if err == nil {
		return feature, nil
	}
	if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		return domain.Feature{}, err
	}
	return domain.Feature{}, domain.NewError(domain.DomainErrorCodeNotFound, "feature %q was not found", idOrSlug)
}

// GetNode resolves a public feature ID or slug or a public task ID without
// exposing the storage UUID or requiring callers to choose the resource first.
func (s *Service) GetNode(ctx context.Context, id string) (any, error) {
	if strings.HasPrefix(id, "T-") {
		return s.repository.GetTask(ctx, id)
	}
	feature, err := s.ResolveFeature(ctx, id)
	if err == nil {
		return feature, nil
	}
	if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		return nil, err
	}
	return nil, domain.NewError(domain.DomainErrorCodeNotFound, "feature or task %q was not found", id)
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
		domain.TaskStatusAuto,
		domain.TaskStatusNotStarted,
		domain.TaskStatusInProgress,
		domain.TaskStatusCompleted,
		domain.TaskStatusClosed,
	) {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	}
	return s.repository.UpdateTask(ctx, task)
}

func (s *Service) DeleteTask(ctx context.Context, id string, cascade bool) error {
	return s.repository.DeleteTask(ctx, id, cascade)
}

func (s *Service) GetImplementationPlan(ctx context.Context, taskID string) (domain.Document, error) {
	if _, err := s.repository.GetTask(ctx, taskID); err != nil {
		return domain.Document{}, err
	}
	return s.repository.GetImplementationPlan(ctx, taskID)
}

func (s *Service) UpsertImplementationPlan(
	ctx context.Context,
	taskID string,
	document domain.Document,
) (domain.Document, error) {
	if _, err := s.repository.GetTask(ctx, taskID); err != nil {
		return domain.Document{}, err
	}
	document.TaskID = taskID
	document.FeatureID = ""
	document.IsImplementationPlan = true
	if err := validateDocumentSource(&document, true); err != nil {
		return domain.Document{}, err
	}
	return s.repository.UpsertImplementationPlan(ctx, taskID, document)
}

func (s *Service) DeleteImplementationPlan(ctx context.Context, taskID string) error {
	if _, err := s.repository.GetTask(ctx, taskID); err != nil {
		return err
	}
	return s.repository.DeleteImplementationPlan(ctx, taskID)
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
	host, owner, repo, number, parsedCanonical, err := githubprovider.ParsePullRequestURLDetails(rawURL)
	if err != nil {
		return domain.PullRequest{}, domain.NewError(domain.DomainErrorCodeInvalidPullRequestURL, "%s", err)
	}
	canonical := parsedCanonical
	if s.provider == nil && s.configStore != nil {
		settings, loadErr := s.configStore.Load()
		if loadErr != nil {
			return domain.PullRequest{}, configDomainError(loadErr)
		}
		hostConfig, ok := settings.HostFor(host)
		if !ok {
			return domain.PullRequest{}, domain.NewError(
				domain.DomainErrorCodeInvalidPullRequestURL,
				"GitHub host %q is not configured",
				host,
			)
		}
		host = hostConfig.Host
		canonical = canonicalPullRequestURL(hostConfig, owner, repo, number)
	} else if host == "" {
		host = "github.com"
	}
	return s.repository.UpsertPullRequest(
		ctx,
		domain.PullRequest{
			TaskID:       taskID,
			Host:         host,
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
	title, locator, content string,
	isImplementationPlan bool,
) (domain.Document, error) {
	if (featureID == "") == (taskID == "") {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"set exactly one of feature_id or task_id",
		)
	}
	document := domain.Document{
		FeatureID: featureID, TaskID: taskID, Kind: kind, Title: strings.TrimSpace(title),
		Locator: locator, Content: content, IsImplementationPlan: isImplementationPlan,
	}
	if err := validateDocumentSource(&document, false); err != nil {
		return domain.Document{}, err
	}
	if isImplementationPlan && featureID != "" {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"feature documents cannot be implementation plans",
		)
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
		if isImplementationPlan {
			if _, err := s.repository.GetImplementationPlan(ctx, taskID); err == nil {
				return domain.Document{}, domain.NewError(
					domain.DomainErrorCodeDuplicateImplementationPlan,
					"task %q already has an implementation plan", taskID,
				)
			} else if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
				return domain.Document{}, err
			}
		}
	}
	return s.repository.CreateDocument(
		ctx, featureID, taskID, document.Kind, document.Title, document.Locator, document.Content, isImplementationPlan,
	)
}

func (s *Service) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	return s.repository.GetDocument(ctx, id)
}

func (s *Service) UpdateDocument(
	ctx context.Context,
	id string,
	title *string,
	source *domain.Document,
	isImplementationPlan *bool,
) (domain.Document, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return domain.Document{}, err
	}
	if title != nil {
		document.Title = strings.TrimSpace(*title)
	}
	if source != nil {
		document.Kind = source.Kind
		document.Locator = source.Locator
		document.Content = source.Content
	}
	if isImplementationPlan != nil {
		document.IsImplementationPlan = *isImplementationPlan
	}
	if document.IsImplementationPlan && document.TaskID == "" {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"feature documents cannot be implementation plans",
		)
	}
	if err := validateDocumentSource(&document, false); err != nil {
		return domain.Document{}, err
	}
	var updatedTitle *string
	if title != nil {
		updatedTitle = &document.Title
	}
	var updatedSource *domain.Document
	if source != nil {
		updatedSource = &domain.Document{
			Kind:    document.Kind,
			Locator: document.Locator,
			Content: document.Content,
		}
	}
	return s.repository.UpdateDocument(ctx, id, updatedTitle, updatedSource, isImplementationPlan)
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.repository.DeleteDocument(ctx, id)
}

// ReadDocumentContent only reads paths that were explicitly registered or
// Markdown stored by PRX. The size limit keeps a preview request from consuming an
// unbounded amount of memory in the server or browser.
func (s *Service) ReadDocumentContent(ctx context.Context, id string) (string, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return "", err
	}
	if document.Kind == domain.DocumentKindMarkdown {
		return document.Content, nil
	}
	if document.Kind != domain.DocumentKindLocalFile {
		return "", domain.NewError(
			domain.DomainErrorCodeInvalidDocumentKind,
			"URL documents do not have readable content",
		)
	}
	file, err := os.Open(document.Locator)
	if err != nil {
		return "", domain.NewError(domain.DomainErrorCodeDocumentReadFailed, "could not read the local file")
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxDocumentContentBytes+1))
	if err != nil {
		return "", domain.NewError(domain.DomainErrorCodeDocumentReadFailed, "could not read the local file")
	}
	if len(content) > maxDocumentContentBytes {
		return "", domain.NewError(domain.DomainErrorCodeDocumentTooLarge, "document preview is limited to 1 MiB")
	}
	if !utf8.Valid(content) {
		return "", domain.NewError(domain.DomainErrorCodeDocumentNotText, "local file is not valid UTF-8 text")
	}
	return string(content), nil
}

func validateDocumentSource(document *domain.Document, implementationPlanCommand bool) error {
	if !oneOf(document.Kind, domain.DocumentKindURL, domain.DocumentKindLocalFile, domain.DocumentKindMarkdown) {
		return domain.NewError(
			domain.DomainErrorCodeInvalidDocumentKind,
			"document kind must be url, local_file, or markdown",
		)
	}
	document.Locator = strings.TrimSpace(document.Locator)
	if document.Kind == domain.DocumentKindMarkdown {
		document.Locator = ""
		if strings.TrimSpace(document.Content) == "" {
			code := domain.DomainErrorCodeInvalidDocument
			if implementationPlanCommand {
				code = domain.DomainErrorCodeInvalidImplementationPlan
			}
			return domain.NewError(code, "Markdown content is required")
		}
		if !utf8.ValidString(document.Content) {
			return domain.NewError(domain.DomainErrorCodeDocumentNotText, "Markdown content must be valid UTF-8")
		}
		if len([]byte(document.Content)) > maxDocumentContentBytes {
			code := domain.DomainErrorCodeDocumentTooLarge
			if implementationPlanCommand {
				code = domain.DomainErrorCodeImplementationPlanTooLarge
			}
			return domain.NewError(code, "Markdown content is limited to 1 MiB")
		}
		return nil
	}
	document.Content = ""
	if document.Locator == "" {
		return domain.NewError(domain.DomainErrorCodeInvalidDocument, "document locator is required")
	}
	if document.Kind == domain.DocumentKindURL {
		parsed, err := url.Parse(document.Locator)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return domain.NewError(domain.DomainErrorCodeInvalidDocumentURL, "document URL must use http or https")
		}
	}
	return nil
}

func (s *Service) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	snapshot, err := s.repository.Snapshot(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	return deriveSnapshot(snapshot), nil
}

// deriveSnapshot turns a stored snapshot into the derived one every caller
// needs. Synchronization shares it with the read path so the feature statuses
// it selects on are the same ones clients see.
func deriveSnapshot(snapshot domain.Snapshot) domain.Snapshot {
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
		if domain.IsTaskFinished(task.DisplayState) {
			feature.FinishedCount++
		}
	}
	for i := range snapshot.Features {
		feature := &snapshot.Features[i]
		feature.DisplayStatus = domain.FeatureDisplayStatus(feature.Status, feature.TaskCount, feature.FinishedCount)
	}
	return snapshot
}

func (s *Service) syncSelected(
	ctx context.Context,
	featureID, taskID string,
	_ bool,
) (succeeded, failed int, err error) {
	if s.provider == nil && s.configStore == nil {
		return 0, 0, domain.NewError(domain.DomainErrorCodeGitHubAuth, "GitHub provider is not configured")
	}
	stored, err := s.repository.Snapshot(ctx)
	if err != nil {
		return 0, 0, err
	}
	snapshot := deriveSnapshot(stored)
	featureResolved := ""
	if featureID != "" {
		feature, err := s.ResolveFeature(ctx, featureID)
		if err != nil {
			return 0, 0, err
		}
		featureResolved = feature.ID
	}
	if taskID != "" {
		if _, err := s.repository.GetTask(ctx, taskID); err != nil {
			return 0, 0, err
		}
	}
	taskFeature := map[string]string{}
	for _, task := range snapshot.Tasks {
		taskFeature[task.ID] = task.FeatureID
	}
	// A completed feature leaves automatic and unscoped manual refreshes for the
	// same reason an archived one does: its pull requests are no longer part of
	// the work in flight. An explicit feature or task keeps maintaining them.
	activeFeatures := map[string]bool{}
	for _, feature := range snapshot.Features {
		activeFeatures[feature.ID] = !feature.Archived &&
			feature.DisplayStatus != domain.FeatureStatusCompleted
	}
	allFeatures := featureID == "" && taskID == ""
	eligible := snapshot.PullRequests[:0]
	for _, pullRequest := range snapshot.PullRequests {
		if allFeatures && !activeFeatures[taskFeature[pullRequest.TaskID]] {
			continue
		}
		eligible = append(eligible, pullRequest)
	}
	snapshot.PullRequests = eligible
	if s.provider == nil {
		settings, loadErr := s.configStore.Load()
		if loadErr != nil {
			return 0, 0, configDomainError(loadErr)
		}
		resolver, resolverErr := githubprovider.NewResolver(settings)
		if resolverErr != nil {
			return 0, 0, configDomainError(resolverErr)
		}
		return s.syncLive(ctx, snapshot, taskFeature, featureResolved, taskID, resolver)
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
			if updated.TaskID == "" {
				updated = pr
			}
			updated, needsAttention := syncFailureValue(
				pr,
				updated,
				fetchErr,
				s.currentTime(),
			)
			if _, persistErr := s.repository.UpsertPullRequest(ctx, updated); persistErr != nil {
				return succeeded, failed, persistErr
			}
			if needsAttention {
				failed++
			}
			continue
		}
		updated.TaskID = pr.TaskID
		updated.SyncError = ""
		updated.Stale = false
		if _, err := s.repository.UpsertPullRequest(ctx, updated); err != nil {
			return succeeded, failed, err
		}
		succeeded++
	}
	return succeeded, failed, nil
}

func (s *Service) Validate(ctx context.Context) []string { return s.repository.Validate(ctx) }

func oneOf[T comparable](value T, values ...T) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
