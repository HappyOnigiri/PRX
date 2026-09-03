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
	feature, err := service.CreateFeature(ctx, "checkout", "Checkout", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Payment API", "", domain.TaskKindPR, "")
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
	// The report shortens the home directory, so the expectation is built with
	// the same rule rather than with the raw path.
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
	if !report.Config.Valid || len(report.Config.Hosts) != 1 {
		t.Fatalf("config=%+v", report.Config)
	}
	// The pull request has never been refreshed, so the report has to say so
	// rather than presenting an installation with nothing to look at.
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
	// The rest of the report still has to be usable when one section fails.
	if !report.Storage.IntegrityValid || report.Storage.Error != "" {
		t.Fatalf("storage=%+v", report.Storage)
	}
}

// A repository that implements neither the diagnostics nor the synchronization
// interface still produces a report, because a partial report is what makes a
// broken installation diagnosable.
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
