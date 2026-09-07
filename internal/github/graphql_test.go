package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

var (
	repositoryAliasPattern = regexp.MustCompile(`r([0-9]+):repository`)
	pullAliasPattern       = regexp.MustCompile(`p([0-9]+):pullRequest`)
)

func TestLiveProviderFetchBatchAggregatesRepositoriesAndMapsPartialErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		query := decodeGraphQLQuery(t, request)
		data := graphQLDataForQuery(query)
		data["r1"] = nil
		writeGraphQLResponse(t, writer, map[string]any{
			"data": data,
			"errors": []any{map[string]any{
				"message": "repository not found", "path": []any{"r1"},
				"extensions": map[string]any{"type": "NOT_FOUND"},
			}},
		})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	values := []domain.PullRequest{
		{TaskID: "first", Owner: "acme", Repository: "api", Number: 1},
		{TaskID: "second", Owner: "acme", Repository: "web", Number: 2},
	}
	result, err := provider.FetchBatch(context.Background(), values)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("GraphQL requests=%d, want 1", requests.Load())
	}
	if result.PullRequests["first"].Author != "octocat" || result.PullRequests["first"].Stale {
		t.Fatalf("first pull request=%+v", result.PullRequests["first"])
	}
	if ClassOf(result.Errors["second"]) != ErrorClassNotFound {
		t.Fatalf("second error class=%s err=%v", ClassOf(result.Errors["second"]), result.Errors["second"])
	}
}

func TestLiveProviderFetchBatchChunksAndPaginatesConnections(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		query := decodeGraphQLQuery(t, request)
		if regexp.MustCompile(`node\(id:`).MatchString(query) {
			writeGraphQLResponse(t, writer, map[string]any{
				"data": map[string]any{"node": map[string]any{
					"assignees": map[string]any{
						"nodes":    []any{map[string]any{"login": "hubot"}},
						"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
					},
				}},
			})
			return
		}
		data := graphQLDataForQuery(query)
		if requests.Load() == 1 {
			firstRepo := data["r0"].(map[string]any)
			firstPull := firstRepo["p0"].(map[string]any)
			firstPull["assignees"] = map[string]any{
				"nodes":    []any{map[string]any{"login": "octocat"}},
				"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "cursor-1"},
			}
		}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	values := make([]domain.PullRequest, 21)
	for index := range values {
		values[index] = domain.PullRequest{
			TaskID: fmt.Sprintf("task-%02d", index), Owner: "acme",
			Repository: fmt.Sprintf("repo-%02d", index), Number: int64(index + 1),
		}
	}
	result, err := provider.FetchBatch(context.Background(), values)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("GraphQL requests=%d, want two chunks and one page", requests.Load())
	}
	if len(result.PullRequests) != len(values) || len(result.PullRequests["task-00"].Assignees) != 2 {
		t.Fatalf("batch result count=%d first=%+v", len(result.PullRequests), result.PullRequests["task-00"])
	}
}

func TestLiveProviderFetchBatchSkipsPaginationForTerminalPullRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		query := decodeGraphQLQuery(t, request)
		if regexp.MustCompile(`node\(id:`).MatchString(query) {
			t.Errorf("terminal pull request triggered a connection page: %s", query)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pull["state"] = "CLOSED"
		pull["assignees"] = nil
		pull["latestReviews"] = nil
		pull["reviewRequests"] = nil
		pull["mergeable"] = nil
		writeGraphQLResponse(t, writer, map[string]any{
			"data": data,
			"errors": []any{map[string]any{
				"message": "review metadata unavailable",
				"path":    []any{"r0", "p0", "latestReviews"},
			}},
		})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "terminal", Owner: "acme", Repository: "api", Number: 7,
		Assignees: []string{"known-assignee"}, ReviewState: domain.ReviewStateApproved,
		Mergeability: domain.MergeabilityConflicting,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("GraphQL requests=%d, want 1", requests.Load())
	}
	if got := result.PullRequests["terminal"]; got.State != domain.PullRequestStateClosed ||
		got.Stale || got.SyncError != "" || got.ReviewState != domain.ReviewStateApproved ||
		got.Mergeability != domain.MergeabilityConflicting ||
		len(got.Assignees) != 1 || got.Assignees[0] != "known-assignee" || len(result.Errors) != 0 {
		t.Fatalf("terminal result=%+v errors=%v partial=%v", got, result.Errors, result.PartialPullRequests)
	}
}

