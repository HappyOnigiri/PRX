package cli

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/spf13/cobra"

	prx "github.com/HappyOnigiri/PRX"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *state) debugCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "debug",
		Short:   "Show a diagnostic report of this PRX installation",
		Example: "prx debug\nprx debug --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := s.debugReport(cmd.Context())
			return s.write(report, func(out io.Writer) error {
				_, err := io.WriteString(out, domain.FormatDebugReport(report))
				return err
			})
		},
	}
}

// debugReport はストレージと保存データを読めるサービスを優先して使う。サービスを
// 開けなかった場合は、CLI 実行から見える範囲でレポートを組み立てる。
// レポートはまさにその状況のために存在するため。
func (s *state) debugReport(ctx context.Context) domain.DebugReport {
	if s.service != nil {
		report, err := s.service.Debug(ctx)
		if err == nil {
			return report
		}
		return s.unavailableDebugReport(err)
	}
	return s.unavailableDebugReport(s.serviceOpenErr)
}

func (s *state) unavailableDebugReport(cause error) domain.DebugReport {
	now := time.Now().UTC()
	message := "the application service is unavailable"
	if cause != nil {
		message = cause.Error()
	}
	report := domain.DebugReport{
		Build: domain.NewDebugBuild(prx.Version()),
		Runtime: domain.NewDebugRuntime(domain.DebugRuntimeInput{
			Mode:          "cli",
			Demo:          s.demo,
			GitHubFixture: s.fixture != "",
		}, now),
		Paths:   domain.NewDebugPaths(s.debugPathsInput(cause)),
		Config:  s.debugConfig(),
		Storage: domain.NewDebugStorage(domain.DebugStorageInput{Error: message}),
		Records: domain.DebugData{Error: message},
		GitHubSync: domain.NewDebugGitHubSync(
			domain.DebugGitHubSyncInput{Error: message},
			now,
		),
	}
	report.Problems = domain.DetectDebugProblems(report, now)
	return report
}

func (s *state) debugPathsInput(cause error) domain.DebugPathsInput {
	input := domain.DebugPathsInput{
		DatabasePath:       s.dbPath,
		DatabasePathSource: s.dbPathSource,
		ConfigPathSource:   s.configPathSource,
		Demo:               s.demo,
	}
	// オープン失敗のエラーは解決済みの場所を持っており、CLI に明示的なパスが
	// 渡されなかったときに読み手が必要とするのはその値。
	var openErr *ServiceOpenError
	if errors.As(cause, &openErr) && openErr.DatabasePath != "" {
		input.DatabasePath = openErr.DatabasePath
	}
	if store, err := config.NewStore(s.configPath); err == nil {
		input.ConfigPath = store.Path()
	}
	return input
}

func (s *state) debugConfig() domain.DebugConfig {
	store, err := config.NewStore(s.configPath)
	if err != nil {
		return domain.NewDebugConfig(domain.DebugConfigInput{LoadError: err.Error()})
	}
	return domain.NewDebugConfig(store.DebugInput())
}
