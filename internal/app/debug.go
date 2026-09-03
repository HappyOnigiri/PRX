package app

import (
	"context"
	"time"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

// DiagnosticsRepository is implemented by the SQLite repository when a real
// database is open. It stays optional for the same reason the GitHub sync state
// interface does: the small repository fakes used by application tests would all
// have to grow these methods otherwise.
type DiagnosticsRepository interface {
	Path() string
	AppliedSchemaVersion(ctx context.Context) (int, error)
	EmbeddedSchemaVersion() (int, error)
	DatabaseFile() domain.DebugDatabaseFile
	ListGitHubRepositoryAuthCache(ctx context.Context) ([]domain.DebugAuthCacheEntry, error)
}

// ProcessInfo carries the facts only the wiring layer knows: how this process
// was started and which locations it was pointed at.
type ProcessInfo struct {
	Mode               string
	Demo               bool
	GitHubFixture      bool
	DatabasePath       string
	DatabasePathSource string
	ConfigPathSource   string
}

// serveEndpoint is the address the server accepted, which is known only after
// the listener is bound.
type serveEndpoint struct {
	address   string
	startedAt time.Time
}

// SetProcessInfo records how this process was started. It is called once while
// the service is being wired, before any request can reach it.
func (s *Service) SetProcessInfo(info ProcessInfo) { s.processInfo = info }

// SetServeEndpoint records the bound listen address. Every other field of the
// service is immutable after construction; this one is written once the listener
// exists and is read afterwards by HTTP handlers, so it is stored atomically
// rather than relying on the ordering of server startup. Its arguments stay
// primitive because the CLI that calls it cannot import this package.
func (s *Service) SetServeEndpoint(address string, startedAt time.Time) {
	s.serveEndpoint.Store(&serveEndpoint{address: address, startedAt: startedAt.UTC()})
}

// Debug assembles the diagnostic report. Every section fails independently: a
// report is most valuable when something is broken, so one unreadable section
// must not remove the others.
//
// It deliberately performs no synchronization. SyncIfDue would clear the run
// error the reader asked about, rewrite the staleness of every pull request, and
// block on an unreachable host.
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
