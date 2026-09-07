package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

type repositoryStub struct{}

func (repositoryStub) CreateProject(context.Context, string, string) (domain.Project, error) {
	return domain.Project{}, errors.New("unexpected CreateProject call")
}

func (repositoryStub) UpdateProject(context.Context, domain.Project) (domain.Project, error) {
	return domain.Project{}, errors.New("unexpected UpdateProject call")
}

func (repositoryStub) GetProject(_ context.Context, id string) (domain.Project, error) {
	return domain.Project{}, domain.NewError(domain.DomainErrorCodeNotFound, "project %q was not found", id)
}

func (repositoryStub) DeleteProject(context.Context, string, bool) error {
	return errors.New("unexpected DeleteProject call")
}

func (repositoryStub) CreateFeature(context.Context, string, string, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected CreateFeature call")
}

func (repositoryStub) UpdateFeature(context.Context, domain.Feature) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected UpdateFeature call")
}

func (repositoryStub) GetFeature(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("unexpected GetFeature call")
}

func (repositoryStub) DeleteFeature(context.Context, string, bool) error {
	return errors.New("unexpected DeleteFeature call")
}

func (repositoryStub) CreateTask(
	context.Context,
	string,
	string,
	string,
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

func (repositoryStub) GetImplementationPlan(context.Context, string) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected GetImplementationPlan call")
}

func (repositoryStub) UpsertImplementationPlan(context.Context, string, domain.Document) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected UpsertImplementationPlan call")
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
	domain.DocumentParent,
	domain.DocumentKind,
	string,
	string,
	string,
	bool,
) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected CreateDocument call")
}

func (repositoryStub) GetDocument(context.Context, string) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected GetDocument call")
}

