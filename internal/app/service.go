package app

import (
	"context"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
)

const (
	maxDocumentContentBytes = 1 << 20
)

// Repository は application service が使う永続化の境界。
// このインタフェースを app パッケージに置くことで、SQLite を開かずに service を
// テストでき、他の永続化実装も自由に満たせる。
type Repository interface {
	CreateProject(ctx context.Context, title, description string) (domain.Project, error)
	UpdateProject(ctx context.Context, project domain.Project) (domain.Project, error)
	GetProject(ctx context.Context, id string) (domain.Project, error)
	DeleteProject(ctx context.Context, id string, cascade bool) error

	CreateFeature(ctx context.Context, title, description, projectID string) (domain.Feature, error)
	UpdateFeature(ctx context.Context, feature domain.Feature) (domain.Feature, error)
	GetFeature(ctx context.Context, id string) (domain.Feature, error)
	DeleteFeature(ctx context.Context, id string, cascade bool) error

	CreateTask(ctx context.Context, featureID, title, scope, assignee string) (domain.Task, error)
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

	Snapshot(ctx context.Context) (domain.Snapshot, error)
	Validate(ctx context.Context) []string
}

// GitHubAuthCache は、実際の GitHub 同期が有効なとき SQLite repository が実装する。
// 任意のままにすることで、application 層のテストが使う小さな repository の fake を
// そのまま保てる。
type GitHubAuthCache interface {
	GetGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository string) (string, bool, error)
	UpsertGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository, authMethodID string) error
	DeleteGitHubRepositoryAuthCache(ctx context.Context, host, owner, repository string) error
}

type GitHubSyncStateRepository interface {
	GitHubSyncState(ctx context.Context) (domain.GitHubSyncState, error)
	AcquireGitHubAutoSync(ctx context.Context, runID string, attemptedAt time.Time, dueBeforeUnix int64) (bool, error)
	StartGitHubSync(ctx context.Context, runID string, attemptedAt time.Time) error
	// CompleteGitHubSync は、その実行がまだ共有状態を保持していたかを返す。
	// 並行する refresh は run identifier を上書きするので、呼び出し側は後から
	// 読み戻した状態を自分の結果として提示してはならない。
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
	// processInfo は service の配線中に一度だけ書かれ、診断レポートだけが読む。
	processInfo ProcessInfo
	// serveEndpoint は唯一の可変フィールド。listen アドレスは listener の bind 後に
	// 判明し、その後 HTTP ハンドラが読む。
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

// CreateFeature は project を必須とする。feature は必ずどこかに属するので、
// 後から割り当てるのではなく、呼び出し側が最初にコンテナを指定する。
func (s *Service) CreateFeature(
	ctx context.Context,
	title, description, projectID string,
) (domain.Feature, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Feature{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "feature title is required")
	}
	resolvedProject, err := s.resolveProjectAssignment(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return domain.Feature{}, err
	}
	created, err := s.repository.CreateFeature(ctx, title, strings.TrimSpace(description), resolvedProject)
	if err != nil {
		return domain.Feature{}, err
	}
	return s.withReadOnly(ctx, created)
}

