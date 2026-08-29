package rpc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/rpc"
	"github.com/HappyOnigiri/PRX/internal/store"
)

func TestRPCReadsOnlyRegisteredMarkdownDocuments(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "markdown-preview", Title: "Markdown preview"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "Documented task",
				Kind:      prxv1.TaskKind_TASK_KIND_MANUAL,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.md")
	const content = "# Plan\n\nShip safely.\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: task.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH,
				Title:  "Plan",
				Value:  path,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := client.ReadMarkdownDocument(
		ctx,
		connect.NewRequest(&prxv1.ReadMarkdownDocumentRequest{Id: document.Msg.GetDocument().GetId()}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Msg.GetContent() != content {
		t.Fatalf("content=%q", preview.Msg.GetContent())
	}

	urlDocument, err := client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: task.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_URL,
				Title:  "Runbook",
				Value:  "https://example.com/runbook",
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ReadMarkdownDocument(
		ctx,
		connect.NewRequest(&prxv1.ReadMarkdownDocumentRequest{Id: urlDocument.Msg.GetDocument().GetId()}),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_KIND {
		t.Fatalf("URL preview code=%s err=%v", got, err)
	}

	missingDocument, err := client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: task.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH,
				Title:  "Missing",
				Value:  filepath.Join(t.TempDir(), "missing.md"),
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ReadMarkdownDocument(
		ctx,
		connect.NewRequest(&prxv1.ReadMarkdownDocumentRequest{Id: missingDocument.Msg.GetDocument().GetId()}),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_READ_FAILED {
		t.Fatalf("missing preview code=%s err=%v", got, err)
	}

	largePath := filepath.Join(t.TempDir(), "large.md")
	if err := os.WriteFile(largePath, make([]byte, (1<<20)+1), 0o600); err != nil {
		t.Fatal(err)
	}
	largeDocument, err := client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: task.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_MARKDOWN_PATH,
				Title:  "Large",
				Value:  largePath,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ReadMarkdownDocument(
		ctx,
		connect.NewRequest(&prxv1.ReadMarkdownDocumentRequest{Id: largeDocument.Msg.GetDocument().GetId()}),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_DOCUMENT_TOO_LARGE {
		t.Fatalf("large preview code=%s err=%v", got, err)
	}
}

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
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "rpc-feature", Title: "RPC feature"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if feature.Msg.GetFeature().GetStatus() != prxv1.FeatureStatus_FEATURE_STATUS_ACTIVE {
		t.Fatalf("feature status=%s", feature.Msg.GetFeature().GetStatus())
	}
	a, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "A",
				Kind:      prxv1.TaskKind_TASK_KIND_PULL_REQUEST,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "B",
				Kind:      prxv1.TaskKind_TASK_KIND_PULL_REQUEST,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.AddDependency(
		ctx,
		connect.NewRequest(
			&prxv1.AddDependencyRequest{BlockerTaskId: a.Msg.GetTask().GetId(), BlockedTaskId: b.Msg.GetTask().GetId()},
		),
	); err != nil {
		t.Fatal(err)
	}
	_, err = client.AddDependency(
		ctx,
		connect.NewRequest(
			&prxv1.AddDependencyRequest{BlockerTaskId: b.Msg.GetTask().GetId(), BlockedTaskId: a.Msg.GetTask().GetId()},
		),
	)
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
		foundCycleDetail = errorDetail.GetCode() == prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_CYCLE &&
			len(errorDetail.GetPath()) >= 3
	}
	if !foundCycleDetail {
		t.Fatalf("cycle RPC error missing structured detail: %v", connectErr.Details())
	}
	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.GetSnapshot().GetTasks()) != 2 || len(snapshot.Msg.GetSnapshot().GetDependencies()) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot.Msg.GetSnapshot())
	}
	if snapshot.Msg.GetSnapshot().GetTasks()[1].GetKind() != prxv1.TaskKind_TASK_KIND_PULL_REQUEST ||
		snapshot.Msg.GetSnapshot().GetTasks()[1].GetBlockedReason() == nil {
		t.Fatalf("unexpected structured task state: %+v", snapshot.Msg.GetSnapshot().GetTasks()[1])
	}
	if snapshot.Msg.GetSnapshot().GetTasks()[1].GetBlockedReason().GetCode() !=
		prxv1.BlockedReasonCode_BLOCKED_REASON_CODE_WAITING_FOR_BLOCKER {
		t.Fatalf("blocked reason=%+v", snapshot.Msg.GetSnapshot().GetTasks()[1].GetBlockedReason())
	}
}

