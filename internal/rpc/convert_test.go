package rpc

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestProtoFeatureStatusMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.FeatureStatus
		want  prxv1.FeatureStatus
	}{
		{"active", domain.FeatureStatusActive, prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE},
		{"paused", domain.FeatureStatusPaused, prxv1.FeatureStatus_FEATURE_STATUS_PAUSED},
		{"completed", domain.FeatureStatusCompleted, prxv1.FeatureStatus_FEATURE_STATUS_COMPLETED},
		{"cancelled", domain.FeatureStatusCancelled, prxv1.FeatureStatus_FEATURE_STATUS_CANCELLED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoFeatureStatus(test.value); got != test.want {
				t.Fatalf("protoFeatureStatus(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoTaskKindMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.TaskKind
		want  prxv1.TaskKind
	}{
		{"pull request", domain.TaskKindPR, prxv1.TaskKind_TASK_KIND_PULL_REQUEST},
		{"manual", domain.TaskKindManual, prxv1.TaskKind_TASK_KIND_MANUAL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoTaskKind(test.value); got != test.want {
				t.Fatalf("protoTaskKind(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoTaskStatusMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.TaskStatus
		want  prxv1.TaskStatus
	}{
		{"planned", domain.TaskStatusPlanned, prxv1.TaskStatus_TASK_STATUS_PLANNED},
		{"in progress", domain.TaskStatusInProgress, prxv1.TaskStatus_TASK_STATUS_IN_PROGRESS},
		{"completed", domain.TaskStatusCompleted, prxv1.TaskStatus_TASK_STATUS_COMPLETED},
		{"cancelled", domain.TaskStatusCancelled, prxv1.TaskStatus_TASK_STATUS_CANCELLED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoTaskStatus(test.value); got != test.want {
				t.Fatalf("protoTaskStatus(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoTaskDisplayStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.TaskDisplayState
		want  prxv1.TaskDisplayState
	}{
		{"planned", domain.TaskDisplayStatePlanned, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_PLANNED},
		{"in progress", domain.TaskDisplayStateInProgress, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_IN_PROGRESS},
		{"completed", domain.TaskDisplayStateCompleted, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_COMPLETED},
		{"cancelled", domain.TaskDisplayStateCancelled, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CANCELLED},
		{"unlinked", domain.TaskDisplayStateUnlinked, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNLINKED},
		{"merged", domain.TaskDisplayStateMerged, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_MERGED},
		{"closed", domain.TaskDisplayStateClosed, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CLOSED},
		{"draft", domain.TaskDisplayStateDraft, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_DRAFT},
		{"conflict", domain.TaskDisplayStateConflict, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CONFLICT},
		{"changes requested", domain.TaskDisplayStateChangesRequested, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_CHANGES_REQUESTED},
		{"approved", domain.TaskDisplayStateApproved, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_APPROVED},
		{"review waiting", domain.TaskDisplayStateReviewWaiting, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_REVIEW_WAITING},
		{"open", domain.TaskDisplayStateOpen, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_OPEN},
		{"unknown", domain.TaskDisplayStateUnknown, prxv1.TaskDisplayState_TASK_DISPLAY_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoTaskDisplayState(test.value); got != test.want {
				t.Fatalf("protoTaskDisplayState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoPullRequestStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.PullRequestState
		want  prxv1.PullRequestState
	}{
		{"open", domain.PullRequestStateOpen, prxv1.PullRequestState_PULL_REQUEST_STATE_OPEN},
		{"closed", domain.PullRequestStateClosed, prxv1.PullRequestState_PULL_REQUEST_STATE_CLOSED},
		{"merged", domain.PullRequestStateMerged, prxv1.PullRequestState_PULL_REQUEST_STATE_MERGED},
		{"unknown", domain.PullRequestStateUnknown, prxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoPullRequestState(test.value); got != test.want {
				t.Fatalf("protoPullRequestState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoReviewStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.ReviewState
		want  prxv1.ReviewState
	}{
		{"none", domain.ReviewStateNone, prxv1.ReviewState_REVIEW_STATE_NONE},
		{"required", domain.ReviewStateRequired, prxv1.ReviewState_REVIEW_STATE_REQUIRED},
		{"approved", domain.ReviewStateApproved, prxv1.ReviewState_REVIEW_STATE_APPROVED},
		{"changes requested", domain.ReviewStateChangesRequested, prxv1.ReviewState_REVIEW_STATE_CHANGES_REQUESTED},
		{"unknown", domain.ReviewStateUnknown, prxv1.ReviewState_REVIEW_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoReviewState(test.value); got != test.want {
				t.Fatalf("protoReviewState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoMergeabilityMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.Mergeability
		want  prxv1.Mergeability
	}{
		{"mergeable", domain.MergeabilityMergeable, prxv1.Mergeability_MERGEABILITY_MERGEABLE},
		{"conflicting", domain.MergeabilityConflicting, prxv1.Mergeability_MERGEABILITY_CONFLICTING},
		{"unknown", domain.MergeabilityUnknown, prxv1.Mergeability_MERGEABILITY_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoMergeability(test.value); got != test.want {
				t.Fatalf("protoMergeability(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoPullRequestDisplayStateMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.PullRequestDisplayState
		want  prxv1.PullRequestDisplayState
	}{
		{"merged", domain.PullRequestDisplayStateMerged, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_MERGED},
		{"closed", domain.PullRequestDisplayStateClosed, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CLOSED},
		{"draft", domain.PullRequestDisplayStateDraft, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_DRAFT},
		{"conflict", domain.PullRequestDisplayStateConflict, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CONFLICT},
		{"changes requested", domain.PullRequestDisplayStateChangesRequested, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_CHANGES_REQUESTED},
		{"approved", domain.PullRequestDisplayStateApproved, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_APPROVED},
		{"review waiting", domain.PullRequestDisplayStateReviewWaiting, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_REVIEW_WAITING},
		{"open", domain.PullRequestDisplayStateOpen, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_OPEN},
		{"unknown", domain.PullRequestDisplayStateUnknown, prxv1.PullRequestDisplayState_PULL_REQUEST_DISPLAY_STATE_UNKNOWN},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoPullRequestDisplayState(test.value); got != test.want {
				t.Fatalf("protoPullRequestDisplayState(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoDocumentKindMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.DocumentKind
		want  prxv1.DocumentKind
	}{
		{"URL", domain.DocumentKindURL, prxv1.DocumentKind_DOCUMENT_KIND_URL},
		{"Markdown path", domain.DocumentKindMarkdownPath, prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := protoDocumentKind(test.value); got != test.want {
				t.Fatalf("protoDocumentKind(%q)=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestProtoBlockedReasonMapsEveryKnownValue(t *testing.T) {
	tests := []struct {
		name  string
		value domain.BlockedReasonCode
		want  prxv1.BlockedReasonCode
	}{
		{"dependency data incomplete", domain.BlockedReasonCodeDependencyDataIncomplete, prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_DEPENDENCY_DATA_INCOMPLETE},
		{"blocker stale", domain.BlockedReasonCodeBlockerStale, prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_BLOCKER_STALE},
		{"waiting for blocker", domain.BlockedReasonCodeWaitingForBlocker, prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := protoBlockedReason(domain.Task{BlockedCode: test.value, BlockerTaskID: "blocker"})
			if got == nil || got.GetCode() != test.want || got.GetBlockerTaskId() != "blocker" {
				t.Fatalf("protoBlockedReason(%q)=%+v, want code %s", test.value, got, test.want)
			}
		})
	}
}

func TestRPCErrorDetailsMapEveryKnownDomainErrorCode(t *testing.T) {
	tests := []struct {
		name  string
		value domain.DomainErrorCode
		want  prxv1.DomainErrorCode
	}{
		{"cross feature dependency", domain.DomainErrorCodeCrossFeatureDependency, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CROSS_FEATURE_DEPENDENCY},
		{"cycle", domain.DomainErrorCodeCycle, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE},
		{"duplicate dependency", domain.DomainErrorCodeDuplicateDependency, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_DEPENDENCY},
		{"duplicate pull request", domain.DomainErrorCodeDuplicatePullRequest, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DUPLICATE_PULL_REQUEST},
		{"GitHub auth", domain.DomainErrorCodeGitHubAuth, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_GITHUB_AUTH},
		{"invalid database", domain.DomainErrorCodeInvalidDatabase, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DATABASE},
		{"invalid document", domain.DomainErrorCodeInvalidDocument, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT},
		{"invalid document kind", domain.DomainErrorCodeInvalidDocumentKind, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND},
		{"invalid kind", domain.DomainErrorCodeInvalidKind, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_KIND},
		{"invalid parent", domain.DomainErrorCodeInvalidParent, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PARENT},
		{"invalid pull request URL", domain.DomainErrorCodeInvalidPullRequestURL, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_PULL_REQUEST_URL},
		{"invalid seed", domain.DomainErrorCodeInvalidSeed, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SEED},
		{"invalid slug", domain.DomainErrorCodeInvalidSlug, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_SLUG},
		{"invalid status", domain.DomainErrorCodeInvalidStatus, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS},
		{"invalid title", domain.DomainErrorCodeInvalidTitle, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_TITLE},
		{"not found", domain.DomainErrorCodeNotFound, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND},
		{"references exist", domain.DomainErrorCodeReferencesExist, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST},
		{"pull request on manual task", domain.DomainErrorCodePullRequestOnManualTask, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PULL_REQUEST_ON_MANUAL_TASK},
		{"PR task completes on merge", domain.DomainErrorCodePRTaskCompletesOnMerge, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PR_TASK_COMPLETES_ON_MERGE},
		{"invalid document URL", domain.DomainErrorCodeInvalidDocumentURL, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_URL},
		{"document read failed", domain.DomainErrorCodeDocumentReadFailed, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_READ_FAILED},
		{"document too large", domain.DomainErrorCodeDocumentTooLarge, prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_TOO_LARGE},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rpcError(domain.NewError(test.value, "domain error"))
			if got := errorDetailCode(t, err); got != test.want {
				t.Fatalf("rpcError(%q) detail code=%s, want %s", test.value, got, test.want)
			}
		})
	}
}

func errorDetailCode(t *testing.T, err error) prxv1.DomainErrorCode {
	t.Helper()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error type=%T value=%v", err, err)
	}
	for _, detail := range connectErr.Details() {
		value, detailErr := detail.Value()
		if detailErr != nil {
			t.Fatal(detailErr)
		}
		if errorDetail, ok := value.(*prxv1.ErrorDetail); ok {
			return errorDetail.GetCode()
		}
	}
	return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
}
