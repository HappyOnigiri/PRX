package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

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
	}
}

func writeGraphQLResponse(t *testing.T, writer http.ResponseWriter, value any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatal(err)
	}
}
