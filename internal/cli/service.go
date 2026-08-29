package cli

import (
	"context"
	"io"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// OpenService constructs the application service and returns the resource that
// must be closed after one CLI command finishes.
type OpenService func(context.Context, string, string, bool) (Service, io.Closer, error)

// Service is the application boundary used by CLI commands and the RPC server
// exposed by the serve command.
type Service interface {
	CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error)
	UpdateFeature(
		ctx context.Context,
		id string,
		slug, title, description *string,
		status *domain.FeatureStatus,
		archived *bool,
	) (domain.Feature, error)
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

	AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error)
	RemoveDependency(ctx context.Context, blocker, blocked string) error

	AttachPullRequest(ctx context.Context, taskID, rawURL string) (domain.PullRequest, error)
	DetachPullRequest(ctx context.Context, taskID string) error

	AddDocument(
		ctx context.Context,
		featureID, taskID string,
		kind domain.DocumentKind,
		title, value string,
	) (domain.Document, error)
	DeleteDocument(ctx context.Context, id string) error
	ReadMarkdownDocument(ctx context.Context, id string) (string, error)

	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Sync(ctx context.Context, featureID, taskID string) (int, int, error)
	Validate(ctx context.Context) []string
	SeedDemo(ctx context.Context, slug string, count int) error
}
