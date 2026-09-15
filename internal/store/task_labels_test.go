package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestTaskLabelOverridesRoundTripAndPartialUpdates(t *testing.T) {
	ctx := context.Background()
	database, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Label project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Label feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	text := "Working"
	color := "#A1B2C3"
	projectUpdate := make(domain.TaskLabelOverridesUpdate)
	projectUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &text, Color: &color}
	project, err = service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{
		TaskLabelOverrides: &projectUpdate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if value := project.TaskLabelOverrides[domain.TaskLabelStatusInProgress]; value.Text != text ||
		value.Color != "#a1b2c3" {
		t.Fatalf("project override=%+v", value)
	}
	featureText := "Coding"
	featureUpdate := make(domain.TaskLabelOverridesUpdate)
	featureUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &featureText}
	feature, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		TaskLabelOverrides: &featureUpdate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if value := feature.TaskLabelOverrides[domain.TaskLabelStatusInProgress]; value.Text != featureText {
		t.Fatalf("feature override=%+v", value)
	}
	reloadedProject, err := database.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	reloadedFeature, err := database.GetFeature(ctx, feature.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedProject.TaskLabelOverrides[domain.TaskLabelStatusInProgress].Color != "#a1b2c3" ||
		reloadedFeature.TaskLabelOverrides[domain.TaskLabelStatusInProgress].Text != featureText {
		t.Fatalf(
			"reloaded project=%+v feature=%+v",
			reloadedProject.TaskLabelOverrides,
			reloadedFeature.TaskLabelOverrides,
		)
	}
	empty := ""
	clearUpdate := make(domain.TaskLabelOverridesUpdate)
	clearUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &empty}
	feature, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		TaskLabelOverrides: &clearUpdate,
	})
	if err != nil {
		t.Fatal(err)
	}
	value := feature.TaskLabelOverrides[domain.TaskLabelStatusInProgress]
	if value.Text != "" {
		t.Fatalf("text was not cleared: %+v", value)
	}

	second, err := service.CreateProject(ctx, "Second label project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{ProjectID: &second.ID})
	if err != nil {
		t.Fatal(err)
	}
	if feature.ProjectID != second.ID || feature.TaskLabelOverrides[domain.TaskLabelStatusInProgress].Text != "" {
		t.Fatalf("moved feature=%+v", feature)
	}
}

func TestTaskLabelOverridesRejectInvalidValuesAndArchivedWrites(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Label project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Label feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	badText := "bad\ntext"
	badUpdate := make(domain.TaskLabelOverridesUpdate)
	badUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &badText}
	_, err = service.UpdateProject(ctx, project.ID, domain.ProjectUpdate{
		TaskLabelOverrides: &badUpdate,
	})
	if domain.ErrorCode(err) != domain.DomainErrorCodeInvalidTaskLabel {
		t.Fatalf("invalid text error=%v code=%s", err, domain.ErrorCode(err))
	}
	archived := true
	if _, err := service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{Archived: &archived}); err != nil {
		t.Fatal(err)
	}
	text := "Coding"
	archivedUpdate := make(domain.TaskLabelOverridesUpdate)
	archivedUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &text}
	_, err = service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		TaskLabelOverrides: &archivedUpdate,
	})
	if domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
		t.Fatalf("archived feature error=%v code=%s", err, domain.ErrorCode(err))
	}
}

func TestTaskLabelAppearancesResolveGlobalProjectAndFeature(t *testing.T) {
	ctx := context.Background()
	database, err := openTestDatabaseForLabels(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configStore, err := config.NewStore(configPath)
	if err != nil {
		t.Fatal(err)
	}
	globalText := "Working"
	globalUpdate := make(domain.TaskLabelOverridesUpdate)
	globalUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &globalText}
	if _, err := configStore.Update(func(value *config.Config) error {
		return value.SetTaskLabelOverrides(globalUpdate)
	}); err != nil {
		t.Fatal(err)
	}
	service := app.NewWithConfig(database, nil, configStore)
	project, err := service.CreateProject(ctx, "Appearance project", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Appearance feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	featureText := "Coding"
	featureUpdate := make(domain.TaskLabelOverridesUpdate)
	featureUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &featureText}
	if _, err := service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{
		TaskLabelOverrides: &featureUpdate,
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var appearance domain.TaskLabelAppearance
	for _, item := range snapshot.Features {
		if item.ID == feature.ID {
			appearance = item.TaskLabelAppearances[domain.TaskLabelStatusInProgress]
		}
	}
	if appearance.Text != featureText || !appearance.TextOverridden {
		t.Fatalf("appearance=%+v", appearance)
	}
}

func openTestDatabaseForLabels(t *testing.T) (*store.Store, error) {
	t.Helper()
	return store.Open(context.Background(), filepath.Join(t.TempDir(), "labels.db"))
}
