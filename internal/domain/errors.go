package domain

import (
	"errors"
	"fmt"
)

type DomainErrorCode string

const (
	DomainErrorCodeCrossFeatureDependency     DomainErrorCode = "cross_feature_dependency"
	DomainErrorCodeCycle                      DomainErrorCode = "cycle"
	DomainErrorCodeDuplicateDependency        DomainErrorCode = "duplicate_dependency"
	DomainErrorCodeDuplicatePullRequest       DomainErrorCode = "duplicate_pull_request"
	DomainErrorCodeGitHubAuth                 DomainErrorCode = "github_auth"
	DomainErrorCodeInvalidDatabase            DomainErrorCode = "invalid_database"
	DomainErrorCodeInvalidDocument            DomainErrorCode = "invalid_document"
	DomainErrorCodeInvalidDocumentKind        DomainErrorCode = "invalid_document_kind"
	DomainErrorCodeInvalidKind                DomainErrorCode = "invalid_kind"
	DomainErrorCodeInvalidParent              DomainErrorCode = "invalid_parent"
	DomainErrorCodeInvalidPullRequestURL      DomainErrorCode = "invalid_pull_request_url"
	DomainErrorCodeInvalidSlug                DomainErrorCode = "invalid_slug"
	DomainErrorCodeInvalidStatus              DomainErrorCode = "invalid_status"
	DomainErrorCodeInvalidTitle               DomainErrorCode = "invalid_title"
	DomainErrorCodeNotFound                   DomainErrorCode = "not_found"
	DomainErrorCodeReferencesExist            DomainErrorCode = "references_exist"
	DomainErrorCodePullRequestOnManualTask    DomainErrorCode = "pull_request_on_manual_task"
	DomainErrorCodeInvalidDocumentURL         DomainErrorCode = "invalid_document_url"
	DomainErrorCodeDocumentReadFailed         DomainErrorCode = "document_read_failed"
	DomainErrorCodeDocumentTooLarge           DomainErrorCode = "document_too_large"
	DomainErrorCodeInvalidImplementationPlan  DomainErrorCode = "invalid_implementation_plan"
	DomainErrorCodeImplementationPlanTooLarge DomainErrorCode = "implementation_plan_too_large"
	DomainErrorCodeInvalidConfig              DomainErrorCode = "invalid_config"
	DomainErrorCodeInternal                   DomainErrorCode = "internal"
)

type Error struct {
	Code    DomainErrorCode `json:"code"`
	Message string          `json:"message"`
	Path    []string        `json:"path,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func NewError(code DomainErrorCode, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCode(err error) DomainErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return DomainErrorCodeInternal
}