func TestLiveProviderFetchBatchPreservesPartialPullRequestOnPaginationError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		query := decodeGraphQLQuery(t, request)
		if regexp.MustCompile(`node\(id:`).MatchString(query) {
			writeGraphQLResponse(t, writer, map[string]any{
				"errors": []any{map[string]any{"message": "review metadata unavailable"}},
			})
			return
		}
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pull["assignees"] = map[string]any{
			"nodes":    []any{map[string]any{"login": "octocat"}},
			"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "assignee-cursor"},
		}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "partial", Owner: "acme", Repository: "api", Number: 8,
	}})
	if err != nil {
		t.Fatal(err)
	}
	partial := result.PartialPullRequests["partial"]
	if requests.Load() != 2 || partial.Author != "octocat" ||
		result.Errors["partial"] == nil || len(result.PullRequests) != 0 {
		t.Fatalf(
			"partial result=%+v errors=%v pulls=%v requests=%d",
			partial,
			result.Errors,
			result.PullRequests,
			requests.Load(),
		)
	}
}

// プロキシや GitHub Enterprise は、REST は PR を返すのに GraphQL には権限や
// メソッドのエラーを返すことがある。フォールバックを 404 に限ってはならない。
func TestLiveProviderFetchBatchFallsBackToRESTOnAnyGraphQLHTTPError(t *testing.T) {
	for name, status := range map[string]int{
		"forbidden":          http.StatusForbidden,
		"unauthorized":       http.StatusUnauthorized,
		"method not allowed": http.StatusMethodNotAllowed,
	} {
		t.Run(name, func(t *testing.T) {
			var restRequests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(
				func(writer http.ResponseWriter, request *http.Request) {
					if request.URL.Path == "/graphql" {
						writer.WriteHeader(status)
						_, _ = writer.Write([]byte(`{"message":"blocked"}`))
						return
					}
					restRequests.Add(1)
					writeRESTPullRequestResponse(t, writer, request)
				},
			))
			defer server.Close()
			provider, err := NewLiveProvider(context.Background(), LiveProviderOptions{
				Token:      "test-token",
				APIURL:     server.URL + "/api/v3/",
				GraphQLURL: server.URL + "/graphql",
				HTTPClient: server.Client(),
			})
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{
				{TaskID: "first", Owner: "acme", Repository: "api", Number: 1},
			})
			if err != nil {
				t.Fatal(err)
			}
			if restRequests.Load() == 0 {
				t.Fatal("GraphQL failure did not fall back to REST")
			}
			if result.PullRequests["first"].Author != "octocat" || result.Errors["first"] != nil {
				t.Fatalf("fallback result=%+v err=%v", result.PullRequests["first"], result.Errors["first"])
			}
		})
	}
}

// FetchBatch は後続チャンクの失敗と併せて、それ以前のチャンクの結果も報告する。
// 呼び出し元が実際に更新できた PR を保存できるようにするため。
func TestLiveProviderFetchBatchKeepsEarlierChunksWhenALaterChunkFails(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requests.Add(1) > 1 {
			writeGraphQLResponse(t, writer, map[string]any{
				"errors": []any{map[string]any{"message": "server is unavailable"}},
			})
			return
		}
		writeGraphQLResponse(t, writer, map[string]any{
			"data": graphQLDataForQuery(decodeGraphQLQuery(t, request)),
		})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	values := make([]domain.PullRequest, 21)
	for index := range values {
		values[index] = domain.PullRequest{
			TaskID: fmt.Sprintf("task-%02d", index), Owner: "acme",
			Repository: fmt.Sprintf("repo-%02d", index), Number: int64(index + 1),
		}
	}
	result, err := provider.FetchBatch(context.Background(), values)
	if err == nil {
		t.Fatal("second chunk failure was not reported")
	}
	if len(result.PullRequests) != graphQLBatchSize {
		t.Fatalf("kept pull requests=%d, want %d", len(result.PullRequests), graphQLBatchSize)
	}
	if _, ok := result.PullRequests["task-20"]; ok {
		t.Fatal("failed chunk produced a pull request")
	}
}

