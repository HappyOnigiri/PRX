package rpc

import (
	"context"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// Service is the smallest application boundary required to translate the
// ConnectRPC API into domain operations.
type Service interface {
	Snapshot(context.Context) (domain.Snapshot, error)

	CreateFeature(context.Context, string, string, string) (domain.Feature, error)
	UpdateFeature(context.Context, string, *string, *string, *string, *string, *bool) (domain.Feature, error)
	DeleteFeature(context.Context, string, bool) error

	CreateTask(context.Context, string, string, string, string, string) (domain.Task, error)
	UpdateTask(context.Context, string, *string, *string, *string, *string) (domain.Task, error)
	DeleteTask(context.Context, string, bool) error

	AddDependency(context.Context, string, string) (domain.Dependency, error)
	RemoveDependency(context.Context, string, string) error

	AttachPullRequest(context.Context, string, string) (domain.PullRequest, error)
	DetachPullRequest(context.Context, string) error

	AddDocument(context.Context, string, string, string, string, string) (domain.Document, error)
	DeleteDocument(context.Context, string) error
	ReadMarkdownDocument(context.Context, string) (string, error)

	Sync(context.Context, string, string) (int, int, error)
	Validate(context.Context) []string
}
