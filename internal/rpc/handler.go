package rpc

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

type Handler struct {
	prxv1connect.UnimplementedPRXServiceHandler
	service     Service
	configStore *config.Store
}

func New(service Service, stores ...*config.Store) (string, http.Handler) {
	var configStore *config.Store
	if len(stores) > 0 {
		configStore = stores[0]
	}
	if configStore == nil {
		if provider, ok := service.(interface{ ConfigStore() *config.Store }); ok {
			configStore = provider.ConfigStore()
		}
	}
	// Requiring the Connect protocol header keeps the RPCs out of reach of
	// simple cross-origin requests, which browsers send without a preflight.
	return prxv1connect.NewPRXServiceHandler(
		&Handler{service: service, configStore: configStore},
		connect.WithRequireConnectProtocolHeader(),
	)
}

func rpcError(err error) error {
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		return connect.NewError(connect.CodeInternal, err)
	}
	code := connect.CodeInvalidArgument
	switch domainErr.Code {
	case domain.DomainErrorCodeCrossFeatureDependency,
		domain.DomainErrorCodeInvalidDatabase,
		domain.DomainErrorCodeInvalidDocument,
		domain.DomainErrorCodeInvalidDocumentKind,
		domain.DomainErrorCodeInvalidKind,
		domain.DomainErrorCodeInvalidParent,
		domain.DomainErrorCodeInvalidPullRequestURL,
		domain.DomainErrorCodeInvalidSeed,
		domain.DomainErrorCodeInvalidSlug,
		domain.DomainErrorCodeInvalidStatus,
		domain.DomainErrorCodeInvalidTitle,
		domain.DomainErrorCodePullRequestOnManualTask,
		domain.DomainErrorCodePRTaskCompletesOnMerge,
		domain.DomainErrorCodeInvalidDocumentURL,
		domain.DomainErrorCodeDocumentReadFailed,
		domain.DomainErrorCodeDocumentTooLarge,
		domain.DomainErrorCodeInvalidConfig:
		code = connect.CodeInvalidArgument
	case domain.DomainErrorCodeNotFound:
		code = connect.CodeNotFound
	case domain.DomainErrorCodeDuplicateDependency,
		domain.DomainErrorCodeDuplicatePullRequest,
		domain.DomainErrorCodeReferencesExist,
		domain.DomainErrorCodeCycle:
		code = connect.CodeFailedPrecondition
	case domain.DomainErrorCodeGitHubAuth:
		code = connect.CodeUnauthenticated
	case domain.DomainErrorCodeInternal:
		code = connect.CodeInternal
	}
	connectErr := connect.NewError(code, err)
	detail, detailErr := connect.NewErrorDetail(
		&prxv1.ErrorDetail{Code: protoDomainErrorCode(domainErr.Code), Path: domainErr.Path},
	)
	if detailErr == nil {
		connectErr.AddDetail(detail)
	}
	return connectErr
}

func (h *Handler) GetSnapshot(
	ctx context.Context,
	_ *connect.Request[prxv1.GetSnapshotRequest],
) (*connect.Response[prxv1.GetSnapshotResponse], error) {
	snapshot, err := h.service.Snapshot(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetSnapshotResponse{Snapshot: protoSnapshot(snapshot)}), nil
}

func (h *Handler) CreateFeature(
	ctx context.Context,
	req *connect.Request[prxv1.CreateFeatureRequest],
) (*connect.Response[prxv1.CreateFeatureResponse], error) {
	value, err := h.service.CreateFeature(ctx, req.Msg.GetSlug(), req.Msg.GetTitle(), req.Msg.GetDescription())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.CreateFeatureResponse{Feature: protoFeature(value)}), nil
}

func (h *Handler) UpdateFeature(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateFeatureRequest],
) (*connect.Response[prxv1.UpdateFeatureResponse], error) {
	statusValue := req.Msg.GetStatus()
	var statusPointer *prxv1.FeatureStatus
	if req.Msg.Status != nil {
		statusPointer = &statusValue
	}
	status, err := domainFeatureStatus(statusPointer)
	if err != nil {
		return nil, rpcError(err)
	}
	slugValue := req.Msg.GetSlug()
	var slugPointer *string
	if req.Msg.Slug != nil {
		slugPointer = &slugValue
	}
	titleValue := req.Msg.GetTitle()
	var titlePointer *string
	if req.Msg.Title != nil {
		titlePointer = &titleValue
	}
	descriptionValue := req.Msg.GetDescription()
	var descriptionPointer *string
	if req.Msg.Description != nil {
		descriptionPointer = &descriptionValue
	}
	archivedValue := req.Msg.GetArchived()
	var archivedPointer *bool
	if req.Msg.Archived != nil {
		archivedPointer = &archivedValue
	}
	value, err := h.service.UpdateFeature(
		ctx,
		req.Msg.GetId(),
		slugPointer,
		titlePointer,
		descriptionPointer,
		status,
		archivedPointer,
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.UpdateFeatureResponse{Feature: protoFeature(value)}), nil
}

func (h *Handler) DeleteFeature(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteFeatureRequest],
) (*connect.Response[prxv1.DeleteFeatureResponse], error) {
	if err := h.service.DeleteFeature(ctx, req.Msg.GetId(), req.Msg.GetCascade()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.DeleteFeatureResponse{}), nil
}

