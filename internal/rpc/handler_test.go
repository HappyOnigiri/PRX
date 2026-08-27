package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	prmapv1 "github.com/HappyOnigiri/PRX/gen/prmap/v1"
	"github.com/HappyOnigiri/PRX/gen/prmap/v1/prmapv1connect"
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
	client := prmapv1connect.NewPRMapServiceClient(server.Client(), server.URL)
	feature, err := client.CreateFeature(ctx, connect.NewRequest(&prmapv1.CreateFeatureRequest{Slug: "rpc-feature", Title: "RPC feature"}))
	if err != nil {
		t.Fatal(err)
	}
	a, err := client.CreateTask(ctx, connect.NewRequest(&prmapv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "A", Kind: "pr"}))
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.CreateTask(ctx, connect.NewRequest(&prmapv1.CreateTaskRequest{FeatureId: feature.Msg.Feature.Id, Title: "B", Kind: "pr"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.AddDependency(ctx, connect.NewRequest(&prmapv1.AddDependencyRequest{BlockerTaskId: a.Msg.Task.Id, BlockedTaskId: b.Msg.Task.Id})); err != nil {
		t.Fatal(err)
	}
	_, err = client.AddDependency(ctx, connect.NewRequest(&prmapv1.AddDependencyRequest{BlockerTaskId: b.Msg.Task.Id, BlockedTaskId: a.Msg.Task.Id}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("cycle RPC code=%s err=%v", connect.CodeOf(err), err)
	}
	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&prmapv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.Snapshot.Tasks) != 2 || len(snapshot.Msg.Snapshot.Dependencies) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot.Msg.Snapshot)
	}
}