func writeRESTPullRequestResponse(t *testing.T, writer http.ResponseWriter, request *http.Request) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	switch {
	case strings.HasSuffix(request.URL.Path, "/reviews"):
		_, _ = writer.Write([]byte(`[]`))
	case strings.HasSuffix(request.URL.Path, "/requested_reviewers"):
		_, _ = writer.Write([]byte(`{"users":[],"teams":[]}`))
	case strings.HasSuffix(request.URL.Path, "/commits"):
		_, _ = writer.Write([]byte(
			`[{"sha":"c0ffee","commit":{"committer":{"date":"2026-01-01T00:00:00Z"}}}]`,
		))
	case strings.HasSuffix(request.URL.Path, "/status"):
		_, _ = writer.Write([]byte(`{"state":"success","total_count":1}`))
	case strings.HasSuffix(request.URL.Path, "/check-runs"):
		_, _ = writer.Write([]byte(`{"total_count":0,"check_runs":[]}`))
	default:
		_, _ = writer.Write([]byte(`{"node_id":"PR_node","state":"open","merged":false,` +
			`"draft":false,"mergeable":true,"updated_at":"2026-01-01T00:00:00Z",` +
			`"user":{"login":"octocat"},"assignees":[]}`))
	}
}

func newGraphQLTestProvider(t *testing.T, server *httptest.Server) *LiveProvider {
	t.Helper()
	provider, err := NewLiveProvider(context.Background(), LiveProviderOptions{
		Token: "test-token", GraphQLURL: server.URL, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func decodeGraphQLQuery(t *testing.T, request *http.Request) string {
	t.Helper()
	if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer test-token" {
		t.Fatalf("request method=%s authorization=%q", request.Method, request.Header.Get("Authorization"))
	}
	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Query
}

func graphQLDataForQuery(query string) map[string]any {
	data := map[string]any{}
	pulls := pullAliasPattern.FindAllStringSubmatch(query, -1)
	for _, repositoryMatch := range repositoryAliasPattern.FindAllStringSubmatch(query, -1) {
		repository := map[string]any{}
		for _, pullMatch := range pulls {
			repository["p"+pullMatch[1]] = validGraphQLPullRequest()
		}
		data["r"+repositoryMatch[1]] = repository
	}
	return data
}

func validGraphQLPullRequest() map[string]any {
	pageInfo := map[string]any{"hasNextPage": false, "endCursor": ""}
	return map[string]any{
		"id": "PR_node", "author": map[string]any{"login": "octocat"},
		"assignees": map[string]any{"nodes": []any{}, "pageInfo": pageInfo},
		"state":     "OPEN", "merged": false, "isDraft": false, "mergeable": "MERGEABLE",
		"updatedAt": "2026-01-01T00:00:00Z",
		"latestReviews": map[string]any{
			"nodes": []any{map[string]any{
				"author": map[string]any{"login": "reviewer"}, "state": "APPROVED",
			}},
			"pageInfo": pageInfo,
		},
		"reviewRequests": map[string]any{"nodes": []any{}, "pageInfo": pageInfo},
		"commits": map[string]any{"nodes": []any{map[string]any{
			"commit": map[string]any{
				"committedDate":     "2026-01-01T00:00:00Z",
				"statusCheckRollup": map[string]any{"state": "SUCCESS"},
			},
		}}},
	}
}

func writeGraphQLResponse(t *testing.T, writer http.ResponseWriter, value any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatal(err)
	}
}

// GraphQL 経路の 3 項目は、レビュー中の判定に直接効く。変更要求だけを時刻の対象に
// し、push 時刻には最新コミットのコミット日時を使う。
func TestLiveProviderFetchBatchReadsReviewTimingFields(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := decodeGraphQLQuery(t, request)
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pageInfo := map[string]any{"hasNextPage": false, "endCursor": ""}
		pull["latestReviews"] = map[string]any{
			"nodes": []any{
				map[string]any{
					"author": map[string]any{"login": "commenter"},
					"state":  "COMMENTED", "submittedAt": "2026-03-05T00:00:00Z",
				},
				map[string]any{
					"author": map[string]any{"login": "reviewer"},
					"state":  "CHANGES_REQUESTED", "submittedAt": "2026-03-01T00:00:00Z",
				},
			},
			"pageInfo": pageInfo,
		}
		pull["reviewRequests"] = map[string]any{
			"nodes":    []any{map[string]any{"requestedReviewer": map[string]any{"login": "mona"}}},
			"pageInfo": pageInfo,
		}
		pull["commits"] = map[string]any{"nodes": []any{map[string]any{
			"commit": map[string]any{"committedDate": "2026-03-02T00:00:00Z"},
		}}}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "timing", Owner: "acme", Repository: "api", Number: 9,
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := result.PullRequests["timing"]
	if !got.ReviewRequestPending ||
		got.ChangesRequestedAt == nil || got.ChangesRequestedAt.Format(time.RFC3339) != "2026-03-01T00:00:00Z" ||
		got.LastPushedAt == nil || got.LastPushedAt.Format(time.RFC3339) != "2026-03-02T00:00:00Z" {
		t.Fatalf("timing result=%+v", got)
	}
}

// latestReviews の後続ページを取得できないときは、先頭ページだけで変更要求時刻を
// 上書きしない。巻き戻すと、未対応の task がレビュー中に見える。
func TestLiveProviderFetchBatchKeepsChangesRequestedAtWhenReviewPagesFail(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := decodeGraphQLQuery(t, request)
		if regexp.MustCompile(`node\(id:`).MatchString(query) {
			writeGraphQLResponse(t, writer, map[string]any{
				"errors": []any{map[string]any{"message": "review metadata unavailable"}},
			})
			return
		}
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pull["latestReviews"] = map[string]any{
			"nodes": []any{map[string]any{
				"author": map[string]any{"login": "reviewer"},
				"state":  "CHANGES_REQUESTED", "submittedAt": "2026-01-01T00:00:00Z",
			}},
			"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "review-cursor"},
		}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	known := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "pages", Owner: "acme", Repository: "api", Number: 10,
		ChangesRequestedAt: &known,
	}})
	if err != nil {
		t.Fatal(err)
	}
	partial := result.PartialPullRequests["pages"]
	if result.Errors["pages"] == nil || partial.ChangesRequestedAt == nil ||
		!partial.ChangesRequestedAt.Equal(known) {
		t.Fatalf("partial=%+v errors=%v", partial, result.Errors)
	}
}

