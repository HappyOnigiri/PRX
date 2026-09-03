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
	"github.com/HappyOnigiri/PRX/internal/filepicker"
)

// LocalFilePicker opens a server-local native chooser without reading the file.
type LocalFilePicker interface {
	SelectFile(ctx context.Context) (path string, canceled bool, err error)
}

type Handler struct {
	prxv1connect.UnimplementedPRXServiceHandler
	service     Service
	configStore *config.Store
	filePicker  LocalFilePicker
	pickerBusy  chan struct{}
}

func New(service Service) (string, http.Handler) {
	return NewWithFilePicker(service, filepicker.New())
}

// NewWithFilePicker constructs the RPC handler with an injectable native chooser.
func NewWithFilePicker(service Service, picker LocalFilePicker) (string, http.Handler) {
	var configStore *config.Store
	if configStore == nil {
		if provider, ok := service.(interface{ ConfigStore() *config.Store }); ok {
			configStore = provider.ConfigStore()
		}
	}
	// Requiring the Connect protocol header keeps the RPCs out of reach of
	// simple cross-origin requests, which browsers send without a preflight.
	return prxv1connect.NewPRXServiceHandler(
		&Handler{
			service: service, configStore: configStore, filePicker: picker,
			pickerBusy: make(chan struct{}, 1),
		},
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
		domain.DomainErrorCodeInvalidSlug,
		domain.DomainErrorCodeInvalidStatus,
		domain.DomainErrorCodeInvalidTitle,
		domain.DomainErrorCodePullRequestOnManualTask,
		domain.DomainErrorCodeInvalidDocumentURL,
		domain.DomainErrorCodeDocumentReadFailed,
		domain.DomainErrorCodeDocumentTooLarge,
		domain.DomainErrorCodeDocumentNotText,
		domain.DomainErrorCodeInvalidImplementationPlan,
		domain.DomainErrorCodeImplementationPlanTooLarge,
		domain.DomainErrorCodeInvalidConfig:
		code = connect.CodeInvalidArgument
	case domain.DomainErrorCodeNotFound:
		code = connect.CodeNotFound
	case domain.DomainErrorCodeDuplicateDependency,
		domain.DomainErrorCodeDuplicatePullRequest,
		domain.DomainErrorCodeReferencesExist,
		domain.DomainErrorCodeDuplicateImplementationPlan,
		domain.DomainErrorCodeArchivedReadOnly,
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

func (h *Handler) CreateProject(
	ctx context.Context,
	req *connect.Request[prxv1.CreateProjectRequest],
) (*connect.Response[prxv1.CreateProjectResponse], error) {
	value, err := h.service.CreateProject(ctx, req.Msg.GetSlug(), req.Msg.GetTitle(), req.Msg.GetDescription())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.CreateProjectResponse{Project: protoProject(value)}), nil
}

func (h *Handler) UpdateProject(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateProjectRequest],
) (*connect.Response[prxv1.UpdateProjectResponse], error) {
	value, err := h.service.UpdateProject(ctx, req.Msg.GetId(), domain.ProjectUpdate{
		Slug:        optionalValue(req.Msg.Slug != nil, req.Msg.GetSlug()),
		Title:       optionalValue(req.Msg.Title != nil, req.Msg.GetTitle()),
		Description: optionalValue(req.Msg.Description != nil, req.Msg.GetDescription()),
		Archived:    optionalValue(req.Msg.Archived != nil, req.Msg.GetArchived()),
	})
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.UpdateProjectResponse{Project: protoProject(value)}), nil
}

func (h *Handler) DeleteProject(
	ctx context.Context,
	req *connect.Request[prxv1.DeleteProjectRequest],
) (*connect.Response[prxv1.DeleteProjectResponse], error) {
	if err := h.service.DeleteProject(ctx, req.Msg.GetId(), req.Msg.GetCascade()); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.DeleteProjectResponse{}), nil
}

func (h *Handler) CreateFeature(
	ctx context.Context,
	req *connect.Request[prxv1.CreateFeatureRequest],
) (*connect.Response[prxv1.CreateFeatureResponse], error) {
	value, err := h.service.CreateFeature(
		ctx,
		req.Msg.GetSlug(),
		req.Msg.GetTitle(),
		req.Msg.GetDescription(),
		req.Msg.GetProjectId(),
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.CreateFeatureResponse{Feature: protoFeature(value)}), nil
}

