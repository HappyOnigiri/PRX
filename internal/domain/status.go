package domain

func PRDisplayState(pr *PullRequest) TaskDisplayState {
	if pr == nil {
		return TaskDisplayStateUnlinked
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

func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status == TaskStatusCancelled {
		return false
	}
	if task.Kind == TaskKindManual {
		return task.Status == TaskStatusCompleted
	}
	return pr != nil && !pr.Stale && pr.State == PullRequestStateMerged
}

func IsIncomplete(task Task, pr *PullRequest) bool {
	if task.Status == TaskStatusCancelled || task.Status == TaskStatusCompleted {
		return false
	}
	return task.Kind != TaskKindPR || pr == nil || pr.State != PullRequestStateMerged
}