// UpdateFeature は呼び出し側が指定した全フィールドを適用する。nil ポインタは未指定、
// 空文字列は値を消す要求を意味する。空の ProjectID は拒否する。
// feature はどの project にも属さない状態にはなれないため。
func (s *Service) UpdateFeature(
	ctx context.Context,
	id string,
	update domain.FeatureUpdate,
) (domain.Feature, error) {
	feature, err := s.ResolveFeature(ctx, id)
	if err != nil {
		return domain.Feature{}, err
	}
	if !archivedFeatureFlagOnly(update) {
		if err := s.guardFeature(ctx, feature); err != nil {
			return domain.Feature{}, err
		}
	}
	if update.ProjectID != nil {
		resolvedProject, err := s.resolveProjectAssignment(ctx, strings.TrimSpace(*update.ProjectID))
		if err != nil {
			return domain.Feature{}, err
		}
		feature.ProjectID = resolvedProject
	}
	if update.Title != nil {
		feature.Title = strings.TrimSpace(*update.Title)
	}
	if update.Description != nil {
		feature.Description = *update.Description
	}
	if update.Status != nil && *update.Status != "" {
		feature.Status = *update.Status
	}
	if update.Archived != nil {
		feature.Archived = *update.Archived
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
	updated, err := s.repository.UpdateFeature(ctx, feature)
	if err != nil {
		return domain.Feature{}, err
	}
	return s.withReadOnly(ctx, updated)
}

// ResolveFeature は公開 ID で feature を引く。解決した feature は導出済みの
// ReadOnly を持つので、単一 feature を報告する呼び出し側も snapshot 読み取りと
// 同じ値を公開できる。
func (s *Service) ResolveFeature(ctx context.Context, id string) (domain.Feature, error) {
	feature, err := s.repository.GetFeature(ctx, id)
	if err != nil {
		return domain.Feature{}, err
	}
	return s.withReadOnly(ctx, feature)
}

// GetNode は project・feature・task の公開 ID を、ストレージの UUID を晒さず、
// 呼び出し側に種別を先に選ばせることもなく解決する。
// 公開 ID の接頭辞が種別を示すので、オペランドが曖昧になることはない。
func (s *Service) GetNode(ctx context.Context, id string) (any, error) {
	switch {
	case strings.HasPrefix(id, "T-"):
		return s.repository.GetTask(ctx, id)
	case strings.HasPrefix(id, "P-"):
		return s.ResolveProject(ctx, id)
	case strings.HasPrefix(id, "F-"):
		return s.ResolveFeature(ctx, id)
	}
	return nil, domain.NewError(domain.DomainErrorCodeNotFound, "project, feature, or task %q was not found", id)
}

// DeleteFeature は意図的にガードしていない。アーカイブ済みの作業を捨てる操作は、
// 読み取り専用の障壁が通すものの 1 つ。
func (s *Service) DeleteFeature(ctx context.Context, id string, cascade bool) error {
	feature, err := s.ResolveFeature(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteFeature(ctx, feature.ID, cascade)
}

func (s *Service) CreateTask(
	ctx context.Context,
	featureID, title, scope, assignee string,
) (domain.Task, error) {
	feature, err := s.ResolveFeature(ctx, featureID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := s.guardFeature(ctx, feature); err != nil {
		return domain.Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidTitle, "task title is required")
	}
	return s.repository.CreateTask(ctx, feature.ID, title, strings.TrimSpace(scope), strings.TrimSpace(assignee))
}

// UpdateTask は呼び出し側が指定した全フィールドを適用する。nil ポインタは未指定、
// 空文字列は値を消す要求を意味する。
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
	if err := s.guardTask(ctx, task); err != nil {
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
		domain.TaskStatusNotStarted,
		domain.TaskStatusDesigning,
		domain.TaskStatusInProgress,
		domain.TaskStatusCompleted,
		domain.TaskStatusClosed,
	) {
		return domain.Task{}, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	}
	return s.repository.UpdateTask(ctx, task)
}

func (s *Service) DeleteTask(ctx context.Context, id string, cascade bool) error {
	if err := s.guardTaskID(ctx, id); err != nil {
		return err
	}
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
	if err := s.guardTaskID(ctx, taskID); err != nil {
		return domain.Document{}, err
	}
	document.TaskID = taskID
	document.ProjectID = ""
	document.FeatureID = ""
	document.IsImplementationPlan = true
	if err := validateDocumentSource(&document, true); err != nil {
		return domain.Document{}, err
	}
	return s.repository.UpsertImplementationPlan(ctx, taskID, document)
}

func (s *Service) DeleteImplementationPlan(ctx context.Context, taskID string) error {
	if err := s.guardTaskID(ctx, taskID); err != nil {
		return err
	}
	return s.repository.DeleteImplementationPlan(ctx, taskID)
}

// AddDependency は blocker だけをガードする。repository が feature をまたぐ辺を
// 拒むので、両端は同じ feature と同じ読み取り専用状態を共有する。
func (s *Service) AddDependency(ctx context.Context, blocker, blocked string) (domain.Dependency, error) {
	if err := s.guardTaskID(ctx, blocker); err != nil {
		return domain.Dependency{}, err
	}
	return s.repository.AddDependency(ctx, blocker, blocked)
}

func (s *Service) RemoveDependency(ctx context.Context, blocker, blocked string) error {
	if err := s.guardTaskID(ctx, blocker); err != nil {
		return err
	}
	return s.repository.RemoveDependency(ctx, blocker, blocked)
}

func (s *Service) AttachPullRequest(ctx context.Context, taskID, rawURL string) (domain.PullRequest, error) {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	if err := s.guardTask(ctx, task); err != nil {
		return domain.PullRequest{}, err
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
	attached, err := s.repository.UpsertPullRequest(
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
			CheckState:   domain.CheckStateUnknown,
			Stale:        true,
		},
	)
	if err != nil {
		return domain.PullRequest{}, err
	}
	return s.refreshAttachedPullRequest(ctx, attached), nil
}

// attachSyncTimeout は attach 後の refresh に上限を設ける。この refresh は書き込みの
// 副作用であって呼び出し側が求めたものではないので、到達できない host のせいで
// attach をいつまでも待たせてはならない。
const attachSyncTimeout = 30 * time.Second

// refreshAttachedPullRequest は attach 直後の pull request を取得し、記録したばかりの
// 作業が stale として表示されないようにする。task 単位の refresh を使うベストエフォート
// 処理で、GitHub に到達できなくても attach 自体は成功する。
func (s *Service) refreshAttachedPullRequest(
	ctx context.Context,
	attached domain.PullRequest,
) domain.PullRequest {
	syncContext, cancel := context.WithTimeout(ctx, attachSyncTimeout)
	defer cancel()
	_, _, _ = s.Sync(syncContext, "", attached.TaskID)
	snapshot, err := s.repository.Snapshot(ctx)
	if err != nil {
		return attached
	}
	for _, pullRequest := range snapshot.PullRequests {
		if pullRequest.TaskID == attached.TaskID {
			return pullRequest
		}
	}
	return attached
}

func (s *Service) DetachPullRequest(ctx context.Context, taskID string) error {
	if err := s.guardTaskID(ctx, taskID); err != nil {
		return err
	}
	return s.repository.DeletePullRequest(ctx, taskID)
}

func (s *Service) AddDocument(
	ctx context.Context,
	parent domain.DocumentParent,
	kind domain.DocumentKind,
	title, locator, content string,
	isImplementationPlan bool,
) (domain.Document, error) {
	if parent.Count() != 1 {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"set exactly one of project_id, feature_id, or task_id",
		)
	}
	document := domain.Document{
		ProjectID: parent.ProjectID, FeatureID: parent.FeatureID, TaskID: parent.TaskID,
		Kind: kind, Title: strings.TrimSpace(title),
		Locator: locator, Content: content, IsImplementationPlan: isImplementationPlan,
	}
	if err := validateDocumentSource(&document, false); err != nil {
		return domain.Document{}, err
	}
	if isImplementationPlan && parent.TaskID == "" {
		return domain.Document{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"only task documents can be implementation plans",
		)
	}
	resolved, err := s.resolveDocumentParent(ctx, parent, isImplementationPlan)
	if err != nil {
		return domain.Document{}, err
	}
	return s.repository.CreateDocument(
		ctx, resolved, document.Kind, document.Title, document.Locator, document.Content, isImplementationPlan,
	)
}

