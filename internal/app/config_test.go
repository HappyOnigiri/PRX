package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestAttachPullRequestUsesConfiguredHostURL(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "attach.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Default()
	if err := settings.AddHost(
		config.Host{Host: "ghe.example.com", WebURL: "https://ghe.example.com/git"},
	); err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(settings); err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	feature, err := service.CreateFeature(ctx, "ghe", "GHE", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(ctx, feature.ID, "Enterprise PR", "", domain.TaskKindPR, "")
	if err != nil {
		t.Fatal(err)
	}
	value, err := service.AttachPullRequest(ctx, task.ID, "https://GHE.Example.com/Acme/API/pull/42")
	if err != nil {
		t.Fatal(err)
	}
	if value.Host != "ghe.example.com" || value.URL != "https://ghe.example.com/git/Acme/API/pull/42" {
		t.Fatalf("attached pull request=%+v", value)
	}
	if _, err := service.AttachPullRequest(
		ctx,
		task.ID,
		"https://unknown.example.com/acme/api/pull/42",
	); configErrorCode(
		err,
	) != domain.DomainErrorCodeInvalidPullRequestURL {
		t.Fatalf("unconfigured host error=%v code=%s", err, configErrorCode(err))
	}
}

func TestSyncReportsInvalidGitHubConfig(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("version: 1\nunknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configStore, err := config.NewStore(configPath)
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	_, _, err = service.Sync(ctx, "", "")
	if configErrorCode(err) != domain.DomainErrorCodeInvalidConfig {
		t.Fatalf("sync error=%v code=%s", err, configErrorCode(err))
	}
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Message == "" {
		t.Fatalf("sync error lacks domain details: %v", err)
	}
}

func configErrorCode(err error) domain.DomainErrorCode {
	return domain.ErrorCode(err)
}