// 終了した pull request では、REST 経路と同じく 3 項目を落とす。
func TestLiveProviderFetchBatchClearsReviewTimingForFinishedPullRequests(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := decodeGraphQLQuery(t, request)
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pull["state"] = "MERGED"
		pull["merged"] = true
		pull["reviewRequests"] = map[string]any{
			"nodes":    []any{map[string]any{"requestedReviewer": map[string]any{"login": "mona"}}},
			"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
		}
		pull["commits"] = map[string]any{"nodes": []any{map[string]any{
			"commit": map[string]any{"committedDate": "2026-03-02T00:00:00Z"},
		}}}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	known := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "merged", Owner: "acme", Repository: "api", Number: 11,
		ChangesRequestedAt: &known, LastPushedAt: &known, ReviewRequestPending: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := result.PullRequests["merged"]
	if got.ReviewRequestPending || got.ChangesRequestedAt != nil || got.LastPushedAt != nil {
		t.Fatalf("merged result=%+v", got)
	}
}

// クエリ検証エラーは HTTP 200 で data を伴わずに返る。path は repository の別名から
// 始まらないので、個別の失敗に割り当てるとメッセージが失われる。
func TestLiveProviderFetchBatchReportsQueryValidationErrors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeGraphQLResponse(t, writer, map[string]any{
			"errors": []any{map[string]any{
				"message": "Field 'pushedAt' doesn't exist on type 'Commit'",
				"path":    []any{"query", "r0", "p0", "commits", "nodes", "commit", "pushedAt"},
				"extensions": map[string]any{
					"code": "undefinedField", "typeName": "Commit", "fieldName": "pushedAt",
				},
			}},
		})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	result, err := provider.FetchBatch(context.Background(), []domain.PullRequest{{
		TaskID: "invalid", Owner: "acme", Repository: "api", Number: 12,
	}})
	if err == nil || !strings.Contains(err.Error(), "pushedAt") {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	if result.Errors["invalid"] != nil {
		t.Fatalf("item error=%v", result.Errors["invalid"])
	}
}