// resolveDocumentParent は指定された親が存在し書き込みを受け付けるかを確認し、
// 公開識別子を正規化して返す。
func (s *Service) resolveDocumentParent(
	ctx context.Context,
	parent domain.DocumentParent,
	isImplementationPlan bool,
) (domain.DocumentParent, error) {
	switch {
	case parent.ProjectID != "":
		project, err := s.ResolveProject(ctx, parent.ProjectID)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		if err := s.guardProject(project); err != nil {
			return domain.DocumentParent{}, err
		}
		return domain.DocumentParent{ProjectID: project.ID}, nil
	case parent.FeatureID != "":
		feature, err := s.ResolveFeature(ctx, parent.FeatureID)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		if err := s.guardFeature(ctx, feature); err != nil {
			return domain.DocumentParent{}, err
		}
		return domain.DocumentParent{FeatureID: feature.ID}, nil
	}
	if err := s.guardTaskID(ctx, parent.TaskID); err != nil {
		return domain.DocumentParent{}, err
	}
	if isImplementationPlan {
		if _, err := s.repository.GetImplementationPlan(ctx, parent.TaskID); err == nil {
			return domain.DocumentParent{}, domain.NewError(
				domain.DomainErrorCodeDuplicateImplementationPlan,
				"task %q already has an implementation plan", parent.TaskID,
			)
		} else if domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
			return domain.DocumentParent{}, err
		}
	}
	return domain.DocumentParent{TaskID: parent.TaskID}, nil
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
	if err := s.guardDocument(ctx, document); err != nil {
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
			"only task documents can be implementation plans",
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
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return err
	}
	if err := s.guardDocument(ctx, document); err != nil {
		return err
	}
	return s.repository.DeleteDocument(ctx, id)
}

