package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	prxv1 "github.com/HappyOnigiri/PRX/gen/prx/v1"
	"github.com/HappyOnigiri/PRX/gen/prx/v1/prxv1connect"
	"github.com/HappyOnigiri/PRX/internal/domain"
	"github.com/HappyOnigiri/PRX/internal/rpc"
)

func TestRPCReturnsTheDiagnosticReportAndItsText(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	feature, err := client.CreateFeature(
		ctx,
		connect.NewRequest(&prxv1.CreateFeatureRequest{
			Title:     "Diagnostics",
			ProjectId: newRPCProject(t, ctx, client, "Diagnostics"),
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	task, err := client.CreateTask(ctx, connect.NewRequest(&prxv1.CreateTaskRequest{
		FeatureId: feature.Msg.GetFeature().GetId(),
		Title:     "Refresh",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.AttachPullRequest(ctx, connect.NewRequest(&prxv1.AttachPullRequestRequest{
		TaskId: task.Msg.GetTask().GetId(),
		Url:    "https://github.com/acme/web/pull/12",
	})); err != nil {
		t.Fatal(err)
	}

	response, err := client.GetDebugReport(ctx, connect.NewRequest(&prxv1.GetDebugReportRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	report := response.Msg.GetReport()
	if report.GetBuild().GetVersion() == "" || report.GetRuntime().GetGeneratedAt() == "" {
		t.Fatalf("report=%+v", report)
	}
	if report.GetRecords().GetFeatures() != 1 || report.GetRecords().GetPullRequests() != 1 {
		t.Fatalf("data=%+v", report.GetRecords())
	}
	if report.GetStorage().GetCliSchemaVersion() != domain.CLIResponseSchemaVersion {
		t.Fatalf("storage=%+v", report.GetStorage())
	}
	if !report.GetStorage().GetDatabaseFile().GetApplicable() {
		t.Fatalf("database file=%+v", report.GetStorage().GetDatabaseFile())
	}
	// WebUI がコピーするのは描画済みのテキストなので、ブラウザ側で組み立て直す
	// のではなく完全な形で届く必要がある。
	for _, section := range []string{"problems:", "build:", "paths:", "storage:", "github_sync:"} {
		if !strings.Contains(response.Msg.GetText(), section) {
			t.Fatalf("text omitted %q:\n%s", section, response.Msg.GetText())
		}
	}
	if len(report.GetProblems()) == 0 {
		t.Fatalf("a pull request that was never refreshed must be reported: %+v", report.GetProblems())
	}
	for _, problem := range report.GetProblems() {
		if problem.GetCode() == prxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_UNSPECIFIED {
			t.Fatalf("problem %+v was not mapped to an enum member", problem)
		}
	}
}

// ドメインの問題コードはすべて個別の enum メンバーとして送られる必要がある。
// でなければクライアントはサーバーの検出結果で分岐できない。
func TestRPCMapsEveryDebugProblemCode(t *testing.T) {
	client := newTestClientForService(t, allProblemsService{})
	response, err := client.GetDebugReport(
		context.Background(),
		connect.NewRequest(&prxv1.GetDebugReportRequest{}),
	)
	if err != nil {
		t.Fatal(err)
	}
	codes := domain.SortedDebugProblemCodes()
	problems := response.Msg.GetReport().GetProblems()
	if len(problems) != len(codes) {
		t.Fatalf("problems=%d codes=%d", len(problems), len(codes))
	}
	seen := map[prxv1.DebugProblemCode]bool{}
	for index, problem := range problems {
		if problem.GetCode() == prxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_UNSPECIFIED {
			t.Fatalf("problem code %q has no enum member", codes[index])
		}
		if seen[problem.GetCode()] {
			t.Fatalf("problem code %q shares an enum member with an earlier code", codes[index])
		}
		seen[problem.GetCode()] = true
	}
}

func newTestClientForService(t *testing.T, service rpc.Service) prxv1connect.PRXServiceClient {
	t.Helper()
	path, handler := rpc.New(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return prxv1connect.NewPRXServiceClient(server.Client(), server.URL)
}

type allProblemsService struct {
	rpc.Service
}

func (allProblemsService) Debug(context.Context) (domain.DebugReport, error) {
	var report domain.DebugReport
	for _, code := range domain.SortedDebugProblemCodes() {
		report.Problems = append(report.Problems, domain.DebugProblem{Code: code, Evidence: "detected"})
	}
	return report, nil
}

// レポートは資格情報を秘密の値抜きで提示する。今後追加されるフィールドでも
// それが保たれるのはスキーマのおかげ。
func TestDebugConfigAuthMethodSchemaCarriesNoSecretFields(t *testing.T) {
	fields := (&prxv1.DebugConfigAuthMethod{}).ProtoReflect().Descriptor().Fields()
	for index := range fields.Len() {
		name := string(fields.Get(index).Name())
		if strings.Contains(name, "token") || strings.Contains(name, "secret_hint") {
			t.Fatalf("field %q can carry credential material", name)
		}
	}
}
