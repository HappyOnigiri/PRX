package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newDebugService(t *testing.T) (*app.Service, string, string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	databasePath := filepath.Join(root, "debug.db")
	configPath := filepath.Join(root, "config.yaml")
	database, err := store.Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	service.SetProcessInfo(app.ProcessInfo{
		Mode:               "cli",
		DatabasePath:       databasePath,
		DatabasePathSource: "flag",
		ConfigPathSource:   "flag",
	})
	return service, databasePath, configPath
}

func TestDebugReportsStorageConfigurationAndData(t *testing.T) {
	ctx := context.Background()
	service, databasePath, configPath := newDebugService(t)
	project, err := service.CreateProject(ctx, "Payments", "")
	if err != nil {
		t.Fatal(err)
	}
	archived := true
	if _, err := service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{Archived: &archived}); err != nil {
		t.Fatal(err)
	}
	feature := newFeature(t, ctx, service, "Checkout")
	task, err := service.CreateTask(ctx, feature.ID, "Payment API", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AttachPullRequest(ctx, task.ID, "https://github.com/acme/web/pull/12"); err != nil {
		t.Fatal(err)
	}

	report, err := service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Build.Version == "" || report.Runtime.Mode != "cli" {
		t.Fatalf("report=%+v", report)
	}
	// レポートはホームディレクトリを短縮するので、期待値も生パスではなく
	// 同じ規則で組み立てる。
	shortener := domain.NewDebugPathShortener()
	if report.Paths.DatabasePath != shortener.Path(databasePath) ||
		report.Paths.ConfigPath != shortener.Path(configPath) {
		t.Fatalf("paths=%+v", report.Paths)
	}
	if !report.Paths.DatabaseFileExists || !report.Paths.ConfigFileExists {
		t.Fatalf("paths=%+v", report.Paths)
	}
	if report.Storage.AppliedSchemaVersion != report.Storage.EmbeddedSchemaVersion ||
		!report.Storage.IntegrityValid {
		t.Fatalf("storage=%+v", report.Storage)
	}
	if !report.Storage.DatabaseFile.Applicable || !report.Storage.DatabaseFile.Writable {
		t.Fatalf("database file=%+v", report.Storage.DatabaseFile)
	}
	if report.Records.Features != 1 || report.Records.Tasks != 1 || report.Records.PullRequests != 1 {
		t.Fatalf("records=%+v", report.Records)
	}
	// 書き込みが拒否された理由は、行き先のアーカイブ済み project で説明される。
	// よってレポートは project 数と現れている状態を持つ必要がある。
	// 件数が同じなら名前順なので、active な project が先に来る。
	if report.Records.Projects != 2 ||
		len(report.Records.ProjectStates) != 2 ||
		report.Records.ProjectStates[0].Name != "active" ||
		report.Records.ProjectStates[1].Name != "archived" {
		t.Fatalf("records=%+v", report.Records)
	}
	if !report.Config.Valid || len(report.Config.Hosts) != 1 {
		t.Fatalf("config=%+v", report.Config)
	}
	// pull request は一度も更新されていないので、レポートはその事実を示すべきで、
	// 見るものが何もない環境のように見せてはならない。
	if report.GitHubSync.Status.LastUpdatedAt != nil {
		t.Fatalf("sync=%+v", report.GitHubSync)
	}
	if !hasDebugProblem(report, domain.DebugProblemCodeGitHubSyncNeverCompleted) {
		t.Fatalf("problems=%+v", report.Problems)
	}
}

