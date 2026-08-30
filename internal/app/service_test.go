package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

type repositoryStub struct{}

func (repositoryStub) CreateFeature(context.Context, string, string, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected CreateFeature call")
}

func (repositoryStub) UpdateFeature(context.Context, domain.Feature) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected UpdateFeature call")
}

func (repositoryStub) GetFeature(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected GetFeature call")
}

func (repositoryStub) GetFeatureBySlug(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected GetFeatureBySlug call")
}

func (repositoryStub) DeleteFeature(context.Context, string, bool) error {
	return errors.New("unexpected DeleteFeature call")
}

func (repositoryStub) CreateTask(
	context.Context,
	string,
	string,
	string,
	domain.TaskKind,
	string,
) (domain.Task, error) {
	return domain.Task{}, errors.New("unexpected CreateTask call")
}

func (repositoryStub) GetTask(context.Context, string) (domain.Task, error) {
	return domain.Task{}, errors.New("unexpected GetTask call")
}

func (repositoryStub) UpdateTask(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, errors.New("unexpected UpdateTask call")
}

func (repositoryStub) DeleteTask(context.Context, string, bool) error {
	return errors.New("unexpected DeleteTask call")
}

func (repositoryStub) GetImplementationPlan(context.Context, string) (domain.ImplementationPlan, error) {
	return domain.ImplementationPlan{}, errors.New("unexpected GetImplementationPlan call")
}

func (repositoryStub) UpsertImplementationPlan(context.Context, string, string) (domain.ImplementationPlan, error) {
	return domain.ImplementationPlan{}, errors.New("unexpected UpsertImplementationPlan call")
}

func (repositoryStub) DeleteImplementationPlan(context.Context, string) error {
	return errors.New("unexpected DeleteImplementationPlan call")
}

func (repositoryStub) AddDependency(context.Context, string, string) (domain.Dependency, error) {
	return domain.Dependency{}, errors.New("unexpected AddDependency call")
}

func (repositoryStub) RemoveDependency(context.Context, string, string) error {
	return errors.New("unexpected RemoveDependency call")
}

func (repositoryStub) UpsertPullRequest(context.Context, domain.PullRequest) (domain.PullRequest, error) {
	return domain.PullRequest{}, errors.New("unexpected UpsertPullRequest call")
}

func (repositoryStub) DeletePullRequest(context.Context, string) error {
	return errors.New("unexpected DeletePullRequest call")
}

func (repositoryStub) CreateDocument(
	context.Context,
	string,
	string,
	domain.DocumentKind,
	string,
	string,
) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected CreateDocument call")
}

func (repositoryStub) GetDocument(context.Context, string) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected GetDocument call")
}

func (repositoryStub) DeleteDocument(context.Context, string) error {
	return errors.New("unexpected DeleteDocument call")
}

func (repositoryStub) Snapshot(context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{}, errors.New("unexpected Snapshot call")
}
func (repositoryStub) Validate(context.Context) []string { return []string{"unexpected Validate call"} }

type featureRepository struct {
	repositoryStub
	feature domain.Feature
}

func (r *featureRepository) GetFeature(context.Context, string) (domain.Feature, error) {
	return r.feature, nil
}

type taskRepository struct {
	repositoryStub
	task domain.Task
}

func (r *taskRepository) GetTask(context.Context, string) (domain.Task, error) {
	return r.task, nil
}

type featureSlugRepository struct {
	repositoryStub
	feature domain.Feature
}

func (r *featureSlugRepository) GetFeature(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("feature ID was not found")
}

func (r *featureSlugRepository) GetFeatureBySlug(context.Context, string) (domain.Feature, error) {
	return r.feature, nil
}