// ReadDocumentContent は、明示的に登録されたパスと PRX が保存した Markdown だけを読む。
// サイズ上限により、プレビュー要求がサーバーやブラウザのメモリを
// 無制限に消費するのを防ぐ。
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

// deriveSnapshot は保存済み snapshot を、全呼び出し側が必要とする導出済みの形に変える。
// 同期処理も読み取り経路とこれを共有するため、選別に使う feature のステータスは
// クライアントが見るものと一致する。
func deriveSnapshot(snapshot domain.Snapshot) domain.Snapshot {
	snapshot.Tasks = domain.Derive(snapshot.Tasks, snapshot.Dependencies, snapshot.PullRequests)
	taskIndex := map[string]int{}
	for i, task := range snapshot.Tasks {
		taskIndex[task.ID] = i
		if task.Ready {
			snapshot.ReadyTasks = append(snapshot.ReadyTasks, task)
		}
		if task.DisplayState == domain.TaskDisplayStateInReview {
			snapshot.ReviewWaitingTasks = append(snapshot.ReviewWaitingTasks, task)
		}
		if slices.Contains(task.BlockLabels, domain.TaskBlockLabelConflict) {
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
		// feature が見つからない task は、データベースの参照整合性が壊れた印。
		// 先頭 feature に加算したり空スライスの範囲外を参照したりせず、読み飛ばす。
		i, ok := featureIndex[task.FeatureID]
		if !ok {
			continue
		}
		feature := &snapshot.Features[i]
		feature.TaskCount++
		if task.Ready {
			feature.ReadyCount++
		}
		if task.DisplayState == domain.TaskDisplayStateInReview {
			feature.ReviewWaitingCount++
		}
		if slices.Contains(task.BlockLabels, domain.TaskBlockLabelConflict) {
			feature.ConflictCount++
		}
		if task.DisplayState == domain.TaskDisplayStateMerged {
			feature.MergedCount++
		}
		if domain.IsTaskFinished(task.DisplayState) {
			feature.FinishedCount++
		}
	}
	archivedProjects := map[string]bool{}
	for _, project := range snapshot.Projects {
		archivedProjects[project.ID] = project.Archived
	}
	for i := range snapshot.Features {
		feature := &snapshot.Features[i]
		feature.DisplayStatus = domain.FeatureDisplayStatus(feature.Status, feature.TaskCount, feature.FinishedCount)
		// project のアーカイブは配下の feature にも及ぶ。2 つのフラグを合成するのは
		// ここだけで、クライアントは ReadOnly を読むだけでよい。
		feature.ReadOnly = feature.Archived || archivedProjects[feature.ProjectID]
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
	// 完了した feature は、アーカイブ済みと同じ理由で自動 refresh と範囲指定なしの
	// 手動 refresh の対象から外れる。pull request がもう進行中ではないため。
	// ReadOnly は feature 自身とアーカイブ済み project の両方を覆う。
	activeFeatures := map[string]bool{}
	for _, feature := range snapshot.Features {
		activeFeatures[feature.ID] = !feature.ReadOnly &&
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
