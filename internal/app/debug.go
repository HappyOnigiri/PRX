package app

import (
	"context"
	"time"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// DiagnosticsRepository は実データベースを開いているとき SQLite repository が実装する。
// GitHub sync state のインタフェースと同じ理由で任意のままにしている。
// そうしないと小さな repository の fake すべてがこれらを持つことになる。
type DiagnosticsRepository interface {
	Path() string
	AppliedSchemaVersion(ctx context.Context) (int, error)
	EmbeddedSchemaVersion() (int, error)
	DatabaseFile() domain.DebugDatabaseFile
	ListGitHubRepositoryAuthCache(ctx context.Context) ([]domain.DebugAuthCacheEntry, error)
}

// ProcessInfo は配線層だけが知る事実、すなわちこのプロセスの起動方法と、
// 指し示されたパスを運ぶ。
type ProcessInfo struct {
	Mode               string
	Demo               bool
	GitHubFixture      bool
	DatabasePath       string
	DatabasePathSource string
	ConfigPathSource   string
}

// serveEndpoint はサーバーが受け付けたアドレス。listener を bind した後にしか
// 判明しない。
type serveEndpoint struct {
	address   string
	startedAt time.Time
}

// SetProcessInfo はこのプロセスの起動方法を記録する。service の配線中に一度だけ、
// リクエストが到達しうる前に呼ばれる。
func (s *Service) SetProcessInfo(info ProcessInfo) { s.processInfo = info }

// SetServeEndpoint は bind した listen アドレスを記録する。service のフィールドで
// 唯一、listener 生成後に書かれるため、サーバー起動順序に頼らず atomic に保存する。
func (s *Service) SetServeEndpoint(address string, startedAt time.Time) {
	s.serveEndpoint.Store(&serveEndpoint{address: address, startedAt: startedAt.UTC()})
}

// Debug は診断レポートを組み立てる。壊れているときこそ価値があるので各セクションは
// 独立に失敗し、同期は行わない。同期すると読み手が知りたい実行エラーが消えてしまう。
func (s *Service) Debug(ctx context.Context) (domain.DebugReport, error) {
	now := s.currentTime()
	report := domain.DebugReport{
		Build:   domain.NewDebugBuild(prx.Version()),
		Runtime: domain.NewDebugRuntime(s.debugRuntimeInput(), now),
		Paths:   domain.NewDebugPaths(s.debugPathsInput()),
		Config:  s.debugConfig(),
		Storage: s.debugStorage(ctx),
	}
	snapshot, snapshotErr := s.repository.Snapshot(ctx)
	if snapshotErr != nil {
		report.Records = domain.DebugData{Error: snapshotErr.Error()}
	} else {
		snapshot = deriveSnapshot(snapshot)
		report.Records = domain.NewDebugData(snapshot)
	}
	report.GitHubSync = s.debugGitHubSync(ctx, snapshot.PullRequests, now)
	report.Problems = domain.DetectDebugProblems(report, now)
	return report, nil
}

func (s *Service) debugRuntimeInput() domain.DebugRuntimeInput {
	input := domain.DebugRuntimeInput{
		Mode:          s.processInfo.Mode,
		Demo:          s.processInfo.Demo,
		GitHubFixture: s.processInfo.GitHubFixture,
	}
	if input.Mode == "" {
		input.Mode = "cli"
	}
	if endpoint := s.serveEndpoint.Load(); endpoint != nil {
		input.Mode = "serve"
		input.ListenAddress = endpoint.address
		started := endpoint.startedAt
		input.StartedAt = &started
	}
	return input
}

func (s *Service) debugPathsInput() domain.DebugPathsInput {
	input := domain.DebugPathsInput{
		DatabasePath:       s.processInfo.DatabasePath,
		DatabasePathSource: s.processInfo.DatabasePathSource,
		ConfigPathSource:   s.processInfo.ConfigPathSource,
		Demo:               s.processInfo.Demo,
	}
	if repository, ok := s.repository.(DiagnosticsRepository); ok {
		input.DatabasePath = repository.Path()
	}
	if s.configStore != nil {
		input.ConfigPath = s.configStore.Path()
	}
	return input
}

func (s *Service) debugConfig() domain.DebugConfig {
	if s.configStore == nil {
		return domain.NewDebugConfig(domain.DebugConfigInput{LoadError: "configuration is unavailable"})
	}
	return domain.NewDebugConfig(s.configStore.DebugInput())
}

func (s *Service) debugStorage(ctx context.Context) domain.DebugStorage {
	repository, ok := s.repository.(DiagnosticsRepository)
	if !ok {
		return domain.NewDebugStorage(domain.DebugStorageInput{Error: "storage diagnostics are unavailable"})
	}
	input := domain.DebugStorageInput{
		IntegrityErrors: s.repository.Validate(ctx),
		DatabaseFile:    repository.DatabaseFile(),
	}
	applied, err := repository.AppliedSchemaVersion(ctx)
	if err != nil {
		input.Error = err.Error()
	}
	input.AppliedSchemaVersion = applied
	embedded, err := repository.EmbeddedSchemaVersion()
	if err != nil && input.Error == "" {
		input.Error = err.Error()
	}
	input.EmbeddedSchemaVersion = embedded
	return domain.NewDebugStorage(input)
}

func (s *Service) debugGitHubSync(
	ctx context.Context,
	pullRequests []domain.PullRequest,
	now time.Time,
) domain.DebugGitHubSync {
	input := domain.DebugGitHubSyncInput{PullRequests: pullRequests}
	status, err := s.SyncStatus(ctx)
	if err != nil {
		input.Error = err.Error()
	}
	input.Status = status
	if repository, ok := s.repository.(DiagnosticsRepository); ok {
		cache, cacheErr := repository.ListGitHubRepositoryAuthCache(ctx)
		if cacheErr != nil && input.Error == "" {
			input.Error = cacheErr.Error()
		}
		input.AuthCache = cache
	}
	return domain.NewDebugGitHubSync(input, now)
}