func TestDebugReportsCredentialsWithoutSecrets(t *testing.T) {
	ctx := context.Background()
	service, _, _ := newDebugService(t)
	settings := config.Default()
	if err := settings.AddAuthMethod(config.AuthMethod{
		ID:    "inline",
		Host:  "github.com",
		Type:  config.AuthMethodTypeInline,
		Token: "github_pat_debug_secret",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.ConfigStore().Save(settings); err != nil {
		t.Fatal(err)
	}

	report, err := service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Config.AuthMethods) != 1 || !report.Config.AuthMethods[0].SecretConfigured {
		t.Fatalf("auth methods=%+v", report.Config.AuthMethods)
	}
	if strings.Contains(domain.FormatDebugReport(report), "github_pat_debug_secret") {
		t.Fatal("the report disclosed an inline token")
	}
}

func TestDebugRecordsTheServeEndpoint(t *testing.T) {
	ctx := context.Background()
	service, _, _ := newDebugService(t)
	started := time.Now().UTC().Add(-time.Minute)
	service.SetServeEndpoint("127.0.0.1:7331", started)

	report, err := service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Runtime.Mode != "serve" || report.Runtime.ListenAddress != "127.0.0.1:7331" {
		t.Fatalf("runtime=%+v", report.Runtime)
	}
	if report.Runtime.StartedAt == nil || report.Runtime.UptimeSeconds < 59 {
		t.Fatalf("runtime=%+v", report.Runtime)
	}
}

func TestDebugReportsAnUnreadableConfiguration(t *testing.T) {
	ctx := context.Background()
	service, _, configPath := newDebugService(t)
	if err := os.WriteFile(configPath, []byte("version: 1\ngithub: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Config.Valid || len(report.Config.Errors) == 0 {
		t.Fatalf("config=%+v", report.Config)
	}
	if !hasDebugProblem(report, domain.DebugProblemCodeConfigUnreadable) {
		t.Fatalf("problems=%+v", report.Problems)
	}
	// 1 つのセクションが失敗しても、レポートの残りは使えなければならない。
	if !report.Storage.IntegrityValid || report.Storage.Error != "" {
		t.Fatalf("storage=%+v", report.Storage)
	}
}

// 診断用インタフェースも同期用インタフェースも実装しない repository でも
// レポートは生成される。壊れた環境を診断可能にするのは、部分的なレポートだから。
func TestDebugReportsPartialSectionsForALimitedRepository(t *testing.T) {
	service := app.New(&debugRepository{}, nil)
	report, err := service.Debug(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Storage.Error == "" || report.Records.Error == "" || report.GitHubSync.Error == "" {
		t.Fatalf("report=%+v", report)
	}
	if report.Build.Version == "" || report.Runtime.Mode != "cli" {
		t.Fatalf("report=%+v", report)
	}
	if !hasDebugProblem(report, domain.DebugProblemCodeStorageUnavailable) {
		t.Fatalf("problems=%+v", report.Problems)
	}
}

func hasDebugProblem(report domain.DebugReport, code domain.DebugProblemCode) bool {
	for _, problem := range report.Problems {
		if problem.Code == code {
			return true
		}
	}
	return false
}

type debugRepository struct {
	repositoryStub
}

func (*debugRepository) Snapshot(context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{}, errors.New("snapshot is unavailable")
}

func (*debugRepository) Validate(context.Context) []string { return nil }

// TestDebugReportsTheInjectedDaemonState は配線層が注入した常駐の観測を、レポートを
// 生成したプロセスの記述とは別のセクションとして報告することを確かめる。
func TestDebugReportsTheInjectedDaemonState(t *testing.T) {
	ctx := context.Background()
	service, databasePath, _ := newDebugService(t)
	if service.DatabasePath() != databasePath {
		t.Fatalf("database path=%q, want %q", service.DatabasePath(), databasePath)
	}
	report, err := service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Daemon.Supported || report.Daemon.PlistStatus != domain.DebugPlistStatusUnknown {
		t.Fatalf("uninjected daemon section=%+v", report.Daemon)
	}
	service.SetDaemonInspector(func(context.Context) domain.DebugDaemonInput {
		return domain.DebugDaemonInput{
			Supported: true, Installed: true, PlistStatus: domain.DebugPlistStatusStale,
			PlistPath: "/tmp/com.user.prx.plist", Running: false,
		}
	})
	report, err = service.Debug(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Daemon.Supported || report.Daemon.PlistStatus != domain.DebugPlistStatusStale {
		t.Fatalf("daemon section=%+v", report.Daemon)
	}
	if report.Runtime.Mode != "cli" {
		t.Fatalf("runtime mode=%q, want the reporting process to stay cli", report.Runtime.Mode)
	}
	codes := make(map[domain.DebugProblemCode]bool, len(report.Problems))
	for _, problem := range report.Problems {
		codes[problem.Code] = true
	}
	if !codes[domain.DebugProblemCodeDaemonPlistStale] || !codes[domain.DebugProblemCodeDaemonNotRunning] {
		t.Fatalf("problems=%+v", report.Problems)
	}
}
