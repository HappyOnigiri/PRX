package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/prompt"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func newPromptClient(t *testing.T) prxv1connect.PRXServiceClient {
	t.Helper()
	root := t.TempDir()
	database, err := store.Open(context.Background(), filepath.Join(root, "prompt.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	configStore, err := config.NewStore(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return newConfigClient(t, app.NewWithConfig(database, nil, configStore))
}

func createPromptTask(t *testing.T, client prxv1connect.PRXServiceClient) string {
	t.Helper()
	ctx := context.Background()
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "prompts", Title: "Prompts"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{
		FeatureId: feature.Msg.GetFeature().GetId(),
		Title:     "Add the checkout API",
		Scope:     "Server only",
		Kind:      prxv1.TaskKind_TASK_KIND_MANUAL,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return task.Msg.GetTask().GetId()
}

func TestRPCPromptTemplatesRoundTripAndDriveTheTaskPrompt(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)
	taskID := createPromptTask(t, client)

	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Msg.GetTemplates().GetDesign() != prompt.DefaultTemplates().Design {
		t.Fatalf("design template=%q, want the built-in template", stored.Msg.GetTemplates().GetDesign())
	}
	// The vocabulary travels with the templates so an editor never has to keep
	// its own copy of what this server accepts.
	if !slices.Equal(stored.Msg.GetSupportedPlaceholders(), prompt.SupportedPlaceholders()) {
		t.Fatalf("supported placeholders=%v, want %v",
			stored.Msg.GetSupportedPlaceholders(), prompt.SupportedPlaceholders())
	}
	if stored.Msg.GetRequiredPlaceholder() != prompt.RequiredPlaceholder() {
		t.Fatalf("required placeholder=%q, want %q",
			stored.Msg.GetRequiredPlaceholder(), prompt.RequiredPlaceholder())
	}

	updated, err := client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{
		Design:         "Design {{task_id}}: {{task_title}}\n",
		Implementation: "Implement {{task_id}} in {{feature_id}}\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.GetTemplates().GetImplementation() != "Implement {{task_id}} in {{feature_id}}\n" {
		t.Fatalf("implementation template=%q", updated.Msg.GetTemplates().GetImplementation())
	}

	design, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil {
		t.Fatal(err)
	}
	if design.Msg.GetKind() != prxv1.TaskPromptKind_TASK_PROMPT_KIND_DESIGN ||
		design.Msg.GetPrompt() != "Design "+taskID+": Add the checkout API\n" ||
		design.Msg.GetTaskId() != taskID {
		t.Fatalf("design prompt=%+v", design.Msg)
	}

	// Registering a plan changes which template the same request renders, with
	// no other input from the caller.
	if _, err := client.AddDocument(ctx, connect.NewRequest(&prxv1.AddDocumentRequest{
		TaskId:               taskID,
		Title:                "Plan",
		Source:               &prxv1.AddDocumentRequest_Markdown{Markdown: "# Plan\n"},
		IsImplementationPlan: true,
	})); err != nil {
		t.Fatal(err)
	}
	implementation, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: taskID}))
	if err != nil {
		t.Fatal(err)
	}
	if implementation.Msg.GetKind() != prxv1.TaskPromptKind_TASK_PROMPT_KIND_IMPLEMENTATION ||
		!strings.HasPrefix(implementation.Msg.GetPrompt(), "Implement "+taskID+" in F-") {
		t.Fatalf("implementation prompt=%+v", implementation.Msg)
	}
}

func TestRPCPromptFailuresUseTheConfigurationVocabulary(t *testing.T) {
	ctx := context.Background()
	client := newPromptClient(t)

	_, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: "T-404"}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_NOT_FOUND {
		t.Fatalf("missing task error=%v", err)
	}

	_, err = client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{
		Design:         "no target",
		Implementation: "Implement {{task_id}}",
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_CONFIG {
		t.Fatalf("invalid template error=%v", err)
	}
	// The rejected write left the stored pair alone.
	stored, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
	if err != nil || stored.Msg.GetTemplates().GetDesign() != prompt.DefaultTemplates().Design {
		t.Fatalf("stored templates=%+v err=%v", stored.Msg.GetTemplates(), err)
	}
}

func TestRPCPromptMethodsAreUnavailableWithoutConfigStore(t *testing.T) {
	path, handler := rpc.New(app.New(noopRepository{}, nil))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"GetPromptTemplates": func() error {
			_, err := client.GetPromptTemplates(ctx, connect.NewRequest(&prxv1.GetPromptTemplatesRequest{}))
			return err
		},
		"UpdatePromptTemplates": func() error {
			_, err := client.UpdatePromptTemplates(ctx, connect.NewRequest(&prxv1.UpdatePromptTemplatesRequest{}))
			return err
		},
		"GetTaskPrompt": func() error {
			_, err := client.GetTaskPrompt(ctx, connect.NewRequest(&prxv1.GetTaskPromptRequest{TaskId: "T-1"}))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); connect.CodeOf(err) != connect.CodeUnimplemented {
				t.Fatalf("code=%s err=%v", connect.CodeOf(err), err)
			}
		})
	}
}
