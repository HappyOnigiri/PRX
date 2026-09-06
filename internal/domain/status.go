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
// tracks. Stored completions and closures count, and so do merged and closed
// pull requests, matching how IsSatisfied treats completion and closure alike.
// It reads the derived state rather than the stored status so a task without a
// finished status follows its pull request.
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

// displayStateFor derives the state presented for a task. A finished stored
// status is a decision about the task itself and outranks the pull request, so
// a task marked completed stays completed while its pull request is still open.
// An unfinished status yields to an attached pull request instead, which is how
// linking one moves a task from in progress on to review without a second edit.
// Designing yields once more, to a registered implementation plan, so the task
// an agent marked as being designed reaches designed by registering the plan
// rather than by a second edit of the status.
func displayStateFor(task Task, pr *PullRequest) TaskDisplayState {
	if task.Status == TaskStatusCompleted {
		return TaskDisplayStateCompleted
	}
	if task.Status == TaskStatusClosed {
		return TaskDisplayStateClosed
	}
	if pr != nil {
		return PRDisplayState(pr)
	}
	if task.Status == TaskStatusInProgress {
		return TaskDisplayStateInProgress
	}
	if task.HasImplementationPlan {
		return TaskDisplayStateDesigned
	}
	if task.Status == TaskStatusDesigning {
		return TaskDisplayStateDesigning
	}
	return TaskDisplayStateNotStarted
}

// IsSatisfied reports whether a task settles the dependencies that wait on it.
// A finished stored status settles them on its own, and so does an attached
// pull request that reached open, closed, or merged. An unfinished status
// without a pull request does not, so a task left in progress keeps its
// dependents waiting.
func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusClosed {
		return true
	}
	return pr != nil && (pr.State == PullRequestStateOpen ||
		pr.State == PullRequestStateClosed || pr.State == PullRequestStateMerged)
}

// isReadyCandidate reads the derived state alone. A task that reached a pull
// request or a finished status is past the point readiness describes, and only
// work whose implementation has not begun asks whether its blockers are clear.
// Designing sits with not started and designed, because designing decides how
// the work will be built rather than starting to build it.
func isReadyCandidate(display TaskDisplayState) bool {
	return display == TaskDisplayStateNotStarted ||
		display == TaskDisplayStateDesigning ||
		display == TaskDisplayStateDesigned
}
