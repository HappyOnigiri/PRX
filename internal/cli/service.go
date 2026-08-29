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
	CreateFeature(context.Context, string, string, string) (domain.Feature, error)
	UpdateFeature(context.Context, string, *string, *string, *string, *domain.FeatureStatus, *bool) (domain.Feature, error)
	ResolveFeature(context.Context, string) (domain.Feature, error)
	DeleteFeature(context.Context, string, bool) error

	CreateTask(context.Context, string, string, string, domain.TaskKind, string) (domain.Task, error)
	UpdateTask(context.Context, string, *string, *string, *domain.TaskStatus, *string) (domain.Task, error)
	DeleteTask(context.Context, string, bool) error

	AddDependency(context.Context, string, string) (domain.Dependency, error)
	RemoveDependency(context.Context, string, string) error

	AttachPullRequest(context.Context, string, string) (domain.PullRequest, error)
	DetachPullRequest(context.Context, string) error

	AddDocument(context.Context, string, string, domain.DocumentKind, string, string) (domain.Document, error)
	DeleteDocument(context.Context, string) error
	ReadMarkdownDocument(context.Context, string) (string, error)

	Snapshot(context.Context) (domain.Snapshot, error)
	Sync(context.Context, string, string) (int, int, error)
	Validate(context.Context) []string
	SeedDemo(context.Context, string, int) error
}
