package cli

import (
	"strings"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestResolveTaskStatusInputUsesStableAndScopedLabels(t *testing.T) {
	appearances := make(domain.TaskLabelAppearances)
	appearances[domain.TaskLabelStatusNotStarted] = domain.TaskLabelAppearance{Text: "Queue"}
	appearances[domain.TaskLabelStatusDesigning] = domain.TaskLabelAppearance{Text: "Queue"}
	appearances[domain.TaskLabelStatusInProgress] = domain.TaskLabelAppearance{Text: "  Coding  "}
	appearances[domain.TaskLabelStatusDesigned] = domain.TaskLabelAppearance{Text: "Derived"}
	snapshot := domain.Snapshot{
		Features: []domain.Feature{{
			ID:                   "F-1",
			TaskLabelAppearances: appearances,
		}},
		Tasks: []domain.Task{{ID: "T-1", FeatureID: "F-1"}},
	}
	status, err := resolveTaskStatusInput(" coding ", "T-1", snapshot)
	if err != nil || status == nil || *status != domain.TaskStatusInProgress {
		t.Fatalf("custom status=%v err=%v", status, err)
	}
	status, err = resolveTaskStatusInput("IN_PROGRESS", "T-1", snapshot)
	if err != nil || status == nil || *status != domain.TaskStatusInProgress {
		t.Fatalf("stable status=%v err=%v", status, err)
	}
	status, err = resolveTaskStatusInput("Derived", "T-1", snapshot)
	if err == nil || status != nil || domain.ErrorCode(err) != domain.DomainErrorCodeInvalidStatus {
		t.Fatalf("derived status=%v err=%v", status, err)
	}
	_, err = resolveTaskStatusInput("queue", "T-1", snapshot)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") ||
		domain.ErrorCode(err) != domain.DomainErrorCodeInvalidStatus {
		t.Fatalf("ambiguous status error=%v", err)
	}
}

func TestTaskHumanRendererUsesCustomTextWithoutChangingJSONValues(t *testing.T) {
	task := domain.Task{
		ID: "T-1", FeatureID: "F-1", Status: domain.TaskStatusInProgress,
		DisplayState: domain.TaskDisplayStateInProgress,
	}
	featureAppearances := make(domain.TaskLabelAppearances)
	featureAppearances[domain.TaskLabelStatusInProgress] = domain.TaskLabelAppearance{Text: "Coding"}
	appearances := map[string]domain.TaskLabelAppearances{"F-1": featureAppearances}
	if got := taskDisplayLabel(task, appearances); got != "Coding" {
		t.Fatalf("display label=%q", got)
	}
}
