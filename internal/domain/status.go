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
	return task.Status == TaskStatusAuto &&
		(display == TaskDisplayStateNotStarted || display == TaskDisplayStateDesigned)
}
