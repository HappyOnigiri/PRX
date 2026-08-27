package domain

func PRDisplayState(pr *PullRequest) string {
	if pr == nil {
		return "unlinked"
	}
	if pr.State == "merged" {
		return "merged"
	}
	if pr.State == "closed" {
		return "closed"
	}
	if pr.Draft {
		return "draft"
	}
	if pr.Mergeability == "conflicting" {
		return "conflict"
	}
	if pr.ReviewState == "changes_requested" {
		return "changes_requested"
	}
	if pr.ReviewState == "approved" {
		return "approved"
	}
	if pr.ReviewState == "required" {
		return "review_waiting"
	}
	if pr.State == "open" {
		return "open"
	}
	return "unknown"
}

func IsSatisfied(task Task, pr *PullRequest) bool {
	if task.Status == TaskCancelled {
		return false
	}
	if task.Kind == TaskKindManual {
		return task.Status == TaskCompleted
	}
	return pr != nil && !pr.Stale && pr.State == "merged"
}

func IsIncomplete(task Task, pr *PullRequest) bool {
	if task.Status == TaskCancelled || task.Status == TaskCompleted {
		return false
	}
	return task.Kind != TaskKindPR || pr == nil || pr.State != "merged"
}
