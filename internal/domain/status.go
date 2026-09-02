package domain

func PRDisplayState(pr *PullRequest) TaskDisplayState {
	if pr == nil {
		return TaskDisplayStateUnknown
	}
	if pr.State == PullRequestStateMerged {
		return TaskDisplayStateMerged
	}
	if pr.State == PullRequestStateClosed {
		return TaskDisplayStateClosed
	}
	if pr.Draft {
		return TaskDisplayStateDraft
	}
	if pr.Mergeability == MergeabilityConflicting {
		return TaskDisplayStateConflict
	}
	if pr.ReviewState == ReviewStateChangesRequested {
		return TaskDisplayStateChangesRequested
	}
	if pr.ReviewState == ReviewStateApproved {
		return TaskDisplayStateApproved
	}
	if pr.ReviewState == ReviewStateRequired {
		return TaskDisplayStateReviewWaiting
	}
	if pr.State == PullRequestStateOpen {
		return TaskDisplayStateOpen
	}
	return TaskDisplayStateUnknown
}

// IsTaskFinished reports whether a derived task state ends the work the task
// tracks. Manual completions and closures count, and so do merged and closed
// pull requests, matching how IsSatisfied treats completion and closure alike.
// It reads the derived state rather than the stored status so an automatic task
// follows its pull request.
func IsTaskFinished(display TaskDisplayState) bool {
	return display == TaskDisplayStateCompleted ||
		display == TaskDisplayStateClosed ||
		display == TaskDisplayStateMerged
}

// FeatureDisplayStatus derives the status presented for a feature. A stored
// status other than auto is a manual override and is returned unchanged, so a
// feature returned to active stays active while its tasks remain finished.
// Auto reports completed once the feature owns at least one task and every one
// of them is finished; a feature without tasks has nothing to complete.
func FeatureDisplayStatus(stored FeatureStatus, taskCount, finishedCount int) FeatureStatus {
	if stored != FeatureStatusAuto {
		return stored
	}
	if taskCount >= 1 && finishedCount == taskCount {
		return FeatureStatusCompleted
	}
	return FeatureStatusActive
}

func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status != TaskStatusAuto {
		return task.Status == TaskStatusCompleted || task.Status == TaskStatusClosed
	}
	if task.Kind != TaskKindPR {
		return false
	}
	return pr != nil && (pr.State == PullRequestStateOpen ||
		pr.State == PullRequestStateClosed || pr.State == PullRequestStateMerged)
}

func isReadyCandidate(task Task, display TaskDisplayState) bool {
	if task.Status == TaskStatusNotStarted {
		return true
	}
	return task.Status == TaskStatusAuto && display == TaskDisplayStateNotStarted
}
