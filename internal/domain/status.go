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

// IsTaskFinished は、導出されたタスク状態がそのタスクの作業の終了を表すかを返す。
// 保存済み status ではなく導出状態を見るため、終了状態の status を持たないタスクは
// pull request に従う。
func IsTaskFinished(display TaskDisplayState) bool {
	return display == TaskDisplayStateCompleted ||
		display == TaskDisplayStateClosed ||
		display == TaskDisplayStateMerged
}

// FeatureDisplayStatus は feature の表示用 status を導出する。auto 以外の保存済み
// status は手動の上書きとみなし、そのまま返す。
// 自動導出の内容は docs/design/domain.md に記載する。
func FeatureDisplayStatus(stored FeatureStatus, taskCount, finishedCount int) FeatureStatus {
	if stored != FeatureStatusAuto {
		return stored
	}
	if taskCount >= 1 && finishedCount == taskCount {
		return FeatureStatusCompleted
	}
	return FeatureStatusActive
}

// displayStateFor は docs/design/domain.md の順序でタスクの表示状態を導出する。
// 終了状態の保存済み status は pull request より優先し、未終了なら pull request に
// 譲り、designing は plan に譲る。
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

// IsSatisfied は、タスクがそれを待つ依存を解消するかを返す。終了状態の保存済み
// status だけでも解消し、open・closed・merged に達した pull request も同様。
// docs/design/domain.md を参照。
func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusClosed {
		return true
	}
	return pr != nil && (pr.State == PullRequestStateOpen ||
		pr.State == PullRequestStateClosed || pr.State == PullRequestStateMerged)
}

// isReadyCandidate は導出状態だけを見る。実装が始まっていない作業だけが blocker の
// 解消を問う。docs/design/domain.md のとおり designing は not started や designed と
// 同じ扱いとする。
func isReadyCandidate(display TaskDisplayState) bool {
	return display == TaskDisplayStateNotStarted ||
		display == TaskDisplayStateDesigning ||
		display == TaskDisplayStateDesigned
}
