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
// tracks. It reads the derived state rather than the stored status, so a task
// without a finished status follows its pull request.
func IsTaskFinished(display TaskDisplayState) bool {
	return display == TaskDisplayStateCompleted ||
		display == TaskDisplayStateClosed ||
		display == TaskDisplayStateMerged
}

// FeatureDisplayStatus derives the status presented for a feature. A stored
// status other than auto is a manual override and is returned unchanged.
// docs/design/domain.md records what the automatic status derives.
func FeatureDisplayStatus(stored FeatureStatus, taskCount, finishedCount int) FeatureStatus {
	if stored != FeatureStatusAuto {
		return stored
	}
	if taskCount >= 1 && finishedCount == taskCount {
		return FeatureStatusCompleted
	}
	return FeatureStatusActive
}

// displayStateFor derives the state presented for a task, in the order
// docs/design/domain.md records: a finished stored status outranks the pull
// request, an unfinished one yields to it, and designing yields to a plan.
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
// A finished stored status settles them on its own, and so does a pull request
// that reached open, closed, or merged. See docs/design/domain.md.
func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusClosed {
		return true
	}
	return pr != nil && (pr.State == PullRequestStateOpen ||
		pr.State == PullRequestStateClosed || pr.State == PullRequestStateMerged)
}

// isReadyCandidate reads the derived state alone: only work whose
// implementation has not begun asks whether its blockers are clear. Designing
// sits with not started and designed, as docs/design/domain.md records.
func isReadyCandidate(display TaskDisplayState) bool {
	return display == TaskDisplayStateNotStarted ||
		display == TaskDisplayStateDesigning ||
		display == TaskDisplayStateDesigned
}
