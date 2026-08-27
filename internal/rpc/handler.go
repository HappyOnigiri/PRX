package rpc

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	prmapv1 "github.com/HappyOnigiri/PRX/gen/prmap/v1"
	"github.com/HappyOnigiri/PRX/gen/prmap/v1/prmapv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

type Handler struct {
	prmapv1connect.UnimplementedPRMapServiceHandler
	service *app.Service
}

func New(service *app.Service) (string, http.Handler) {
	return prmapv1connect.NewPRMapServiceHandler(&Handler{service: service})
}

func rpcError(err error) error {
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		return connect.NewError(connect.CodeInternal, err)
	}
	code := connect.CodeInvalidArgument
	switch domainErr.Code {
	case "not_found":
		code = connect.CodeNotFound
	case "duplicate_dependency", "duplicate_pull_request", "references_exist", "cycle":
		code = connect.CodeFailedPrecondition
	case "github_auth":
		code = connect.CodeUnauthenticated
	}
	return connect.NewError(code, err)
}

func (h *Handler) GetSnapshot(ctx context.Context, _ *connect.Request[prmapv1.GetSnapshotRequest]) (*connect.Response[prmapv1.GetSnapshotResponse], error) {
	snapshot, err := h.service.Snapshot(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.GetSnapshotResponse{Snapshot: protoSnapshot(snapshot)}), nil
}

func (h *Handler) CreateFeature(ctx context.Context, req *connect.Request[prmapv1.CreateFeatureRequest]) (*connect.Response[prmapv1.CreateFeatureResponse], error) {
	value, err := h.service.CreateFeature(ctx, req.Msg.Slug, req.Msg.Title, req.Msg.Description)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.CreateFeatureResponse{Feature: protoFeature(value)}), nil
}

func (h *Handler) UpdateFeature(ctx context.Context, req *connect.Request[prmapv1.UpdateFeatureRequest]) (*connect.Response[prmapv1.UpdateFeatureResponse], error) {
	value, err := h.service.UpdateFeature(ctx, req.Msg.Id, req.Msg.Slug, req.Msg.Title, req.Msg.Description, req.Msg.Status, req.Msg.Archived)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.UpdateFeatureResponse{Feature: protoFeature(value)}), nil
}

func (h *Handler) DeleteFeature(ctx context.Context, req *connect.Request[prmapv1.DeleteFeatureRequest]) (*connect.Response[prmapv1.DeleteFeatureResponse], error) {
	if err := h.service.DeleteFeature(ctx, req.Msg.Id, req.Msg.Cascade); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.DeleteFeatureResponse{}), nil
}

func (h *Handler) CreateTask(ctx context.Context, req *connect.Request[prmapv1.CreateTaskRequest]) (*connect.Response[prmapv1.CreateTaskResponse], error) {
	value, err := h.service.CreateTask(ctx, req.Msg.FeatureId, req.Msg.Title, req.Msg.Scope, req.Msg.Kind, req.Msg.Assignee)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.CreateTaskResponse{Task: protoTask(value)}), nil
}

func (h *Handler) UpdateTask(ctx context.Context, req *connect.Request[prmapv1.UpdateTaskRequest]) (*connect.Response[prmapv1.UpdateTaskResponse], error) {
	value, err := h.service.UpdateTask(ctx, req.Msg.Id, req.Msg.Title, req.Msg.Scope, req.Msg.Status, req.Msg.Assignee)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.UpdateTaskResponse{Task: protoTask(value)}), nil
}

func (h *Handler) DeleteTask(ctx context.Context, req *connect.Request[prmapv1.DeleteTaskRequest]) (*connect.Response[prmapv1.DeleteTaskResponse], error) {
	if err := h.service.DeleteTask(ctx, req.Msg.Id, req.Msg.Cascade); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.DeleteTaskResponse{}), nil
}

func (h *Handler) AddDependency(ctx context.Context, req *connect.Request[prmapv1.AddDependencyRequest]) (*connect.Response[prmapv1.AddDependencyResponse], error) {
	value, err := h.service.AddDependency(ctx, req.Msg.BlockerTaskId, req.Msg.BlockedTaskId)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.AddDependencyResponse{Dependency: protoDependency(value)}), nil
}

func (h *Handler) RemoveDependency(ctx context.Context, req *connect.Request[prmapv1.RemoveDependencyRequest]) (*connect.Response[prmapv1.RemoveDependencyResponse], error) {
	if err := h.service.RemoveDependency(ctx, req.Msg.BlockerTaskId, req.Msg.BlockedTaskId); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.RemoveDependencyResponse{}), nil
}

func (h *Handler) AttachPullRequest(ctx context.Context, req *connect.Request[prmapv1.AttachPullRequestRequest]) (*connect.Response[prmapv1.AttachPullRequestResponse], error) {
	value, err := h.service.AttachPullRequest(ctx, req.Msg.TaskId, req.Msg.Url)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.AttachPullRequestResponse{PullRequest: protoPullRequest(value)}), nil
}

func (h *Handler) DetachPullRequest(ctx context.Context, req *connect.Request[prmapv1.DetachPullRequestRequest]) (*connect.Response[prmapv1.DetachPullRequestResponse], error) {
	if err := h.service.DetachPullRequest(ctx, req.Msg.TaskId); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.DetachPullRequestResponse{}), nil
}

func (h *Handler) AddDocument(ctx context.Context, req *connect.Request[prmapv1.AddDocumentRequest]) (*connect.Response[prmapv1.AddDocumentResponse], error) {
	value, err := h.service.AddDocument(ctx, req.Msg.FeatureId, req.Msg.TaskId, req.Msg.Kind, req.Msg.Title, req.Msg.Value)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.AddDocumentResponse{Document: protoDocument(value)}), nil
}

func (h *Handler) DeleteDocument(ctx context.Context, req *connect.Request[prmapv1.DeleteDocumentRequest]) (*connect.Response[prmapv1.DeleteDocumentResponse], error) {
	if err := h.service.DeleteDocument(ctx, req.Msg.Id); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.DeleteDocumentResponse{}), nil
}

func (h *Handler) Sync(ctx context.Context, req *connect.Request[prmapv1.SyncRequest]) (*connect.Response[prmapv1.SyncResponse], error) {
	succeeded, failed, err := h.service.Sync(ctx, req.Msg.FeatureId, req.Msg.TaskId)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&prmapv1.SyncResponse{Succeeded: int32(succeeded), Failed: int32(failed)}), nil
}

func (h *Handler) Validate(ctx context.Context, _ *connect.Request[prmapv1.ValidateRequest]) (*connect.Response[prmapv1.ValidateResponse], error) {
	errorsFound := h.service.Validate(ctx)
	return connect.NewResponse(&prmapv1.ValidateResponse{Valid: len(errorsFound) == 0, Errors: errorsFound}), nil
}