func (h *Handler) CreateTask(
	ctx context.Context,
	req *connect.Request[prxv1.CreateTaskRequest],
) (*connect.Response[prxv1.CreateTaskResponse], error) {
	kind, err := domainTaskKind(req.Msg.GetKind())
	if err != nil {
		return nil, rpcError(err)
	}
	value, err := h.service.CreateTask(
		ctx,
		req.Msg.GetFeatureId(),
		req.Msg.GetTitle(),
		req.Msg.GetScope(),
		kind,
		req.Msg.GetAssignee(),
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.CreateTaskResponse{Task: protoTask(value)}), nil
}

func (h *Handler) UpdateTask(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateTaskRequest],
) (*connect.Response[prxv1.UpdateTaskResponse], error) {
	statusValue := req.Msg.GetStatus()
	var statusPointer *prxv1.TaskStatus
	if req.Msg.Status != nil {
		statusPointer = &statusValue
	}
	status, err := domainTaskStatus(statusPointer)
	if err != nil {
		return nil, rpcError(err)
	}
	titleValue := req.Msg.GetTitle()
	var titlePointer *string
	if req.Msg.Title != nil {
		titlePointer = &titleValue
	}
	scopeValue := req.Msg.GetScope()
	var scopePointer *string
	if req.Msg.Scope != nil {
		scopePointer = &scopeValue
	}
	assigneeValue := req.Msg.GetAssignee()
	var assigneePointer *string
	if req.Msg.Assignee != nil {
		assigneePointer = &assigneeValue
	}
	value, err := h.service.UpdateTask(ctx, req.Msg.GetId(), titlePointer, scopePointer, status, assigneePointer)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.UpdateTaskResponse{Task: protoTask(value)}), nil
}

func (h *Handler) DeleteTask(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteTaskRequest],
) (*connect.Response[prxv1.DeleteTaskResponse], error) {
	if err := h.service.DeleteTask(ctx, req.Msg.GetId(), req.Msg.GetCascade()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.DeleteTaskResponse{}), nil
}

func (h *Handler) AddDependency(
	ctx context.Context,
	req *connect.Request[prxv1.AddDependencyRequest],
) (*connect.Response[prxv1.AddDependencyResponse], error) {
	value, err := h.service.AddDependency(ctx, req.Msg.GetBlockerTaskId(), req.Msg.GetBlockedTaskId())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.AddDependencyResponse{Dependency: protoDependency(value)}), nil
}

func (h *Handler) RemoveDependency(
	ctx context.Context,
	req *connect.Request[prxv1.RemoveDependencyRequest],
) (*connect.Response[prxv1.RemoveDependencyResponse], error) {
	if err := h.service.RemoveDependency(ctx, req.Msg.GetBlockerTaskId(), req.Msg.GetBlockedTaskId()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.RemoveDependencyResponse{}), nil
}

func (h *Handler) AttachPullRequest(
	ctx context.Context,
	req *connect.Request[prxv1.AttachPullRequestRequest],
) (*connect.Response[prxv1.AttachPullRequestResponse], error) {
	value, err := h.service.AttachPullRequest(ctx, req.Msg.GetTaskId(), req.Msg.GetUrl())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.AttachPullRequestResponse{PullRequest: protoPullRequest(value)}), nil
}

func (h *Handler) DetachPullRequest(
	ctx context.Context,
	req *connect.Request[prxv1.DetachPullRequestRequest],
) (*connect.Response[prxv1.DetachPullRequestResponse], error) {
	if err := h.service.DetachPullRequest(ctx, req.Msg.GetTaskId()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.DetachPullRequestResponse{}), nil
}

func (h *Handler) AddDocument(
	ctx context.Context,
	req *connect.Request[prxv1.AddDocumentRequest],
) (*connect.Response[prxv1.AddDocumentResponse], error) {
	value, err := h.service.AddDocument(
		ctx,
		req.Msg.GetFeatureId(),
		req.Msg.GetTaskId(),
		domainDocumentKind(req.Msg.GetKind()),
		req.Msg.GetTitle(),
		req.Msg.GetValue(),
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.AddDocumentResponse{Document: protoDocument(value)}), nil
}

func (h *Handler) DeleteDocument(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteDocumentRequest],
) (*connect.Response[prxv1.DeleteDocumentResponse], error) {
	if err := h.service.DeleteDocument(ctx, req.Msg.GetId()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.DeleteDocumentResponse{}), nil
}

func (h *Handler) ReadMarkdownDocument(
	ctx context.Context,
	req *connect.Request[prxv1.ReadMarkdownDocumentRequest],
) (*connect.Response[prxv1.ReadMarkdownDocumentResponse], error) {
	content, err := h.service.ReadMarkdownDocument(ctx, req.Msg.GetId())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.ReadMarkdownDocumentResponse{Content: content}), nil
}

func (h *Handler) Sync(
	ctx context.Context,
	req *connect.Request[prxv1.SyncRequest],
) (*connect.Response[prxv1.SyncResponse], error) {
	succeeded, failed, err := h.service.Sync(ctx, req.Msg.GetFeatureId(), req.Msg.GetTaskId())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.SyncResponse{Succeeded: int32(succeeded), Failed: int32(failed)}), nil
}

func (h *Handler) Validate(
	ctx context.Context,
	_ *connect.Request[prxv1.ValidateRequest],
) (*connect.Response[prxv1.ValidateResponse], error) {
	errorsFound := h.service.Validate(ctx)
	return connect.NewResponse(&prxv1.ValidateResponse{Valid: len(errorsFound) == 0, Errors: errorsFound}), nil
}
