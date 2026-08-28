package rpc

import (
	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func protoFeature(v domain.Feature) *prxv1.Feature {
	return &prxv1.Feature{Id: v.ID, Slug: v.Slug, Title: v.Title, Description: v.Description, Status: protoFeatureStatus(v.Status), Archived: v.Archived, CreatedAt: v.CreatedAt.Format(timeFormat), UpdatedAt: v.UpdatedAt.Format(timeFormat), TaskCount: int32(v.TaskCount), ReadyCount: int32(v.ReadyCount), ReviewWaitingCount: int32(v.ReviewWaitingCount), ConflictCount: int32(v.ConflictCount), MergedCount: int32(v.MergedCount)}
}

func protoTask(v domain.Task) *prxv1.Task {
	return &prxv1.Task{Id: v.ID, FeatureId: v.FeatureID, Title: v.Title, Scope: v.Scope, Kind: protoTaskKind(v.Kind), Status: protoTaskStatus(v.Status), Assignee: v.Assignee, CreatedAt: v.CreatedAt.Format(timeFormat), UpdatedAt: v.UpdatedAt.Format(timeFormat), Ready: v.Ready, DisplayState: protoTaskDisplayState(v.DisplayState), BlockedReason: protoBlockedReason(v)}
}

func protoDependency(v domain.Dependency) *prxv1.Dependency {
	return &prxv1.Dependency{BlockerTaskId: v.BlockerTaskID, BlockedTaskId: v.BlockedTaskID, CreatedAt: v.CreatedAt.Format(timeFormat)}
}

func protoPullRequest(v domain.PullRequest) *prxv1.PullRequest {
	result := &prxv1.PullRequest{TaskId: v.TaskID, Owner: v.Owner, Repository: v.Repository, Number: v.Number, Url: v.URL, NodeId: v.NodeID, Author: v.Author, Assignees: v.Assignees, State: protoPullRequestState(v.State), Draft: v.Draft, ReviewState: protoReviewState(v.ReviewState), Mergeability: protoMergeability(v.Mergeability), SyncError: v.SyncError, Stale: v.Stale, DisplayState: protoPullRequestDisplayState(v.DisplayState)}
	if v.GitHubUpdatedAt != nil {
		result.GithubUpdatedAt = v.GitHubUpdatedAt.Format(timeFormat)
	}
	if v.LastSyncedAt != nil {
		result.LastSyncedAt = v.LastSyncedAt.Format(timeFormat)
	}
	return result
}

func protoDocument(v domain.Document) *prxv1.Document {
	return &prxv1.Document{Id: v.ID, FeatureId: v.FeatureID, TaskId: v.TaskID, Kind: protoDocumentKind(v.Kind), Title: v.Title, Value: v.Value, CreatedAt: v.CreatedAt.Format(timeFormat)}
}

func protoSnapshot(v domain.Snapshot) *prxv1.Snapshot {
	result := &prxv1.Snapshot{}
	for _, item := range v.Features {
		result.Features = append(result.Features, protoFeature(item))
	}
	for _, item := range v.Tasks {
		result.Tasks = append(result.Tasks, protoTask(item))
	}
	for _, item := range v.Dependencies {
		result.Dependencies = append(result.Dependencies, protoDependency(item))
	}
	for _, item := range v.PullRequests {
		result.PullRequests = append(result.PullRequests, protoPullRequest(item))
	}
	for _, item := range v.Documents {
		result.Documents = append(result.Documents, protoDocument(item))
	}
	for _, item := range v.ReadyTasks {
		result.ReadyTasks = append(result.ReadyTasks, protoTask(item))
	}
	for _, item := range v.ReviewWaitingTasks {
		result.ReviewWaitingTasks = append(result.ReviewWaitingTasks, protoTask(item))
	}
	for _, item := range v.ConflictTasks {
		result.ConflictTasks = append(result.ConflictTasks, protoTask(item))
	}
	for _, item := range v.StaleTasks {
		result.StaleTasks = append(result.StaleTasks, protoTask(item))
	}
	return result
}

func protoFeatureStatus(value string) prxv1.FeatureStatus {
	switch value {
	case "active":
		return prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE
	case "paused":
		return prxv1.FeatureStatus_FEATURE_STATUS_PAUSED
	case "completed":
		return prxv1.FeatureStatus_FEATURE_STATUS_COMPLETED
	case "cancelled":
		return prxv1.FeatureStatus_FEATURE_STATUS_CANCELLED
	default:
		return prxv1.FeatureStatus_FEATURE_STATUS_UNSPECIFIED
	}
}

func domainFeatureStatus(value *prxv1.FeatureStatus) *string {
	if value == nil {
		return nil
	}
	result := ""
	switch *value {
	case prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE:
		result = "active"
	case prxv1.FeatureStatus_FEATURE_STATUS_PAUSED:
		result = "paused"
	case prxv1.FeatureStatus_FEATURE_STATUS_COMPLETED:
		result = "completed"
	case prxv1.FeatureStatus_FEATURE_STATUS_CANCELLED:
		result = "cancelled"
	}
	return &result
}

func protoTaskKind(value string) prxv1.TaskKind {
	switch value {
	case domain.TaskKindPR:
		return prxv1.TaskKind_TASK_KIND_PULL_REQUEST
	case domain.TaskKindManual:
		return prxv1.TaskKind_TASK_KIND_MANUAL
	default:
		return prxv1.TaskKind_TASK_KIND_UNSPECIFIED
	}
}

func domainTaskKind(value prxv1.TaskKind) string {
	switch value {
	case prxv1.TaskKind_TASK_KIND_PULL_REQUEST:
		return domain.TaskKindPR
	case prxv1.TaskKind_TASK_KIND_MANUAL:
		return domain.TaskKindManual
	default:
		return ""
	}
}

func protoTaskStatus(value string) prxv1.TaskStatus {
	switch value {
	case domain.TaskPlanned:
		return prxv1.TaskStatus_TASK_STATUS_PLANNED
	case domain.TaskInProgress:
		return prxv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.TaskCompleted:
		return prxv1.TaskStatus_TASK_STATUS_COMPLETED
	case domain.TaskCancelled:
		return prxv1.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return prxv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func domainTaskStatus(value *prxv1.TaskStatus) *string {
	if value == nil {
		return nil
	}
	result := ""
	switch *value {
	case prxv1.TaskStatus_TASK_STATUS_PLANNED:
		result = domain.TaskPlanned
	case prxv1.TaskStatus_TASK_STATUS_IN_PROGRESS:
		result = domain.TaskInProgress
	case prxv1.TaskStatus_TASK_STATUS_COMPLETED:
		result = domain.TaskCompleted
	case prxv1.TaskStatus_TASK_STATUS_CANCELLED:
		result = domain.TaskCancelled
	}
	return &result
}

func protoTaskDisplayState(value string) prxv1.TaskDisplayState {
	states := map[string]prxv1.TaskDisplayState{
		"planned":           prxv1.TaskDisplayState_TASK_DISPLAY_STATE_PLANNED,
		"in_progress":       prxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_PROGRESS,
		"completed":         prxv1.TaskDisplayState_TASK_DISPLAY_STATE_COMPLETED,
		"cancelled":         prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CANCELLED,
		"unlinked":          prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNLINKED,
		"merged":            prxv1.TaskDisplayState_TASK_DISPLAY_STATE_MERGED,
		"closed":            prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CLOSED,
		"draft":             prxv1.TaskDisplayState_TASK_DISPLAY_STATE_DRAFT,
		"conflict":          prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CONFLICT,
		"changes_requested": prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CHANGES_REQUESTED,
		"approved":          prxv1.TaskDisplayState_TASK_DISPLAY_STATE_APPROVED,
		"review_waiting":    prxv1.TaskDisplayState_TASK_DISPLAY_STATE_REVIEW_WAITING,
		"open":              prxv1.TaskDisplayState_TASK_DISPLAY_STATE_OPEN,
		"unknown":           prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNSPECIFIED
}

func protoPullRequestState(value string) prxv1.PullRequestState {
	switch value {
	case "open":
		return prxv1.PullRequestState_PULL_REQUEST_STATE_OPEN
	case "closed":
		return prxv1.PullRequestState_PULL_REQUEST_STATE_CLOSED
	case "merged":
		return prxv1.PullRequestState_PULL_REQUEST_STATE_MERGED
	case "unknown":
		return prxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN
	default:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_UNSPECIFIED
	}
}

func protoReviewState(value string) prxv1.ReviewState {
	switch value {
	case "none":
		return prxv1.ReviewState_REVIEW_STATE_NONE
	case "required":
		return prxv1.ReviewState_REVIEW_STATE_REQUIRED
	case "approved":
		return prxv1.ReviewState_REVIEW_STATE_APPROVED
	case "changes_requested":
		return prxv1.ReviewState_REVIEW_STATE_CHANGES_REQUESTED
	case "unknown":
		return prxv1.ReviewState_REVIEW_STATE_UNKNOWN
	default:
		return prxv1.ReviewState_REVIEW_STATE_UNSPECIFIED
	}
}

func protoMergeability(value string) prxv1.Mergeability {
	switch value {
	case "mergeable":
		return prxv1.Mergeability_MERGEABILITY_MERGEABLE
	case "conflicting":
		return prxv1.Mergeability_MERGEABILITY_CONFLICTING
	case "unknown":
		return prxv1.Mergeability_MERGEABILITY_UNKNOWN
	default:
		return prxv1.Mergeability_MERGEABILITY_UNSPECIFIED
	}
}

func protoPullRequestDisplayState(value string) prxv1.PullRequestDisplayState {
	states := map[string]prxv1.PullRequestDisplayState{
		"merged":            prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_MERGED,
		"closed":            prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CLOSED,
		"draft":             prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_DRAFT,
		"conflict":          prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CONFLICT,
		"changes_requested": prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CHANGES_REQUESTED,
		"approved":          prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_APPROVED,
		"review_waiting":    prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_REVIEW_WAITING,
		"open":              prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_OPEN,
		"unknown":           prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNSPECIFIED
}

func protoDocumentKind(value string) prxv1.DocumentKind {
	switch value {
	case "url":
		return prxv1.DocumentKind_DOCUMENT_KIND_URL
	case "markdown_path":
		return prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH
	default:
		return prxv1.DocumentKind_DOCUMENT_KIND_UNSPECIFIED
	}
}

func domainDocumentKind(value prxv1.DocumentKind) string {
	switch value {
	case prxv1.DocumentKind_DOCUMENT_KIND_URL:
		return "url"
	case prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH:
		return "markdown_path"
	default:
		return ""
	}
}

func protoBlockedReason(task domain.Task) *prxv1.BlockedReason {
	code := prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED
	switch task.BlockedCode {
	case domain.BlockedDependencyDataIncomplete:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_DEPENDENCY_DATA_INCOMPLETE
	case domain.BlockedByStaleData:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_BLOCKER_STALE
	case domain.BlockedWaitingForBlocker:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER
	}
	if code == prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED {
		return nil
	}
	return &prxv1.BlockedReason{Code: code, BlockerTaskId: task.BlockerTaskID}
}

func protoDomainErrorCode(value string) prxv1.DomainErrorCode {
	codes := map[string]prxv1.DomainErrorCode{
		"internal":                 prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INTERNAL,
		"cross_feature_dependency": prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CROSS_FEATURE_DEPENDENCY,
		"cycle":                    prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE,
		"duplicate_dependency":     prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_DEPENDENCY,
		"duplicate_pull_request":   prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_PULL_REQUEST,
		"github_auth":              prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_GITHUB_AUTH,
		"invalid_database":         prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DATABASE,
		"invalid_document":         prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT,
		"invalid_document_kind":    prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND,
		"invalid_kind":             prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_KIND,
		"invalid_parent":           prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT,
		"invalid_pull_request_url": prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PULL_REQUEST_URL,
		"invalid_seed":             prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SEED,
		"invalid_slug":             prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SLUG,
		"invalid_status":           prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS,
		"invalid_title":            prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TITLE,
		"not_found":                prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND,
		"references_exist":         prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST,
	}
	if code, ok := codes[value]; ok {
		return code
	}
	return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
