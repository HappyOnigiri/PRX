package config_test

import (
	"path/filepath"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestTaskLabelConfigRoundTripAndPartialUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	text := "  Working  "
	color := "#A1B2C3"
	update := make(domain.TaskLabelOverridesUpdate)
	update[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &text, Color: &color}
	value, err := store.Update(func(settings *config.Config) error {
		return settings.SetTaskLabelOverrides(update)
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.TaskLabels[domain.TaskLabelStatusInProgress] != (domain.TaskLabelOverride{
		Text: "Working", Color: "#a1b2c3",
	}) {
		t.Fatalf("updated labels=%+v", value.TaskLabels)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskLabels[domain.TaskLabelStatusInProgress] != value.TaskLabels[domain.TaskLabelStatusInProgress] {
		t.Fatalf("loaded labels=%+v", loaded.TaskLabels)
	}
	empty := ""
	clearUpdate := make(domain.TaskLabelOverridesUpdate)
	clearUpdate[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverrideUpdate{Text: &empty}
	loaded, err = store.Update(func(settings *config.Config) error {
		return settings.SetTaskLabelOverrides(clearUpdate)
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskLabels[domain.TaskLabelStatusInProgress].Text != "" ||
		loaded.TaskLabels[domain.TaskLabelStatusInProgress].Color != "#a1b2c3" {
		t.Fatalf("partial clear removed too much=%+v", loaded.TaskLabels)
	}
}

func TestTaskLabelConfigRejectsInvalidValues(t *testing.T) {
	for name, update := range map[string]domain.TaskLabelOverridesUpdate{
		"unknown key": labelUpdate("status.missing", domain.TaskLabelOverrideUpdate{Text: stringPointer("Missing")}),
		"invalid text": labelUpdate(
			domain.TaskLabelStatusInProgress,
			domain.TaskLabelOverrideUpdate{Text: stringPointer("bad\ntext")},
		),
		"invalid color": labelUpdate(
			domain.TaskLabelStatusInProgress,
			domain.TaskLabelOverrideUpdate{Color: stringPointer("red")},
		),
	} {
		t.Run(name, func(t *testing.T) {
			value := config.Default()
			if err := value.SetTaskLabelOverrides(
				update,
			); domain.ErrorCode(
				err,
			) != domain.DomainErrorCodeInvalidTaskLabel {
				t.Fatal("invalid task label update was accepted")
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func labelUpdate(key domain.TaskLabelKey, value domain.TaskLabelOverrideUpdate) domain.TaskLabelOverridesUpdate {
	result := make(domain.TaskLabelOverridesUpdate)
	result[key] = value
	return result
}
