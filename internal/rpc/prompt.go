package rpc

import (
	"context"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/prompt"
)

func (h *Handler) GetPromptTemplates(
	_ context.Context,
	_ *connect.Request[prxv1.GetPromptTemplatesRequest],
) (*connect.Response[prxv1.GetPromptTemplatesResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.GetPromptTemplatesResponse{
		Templates: protoPromptTemplates(settings.Prompts),
	}), nil
}

func (h *Handler) UpdatePromptTemplates(
	_ context.Context,
	req *connect.Request[prxv1.UpdatePromptTemplatesRequest],
) (*connect.Response[prxv1.UpdatePromptTemplatesResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Update(func(settings *config.Config) error {
		return settings.SetPrompts(prompt.Templates{
			Design:         req.Msg.GetDesign(),
			Implementation: req.Msg.GetImplementation(),
		})
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.UpdatePromptTemplatesResponse{
		Templates: protoPromptTemplates(settings.Prompts),
	}), nil
}

// GetTaskPrompt renders the prompt from the current server state rather than
// from what the caller believes the task looks like. A WebUI snapshot may
// predate a plan being registered or deleted, and the copied prompt has to
// match the task as it is now.
func (h *Handler) GetTaskPrompt(
	ctx context.Context,
	req *connect.Request[prxv1.GetTaskPromptRequest],
) (*connect.Response[prxv1.GetTaskPromptResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	if err != nil {
		return nil, configRPCError(err)
	}
	snapshot, err := h.service.Snapshot(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	task, ok := findTask(snapshot, req.Msg.GetTaskId())
	if !ok {
		return nil, rpcError(
			domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", req.Msg.GetTaskId()),
		)
	}
	kind, body, err := prompt.Render(task, settings.Prompts)
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.GetTaskPromptResponse{
		TaskId: task.ID, Kind: protoTaskPromptKind(kind), Prompt: body,
	}), nil
}

func findTask(snapshot domain.Snapshot, id string) (domain.Task, bool) {
	for _, task := range snapshot.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return domain.Task{}, false
}

func protoPromptTemplates(value prompt.Templates) *prxv1.PromptTemplates {
	return &prxv1.PromptTemplates{Design: value.Design, Implementation: value.Implementation}
}

func protoTaskPromptKind(value prompt.Kind) prxv1.TaskPromptKind {
	if value == prompt.KindImplementation {
		return prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION
	}
	return prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN
}
