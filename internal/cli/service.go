package cli

import (
	"context"
	"io"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

// ServiceOptions は CLI が選んだ実行時の境界条件を運ぶ。パスの出所も含めるのは、
// 各場所をフラグ・環境変数・既定値のどれが決めたかを診断レポートが説明できるようにするため。
type ServiceOptions struct {
	DatabasePath       string
	DatabasePathSource string
	ConfigPathSource   string
	FixturePath        string
	Live               bool
	Demo               bool
}

// ServiceOpenError はオープンに失敗したデータベースの場所を伝える。オープンが失敗した
// ときこそ `prx debug` が実行される想定であり、解決済みのパスは読み手が最初に必要とする情報。
type ServiceOpenError struct {
	DatabasePath string
	Err          error
}

func (e *ServiceOpenError) Error() string { return e.Err.Error() }

func (e *ServiceOpenError) Unwrap() error { return e.Err }

// OpenService はアプリケーションサービスを構築し、CLI コマンド 1 回の終了後に
// クローズすべきリソースを返す。
type OpenService func(context.Context, ServiceOptions) (Service, io.Closer, error)

// Service は CLI コマンドと serve コマンドが公開する RPC サーバーが使う
// アプリケーション境界。
type Service interface {
	CreateProject(ctx context.Context, title, description string) (domain.Project, error)
	UpdateProject(ctx context.Context, id string, update domain.ProjectUpdate) (domain.Project, error)
	ResolveProject(ctx context.Context, id string) (domain.Project, error)
	DeleteProject(ctx context.Context, id string, cascade bool) error

	CreateFeature(ctx context.Context, title, description, projectID string) (domain.Feature, error)
	UpdateFeature(ctx context.Context, id string, update domain.FeatureUpdate) (domain.Feature, error)
	ResolveFeature(ctx context.Context, id string) (domain.Feature, error)
	GetNode(ctx context.Context, id string) (any, error)
	DeleteFeature(ctx context.Context, id string, cascade bool) error

	CreateTask(ctx context.Context, featureID, title, scope, assignee string) (domain.Task, error)
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
