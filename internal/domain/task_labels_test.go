package domain_test

import (
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestTaskLabelNormalization(t *testing.T) {
	value, err := domain.NormalizeTaskLabelOverride(domain.TaskLabelOverride{
		Text:  "  作業中  ",
		Color: " #A1b2C3 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Text != "作業中" || value.Color != "#a1b2c3" {
		t.Fatalf("normalized=%+v", value)
	}

	for name, bad := range map[string]domain.TaskLabelOverride{
		"too long": {Text: strings.Repeat("a", 33)},
		"newline":  {Text: "line\nfeed"},
		"control":  {Text: "bad\x00text"},
		"color":    {Color: "#12345"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.NormalizeTaskLabelOverride(bad); err == nil {
				t.Fatal("invalid task label was accepted")
			}
		})
	}
}

func TestResolveTaskLabelAppearancesInheritsFieldsIndependently(t *testing.T) {
	global := make(domain.TaskLabelOverrides)
	global[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverride{
		Text: "Working", Color: "#112233",
	}
	project := make(domain.TaskLabelOverrides)
	project[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverride{Color: "#445566"}
	feature := make(domain.TaskLabelOverrides)
	feature[domain.TaskLabelStatusInProgress] = domain.TaskLabelOverride{Text: "Coding"}

	appearances := domain.ResolveTaskLabelAppearances(global, project, feature)
	value := appearances[domain.TaskLabelStatusInProgress]
	if value.Text != "Coding" || value.Color != "#445566" ||
		!value.TextOverridden || !value.ColorOverridden {
		t.Fatalf("appearance=%+v", value)
	}

	value = appearances[domain.TaskLabelStatusCompleted]
	if value.Text != "completed" || value.TextOverridden || value.ColorOverridden {
		t.Fatalf("built-in appearance=%+v", value)
	}
}

func TestValidateTaskLabelOverridesRejectsUnknownKeys(t *testing.T) {
	values := make(domain.TaskLabelOverrides)
	values[domain.TaskLabelKey("status.missing")] = domain.TaskLabelOverride{Text: "Missing"}
	err := domain.ValidateTaskLabelOverrides(values)
	if err == nil || !strings.Contains(err.Error(), "unknown task label key") {
		t.Fatalf("error=%v", err)
	}
}

func TestSortedTaskLabelKeys(t *testing.T) {
	values := make(domain.TaskLabelOverrides)
	values[domain.TaskLabelStatusClosed] = domain.TaskLabelOverride{Text: "Closed"}
	values[domain.TaskLabelStatusNotStarted] = domain.TaskLabelOverride{Text: "Queue"}
	keys := domain.SortedTaskLabelKeys(values)
	if len(keys) != 2 || keys[0] != domain.TaskLabelStatusClosed || keys[1] != domain.TaskLabelStatusNotStarted {
		t.Fatalf("sorted keys=%v", keys)
	}
}

func TestTaskLabelKeyHelpersUseUnknownBuckets(t *testing.T) {
	if got := domain.TaskLabelKeyForDisplayState(
		domain.TaskDisplayState("future"),
	); got != domain.TaskLabelStatusUnknown {
		t.Fatalf("unknown display state key=%q", got)
	}
	if got := domain.TaskLabelKeyForBlockLabel(domain.TaskBlockLabel("future")); got != domain.TaskLabelBlockUnknown {
		t.Fatalf("unknown block key=%q", got)
	}
}
