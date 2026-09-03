package rpc

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// Service is the smallest application boundary required to translate the
// ConnectRPC API into domain operations.
type Service interface {
	Snapshot(ctx context.Context) (domain.Snapshot, error)

	CreateProject(ctx context.Context, slug, title, description string) (domain.Project, error)
	UpdateProject(
		ctx context.Context,
		id string,
		slug, title, description *string,
		archived *bool,
	) (domain.Project, error)
	DeleteProject(ctx context.Context, id string, cascade bool) error

	CreateFeature(ctx context.Context, slug, title, description, projectID string) (domain.Feature, error)
	UpdateFeature(
		ctx context.Context,
		id string,
		slug, title, description *string,
		status *domain.FeatureStatus,
		archived *bool,
		projectID *string,
	) (domain.Feature, error)
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
	Sync(ctx context.Context, featureID, taskID string) (int, int, error)
	SyncIfDue(ctx context.Context) (bool, domain.GitHubSyncStatus, error)
	SyncStatus(ctx context.Context) (domain.GitHubSyncStatus, error)
	Validate(ctx context.Context) []string
}
