package domain

import "testing"

func TestTaskFinishedCoversEveryDisplayState(t *testing.T) {
	tests := []struct {
		display TaskDisplayState
		want    bool
	}{
		{TaskDisplayStateNotStarted, false},
		{TaskDisplayStateInProgress, false},
		{TaskDisplayStateCompleted, true},
		{TaskDisplayStateClosed, true},
		{TaskDisplayStateMerged, true},
		{TaskDisplayStateDraft, false},
		{TaskDisplayStateConflict, false},
		{TaskDisplayStateChangesRequested, false},
		{TaskDisplayStateApproved, false},
		{TaskDisplayStateReviewWaiting, false},
		{TaskDisplayStateOpen, false},
		{TaskDisplayStateUnknown, false},
	}
	for _, test := range tests {
		t.Run(string(test.display), func(t *testing.T) {
			if got := IsTaskFinished(test.display); got != test.want {
				t.Fatalf("IsTaskFinished(%q)=%v, want %v", test.display, got, test.want)
			}
		})
	}
}

func TestFeatureDisplayStatusPrefersManualOverride(t *testing.T) {
	tests := []struct {
		name          string
		stored        FeatureStatus
		taskCount     int
		finishedCount int
		want          FeatureStatus
	}{
		{"auto without tasks stays active", FeatureStatusAuto, 0, 0, FeatureStatusActive},
		{"auto with unfinished tasks stays active", FeatureStatusAuto, 3, 2, FeatureStatusActive},
		{"auto with every task finished completes", FeatureStatusAuto, 3, 3, FeatureStatusCompleted},
		{"auto with one finished task completes", FeatureStatusAuto, 1, 1, FeatureStatusCompleted},
		{"manual active never completes", FeatureStatusActive, 2, 2, FeatureStatusActive},
		{"manual paused survives completion", FeatureStatusPaused, 2, 2, FeatureStatusPaused},
		{"manual cancelled survives completion", FeatureStatusCancelled, 2, 2, FeatureStatusCancelled},
		{"manual completed ignores unfinished tasks", FeatureStatusCompleted, 2, 0, FeatureStatusCompleted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := FeatureDisplayStatus(test.stored, test.taskCount, test.finishedCount)
			if got != test.want {
				t.Fatalf("FeatureDisplayStatus(%q, %d, %d)=%q, want %q",
					test.stored, test.taskCount, test.finishedCount, got, test.want)
			}
		})
	}
}
