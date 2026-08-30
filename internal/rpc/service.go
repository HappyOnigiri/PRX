package rpc

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// Service is the smallest application boundary required to translate the
// ConnectRPC API into domain operations.
type Service interface {
	Snapshot(ctx context.Context) (domain.Snapshot, error)

	CreateFeature(ctx context.Context, slug, title, description string) (domain.Feature, error)
	UpdateFeature(
		ctx context.Context,
		id string,
		slug, title, description *string,
		status *domain.FeatureStatus,
		archived *bool,
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

	GetImplementationPlan(ctx context.Context, taskID string) (domain.ImplementationPlan, error)
	UpsertImplementationPlan(ctx context.Context, taskID, content string) (domain.ImplementationPlan, error)
	DeleteImplementationPlan(ctx context.Context, taskID string) error

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

	Sync(ctx context.Context, featureID, taskID string) (int, int, error)
	Validate(ctx context.Context) []string
}
