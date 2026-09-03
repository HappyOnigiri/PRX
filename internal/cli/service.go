package cli

import (
	"context"
	"io"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// ServiceOptions carries the runtime boundaries selected by the CLI.
// The path sources are carried so the diagnostic report can explain which of the
// flag, the environment, or the default selected each location.
type ServiceOptions struct {
	DatabasePath       string
	DatabasePathSource string
	ConfigPathSource   string
	FixturePath        string
	Live               bool
	Demo               bool
}

// ServiceOpenError reports the database location an unsuccessful open attempted.
// `prx debug` is expected to run when opening fails, and the resolved path is
// the first thing its reader needs.
type ServiceOpenError struct {
	DatabasePath string
	Err          error
}

func (e *ServiceOpenError) Error() string { return e.Err.Error() }

func (e *ServiceOpenError) Unwrap() error { return e.Err }

// OpenService constructs the application service and returns the resource that
// must be closed after one CLI command finishes.
type OpenService func(context.Context, ServiceOptions) (Service, io.Closer, error)

// Service is the application boundary used by CLI commands and the RPC server
// exposed by the serve command.
type Service interface {
	CreateProject(ctx context.Context, slug, title, description string) (domain.Project, error)
	UpdateProject(ctx context.Context, id string, update domain.ProjectUpdate) (domain.Project, error)
	ResolveProject(ctx context.Context, idOrSlug string) (domain.Project, error)
	DeleteProject(ctx context.Context, id string, cascade bool) error

	CreateFeature(ctx context.Context, slug, title, description, projectID string) (domain.Feature, error)
	UpdateFeature(ctx context.Context, id string, update domain.FeatureUpdate) (domain.Feature, error)
	ResolveFeature(ctx context.Context, idOrSlug string) (domain.Feature, error)
	GetNode(ctx context.Context, id string) (any, error)
	DeleteFeature(ctx context.Context, id string, cascade bool) error

	CreateTask(
		ctx context.Context,
		featureID, title, scope string,
		kind domain.TaskKind,
		assignee string,
	) (domain.Task, error)
	UpdateTask(
		ctx context.Context,
		id string,
		title, scope *string,
		status *domain.TaskStatus,
		assignee *string,
	) (domain.Task, error)
	DeleteTask(ctx context.Context, id string, cascade bool) error

	GetImplementationPlan(ctx context.Context, taskID string) (domain.Document, error)
	UpsertImplementationPlan(ctx context.Context, taskID string, document domain.Document) (domain.Document, error)
	DeleteImplementationPlan(ctx context.Context, taskID string) error

	AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error)
	RemoveDependency(ctx context.Context, blocker, blocked string) error

	AttachPullRequest(ctx context.Context, taskID, rawURL string) (domain.PullRequest, error)
	DetachPullRequest(ctx context.Context, taskID string) error

	AddDocument(
		ctx context.Context,
		parent domain.DocumentParent,
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
	ReadDocumentContent(ctx context.Context, id string) (string, error)

	Debug(ctx context.Context) (domain.DebugReport, error)
	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Sync(ctx context.Context, featureID, taskID string) (int, int, error)
	SyncIfDue(ctx context.Context) (bool, domain.GitHubSyncStatus, error)
	SyncStatus(ctx context.Context) (domain.GitHubSyncStatus, error)
	Validate(ctx context.Context) []string
}
