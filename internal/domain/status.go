package domain

// PRDisplayState は pull request 自身の表示状態を導出する。task の表示状態とは
// 語彙が別で、taskDisplayStateFromPR がそれぞれ独立に導出する。
func PRDisplayState(pr *PullRequest) PullRequestDisplayState {
	if pr == nil {
		return PullRequestDisplayStateUnknown
	}
	if pr.State == PullRequestStateMerged {
		return PullRequestDisplayStateMerged
	}
	if pr.State == PullRequestStateClosed {
		return PullRequestDisplayStateClosed
	}
	if pr.Draft {
		return PullRequestDisplayStateDraft
	}
	if pr.Mergeability == MergeabilityConflicting {
		return PullRequestDisplayStateConflict
	}
	if pr.ReviewState == ReviewStateChangesRequested {
		return PullRequestDisplayStateChangesRequested
	}
	if pr.ReviewState == ReviewStateApproved {
		return PullRequestDisplayStateApproved
	}
	if pr.ReviewState == ReviewStateRequired {
		return PullRequestDisplayStateReviewWaiting
	}
	if pr.State == PullRequestStateOpen {
		return PullRequestDisplayStateOpen
	}
	return PullRequestDisplayStateUnknown
}

// taskDisplayStateFromPR は pull request から task のステータスを導出する。承認済みを
// レビュー中より優先するのは、承認が「変更要求が 1 件もなく誰かが通した」を意味し、
// 読み手にとって最も重い情報だからである。docs/design/domain.md を参照。
func taskDisplayStateFromPR(pr *PullRequest) TaskDisplayState {
	switch {
	case pr == nil:
		return TaskDisplayStateUnknown
	case pr.State == PullRequestStateMerged:
		return TaskDisplayStateMerged
	case pr.State == PullRequestStateClosed:
		return TaskDisplayStateClosed
	case pr.ReviewState == ReviewStateApproved:
		return TaskDisplayStateApproved
	case pr.ReviewRequestPending || respondedAfterReview(pr):
		return TaskDisplayStateInReview
	case pr.State == PullRequestStateOpen:
		return TaskDisplayStateImplemented
	default:
		return TaskDisplayStateUnknown
	}
}

// respondedAfterReview は変更要求の後に push があったかを返す。同一時刻はレビューが
// 後とみなす。どちらかの時刻が欠けていれば判定しない。
func respondedAfterReview(pr *PullRequest) bool {
	if pr.ChangesRequestedAt == nil || pr.LastPushedAt == nil {
		return false
	}
	return pr.LastPushedAt.After(*pr.ChangesRequestedAt)
}

// blockLabelsFor は task の進行を妨げている事情を、固定の順序で挙げる。
// 終了した task では呼び出し側が評価しない。
func blockLabelsFor(pr *PullRequest, dependencyUnresolved bool) []TaskBlockLabel {
	labels := []TaskBlockLabel{}
	if dependencyUnresolved {
		labels = append(labels, TaskBlockLabelDependencyUnresolved)
	}
	if pr != nil && pr.Mergeability == MergeabilityConflicting {
		labels = append(labels, TaskBlockLabelConflict)
	}
	if pr != nil && pr.ReviewState == ReviewStateChangesRequested {
		labels = append(labels, TaskBlockLabelChangesRequested)
	}
	return labels
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
		return taskDisplayStateFromPR(pr)
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