func TestGetNodeResolvesFeatureIDsAndSlugsAndTaskIDs(t *testing.T) {
	feature := domain.Feature{ID: "F-1", Slug: "checkout"}
	featureValue, err := app.New(&featureRepository{feature: feature}, nil).GetNode(context.Background(), feature.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := featureValue.(domain.Feature); !ok || got.ID != feature.ID || got.Slug != feature.Slug {
		t.Fatalf("feature node=%#v", featureValue)
	}
	slugValue, err := app.New(&featureSlugRepository{feature: feature}, nil).
		GetNode(context.Background(), feature.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := slugValue.(domain.Feature); !ok || got.ID != feature.ID || got.Slug != feature.Slug {
		t.Fatalf("feature slug node=%#v", slugValue)
	}

	task := domain.Task{ID: "T-1", FeatureID: feature.ID, Title: "Implement"}
	taskValue, err := app.New(&taskRepository{task: task}, nil).GetNode(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := taskValue.(domain.Task); !ok || got.ID != task.ID || got.FeatureID != task.FeatureID {
		t.Fatalf("task node=%#v", taskValue)
	}

	if _, err := app.New(repositoryStub{}, nil).
		GetNode(context.Background(), "unknown"); errorCode(
		t,
		err,
	) != domain.DomainErrorCodeNotFound {
		t.Fatalf("unknown node error=%v", err)
	}
}

func (r *taskRepository) UpdateTask(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}

type planRepository struct {
	taskRepository
	plan domain.ImplementationPlan
}

func (r *planRepository) GetImplementationPlan(context.Context, string) (domain.ImplementationPlan, error) {
	return r.plan, nil
}

func (r *planRepository) UpsertImplementationPlan(
	_ context.Context,
	taskID, content string,
) (domain.ImplementationPlan, error) {
	r.plan = domain.ImplementationPlan{TaskID: taskID, Content: content}
	return r.plan, nil
}

func (r *planRepository) DeleteImplementationPlan(context.Context, string) error {
	r.plan = domain.ImplementationPlan{}
	return nil
}

type syncRepository struct {
	repositoryStub
	snapshot domain.Snapshot
	updated  domain.PullRequest
}

func (r *syncRepository) Snapshot(context.Context) (domain.Snapshot, error) {
	return r.snapshot, nil
}

func (r *syncRepository) UpsertPullRequest(_ context.Context, value domain.PullRequest) (domain.PullRequest, error) {
	r.updated = value
	return value, nil
}

type syncProvider struct {
	updated domain.PullRequest
	err     error
}

func (p syncProvider) Fetch(context.Context, domain.PullRequest) (domain.PullRequest, error) {
	return p.updated, p.err
}

type createFeatureRepository struct {
	repositoryStub
	gotSlug        string
	gotTitle       string
	gotDescription string
}

func (r *createFeatureRepository) CreateFeature(
	_ context.Context,
	slug, title, description string,
) (domain.Feature, error) {
	r.gotSlug = slug
	r.gotTitle = title
	r.gotDescription = description
	return domain.Feature{Slug: slug, Title: title, Description: description}, nil
}

func TestCreateFeatureValidatesBeforeRepository(t *testing.T) {
	for _, test := range []struct {
		name  string
		slug  string
		title string
		code  domain.DomainErrorCode
	}{
		{name: "invalid slug", slug: "not a slug", title: "Feature", code: "invalid_slug"},
		{name: "missing title", slug: "feature", title: "  ", code: "invalid_title"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := app.New(repositoryStub{}, nil)
			_, err := service.CreateFeature(context.Background(), test.slug, test.title, "")
			if got := errorCode(t, err); got != test.code {
				t.Fatalf("error code=%q, want %q", got, test.code)
			}
		})
	}
}

func TestCreateFeatureNormalizesBeforeRepository(t *testing.T) {
	repository := &createFeatureRepository{}
	service := app.New(repository, nil)

	_, err := service.CreateFeature(context.Background(), "  Release-API  ", "  Release API  ", "  Description  ")
	if err != nil {
		t.Fatal(err)
	}
	if repository.gotSlug != "release-api" || repository.gotTitle != "Release API" ||
		repository.gotDescription != "Description" {
		t.Fatalf(
			"repository received slug=%q title=%q description=%q",
			repository.gotSlug,
			repository.gotTitle,
			repository.gotDescription,
		)
	}
}

func TestCreateTaskRejectsUnknownKind(t *testing.T) {
	repository := &featureRepository{feature: domain.Feature{ID: "feature-id"}}
	service := app.New(repository, nil)

	_, err := service.CreateTask(context.Background(), "feature-id", "Task", "", domain.TaskKind("unknown"), "")
	if got := errorCode(t, err); got != "invalid_kind" {
		t.Fatalf("error code=%q, want invalid_kind", got)
	}
}

func TestUpdateTaskAcceptsManualOverridesForPRTask(t *testing.T) {
	repository := &taskRepository{
		task: domain.Task{ID: "task-id", Title: "Ship", Kind: domain.TaskKindPR, Status: domain.TaskStatusAuto},
	}
	service := app.New(repository, nil)
	completed := domain.TaskStatusCompleted

	updated, err := service.UpdateTask(context.Background(), "task-id", nil, nil, &completed, nil)
	if err != nil || updated.Status != domain.TaskStatusCompleted {
		t.Fatalf("updated task=%+v err=%v", updated, err)
	}
}

func TestImplementationPlanValidationPreservesContent(t *testing.T) {
	repository := &planRepository{
		taskRepository: taskRepository{task: domain.Task{ID: "task-id", Title: "Plan", Kind: domain.TaskKindManual}},
	}
	service := app.New(repository, nil)
	content := "  # Plan\n\nKeep the surrounding whitespace.  \n"
	plan, err := service.UpsertImplementationPlan(context.Background(), "task-id", content)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Content != content || repository.plan.Content != content {
		t.Fatalf("plan content=%q repository content=%q, want original content", plan.Content, repository.plan.Content)
	}
	if _, err := service.UpsertImplementationPlan(
		context.Background(),
		"task-id",
		" \n\t ",
	); errorCode(
		t,
		err,
	) != domain.DomainErrorCodeInvalidImplementationPlan {
		t.Fatalf("blank plan code=%q", errorCode(t, err))
	}
	if _, err := service.UpsertImplementationPlan(
		context.Background(),
		"task-id",
		string(make([]byte, (1<<20)+1)),
	); errorCode(
		t,
		err,
	) != domain.DomainErrorCodeImplementationPlanTooLarge {
		t.Fatalf("large plan code=%q", errorCode(t, err))
	}
}

func TestSyncPersistsProviderPartialResultOnError(t *testing.T) {
	initial := domain.PullRequest{
		TaskID:       "task-id",
		State:        domain.PullRequestStateUnknown,
		ReviewState:  domain.ReviewStateUnknown,
		Mergeability: domain.MergeabilityUnknown,
		Stale:        true,
	}
	partial := initial
	partial.State = domain.PullRequestStateOpen
	partial.NodeID = "new-node"
	repository := &syncRepository{snapshot: domain.Snapshot{PullRequests: []domain.PullRequest{initial}}}
	service := app.New(repository, syncProvider{updated: partial, err: errors.New("review failed")})
	if succeeded, failed, err := service.Sync(
		context.Background(),
		"",
		"",
	); succeeded != 0 || failed != 1 ||
		err != nil {
		t.Fatalf("sync result=(%d, %d, %v)", succeeded, failed, err)
	}
	if repository.updated.State != domain.PullRequestStateOpen || repository.updated.NodeID != "new-node" ||
		repository.updated.SyncError != "review failed" || !repository.updated.Stale {
		t.Fatalf("persisted partial result=%+v", repository.updated)
	}
}

func TestAddDocumentValidatesWithoutRepository(t *testing.T) {
	for _, test := range []struct {
		name      string
		featureID string
		taskID    string
		kind      domain.DocumentKind
		value     string
		code      domain.DomainErrorCode
	}{
		{name: "missing parent", kind: "url", value: "https://example.com", code: "invalid_parent"},
		{
			name:      "two parents",
			featureID: "feature-id",
			taskID:    "task-id",
			kind:      "url",
			value:     "https://example.com",
			code:      "invalid_parent",
		},
		{name: "unknown kind", taskID: "task-id", kind: "file", value: "docs/plan.md", code: "invalid_document_kind"},
		{name: "missing value", taskID: "task-id", kind: "url", value: "  ", code: "invalid_document"},
		{name: "invalid URL", taskID: "task-id", kind: "url", value: "ftp://example.com", code: "invalid_document_url"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := app.New(repositoryStub{}, nil)
			_, err := service.AddDocument(
				context.Background(),
				test.featureID,
				test.taskID,
				test.kind,
				"Document",
				test.value,
			)
			if got := errorCode(t, err); got != test.code {
				t.Fatalf("error code=%q, want %q", got, test.code)
			}
		})
	}
}

func TestAttachPullRequestRejectsManualTask(t *testing.T) {
	repository := &taskRepository{task: domain.Task{ID: "task-id", Kind: domain.TaskKindManual}}
	service := app.New(repository, nil)

	_, err := service.AttachPullRequest(context.Background(), "task-id", "https://github.com/acme/api/pull/42")
	if got := errorCode(t, err); got != "pull_request_on_manual_task" {
		t.Fatalf("error code=%q, want pull_request_on_manual_task", got)
	}
}

func errorCode(t *testing.T, err error) domain.DomainErrorCode {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error type=%T value=%v", err, err)
	}
	return domainErr.Code
}
