package rpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestRPCGitHubConfigCRUDNeverReturnsInlineToken(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rpc-config.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	configStore, err := config.NewStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Default()
	if err := settings.AddAuthMethod(config.AuthMethod{
		ID: "work", Host: "github.com", Type: config.AuthMethodTypeInline, Token: "github_pat_rpc_secret",
	}); err != nil {
		t.Fatal(err)
	}
	if err := configStore.Save(settings); err != nil {
		t.Fatal(err)
	}
	client := newConfigClient(t, app.NewWithConfig(database, nil, configStore))

	configuration, err := client.GetConfig(ctx, connect.NewRequest(&prxv1.GetConfigRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.Msg.GetConfig().GetAuthMethods()) != 1 ||
		configuration.Msg.GetConfig().GetAuthMethods()[0].GetSecretHint() != "gith…cret" ||
		configuration.Msg.GetConfig().GetAutoSyncIntervalSeconds() != 3600 {
		t.Fatalf("public config=%+v", configuration.Msg.GetConfig())
	}
	encoded, err := protojson.Marshal(configuration.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || containsSecret(string(encoded), "github_pat_rpc_secret") {
		t.Fatalf("RPC response exposed inline token: %s", encoded)
	}

	host, err := client.AddGitHubHost(ctx, connect.NewRequest(&prxv1.AddGitHubHostRequest{Host: "ghe.example.com"}))
	if err != nil || host.Msg.GetHost().GetApiUrl() != "https://ghe.example.com/api/v3/" {
		t.Fatalf("add host=%+v err=%v", host.Msg.GetHost(), err)
	}
	if host.Msg.GetHost().GetGraphqlUrl() != "https://ghe.example.com/api/graphql" {
		t.Fatalf("GraphQL URL=%q", host.Msg.GetHost().GetGraphqlUrl())
	}
	interval, err := client.UpdateGitHubSyncConfig(
		ctx,
		connect.NewRequest(&prxv1.UpdateGitHubSyncConfigRequest{IntervalSeconds: 600}),
	)
	if err != nil || interval.Msg.GetConfig().GetAutoSyncIntervalSeconds() != 600 {
		t.Fatalf("updated interval=%+v err=%v", interval.Msg.GetConfig(), err)
	}
	statusBefore, err := client.GetGitHubSyncStatus(
		ctx,
		connect.NewRequest(&prxv1.GetGitHubSyncStatusRequest{}),
	)
	if err != nil || statusBefore.Msg.GetStatus().LastUpdatedAt != nil {
		t.Fatalf("initial sync status=%+v err=%v", statusBefore.Msg.GetStatus(), err)
	}
	automatic, err := client.SyncGitHubIfDue(
		ctx,
		connect.NewRequest(&prxv1.SyncGitHubIfDueRequest{}),
	)
	if err != nil || !automatic.Msg.GetRan() || automatic.Msg.GetStatus().LastUpdatedAt == nil {
		t.Fatalf("automatic sync=%+v err=%v", automatic.Msg, err)
	}
	apiURL := "https://ghe.example.com/api/v3/"
	updatedHost, err := client.UpdateGitHubHost(ctx, connect.NewRequest(&prxv1.UpdateGitHubHostRequest{
		Host: "ghe.example.com", ApiUrl: &apiURL,
	}))
	if err != nil || updatedHost.Msg.GetHost().GetApiUrl() != apiURL {
		t.Fatalf("update host=%+v err=%v", updatedHost.Msg.GetHost(), err)
	}

	added, err := client.AddGitHubAuthMethod(ctx, connect.NewRequest(&prxv1.AddGitHubAuthMethodRequest{
		Id: "ghe-inline", Host: "ghe.example.com",
		Type:  prxv1.GithubAuthMethodType_GITHUB_AUTH_METHOD_TYPE_INLINE,
		Token: stringPointer("ghe_rpc_secret"),
	}))
	if err != nil || added.Msg.GetAuthMethod().GetSecretHint() != "ghe_…cret" {
		t.Fatalf("add auth=%+v err=%v", added.Msg.GetAuthMethod(), err)
	}
	newID := "ghe-renamed"
	updated, err := client.UpdateGitHubAuthMethod(ctx, connect.NewRequest(&prxv1.UpdateGitHubAuthMethodRequest{
		Id: "ghe-inline", NewId: &newID,
	}))
	if err != nil || updated.Msg.GetAuthMethod().GetId() != newID ||
		updated.Msg.GetAuthMethod().GetSecretHint() != "ghe_…cret" {
		t.Fatalf("update auth=%+v err=%v", updated.Msg.GetAuthMethod(), err)
	}
	stored, err := configStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if method, ok := stored.AuthMethod(newID); !ok || method.Token != "ghe_rpc_secret" {
		t.Fatalf("omitted token was not preserved: %+v", stored.GitHub.AuthMethods)
	}
	if _, err := client.ReorderGitHubAuthMethods(
		ctx,
		connect.NewRequest(&prxv1.ReorderGitHubAuthMethodsRequest{Ids: []string{newID, "work"}}),
	); err != nil {
		t.Fatal(err)
	}
	validation, err := client.ValidateConfig(ctx, connect.NewRequest(&prxv1.ValidateConfigRequest{}))
	if err != nil || !validation.Msg.GetValid() || len(validation.Msg.GetErrors()) != 0 ||
		len(validation.Msg.GetWarnings()) != 0 {
		t.Fatalf("validation=%+v err=%v", validation.Msg, err)
	}
	if _, err := client.DeleteGitHubAuthMethod(
		ctx,
		connect.NewRequest(&prxv1.DeleteGitHubAuthMethodRequest{Id: newID}),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := client.DeleteGitHubHost(
		ctx,
		connect.NewRequest(&prxv1.DeleteGitHubHostRequest{Host: "ghe.example.com"}),
	); err != nil {
		t.Fatal(err)
	}

	_, err = client.AddGitHubAuthMethod(ctx, connect.NewRequest(&prxv1.AddGitHubAuthMethodRequest{
		Id: "invalid", Host: "github.com", Type: prxv1.GithubAuthMethodType(99),
	}))
	if errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_CONFIG {
		t.Fatalf("invalid auth type error=%v", err)
	}
	jsonValue, err := json.Marshal(configuration.Msg.GetConfig())
	if err != nil || containsSecret(string(jsonValue), "github_pat_rpc_secret") {
		t.Fatalf("JSON config exposed token=%s err=%v", jsonValue, err)
	}
}

func TestRPCConfigMethodsAreUnavailableWithoutConfigStore(t *testing.T) {
	path, handler := rpc.New(app.New(noopRepository{}, nil))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
	_, err := client.GetConfig(context.Background(), connect.NewRequest(&prxv1.GetConfigRequest{}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("config without store code=%s err=%v", connect.CodeOf(err), err)
	}
}

func newConfigClient(t *testing.T, service *app.Service) prxv1connect.PRXServiceClient {
	t.Helper()
	path, handler := rpc.New(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
}

func containsSecret(value, secret string) bool {
	return len(secret) > 0 && strings.Contains(value, secret)
}

func stringPointer(value string) *string { return &value }

type noopRepository struct{}

func (noopRepository) CreateFeature(context.Context, string, string, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("not implemented")
}

func (noopRepository) UpdateFeature(context.Context, domain.Feature) (domain.Feature, error) {
	return domain.Feature{}, errors.New("not implemented")
}

func (noopRepository) GetFeature(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("not implemented")
}

func (noopRepository) GetFeatureBySlug(context.Context, string) (domain.Feature, error) {
	return domain.Feature{}, errors.New("not implemented")
}

func (noopRepository) DeleteFeature(context.Context, string, bool) error {
	return errors.New("not implemented")
}

func (noopRepository) CreateTask(
	context.Context,
	string,
	string,
	string,
	domain.TaskKind,
	string,
) (domain.Task, error) {
	return domain.Task{}, errors.New("not implemented")
}

func (noopRepository) GetTask(context.Context, string) (domain.Task, error) {
	return domain.Task{}, errors.New("not implemented")
}

func (noopRepository) UpdateTask(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, errors.New("not implemented")
}

func (noopRepository) DeleteTask(context.Context, string, bool) error {
	return errors.New("not implemented")
}

func (noopRepository) GetImplementationPlan(context.Context, string) (domain.ImplementationPlan, error) {
	return domain.ImplementationPlan{}, errors.New("not implemented")
}

func (noopRepository) UpsertImplementationPlan(context.Context, string, string) (domain.ImplementationPlan, error) {
	return domain.ImplementationPlan{}, errors.New("not implemented")
}

func (noopRepository) DeleteImplementationPlan(context.Context, string) error {
	return errors.New("not implemented")
}

func (noopRepository) AddDependency(context.Context, string, string) (domain.Dependency, error) {
	return domain.Dependency{}, errors.New("not implemented")
}

func (noopRepository) RemoveDependency(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (noopRepository) UpsertPullRequest(context.Context, domain.PullRequest) (domain.PullRequest, error) {
	return domain.PullRequest{}, errors.New("not implemented")
}

func (noopRepository) DeletePullRequest(context.Context, string) error {
	return errors.New("not implemented")
}

func (noopRepository) CreateDocument(
	context.Context,
	string,
	string,
	domain.DocumentKind,
	string,
	string,
) (domain.Document, error) {
	return domain.Document{}, errors.New("not implemented")
}

func (noopRepository) GetDocument(context.Context, string) (domain.Document, error) {
	return domain.Document{}, errors.New("not implemented")
}

func (noopRepository) DeleteDocument(context.Context, string) error {
	return errors.New("not implemented")
}

func (noopRepository) Snapshot(context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{}, errors.New("not implemented")
}
func (noopRepository) Validate(context.Context) []string { return nil }

// TestRPCValidateConfigReportsUnknownFields keeps the server usable against a
// configuration written by a newer PRX and gives the WebUI the same warnings the
// CLI prints.
func TestRPCValidateConfigReportsUnknownFields(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rpc-warning.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nfuture_setting: 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configStore, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	client := newConfigClient(t, app.NewWithConfig(database, nil, configStore))

	if _, err := client.GetConfig(ctx, connect.NewRequest(&prxv1.GetConfigRequest{})); err != nil {
		t.Fatalf("unknown fields blocked a configuration read: %v", err)
	}
	validation, err := client.ValidateConfig(ctx, connect.NewRequest(&prxv1.ValidateConfigRequest{}))
	if err != nil || !validation.Msg.GetValid() || len(validation.Msg.GetErrors()) != 0 {
		t.Fatalf("validation=%+v err=%v", validation.Msg, err)
	}
	warnings := validation.Msg.GetWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], `"future_setting"`) {
		t.Fatalf("warnings=%q", warnings)
	}
}
