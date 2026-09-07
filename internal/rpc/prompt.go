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
		Templates:             protoPromptTemplates(settings.Prompts),
		SupportedPlaceholders: prompt.SupportedPlaceholders(),
		RequiredPlaceholder:   prompt.RequiredPlaceholder(),
		BuiltIn:               protoPromptTemplates(prompt.DefaultTemplates()),

		BatchSupportedPlaceholders: prompt.BatchSupportedPlaceholders(),
		BatchRequiredPlaceholder:   prompt.BatchRequiredPlaceholder(),
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
			Batch:          req.Msg.GetBatch(),
		})
	})
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.UpdatePromptTemplatesResponse{
		Templates: protoPromptTemplates(settings.Prompts),
	}), nil
}

// GetTaskPrompt は呼び出し元が思っているタスクの姿ではなく、現在のサーバー状態から
// プロンプトを生成する。WebUI のスナップショットは計画の登録や削除より前の
// 時点のものかもしれないため。
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

// GetBatchPrompt は複数タスクをまとめた 1 つのプロンプトを生成する。各タスクは
// 現在のサーバー状態から解決し、指定された feature に属するか検査する。
// docs/design/agent-prompts.md を参照。
func (h *Handler) GetBatchPrompt(
	ctx context.Context,
	req *connect.Request[prxv1.GetBatchPromptRequest],
) (*connect.Response[prxv1.GetBatchPromptResponse], error) {
	store, err := h.requireConfig()
	if err != nil {
		return nil, err
	}
	settings, err := store.Load()
	if err != nil {
		return nil, configRPCError(err)
	}
	featureID := req.Msg.GetFeatureId()
	taskIDs := req.Msg.GetTaskIds()
	// 選択の失敗はどちらも親が不正として報告する。リクエストはこの batch に
	// 含むタスクを指定するもので、1 つも指定していないか、feature が持たない
	// タスクを指定したかのいずれか。
	if len(taskIDs) == 0 {
		return nil, rpcError(domain.NewError(
			domain.DomainErrorCodeInvalidParent,
			"a batch prompt needs at least one task of feature %q", featureID,
		))
	}
	snapshot, err := h.service.Snapshot(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	tasks := make([]domain.Task, 0, len(taskIDs))
	for _, id := range taskIDs {
		task, ok := findTask(snapshot, id)
		if !ok {
			return nil, rpcError(
				domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", id),
			)
		}
		if task.FeatureID != featureID {
			return nil, rpcError(domain.NewError(
				domain.DomainErrorCodeInvalidParent,
				"task %q does not belong to feature %q", id, featureID,
			))
		}
		tasks = append(tasks, task)
	}
	if err := requireBlockersInBatch(tasks); err != nil {
		return nil, err
	}
	body, err := prompt.RenderBatch(featureID, tasks, settings.Prompts)
	if err != nil {
		return nil, configRPCError(err)
	}
	return connect.NewResponse(&prxv1.GetBatchPromptResponse{
		FeatureId: featureID, TaskIds: taskIDs, Prompt: body,
	}), nil
}

// requireBlockersInBatch は、待ち相手のブロッカーを含まないままブロックされた
// タスクを要求する batch を拒否する。
// docs/design/agent-prompts.md を参照。
func requireBlockersInBatch(tasks []domain.Task) error {
	included := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		included[task.ID] = struct{}{}
	}
	for _, task := range tasks {
		for _, blockerID := range task.PendingBlockerTaskIDs {
			if _, ok := included[blockerID]; ok {
				continue
			}
			return rpcError(domain.NewError(
				domain.DomainErrorCodeInvalidParent,
				"task %q waits for task %q, which the batch does not include", task.ID, blockerID,
			))
		}
	}
	return nil
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
	return &prxv1.PromptTemplates{
		Design: value.Design, Implementation: value.Implementation, Batch: value.Batch,
	}
}

func protoTaskPromptKind(value prompt.Kind) prxv1.TaskPromptKind {
	if value == prompt.KindImplementation {
		return prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION
	}
	return prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN
}