func TestRPCLifecyclePersistsAndDeletesResources(t *testing.T) {
	ctx := context.Background()
	const mergedURL = "https://github.com/acme/api/pull/42"
	const failedURL = "https://github.com/acme/api/pull/43"
	fixturePath := filepath.Join(t.TempDir(), "github.json")
	fixture := `{"` + mergedURL + `":{"state":"merged","review_state":"approved",` +
		`"mergeability":"mergeable","author":"octocat","assignees":["alice"]},` +
		`"` + failedURL + `":{"error":"GitHub unavailable"}}`
	if err := os.WriteFile(fixturePath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	client := newTestClientWithFixture(t, fixturePath)

	featureResponse, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "rpc-lifecycle", Title: "RPC lifecycle"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	featureID := featureResponse.Msg.GetFeature().GetId()
	createTask := func(title string, kind prxv1.TaskKind) *prxv1.Task {
		response, err := client.CreateTask(
			ctx,
			connect.NewRequest(&prxv1.CreateTaskRequest{FeatureId: featureID, Title: title, Kind: kind}),
		)
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetTask()
	}
	blocker := createTask("Merged blocker", prxv1.TaskKind_TASK_KIND_PULL_REQUEST)
	blocked := createTask("Blocked task", prxv1.TaskKind_TASK_KIND_MANUAL)
	failedTask := createTask("Failed sync", prxv1.TaskKind_TASK_KIND_PULL_REQUEST)
	deletedTask := createTask("Deleted task", prxv1.TaskKind_TASK_KIND_MANUAL)

	if _, err := client.AddDependency(
		ctx,
		connect.NewRequest(&prxv1.AddDependencyRequest{
			BlockerTaskId: blocker.GetId(),
			BlockedTaskId: blocked.GetId(),
		}),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RemoveDependency(
		ctx,
		connect.NewRequest(&prxv1.RemoveDependencyRequest{
			BlockerTaskId: blocker.GetId(),
			BlockedTaskId: blocked.GetId(),
		}),
	); err != nil {
		t.Fatal(err)
	}
	getSnapshot := func() *prxv1.Snapshot {
		response, err := client.GetSnapshot(ctx, connect.NewRequest(&prxv1.GetSnapshotRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		return response.Msg.GetSnapshot()
	}
	if snapshot := getSnapshot(); len(snapshot.GetDependencies()) != 0 {
		t.Fatalf("dependencies after removal=%d, want 0", len(snapshot.GetDependencies()))
	}
	if _, err := client.AddDependency(
		ctx,
		connect.NewRequest(&prxv1.AddDependencyRequest{
			BlockerTaskId: blocker.GetId(),
			BlockedTaskId: blocked.GetId(),
		}),
	); err != nil {
		t.Fatal(err)
	}

	attached, err := client.AttachPullRequest(
		ctx,
		connect.NewRequest(&prxv1.AttachPullRequestRequest{TaskId: blocker.GetId(), Url: mergedURL}),
	)
	if err != nil {
		t.Fatal(err)
	}
	attachedPR := attached.Msg.GetPullRequest()
	if attachedPR.GetOwner() != "acme" || attachedPR.GetRepository() != "api" || attachedPR.GetNumber() != 42 ||
		attachedPR.GetUrl() != mergedURL || attachedPR.GetState() != prxv1.PullRequestState_PULL_REQUEST_STATE_UNKNOWN ||
		!attachedPR.GetStale() || attachedPR.GetGithubUpdatedAt() != "" || attachedPR.GetLastSyncedAt() != "" {
		t.Fatalf("attached pull request=%+v", attachedPR)
	}
	if _, err := client.AttachPullRequest(
		ctx,
		connect.NewRequest(&prxv1.AttachPullRequestRequest{TaskId: failedTask.GetId(), Url: failedURL}),
	); err != nil {
		t.Fatal(err)
	}

	syncResponse, err := client.Sync(
		ctx,
		connect.NewRequest(&prxv1.SyncRequest{FeatureId: featureID}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if syncResponse.Msg.GetSucceeded() != 1 || syncResponse.Msg.GetFailed() != 1 {
		t.Fatalf("sync response=%+v, want one success and one failure", syncResponse.Msg)
	}
	snapshot := getSnapshot()
	if len(snapshot.GetPullRequests()) != 2 {
		t.Fatalf("pull requests=%d, want 2", len(snapshot.GetPullRequests()))
	}
	var mergedPR, failedPR *prxv1.PullRequest
	for _, pullRequest := range snapshot.GetPullRequests() {
		switch pullRequest.GetTaskId() {
		case blocker.GetId():
			mergedPR = pullRequest
		case failedTask.GetId():
			failedPR = pullRequest
		}
	}
	if mergedPR == nil || mergedPR.GetState() != prxv1.PullRequestState_PULL_REQUEST_STATE_MERGED ||
		mergedPR.GetNodeId() != "fixture:42" || mergedPR.GetAuthor() != "octocat" ||
		len(mergedPR.GetAssignees()) != 1 || mergedPR.GetAssignees()[0] != "alice" ||
		mergedPR.GetReviewState() != prxv1.ReviewState_REVIEW_STATE_APPROVED ||
		mergedPR.GetMergeability() != prxv1.Mergeability_MERGEABILITY_MERGEABLE ||
		mergedPR.GetGithubUpdatedAt() == "" || mergedPR.GetLastSyncedAt() == "" ||
		mergedPR.GetSyncError() != "" || mergedPR.GetStale() {
		t.Fatalf("synced pull request=%+v", mergedPR)
	}
	if failedPR == nil || failedPR.GetSyncError() != "GitHub unavailable" || !failedPR.GetStale() ||
		failedPR.GetLastSyncedAt() == "" {
		t.Fatalf("failed pull request=%+v", failedPR)
	}

	deletedDocument, err := client.AddDocument(
		ctx,
		connect.NewRequest(&prxv1.AddDocumentRequest{
			FeatureId: featureID,
			Kind:      prxv1.DocumentKind_DOCUMENT_KIND_URL,
			Title:     "Deleted document",
			Value:     "https://example.com/deleted",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.AddDocument(
		ctx,
		connect.NewRequest(&prxv1.AddDocumentRequest{
			TaskId: failedTask.GetId(),
			Kind:   prxv1.DocumentKind_DOCUMENT_KIND_URL,
			Title:  "Cascade document",
			Value:  "https://example.com/cascade",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.DeleteDocument(
		ctx,
		connect.NewRequest(&prxv1.DeleteDocumentRequest{Id: deletedDocument.Msg.GetDocument().GetId()}),
	); err != nil {
		t.Fatal(err)
	}
	snapshot = getSnapshot()
	if len(snapshot.GetDocuments()) != 1 || snapshot.GetDocuments()[0].GetTitle() != "Cascade document" {
		t.Fatalf("documents after deletion=%+v", snapshot.GetDocuments())
	}

	if _, err := client.DetachPullRequest(
		ctx,
		connect.NewRequest(&prxv1.DetachPullRequestRequest{TaskId: blocker.GetId()}),
	); err != nil {
		t.Fatal(err)
	}
	snapshot = getSnapshot()
	if len(snapshot.GetPullRequests()) != 1 || snapshot.GetPullRequests()[0].GetTaskId() != failedTask.GetId() {
		t.Fatalf("pull requests after detach=%+v", snapshot.GetPullRequests())
	}

	if _, err := client.DeleteTask(
		ctx,
		connect.NewRequest(&prxv1.DeleteTaskRequest{Id: deletedTask.GetId()}),
	); err != nil {
		t.Fatal(err)
	}
	snapshot = getSnapshot()
	for _, task := range snapshot.GetTasks() {
		if task.GetId() == deletedTask.GetId() {
			t.Fatalf("deleted task remains in snapshot: %+v", task)
		}
	}

	_, err = client.DeleteFeature(
		ctx,
		connect.NewRequest(&prxv1.DeleteFeatureRequest{Id: featureID}),
	)
	if err == nil || connect.CodeOf(err) != connect.CodeFailedPrecondition ||
		errorDetailCode(t, err) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_REFERENCES_EXIST {
		t.Fatalf("non-cascade feature deletion code=%s err=%v", connect.CodeOf(err), err)
	}
	snapshot = getSnapshot()
	if len(snapshot.GetFeatures()) != 1 || snapshot.GetFeatures()[0].GetId() != featureID ||
		len(snapshot.GetTasks()) != 3 || len(snapshot.GetPullRequests()) != 1 || len(snapshot.GetDocuments()) != 1 ||
		len(snapshot.GetDependencies()) != 1 {
		t.Fatalf("state after rejected feature deletion=%+v", snapshot)
	}

	validation, err := client.Validate(ctx, connect.NewRequest(&prxv1.ValidateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !validation.Msg.GetValid() || len(validation.Msg.GetErrors()) != 0 {
		t.Fatalf("validation=%+v, want a valid database", validation.Msg)
	}

	if _, err := client.DeleteFeature(
		ctx,
		connect.NewRequest(&prxv1.DeleteFeatureRequest{Id: featureID, Cascade: true}),
	); err != nil {
		t.Fatal(err)
	}
	snapshot = getSnapshot()
	if len(snapshot.GetFeatures()) != 0 || len(snapshot.GetTasks()) != 0 || len(snapshot.GetDependencies()) != 0 ||
		len(snapshot.GetPullRequests()) != 0 || len(snapshot.GetDocuments()) != 0 {
		t.Fatalf("state after cascade deletion=%+v", snapshot)
	}
}

func TestRPCRejectsUnknownEnumValues(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "enum-feature", Title: "Enum feature"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "A",
				Kind:      prxv1.TaskKind_TASK_KIND_MANUAL,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	unknownKind := prxv1.TaskKind(999)
	if _, err = client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{FeatureId: feature.Msg.GetFeature().GetId(), Title: "B", Kind: unknownKind},
		),
	); errorDetailCode(
		t,
		err,
	) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_KIND {
		t.Fatalf("unknown task kind err=%v", err)
	}

	unknownTaskStatus := prxv1.TaskStatus(999)
	if _, err = client.UpdateTask(
		ctx,
		connect.NewRequest(&prxv1.UpdateTaskRequest{Id: task.Msg.GetTask().GetId(), Status: &unknownTaskStatus}),
	); errorDetailCode(
		t,
		err,
	) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS {
		t.Fatalf("unknown task status err=%v", err)
	}

	unknownFeatureStatus := prxv1.FeatureStatus(999)
	if _, err = client.UpdateFeature(
		ctx,
		connect.NewRequest(
			&prxv1.UpdateFeatureRequest{Id: feature.Msg.GetFeature().GetId(), Status: &unknownFeatureStatus},
		),
	); errorDetailCode(
		t,
		err,
	) != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_STATUS {
		t.Fatalf("unknown feature status err=%v", err)
	}

	snapshot, err := client.GetSnapshot(ctx, connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Msg.GetSnapshot().GetTasks()) != 1 {
		t.Fatalf("rejected requests changed state: %+v", snapshot.Msg.GetSnapshot().GetTasks())
	}
	if snapshot.Msg.GetSnapshot().GetTasks()[0].GetStatus() != prxv1.TaskStatus_TASK_STATUS_PLANNED {
		t.Fatalf("task status=%s", snapshot.Msg.GetSnapshot().GetTasks()[0].GetStatus())
	}
}

func TestRPCReportsDistinctErrorCodesPerCause(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{Slug: "cause-feature", Title: "Cause feature"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	manual, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "Sign off",
				Kind:      prxv1.TaskKind_TASK_KIND_MANUAL,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	prTask, err := client.CreateTask(
		ctx,
		connect.NewRequest(
			&prxv1.CreateTaskRequest{
				FeatureId: feature.Msg.GetFeature().GetId(),
				Title:     "Ship API",
				Kind:      prxv1.TaskKind_TASK_KIND_PULL_REQUEST,
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.AttachPullRequest(
		ctx,
		connect.NewRequest(
			&prxv1.AttachPullRequestRequest{
				TaskId: manual.Msg.GetTask().GetId(),
				Url:    "https://github.com/org/repo/pull/42",
			},
		),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PULL_REQUEST_ON_MANUAL_TASK {
		t.Fatalf("pull request on manual task code=%s err=%v", got, err)
	}

	completed := prxv1.TaskStatus_TASK_STATUS_COMPLETED
	_, err = client.UpdateTask(
		ctx,
		connect.NewRequest(&prxv1.UpdateTaskRequest{Id: prTask.Msg.GetTask().GetId(), Status: &completed}),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_PR_TASK_COMPLETES_ON_MERGE {
		t.Fatalf("PR task completion code=%s err=%v", got, err)
	}

	_, err = client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: manual.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_URL,
				Title:  "Spec",
				Value:  "ftp://example.com/spec",
			},
		),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT_URL {
		t.Fatalf("document URL scheme code=%s err=%v", got, err)
	}

	_, err = client.AddDocument(
		ctx,
		connect.NewRequest(
			&prxv1.AddDocumentRequest{
				TaskId: manual.Msg.GetTask().GetId(),
				Kind:   prxv1.DocumentKind_DOCUMENT_KIND_URL,
				Title:  "Spec",
				Value:  "  ",
			},
		),
	)
	if got := errorDetailCode(t, err); got != prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_INVALID_DOCUMENT {
		t.Fatalf("missing document value code=%s err=%v", got, err)
	}
}

func TestRPCMapsInternalDomainErrorToInternal(t *testing.T) {
	path, handler := rpc.New(internalErrorService{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := prxv1connect.NewPRXServiceClient(server.Client(), server.URL)

	_, err := client.GetSnapshot(context.Background(), connect.NewRequest(&prxv1.GetSnapshotRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("internal RPC code=%s err=%v", connect.CodeOf(err), err)
	}
}

type internalErrorService struct {
	rpc.Service
}

func (internalErrorService) Snapshot(context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{}, domain.NewError(domain.DomainErrorCodeInternal, "internal error")
}

func newTestClient(t *testing.T) prxv1connect.PRXServiceClient {
	t.Helper()
	return newTestClientWithFixture(t, "demo")
}

func newTestClientWithFixture(t *testing.T, fixturePath string) prxv1connect.PRXServiceClient {
	t.Helper()
	ctx := context.Background()
	database, err := store.Open(ctx, filepath.Join(t.TempDir(), "rpc.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	provider, err := githubprovider.NewFixtureProvider(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
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
			return errorDetail.GetCode()
		}
	}
	return prxv1.DomainErrorCode_DOMAIN_ERROR_CODE_UNSPECIFIED
}