func (h *Handler) UpdateFeature(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateFeatureRequest],
) (*connect.Response[prxv1.UpdateFeatureResponse], error) {
	status, err := domainFeatureStatus(optionalValue(req.Msg.Status != nil, req.Msg.GetStatus()))
	if err != nil {
		return nil, rpcError(err)
	}
	value, err := h.service.UpdateFeature(ctx, req.Msg.GetId(), domain.FeatureUpdate{
		Slug:        optionalValue(req.Msg.Slug != nil, req.Msg.GetSlug()),
		Title:       optionalValue(req.Msg.Title != nil, req.Msg.GetTitle()),
		Description: optionalValue(req.Msg.Description != nil, req.Msg.GetDescription()),
		Status:      status,
		Archived:    optionalValue(req.Msg.Archived != nil, req.Msg.GetArchived()),
		ProjectID:   optionalValue(req.Msg.ProjectId != nil, req.Msg.GetProjectId()),
	})
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
	source := protoAddDocumentSource(req.Msg)
	value, err := h.service.AddDocument(
		ctx,
		domain.DocumentParent{
			ProjectID: req.Msg.GetProjectId(),
			FeatureID: req.Msg.GetFeatureId(),
			TaskID:    req.Msg.GetTaskId(),
		},
		source.Kind,
		req.Msg.GetTitle(),
		source.Locator,
		source.Content,
		req.Msg.GetIsImplementationPlan(),
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.AddDocumentResponse{Document: protoDocument(value)}), nil
}

func (h *Handler) GetDocument(
	ctx context.Context,
	req *connect.Request[prxv1.GetDocumentRequest],
) (*connect.Response[prxv1.GetDocumentResponse], error) {
	value, err := h.service.GetDocument(ctx, req.Msg.GetId())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetDocumentResponse{Document: protoDocument(value), Content: value.Content}), nil
}

func (h *Handler) UpdateDocument(
	ctx context.Context,
	req *connect.Request[prxv1.UpdateDocumentRequest],
) (*connect.Response[prxv1.UpdateDocumentResponse], error) {
	var source *domain.Document
	if req.Msg.GetSource() != nil {
		value := protoUpdateDocumentSource(req.Msg)
		source = &value
	}
	value, err := h.service.UpdateDocument(
		ctx, req.Msg.GetId(), req.Msg.Title, source, req.Msg.IsImplementationPlan,
	)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.UpdateDocumentResponse{Document: protoDocument(value)}), nil
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

func (h *Handler) ReadDocumentContent(
	ctx context.Context,
	req *connect.Request[prxv1.ReadDocumentContentRequest],
) (*connect.Response[prxv1.ReadDocumentContentResponse], error) {
	content, err := h.service.ReadDocumentContent(ctx, req.Msg.GetId())
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.ReadDocumentContentResponse{Content: content}), nil
}

func (h *Handler) SelectLocalFile(
	ctx context.Context,
	_ *connect.Request[prxv1.SelectLocalFileRequest],
) (*connect.Response[prxv1.SelectLocalFileResponse], error) {
	select {
	case h.pickerBusy <- struct{}{}:
		defer func() { <-h.pickerBusy }()
	default:
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("local file picker is already open"))
	}
	path, canceled, err := h.filePicker.SelectFile(ctx)
	if err != nil {
		var pickerErr *filepicker.Error
		if errors.As(err, &pickerErr) && pickerErr.Kind == filepicker.KindUnavailable {
			return nil, connect.NewError(
				connect.CodeUnavailable,
				errors.New("native file picker is unavailable; enter the path manually"),
			)
		}
		if errors.Is(err, context.Canceled) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, connect.NewError(
			connect.CodeInternal,
			errors.New("native file picker failed; enter the path manually"),
		)
	}
	return connect.NewResponse(&prxv1.SelectLocalFileResponse{Path: path, Canceled: canceled}), nil
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

func (h *Handler) GetGitHubSyncStatus(
	ctx context.Context,
	_ *connect.Request[prxv1.GetGitHubSyncStatusRequest],
) (*connect.Response[prxv1.GetGitHubSyncStatusResponse], error) {
	status, err := h.service.SyncStatus(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetGitHubSyncStatusResponse{Status: protoGitHubSyncStatus(status)}), nil
}

func (h *Handler) SyncGitHubIfDue(
	ctx context.Context,
	_ *connect.Request[prxv1.SyncGitHubIfDueRequest],
) (*connect.Response[prxv1.SyncGitHubIfDueResponse], error) {
	ran, status, err := h.service.SyncIfDue(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.SyncGitHubIfDueResponse{
		Ran: ran, Status: protoGitHubSyncStatus(status),
	}), nil
}

// GetDebugReport returns the diagnostic report together with the text the CLI
// prints. The rendered text crosses the RPC boundary on purpose: the WebUI
// copies it to the clipboard, and a report pasted from the browser has to be the
// same one `prx debug` produces.
func (h *Handler) GetDebugReport(
	ctx context.Context,
	_ *connect.Request[prxv1.GetDebugReportRequest],
) (*connect.Response[prxv1.GetDebugReportResponse], error) {
	report, err := h.service.Debug(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prxv1.GetDebugReportResponse{
		Report: protoDebugReport(report),
		Text:   domain.FormatDebugReport(report),
	}), nil
}

func (h *Handler) Validate(
	ctx context.Context,
	_ *connect.Request[prxv1.ValidateRequest],
) (*connect.Response[prxv1.ValidateResponse], error) {
	errorsFound := h.service.Validate(ctx)
	return connect.NewResponse(&prxv1.ValidateResponse{Valid: len(errorsFound) == 0, Errors: errorsFound}), nil
}
