package rpc

import (
	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func protoFeature(v domain.Feature) *prxv1.Feature {
	return &prxv1.Feature{
		Id:                 v.ID,
		Slug:               v.Slug,
		Title:              v.Title,
		Description:        v.Description,
		Status:             protoFeatureStatus(v.Status),
		Archived:           v.Archived,
		CreatedAt:          v.CreatedAt.Format(timeFormat),
		UpdatedAt:          v.UpdatedAt.Format(timeFormat),
		TaskCount:          int32(v.TaskCount),
		ReadyCount:         int32(v.ReadyCount),
		ReviewWaitingCount: int32(v.ReviewWaitingCount),
		ConflictCount:      int32(v.ConflictCount),
		MergedCount:        int32(v.MergedCount),
	}
}

func protoTask(v domain.Task) *prxv1.Task {
	return &prxv1.Task{
		Id:            v.ID,
		FeatureId:     v.FeatureID,
		Title:         v.Title,
		Scope:         v.Scope,
		Kind:          protoTaskKind(v.Kind),
		Status:        protoTaskStatus(v.Status),
		Assignee:      v.Assignee,
		CreatedAt:     v.CreatedAt.Format(timeFormat),
		UpdatedAt:     v.UpdatedAt.Format(timeFormat),
		Ready:         v.Ready,
		DisplayState:  protoTaskDisplayState(v.DisplayState),
		BlockedReason: protoBlockedReason(v),
	}
}

func protoDependency(v domain.Dependency) *prxv1.Dependency {
	return &prxv1.Dependency{
		BlockerTaskId: v.BlockerTaskID,
		BlockedTaskId: v.BlockedTaskID,
		CreatedAt:     v.CreatedAt.Format(timeFormat),
	}
}

func protoPullRequest(v domain.PullRequest) *prxv1.PullRequest {
	result := &prxv1.PullRequest{
		TaskId:       v.TaskID,
		Owner:        v.Owner,
		Repository:   v.Repository,
		Number:       v.Number,
		Url:          v.URL,
		NodeId:       v.NodeID,
		Author:       v.Author,
		Assignees:    v.Assignees,
		State:        protoPullRequestState(v.State),
		Draft:        v.Draft,
		ReviewState:  protoReviewState(v.ReviewState),
		Mergeability: protoMergeability(v.Mergeability),
		SyncError:    v.SyncError,
		Stale:        v.Stale,
		DisplayState: protoPullRequestDisplayState(v.DisplayState),
	}
	if v.GitHubUpdatedAt != nil {
		result.GithubUpdatedAt = v.GitHubUpdatedAt.Format(timeFormat)
	}
	if v.LastSyncedAt != nil {
		result.LastSyncedAt = v.LastSyncedAt.Format(timeFormat)
	}
	return result
}

func protoDocument(v domain.Document) *prxv1.Document {
	return &prxv1.Document{
		Id:        v.ID,
		FeatureId: v.FeatureID,
		TaskId:    v.TaskID,
		Kind:      protoDocumentKind(v.Kind),
		Title:     v.Title,
		Value:     v.Value,
		CreatedAt: v.CreatedAt.Format(timeFormat),
	}
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

func protoFeatureStatus(value domain.FeatureStatus) prxv1.FeatureStatus {
	switch value {
	case domain.FeatureStatusActive:
		return prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE
	case domain.FeatureStatusPaused:
		return prxv1.FeatureStatus_FEATURE_STATUS_PAUSED
	case domain.FeatureStatusCompleted:
		return prxv1.FeatureStatus_FEATURE_STATUS_COMPLETED
	case domain.FeatureStatusCancelled:
		return prxv1.FeatureStatus_FEATURE_STATUS_CANCELLED
	default:
		return prxv1.FeatureStatus_FEATURE_STATUS_UNSPECIFIED
	}
}

// domainFeatureStatus rejects values the server cannot map instead of falling
// back to the empty string, which the service layer reads as "field omitted".
func domainFeatureStatus(value *prxv1.FeatureStatus) (*domain.FeatureStatus, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.FeatureStatus
	switch *value {
	case prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE:
		result = domain.FeatureStatusActive
	case prxv1.FeatureStatus_FEATURE_STATUS_PAUSED:
		result = domain.FeatureStatusPaused
	case prxv1.FeatureStatus_FEATURE_STATUS_COMPLETED:
		result = domain.FeatureStatusCompleted
	case prxv1.FeatureStatus_FEATURE_STATUS_CANCELLED:
		result = domain.FeatureStatusCancelled
	default:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid feature status")
	}
	return &result, nil
}

func protoTaskKind(value domain.TaskKind) prxv1.TaskKind {
	switch value {
	case domain.TaskKindPR:
		return prxv1.TaskKind_TASK_KIND_PULL_REQUEST
	case domain.TaskKindManual:
		return prxv1.TaskKind_TASK_KIND_MANUAL
	default:
		return prxv1.TaskKind_TASK_KIND_UNSPECIFIED
	}
}

// domainTaskKind maps the unspecified value to the empty string so the service
// layer can apply its default, and rejects everything else it cannot map.
func domainTaskKind(value prxv1.TaskKind) (domain.TaskKind, error) {
	switch value {
	case prxv1.TaskKind_TASK_KIND_UNSPECIFIED:
		return "", nil
	case prxv1.TaskKind_TASK_KIND_PULL_REQUEST:
		return domain.TaskKindPR, nil
	case prxv1.TaskKind_TASK_KIND_MANUAL:
		return domain.TaskKindManual, nil
	default:
		return "", domain.NewError(domain.DomainErrorCodeInvalidKind, "task kind must be pr or manual")
	}
}

func protoTaskStatus(value domain.TaskStatus) prxv1.TaskStatus {
	switch value {
	case domain.TaskStatusPlanned:
		return prxv1.TaskStatus_TASK_STATUS_PLANNED
	case domain.TaskStatusInProgress:
		return prxv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.TaskStatusCompleted:
		return prxv1.TaskStatus_TASK_STATUS_COMPLETED
	case domain.TaskStatusCancelled:
		return prxv1.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return prxv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// domainTaskStatus rejects values the server cannot map instead of falling back
// to the empty string, which the service layer reads as "field omitted".
func domainTaskStatus(value *prxv1.TaskStatus) (*domain.TaskStatus, error) {
	if value == nil {
		return nil, nil
	}
	var result domain.TaskStatus
	switch *value {
	case prxv1.TaskStatus_TASK_STATUS_PLANNED:
		result = domain.TaskStatusPlanned
	case prxv1.TaskStatus_TASK_STATUS_IN_PROGRESS:
		result = domain.TaskStatusInProgress
	case prxv1.TaskStatus_TASK_STATUS_COMPLETED:
		result = domain.TaskStatusCompleted
	case prxv1.TaskStatus_TASK_STATUS_CANCELLED:
		result = domain.TaskStatusCancelled
	default:
		return nil, domain.NewError(domain.DomainErrorCodeInvalidStatus, "invalid task status")
	}
	return &result, nil
}

func protoTaskDisplayState(value domain.TaskDisplayState) prxv1.TaskDisplayState {
	states := map[domain.TaskDisplayState]prxv1.TaskDisplayState{
		domain.TaskDisplayStatePlanned:          prxv1.TaskDisplayState_TASK_DISPLAY_STATE_PLANNED,
		domain.TaskDisplayStateInProgress:       prxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_PROGRESS,
		domain.TaskDisplayStateCompleted:        prxv1.TaskDisplayState_TASK_DISPLAY_STATE_COMPLETED,
		domain.TaskDisplayStateCancelled:        prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CANCELLED,
		domain.TaskDisplayStateUnlinked:         prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNLINKED,
		domain.TaskDisplayStateMerged:           prxv1.TaskDisplayState_TASK_DISPLAY_STATE_MERGED,
		domain.TaskDisplayStateClosed:           prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CLOSED,
		domain.TaskDisplayStateDraft:            prxv1.TaskDisplayState_TASK_DISPLAY_STATE_DRAFT,
		domain.TaskDisplayStateConflict:         prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CONFLICT,
		domain.TaskDisplayStateChangesRequested: prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CHANGES_REQUESTED,
		domain.TaskDisplayStateApproved:         prxv1.TaskDisplayState_TASK_DISPLAY_STATE_APPROVED,
		domain.TaskDisplayStateReviewWaiting:    prxv1.TaskDisplayState_TASK_DISPLAY_STATE_REVIEW_WAITING,
		domain.TaskDisplayStateOpen:             prxv1.TaskDisplayState_TASK_DISPLAY_STATE_OPEN,
		domain.TaskDisplayStateUnknown:          prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNSPECIFIED
}

func protoPullRequestState(value domain.PullRequestState) prxv1.PullRequestState {
	switch value {
	case domain.PullRequestStateOpen:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_OPEN
	case domain.PullRequestStateClosed:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_CLOSED
	case domain.PullRequestStateMerged:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_MERGED
	case domain.PullRequestStateUnknown:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN
	default:
		return prxv1.PullRequestState_PULL_REQUEST_STATE_UNSPECIFIED
	}
}

func protoReviewState(value domain.ReviewState) prxv1.ReviewState {
	switch value {
	case domain.ReviewStateNone:
		return prxv1.ReviewState_REVIEW_STATE_NONE
	case domain.ReviewStateRequired:
		return prxv1.ReviewState_REVIEW_STATE_REQUIRED
	case domain.ReviewStateApproved:
		return prxv1.ReviewState_REVIEW_STATE_APPROVED
	case domain.ReviewStateChangesRequested:
		return prxv1.ReviewState_REVIEW_STATE_CHANGES_REQUESTED
	case domain.ReviewStateUnknown:
		return prxv1.ReviewState_REVIEW_STATE_UNKNOWN
	default:
		return prxv1.ReviewState_REVIEW_STATE_UNSPECIFIED
	}
}

func protoMergeability(value domain.Mergeability) prxv1.Mergeability {
	switch value {
	case domain.MergeabilityMergeable:
		return prxv1.Mergeability_MERGEABILITY_MERGEABLE
	case domain.MergeabilityConflicting:
		return prxv1.Mergeability_MERGEABILITY_CONFLICTING
	case domain.MergeabilityUnknown:
		return prxv1.Mergeability_MERGEABILITY_UNKNOWN
	default:
		return prxv1.Mergeability_MERGEABILITY_UNSPECIFIED
	}
}

func protoPullRequestDisplayState(value domain.PullRequestDisplayState) prxv1.PullRequestDisplayState {
	const (
		changesRequestedState = prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CHANGES_REQUESTED
		reviewWaitingState    = prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_REVIEW_WAITING
	)
	states := map[domain.PullRequestDisplayState]prxv1.PullRequestDisplayState{
		domain.PullRequestDisplayStateMerged:           prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_MERGED,
		domain.PullRequestDisplayStateClosed:           prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CLOSED,
		domain.PullRequestDisplayStateDraft:            prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_DRAFT,
		domain.PullRequestDisplayStateConflict:         prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CONFLICT,
		domain.PullRequestDisplayStateChangesRequested: changesRequestedState,
		domain.PullRequestDisplayStateApproved:         prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_APPROVED,
		domain.PullRequestDisplayStateReviewWaiting:    reviewWaitingState,
		domain.PullRequestDisplayStateOpen:             prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_OPEN,
		domain.PullRequestDisplayStateUnknown:          prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNKNOWN,
	}
	if state, ok := states[value]; ok {
		return state
	}
	return prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNSPECIFIED
}

func protoDocumentKind(value domain.DocumentKind) prxv1.DocumentKind {
	switch value {
	case domain.DocumentKindURL:
		return prxv1.DocumentKind_DOCUMENT_KIND_URL
	case domain.DocumentKindMarkdownPath:
		return prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH
	default:
		return prxv1.DocumentKind_DOCUMENT_KIND_UNSPECIFIED
	}
}

func domainDocumentKind(value prxv1.DocumentKind) domain.DocumentKind {
	switch value {
	case prxv1.DocumentKind_DOCUMENT_KIND_URL:
		return domain.DocumentKindURL
	case prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH:
		return domain.DocumentKindMarkdownPath
	default:
		return ""
	}
}

func protoBlockedReason(task domain.Task) *prxv1.BlockedReason {
	code := prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED
	switch task.BlockedCode {
	case domain.BlockedReasonCodeDependencyDataIncomplete:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_DEPENDENCY_DATA_INCOMPLETE
	case domain.BlockedReasonCodeBlockerStale:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_BLOCKER_STALE
	case domain.BlockedReasonCodeWaitingForBlocker:
		code = prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER
	}
	if code == prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_UNSPECIFIED {
		return nil
	}
	return &prxv1.BlockedReason{Code: code, BlockerTaskId: task.BlockerTaskID}
}

func protoDomainErrorCode(value domain.DomainErrorCode) prxv1.DomainErrorCode {
	switch value {
	case domain.DomainErrorCodeCrossFeatureDependency:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CROSS_FEATURE_DEPENDENCY
	case domain.DomainErrorCodeCycle:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE
	case domain.DomainErrorCodeDuplicateDependency:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_DEPENDENCY
	case domain.DomainErrorCodeDuplicatePullRequest:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_PULL_REQUEST
	case domain.DomainErrorCodeDocumentReadFailed:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_READ_FAILED
	case domain.DomainErrorCodeDocumentTooLarge:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_TOO_LARGE
	case domain.DomainErrorCodeGitHubAuth:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_GITHUB_AUTH
	case domain.DomainErrorCodeInvalidDatabase:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DATABASE
	case domain.DomainErrorCodeInvalidDocument:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT
	case domain.DomainErrorCodeInvalidDocumentKind:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND
	case domain.DomainErrorCodeInvalidDocumentURL:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_URL
	case domain.DomainErrorCodeInvalidKind:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_KIND
	case domain.DomainErrorCodeInvalidParent:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT
	case domain.DomainErrorCodeInvalidPullRequestURL:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PULL_REQUEST_URL
	case domain.DomainErrorCodeInvalidSeed:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SEED
	case domain.DomainErrorCodeInvalidSlug:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SLUG
	case domain.DomainErrorCodeInvalidStatus:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS
	case domain.DomainErrorCodeInvalidTitle:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TITLE
	case domain.DomainErrorCodeNotFound:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND
	case domain.DomainErrorCodePRTaskCompletesOnMerge:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PR_TASK_COMPLETES_ON_MERGE
	case domain.DomainErrorCodePullRequestOnManualTask:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PULL_REQUEST_ON_MANUAL_TASK
	case domain.DomainErrorCodeReferencesExist:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST
	default:
		return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
	}
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
