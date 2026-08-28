package rpc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestRPCSharesDomainValidation(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rpc.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	provider, _ := githubprovider.NewFixtureProvider("demo")
	service := app.New(database, provider)
	path, handler := rpc.New(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
	feature, err := client.CreateFeature(ctx, connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "rpc-feature", Title: "RPC feature"}))
	if err != nil {
		t.Fatal(err)
	}
	if feature.Msg.Feature.Status != prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE {
		t.Fatalf("feature status=%s", feature.Msg.Feature.Status)
	}
	a, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "A", Kind: prxv1.TaskKind_TASK_KIND_PULL_REQUEST}))
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "B", Kind: prxv1.TaskKind_TASK_KIND_PULL_REQUEST}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.AddDependency(ctx, connect.NewRequest(&prxv1.AddDependencyRequest{BlockerTaskId: a.Msg.Task.Id, BlockedTaskId: b.Msg.Task.Id})); err != nil {
		t.Fatal(err)
	}
	_, err = client.AddDependency(ctx, connect.NewRequest(&prxv1.AddDependencyRequest{BlockerTaskId: b.Msg.Task.Id, BlockedTaskId: a.Msg.Task.Id}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("cycle RPC code=%s err=%v", connect.CodeOf(err), err)
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("cycle RPC error type=%T", err)
	}
	foundCycleDetail := false
	for _, detail := range connectErr.Details() {
		value, detailErr := detail.Value()
		if detailErr != nil {
			t.Fatal(detailErr)
		}
		errorDetail, ok := value.(*prxv1.ErrorDetail)
		if !ok {
			continue
		}
		foundCycleDetail = errorDetail.Code == prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE && len(errorDetail.Path) >= 3
	}
	if !foundCycleDetail {
		t.Fatalf("cycle RPC error missing structured detail: %v", connectErr.Details())
	}
	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.Snapshot.Tasks) != 2 || len(snapshot.Msg.Snapshot.Dependencies) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot.Msg.Snapshot)
	}
	if snapshot.Msg.Snapshot.Tasks[1].Kind != prxv1.TaskKind_TASK_KIND_PULL_REQUEST || snapshot.Msg.Snapshot.Tasks[1].BlockedReason == nil {
		t.Fatalf("unexpected structured task state: %+v", snapshot.Msg.Snapshot.Tasks[1])
	}
	if snapshot.Msg.Snapshot.Tasks[1].BlockedReason.Code != prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER {
		t.Fatalf("blocked reason=%+v", snapshot.Msg.Snapshot.Tasks[1].BlockedReason)
	}
}

func TestRPCRejectsUnknownEnumValues(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(ctx, connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "enum-feature", Title: "Enum feature"}))
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "A", Kind: prxv1.TaskKind_TASK_KIND_MANUAL}))
	if err != nil {
		t.Fatal(err)
	}

	unknownKind := prxv1.TaskKind(999)
	if _, err = client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "B", Kind: unknownKind})); errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_KIND {
		t.Fatalf("unknown task kind err=%v", err)
	}

	unknownTaskStatus := prxv1.TaskStatus(999)
	if _, err = client.UpdateTask(ctx, connect.NewRequest(&prxv1.UpdateTaskRequest{Id: task.Msg.Task.Id, Status: &unknownTaskStatus})); errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS {
		t.Fatalf("unknown task status err=%v", err)
	}

	unknownFeatureStatus := prxv1.FeatureStatus(999)
	if _, err = client.UpdateFeature(ctx, connect.NewRequest(&prxv1.UpdateFeatureRequest{Id: feature.Msg.Feature.Id, Status: &unknownFeatureStatus})); errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS {
		t.Fatalf("unknown feature status err=%v", err)
	}

	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.Snapshot.Tasks) != 1 {
		t.Fatalf("rejected requests changed state: %+v", snapshot.Msg.Snapshot.Tasks)
	}
	if snapshot.Msg.Snapshot.Tasks[0].Status != prxv1.TaskStatus_TASK_STATUS_PLANNED {
		t.Fatalf("task status=%s", snapshot.Msg.Snapshot.Tasks[0].Status)
	}
}

func newTestClient(t *testing.T) prxv1connect.PRXServiceClient {
	t.Helper()
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rpc.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, _ := githubprovider.NewFixtureProvider("demo")
	path, handler := rpc.New(app.New(database, provider))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
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
			return errorDetail.Code
		}
	}
	return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
}