// GraphQL のロールアップは 1 値に正規化する。rollup が null ならチェックが 1 件も
// ないので none で、未知の値は成功と言い切らずに unknown へ落とす。
func TestCheckStateFromGraphQL(t *testing.T) {
	withRollup := func(commit map[string]any) graphQLPullRequest {
		var value graphQLPullRequest
		body, err := json.Marshal(map[string]any{
			"commits": map[string]any{"nodes": []any{map[string]any{"commit": commit}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	rollup := func(state string) map[string]any {
		return map[string]any{"statusCheckRollup": map[string]any{"state": state}}
	}
	tests := []struct {
		name   string
		commit map[string]any
		want   domain.CheckState
	}{
		{name: "success", commit: rollup("SUCCESS"), want: domain.CheckStateSuccess},
		{name: "pending", commit: rollup("PENDING"), want: domain.CheckStatePending},
		{name: "expected", commit: rollup("EXPECTED"), want: domain.CheckStatePending},
		{name: "failure", commit: rollup("FAILURE"), want: domain.CheckStateFailure},
		{name: "error", commit: rollup("ERROR"), want: domain.CheckStateFailure},
		{name: "unrecognised", commit: rollup("MYSTERY"), want: domain.CheckStateUnknown},
		{
			name:   "a null roll-up means no checks",
			commit: map[string]any{"statusCheckRollup": nil},
			want:   domain.CheckStateNone,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := checkStateFromGraphQL(withRollup(test.commit)); got != test.want {
				t.Fatalf("check state=%q want %q", got, test.want)
			}
		})
	}
	// コミットが 1 件も返らなければ、紐づくチェックも存在しない。
	if got := checkStateFromGraphQL(graphQLPullRequest{}); got != domain.CheckStateNone {
		t.Fatalf("check state without commits=%q want none", got)
	}
}

// GraphQL 経路も CI を埋め、終了した pull request では落とす。REST 経路が終了時に
// 追加のリクエストをかけないため、値をそろえないと経路で結果が変わる。
func TestLiveProviderFetchBatchReadsAndClearsCheckState(t *testing.T) {
	var merged atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := decodeGraphQLQuery(t, request)
		data := graphQLDataForQuery(query)
		pull := data["r0"].(map[string]any)["p0"].(map[string]any)
		pull["commits"] = map[string]any{"nodes": []any{map[string]any{
			"commit": map[string]any{
				"committedDate":     "2026-03-02T00:00:00Z",
				"statusCheckRollup": map[string]any{"state": "FAILURE"},
			},
		}}}
		if merged.Load() {
			pull["state"] = "MERGED"
			pull["merged"] = true
		}
		writeGraphQLResponse(t, writer, map[string]any{"data": data})
	}))
	defer server.Close()
	provider := newGraphQLTestProvider(t, server)
	current := []domain.PullRequest{{TaskID: "ci", Owner: "acme", Repository: "api", Number: 9}}
	result, err := provider.FetchBatch(context.Background(), current)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PullRequests["ci"].CheckState; got != domain.CheckStateFailure {
		t.Fatalf("check state=%q want failure", got)
	}
	merged.Store(true)
	result, err = provider.FetchBatch(context.Background(), current)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PullRequests["ci"].CheckState; got != domain.CheckStateUnknown {
		t.Fatalf("check state=%q want unknown on a merged pull request", got)
	}
}