func (repositoryStub) UpdateDocument(
	context.Context,
	string,
	*string,
	*domain.Document,
	*bool,
) (domain.Document, error) {
	return domain.Document{}, errors.New("unexpected UpdateDocument call")
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

// 読み取り専用ガードは feature の所属先 project を読むので、stub は
// 「存在しない」ではなく active なコンテナを返す。
func (r *featureRepository) GetProject(_ context.Context, id string) (domain.Project, error) {
	return domain.Project{ID: id}, nil
}

type taskRepository struct {
	repositoryStub
	task domain.Task
	// feature は、task への書き込み前に読み取り専用ガードが読む所有者。
	feature domain.Feature
}

func (r *taskRepository) GetTask(context.Context, string) (domain.Task, error) {
	return r.task, nil
}

func (r *taskRepository) GetFeature(context.Context, string) (domain.Feature, error) {
	return r.feature, nil
}

func (r *taskRepository) GetProject(_ context.Context, id string) (domain.Project, error) {
	return domain.Project{ID: id}, nil
}

type missingFeatureRepository struct {
	repositoryStub
}

func (missingFeatureRepository) GetFeature(_ context.Context, id string) (domain.Feature, error) {
	return domain.Feature{}, domain.NewError(domain.DomainErrorCodeNotFound, "feature %q was not found", id)
}

type failingFeatureRepository struct {
	missingFeatureRepository
	idFailure error
}

func (r *failingFeatureRepository) GetFeature(ctx context.Context, id string) (domain.Feature, error) {
	if r.idFailure == nil {
		return r.missingFeatureRepository.GetFeature(ctx, id)
	}
	return domain.Feature{}, r.idFailure
}

// データベースのロックのようなストレージ障害は、feature が見つからないと報告されず、
// 本来の原因を保つ。
func TestResolveFeatureAndGetNodeKeepStorageFailures(t *testing.T) {
	repository := &failingFeatureRepository{idFailure: errors.New("database is locked")}
	service := app.New(repository, nil)
	_, err := service.ResolveFeature(context.Background(), "F-1")
	if err == nil || err.Error() != "database is locked" {
		t.Fatalf("ResolveFeature error=%v", err)
	}
	_, err = service.GetNode(context.Background(), "F-1")
	if err == nil || err.Error() != "database is locked" {
		t.Fatalf("GetNode error=%v", err)
	}
}

func TestGetNodeResolvesFeatureAndTaskIDs(t *testing.T) {
	feature := domain.Feature{ID: "F-1", Title: "Checkout"}
	featureValue, err := app.New(&featureRepository{feature: feature}, nil).GetNode(context.Background(), feature.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := featureValue.(domain.Feature); !ok || got.ID != feature.ID || got.Title != feature.Title {
		t.Fatalf("feature node=%#v", featureValue)
	}

	task := domain.Task{ID: "T-1", FeatureID: feature.ID, Title: "Implement"}
	taskValue, err := app.New(&taskRepository{task: task}, nil).GetNode(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := taskValue.(domain.Task); !ok || got.ID != task.ID || got.FeatureID != task.FeatureID {
		t.Fatalf("task node=%#v", taskValue)
	}

	// 既知の公開 ID 接頭辞を持たないオペランドはどの種別も指さないので、
	// 検索せずに「存在しない」と報告する。
	if _, err := app.New(missingFeatureRepository{}, nil).
		GetNode(context.Background(), "checkout"); errorCode(
		t,
		err,
	) != domain.DomainErrorCodeNotFound {
		t.Fatalf("unprefixed node error=%v", err)
	}
	if _, err := app.New(missingFeatureRepository{}, nil).
		GetNode(context.Background(), "F-9"); errorCode(t, err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("unknown feature node error=%v", err)
	}
}

func (r *taskRepository) UpdateTask(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}

type planRepository struct {
	taskRepository
	plan domain.Document
}

func (r *planRepository) GetImplementationPlan(context.Context, string) (domain.Document, error) {
	return r.plan, nil
}

func (r *planRepository) UpsertImplementationPlan(
	_ context.Context,
	taskID string, document domain.Document,
) (domain.Document, error) {
	document.TaskID = taskID
	r.plan = document
	return r.plan, nil
}

func (r *planRepository) DeleteImplementationPlan(context.Context, string) error {
	r.plan = domain.Document{}
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
	gotTitle       string
	gotDescription string
	gotProject     string
}

func (r *createFeatureRepository) GetProject(_ context.Context, id string) (domain.Project, error) {
	return domain.Project{ID: id}, nil
}

func (r *createFeatureRepository) CreateFeature(
	_ context.Context,
	title, description, projectID string,
) (domain.Feature, error) {
	r.gotTitle = title
	r.gotDescription = description
	r.gotProject = projectID
	return domain.Feature{Title: title, Description: description, ProjectID: projectID}, nil
}

func TestCreateFeatureValidatesBeforeRepository(t *testing.T) {
	service := app.New(repositoryStub{}, nil)
	_, err := service.CreateFeature(context.Background(), "  ", "", "P-1")
	if got := errorCode(t, err); got != domain.DomainErrorCodeInvalidTitle {
		t.Fatalf("error code=%q, want %q", got, domain.DomainErrorCodeInvalidTitle)
	}

	// 所属は必須なので、project のない feature は repository に届く前に拒否される。
	_, err = service.CreateFeature(context.Background(), "Release API", "", "  ")
	if got := errorCode(t, err); got != domain.DomainErrorCodeInvalidParent {
		t.Fatalf("error code=%q, want %q", got, domain.DomainErrorCodeInvalidParent)
	}
}

func TestCreateFeatureNormalizesBeforeRepository(t *testing.T) {
	repository := &createFeatureRepository{}
	service := app.New(repository, nil)

	_, err := service.CreateFeature(context.Background(), "  Release API  ", "  Description  ", "  P-1  ")
	if err != nil {
		t.Fatal(err)
	}
	if repository.gotTitle != "Release API" ||
		repository.gotDescription != "Description" ||
		repository.gotProject != "P-1" {
		t.Fatalf(
			"repository received title=%q description=%q project=%q",
			repository.gotTitle,
			repository.gotDescription,
			repository.gotProject,
		)
	}
}

func TestUpdateTaskAcceptsManualOverrides(t *testing.T) {
	repository := &taskRepository{
		task: domain.Task{ID: "task-id", Title: "Ship", Status: domain.TaskStatusNotStarted},
	}
	service := app.New(repository, nil)
	completed := domain.TaskStatusCompleted

	updated, err := service.UpdateTask(context.Background(), "task-id", nil, nil, &completed, nil)
	if err != nil || updated.Status != domain.TaskStatusCompleted {
		t.Fatalf("updated task=%+v err=%v", updated, err)
	}

	designing := domain.TaskStatusDesigning
	updated, err = service.UpdateTask(context.Background(), "task-id", nil, nil, &designing, nil)
	if err != nil || updated.Status != domain.TaskStatusDesigning {
		t.Fatalf("updated task=%+v err=%v", updated, err)
	}

	invalid := domain.TaskStatus("designed")
	if _, err := service.UpdateTask(context.Background(), "task-id", nil, nil, &invalid, nil); err == nil {
		t.Fatal("a display state is not a stored status and must be rejected")
	}
}

func TestImplementationPlanValidationPreservesContent(t *testing.T) {
	repository := &planRepository{
		taskRepository: taskRepository{task: domain.Task{ID: "task-id", Title: "Plan"}},
	}
	service := app.New(repository, nil)
	content := "  # Plan\n\nKeep the surrounding whitespace.  \n"
	plan, err := service.UpsertImplementationPlan(
		context.Background(), "task-id", domain.Document{Kind: domain.DocumentKindMarkdown, Content: content},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Content != content || repository.plan.Content != content {
		t.Fatalf("plan content=%q repository content=%q, want original content", plan.Content, repository.plan.Content)
	}
	if _, err := service.UpsertImplementationPlan(
		context.Background(),
		"task-id",
		domain.Document{Kind: domain.DocumentKindMarkdown, Content: " \n\t "},
	); errorCode(
		t,
		err,
	) != domain.DomainErrorCodeInvalidImplementationPlan {
		t.Fatalf("blank plan code=%q", errorCode(t, err))
	}
	if _, err := service.UpsertImplementationPlan(
		context.Background(),
		"task-id",
		domain.Document{Kind: domain.DocumentKindMarkdown, Content: string(make([]byte, (1<<20)+1))},
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
	repository := &syncRepository{snapshot: domain.Snapshot{
		Features:     []domain.Feature{{ID: "feature-id"}},
		Tasks:        []domain.Task{{ID: "task-id", FeatureID: "feature-id"}},
		PullRequests: []domain.PullRequest{initial},
	}}
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

func TestSyncDoesNotCountKnownTerminalProviderFailure(t *testing.T) {
	for name, test := range map[string]struct {
		storedState  domain.PullRequestState
		partialState domain.PullRequestState
		wantState    domain.PullRequestState
		wantFailed   int
		wantError    string
	}{
		"stored terminal state": {
			storedState:  domain.PullRequestStateMerged,
			partialState: domain.PullRequestStateUnknown,
			wantState:    domain.PullRequestStateMerged,
		},
		"partial terminal state": {
			storedState:  domain.PullRequestStateOpen,
			partialState: domain.PullRequestStateClosed,
			wantState:    domain.PullRequestStateClosed,
		},
		"partial non-terminal state takes precedence": {
			storedState:  domain.PullRequestStateMerged,
			partialState: domain.PullRequestStateOpen,
			wantState:    domain.PullRequestStateOpen,
			wantFailed:   1,
			wantError:    "GitHub metadata unavailable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			initial := domain.PullRequest{
				TaskID: "task-id", State: test.storedState,
				SyncError: "previous error", Stale: false,
			}
			partial := initial
			partial.State = test.partialState
			repository := &syncRepository{snapshot: domain.Snapshot{
				Features:     []domain.Feature{{ID: "feature-id"}},
				Tasks:        []domain.Task{{ID: "task-id", FeatureID: "feature-id"}},
				PullRequests: []domain.PullRequest{initial},
			}}
			service := app.New(repository, syncProvider{
				updated: partial,
				err:     errors.New("GitHub metadata unavailable"),
			})
			succeeded, failed, err := service.Sync(context.Background(), "", "")
			if err != nil || succeeded != 0 || failed != test.wantFailed {
				t.Fatalf("sync result=(%d, %d, %v)", succeeded, failed, err)
			}
			if repository.updated.State != test.wantState ||
				repository.updated.SyncError != test.wantError || !repository.updated.Stale ||
				repository.updated.LastSyncedAt == nil {
				t.Fatalf("persisted terminal failure=%+v", repository.updated)
			}
		})
	}
}

func TestAddDocumentValidatesWithoutRepository(t *testing.T) {
	for _, test := range []struct {
		name   string
		parent domain.DocumentParent
		kind   domain.DocumentKind
		value  string
		plan   bool
		code   domain.DomainErrorCode
	}{
		{name: "missing parent", kind: "url", value: "https://example.com", code: "invalid_parent"},
		{
			name:   "two parents",
			parent: domain.DocumentParent{FeatureID: "feature-id", TaskID: "task-id"},
			kind:   "url",
			value:  "https://example.com",
			code:   "invalid_parent",
		},
		{
			name:   "three parents",
			parent: domain.DocumentParent{ProjectID: "P-1", FeatureID: "feature-id", TaskID: "task-id"},
			kind:   "url",
			value:  "https://example.com",
			code:   "invalid_parent",
		},
		{
			name:   "unknown kind",
			parent: domain.DocumentParent{TaskID: "task-id"},
			kind:   "file",
			value:  "docs/plan.md",
			code:   "invalid_document_kind",
		},
		{
			name:   "missing value",
			parent: domain.DocumentParent{TaskID: "task-id"},
			kind:   "url",
			value:  "  ",
			code:   "invalid_document",
		},
		{
			name:   "invalid URL",
			parent: domain.DocumentParent{TaskID: "task-id"},
			kind:   "url",
			value:  "ftp://example.com",
			code:   "invalid_document_url",
		},
		{
			name:   "project document as implementation plan",
			parent: domain.DocumentParent{ProjectID: "P-1"},
			kind:   "url",
			value:  "https://example.com",
			plan:   true,
			code:   "invalid_parent",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := app.New(repositoryStub{}, nil)
			_, err := service.AddDocument(
				context.Background(),
				test.parent,
				test.kind,
				"Document",
				test.value,
				"",
				test.plan,
			)
			if got := errorCode(t, err); got != test.code {
				t.Fatalf("error code=%q, want %q", got, test.code)
			}
		})
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
