package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
	"github.com/HappyOnigiri/PRX/internal/store"
)

type syncServer struct {
	server   *httptest.Server
	respond  func(token string, request *http.Request) int
	mu       sync.Mutex
	requests []syncRequest
}

type syncRequest struct {
	token string
	path  string
}

func newSyncServer(t *testing.T, respond func(string, *http.Request) int) *syncServer {
	t.Helper()
	result := &syncServer{respond: respond}
	result.server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		path := strings.TrimPrefix(request.URL.Path, "/api/v3")
		result.mu.Lock()
		result.requests = append(result.requests, syncRequest{token: token, path: path})
		result.mu.Unlock()
		if status := result.respond(token, request); status != 0 {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(`{"message":"test GitHub response"}`))
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(path, "/reviews"):
			_, _ = writer.Write([]byte(`[]`))
		case strings.HasSuffix(path, "/requested_reviewers"):
			_, _ = writer.Write([]byte(`{"users":[],"teams":[]}`))
		case strings.HasPrefix(path, "/repos/") && strings.HasSuffix(path, "/pulls"):
			_, _ = writer.Write([]byte(`[]`))
		case strings.HasPrefix(path, "/repos/") && strings.Contains(path, "/pulls/"):
			_, _ = writer.Write(
				[]byte(
					`{"number":42,"state":"open","draft":false,"mergeable":true,` +
						`"node_id":"PR_42","user":{"login":"octocat"},"assignees":[],` +
						`"updated_at":"2026-01-01T00:00:00Z"}`,
				),
			)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(result.server.Close)
	return result
}

func (s *syncServer) count(path, token string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, request := range s.requests {
		if request.path == path && (token == "" || request.token == token) {
			count++
		}
	}
	return count
}

func (s *syncServer) tokens() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, 0, len(s.requests))
	for _, request := range s.requests {
		result = append(result, request.token)
	}
	return result
}

func syncHost(t *testing.T, server *syncServer) (config.Host, string) {
	t.Helper()
	parsed, err := url.Parse(server.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return config.Host{
		Host:      parsed.Host,
		WebURL:    server.server.URL,
		APIURL:    server.server.URL + "/api/v3/",
		UploadURL: server.server.URL + "/api/uploads/",
	}, parsed.Host
}

func syncConfig(hosts []config.Host, methods []config.AuthMethod) config.Config {
	return config.Config{
		Version: config.CurrentVersion,
		GitHub:  config.GitHubConfig{Hosts: hosts, AuthMethods: methods},
	}
}

func newSyncService(t *testing.T) (*Service, *store.Store, domain.Snapshot, map[string]string) {
	t.Helper()
	database, err := store.Open(context.Background(), t.TempDir()+"/sync.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	feature, err := database.CreateFeature(context.Background(), "sync", "Sync", "", "")
	if err != nil {
		t.Fatal(err)
	}
	return &Service{
			repository: database,
		}, database, domain.Snapshot{
			Features: []domain.Feature{feature},
		}, map[string]string{}
}

func addSyncPullRequest(
	t *testing.T,
	database *store.Store,
	snapshot *domain.Snapshot,
	taskFeatures map[string]string,
	host, owner, repository string,
	number int64,
) domain.PullRequest {
	t.Helper()
	task, err := database.CreateTask(
		context.Background(),
		snapshot.Features[0].ID,
		fmt.Sprintf("PR %d", number),
		"",
		domain.TaskKindPR,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Tasks = append(snapshot.Tasks, task)
	taskFeatures[task.ID] = task.FeatureID
	value := domain.PullRequest{
		TaskID:       task.ID,
		Host:         host,
		Owner:        owner,
		Repository:   repository,
		Number:       number,
		URL:          fmt.Sprintf("https://%s/%s/%s/pull/%d", host, owner, repository, number),
		State:        domain.PullRequestStateUnknown,
		ReviewState:  domain.ReviewStateUnknown,
		Mergeability: domain.MergeabilityUnknown,
		Stale:        true,
	}
	snapshot.PullRequests = append(snapshot.PullRequests, value)
	return value
}

func routedClient(servers ...*syncServer) *http.Client {
	transports := make(map[string]http.RoundTripper, len(servers))
	for _, server := range servers {
		parsed, _ := url.Parse(server.server.URL)
		transports[parsed.Host] = server.server.Client().Transport
	}
	return &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		transport, ok := transports[request.URL.Host]
		if !ok {
			return nil, fmt.Errorf("no test transport for %s", request.URL.Host)
		}
		return transport.RoundTrip(request)
	})}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSyncLiveFallsBackAfterUnauthorizedAndCachesSuccessfulMethod(t *testing.T) {
	server := newSyncServer(t, func(token string, _ *http.Request) int {
		if token == "bad-token" {
			return http.StatusUnauthorized
		}
		return 0
	})
	hostConfig, host := syncHost(t, server)
	settings := syncConfig(
		[]config.Host{hostConfig},
		[]config.AuthMethod{
			{ID: "bad", Host: host, Type: config.AuthMethodTypeInline, Token: "bad-token"},
			{ID: "good", Host: host, Type: config.AuthMethodTypeInline, Token: "good-token"},
		},
	)
	resolver, err := githubprovider.NewResolver(settings, server.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	service, database, snapshot, taskFeatures := newSyncService(t)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
	succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
	if err != nil || succeeded != 1 || failed != 0 {
		stored, _ := database.GetPullRequest(context.Background(), snapshot.PullRequests[0].TaskID)
		t.Fatalf(
			"first sync succeeded=%d failed=%d err=%v sync_error=%q requests=%+v",
			succeeded,
			failed,
			err,
			stored.SyncError,
			server.requests,
		)
	}
	cached, found, err := database.GetGitHubRepositoryAuthCache(context.Background(), host, "acme", "api")
	if err != nil || !found || cached != "good" {
		t.Fatalf("cache=%q found=%v err=%v", cached, found, err)
	}
	listCalls := server.count("/repos/acme/api/pulls", "")
	if listCalls != 2 {
		t.Fatalf("probe calls=%d, want failed and successful candidate probes", listCalls)
	}

	nextResolver, err := githubprovider.NewResolver(settings, server.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	succeeded, failed, err = service.syncLive(context.Background(), snapshot, taskFeatures, "", "", nextResolver)
	if err != nil || succeeded != 1 || failed != 0 {
		t.Fatalf("cached sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	if got := server.count("/repos/acme/api/pulls", ""); got != listCalls {
		t.Fatalf("cached sync made %d list calls, want %d", got, listCalls)
	}
	for _, token := range server.tokens() {
		if token == "bad-token" && server.count("/repos/acme/api/pulls", "bad-token") > 1 {
			t.Fatalf("unauthorized method was retried in one sync: %v", server.tokens())
		}
	}
}

func TestSyncLiveDisambiguatesCached404BeforeFallback(t *testing.T) {
	t.Run("public repository means PR missing", func(t *testing.T) {
		server := newSyncServer(t, func(token string, request *http.Request) int {
			if token == "cached-token" && strings.Contains(request.URL.Path, "/pulls/42") {
				return http.StatusNotFound
			}
			return 0
		})
		hostConfig, host := syncHost(t, server)
		settings := syncConfig([]config.Host{hostConfig}, []config.AuthMethod{
			{ID: "cached", Host: host, Type: config.AuthMethodTypeInline, Token: "cached-token"},
			{ID: "fallback", Host: host, Type: config.AuthMethodTypeInline, Token: "fallback-token"},
		})
		service, database, snapshot, taskFeatures := newSyncService(t)
		addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
		if err := database.UpsertGitHubRepositoryAuthCache(
			context.Background(),
			host,
			"acme",
			"api",
			"cached",
		); err != nil {
			t.Fatal(err)
		}
		resolver, err := githubprovider.NewResolver(settings, server.server.Client())
		if err != nil {
			t.Fatal(err)
		}
		succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
		if err != nil || succeeded != 0 || failed != 1 {
			t.Fatalf("sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
		}
		if server.count("/repos/acme/api/pulls", "fallback-token") != 0 {
			t.Fatalf("fallback credential was tried for a missing PR: %v", server.tokens())
		}
	})

	t.Run("private repository falls back", func(t *testing.T) {
		server := newSyncServer(t, func(token string, request *http.Request) int {
			if token == "cached-token" {
				return http.StatusNotFound
			}
			return 0
		})
		hostConfig, host := syncHost(t, server)
		settings := syncConfig([]config.Host{hostConfig}, []config.AuthMethod{
			{ID: "cached", Host: host, Type: config.AuthMethodTypeInline, Token: "cached-token"},
			{ID: "fallback", Host: host, Type: config.AuthMethodTypeInline, Token: "fallback-token"},
		})
		service, database, snapshot, taskFeatures := newSyncService(t)
		addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
		if err := database.UpsertGitHubRepositoryAuthCache(
			context.Background(),
			host,
			"acme",
			"api",
			"cached",
		); err != nil {
			t.Fatal(err)
		}
		resolver, err := githubprovider.NewResolver(settings, server.server.Client())
		if err != nil {
			t.Fatal(err)
		}
		succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
		if err != nil || succeeded != 1 || failed != 0 {
			stored, _ := database.GetPullRequest(context.Background(), snapshot.PullRequests[0].TaskID)
			t.Fatalf(
				"sync succeeded=%d failed=%d err=%v sync_error=%q requests=%+v",
				succeeded,
				failed,
				err,
				stored.SyncError,
				server.requests,
			)
		}
		cached, found, err := database.GetGitHubRepositoryAuthCache(context.Background(), host, "acme", "api")
		if err != nil || !found || cached != "fallback" {
			t.Fatalf("fallback cache=%q found=%v err=%v", cached, found, err)
		}
	})
}

func TestSyncLiveDoesNotFallbackOnRateLimit(t *testing.T) {
	server := newSyncServer(t, func(token string, request *http.Request) int {
		if token == "limited-token" && strings.HasSuffix(request.URL.Path, "/repos/acme/api/pulls") {
			return http.StatusTooManyRequests
		}
		return 0
	})
	hostConfig, host := syncHost(t, server)
	settings := syncConfig([]config.Host{hostConfig}, []config.AuthMethod{
		{ID: "limited", Host: host, Type: config.AuthMethodTypeInline, Token: "limited-token"},
		{ID: "other", Host: host, Type: config.AuthMethodTypeInline, Token: "other-token"},
	})
	service, database, snapshot, taskFeatures := newSyncService(t)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
	resolver, err := githubprovider.NewResolver(settings, server.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
	if err != nil || succeeded != 0 || failed != 1 {
		t.Fatalf("sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	if server.count("/repos/acme/api/pulls", "other-token") != 0 {
		t.Fatalf("rate-limited repository tried another credential: %v", server.tokens())
	}
}

func TestSyncLiveReportsWhenAllCredentialsAreUnauthorized(t *testing.T) {
	server := newSyncServer(t, func(string, *http.Request) int {
		return http.StatusUnauthorized
	})
	hostConfig, host := syncHost(t, server)
	settings := syncConfig([]config.Host{hostConfig}, []config.AuthMethod{
		{ID: "first", Host: host, Type: config.AuthMethodTypeInline, Token: "first-token"},
		{ID: "second", Host: host, Type: config.AuthMethodTypeInline, Token: "second-token"},
	})
	service, database, snapshot, taskFeatures := newSyncService(t)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
	resolver, err := githubprovider.NewResolver(settings, server.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
	if err != nil || succeeded != 0 || failed != 1 {
		t.Fatalf("sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	stored, err := database.GetPullRequest(context.Background(), snapshot.PullRequests[0].TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Stale || !strings.Contains(stored.SyncError, "no usable GitHub authentication") {
		t.Fatalf("stored pull request=%+v", stored)
	}
}

func TestSyncLiveKeepsCredentialsOnTheirConfiguredHost(t *testing.T) {
	first := newSyncServer(t, func(token string, _ *http.Request) int {
		if token != "first-token" {
			return http.StatusUnauthorized
		}
		return 0
	})
	second := newSyncServer(t, func(token string, _ *http.Request) int {
		if token != "second-token" {
			return http.StatusUnauthorized
		}
		return 0
	})
	firstConfig, firstHost := syncHost(t, first)
	secondConfig, secondHost := syncHost(t, second)
	settings := syncConfig(
		[]config.Host{firstConfig, secondConfig},
		[]config.AuthMethod{
			{ID: "first", Host: firstHost, Type: config.AuthMethodTypeInline, Token: "first-token"},
			{ID: "second", Host: secondHost, Type: config.AuthMethodTypeInline, Token: "second-token"},
		},
	)
	service, database, snapshot, taskFeatures := newSyncService(t)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, firstHost, "acme", "first", 42)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, secondHost, "acme", "second", 42)
	resolver, err := githubprovider.NewResolver(settings, routedClient(first, second))
	if err != nil {
		t.Fatal(err)
	}
	succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
	if err != nil || succeeded != 2 || failed != 0 {
		t.Fatalf(
			"sync succeeded=%d failed=%d err=%v first=%+v second=%+v",
			succeeded,
			failed,
			err,
			first.requests,
			second.requests,
		)
	}
	for _, token := range first.tokens() {
		if token != "first-token" {
			t.Fatalf("first host received token %q: %v", token, first.tokens())
		}
	}
	for _, token := range second.tokens() {
		if token != "second-token" {
			t.Fatalf("second host received token %q: %v", token, second.tokens())
		}
	}
	firstCache, firstFound, err := database.GetGitHubRepositoryAuthCache(
		context.Background(),
		firstHost,
		"acme",
		"first",
	)
	if err != nil || !firstFound || firstCache != "first" {
		t.Fatalf("first cache=%q found=%v err=%v", firstCache, firstFound, err)
	}
	secondCache, secondFound, err := database.GetGitHubRepositoryAuthCache(
		context.Background(),
		secondHost,
		"acme",
		"second",
	)
	if err != nil || !secondFound || secondCache != "second" {
		t.Fatalf("second cache=%q found=%v err=%v", secondCache, secondFound, err)
	}
}

func TestSyncLiveBatchesRepositoriesOnTheSameHost(t *testing.T) {
	server := newSyncServer(t, func(token string, _ *http.Request) int {
		if token != "shared-token" {
			return http.StatusUnauthorized
		}
		return 0
	})
	hostConfig, host := syncHost(t, server)
	settings := syncConfig([]config.Host{hostConfig}, []config.AuthMethod{{
		ID: "shared", Host: host, Type: config.AuthMethodTypeInline, Token: "shared-token",
	}})
	service, database, snapshot, taskFeatures := newSyncService(t)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "api", 42)
	addSyncPullRequest(t, database, &snapshot, taskFeatures, host, "acme", "web", 43)
	resolver, err := githubprovider.NewResolver(settings, server.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	succeeded, failed, err := service.syncLive(context.Background(), snapshot, taskFeatures, "", "", resolver)
	if err != nil || succeeded != 2 || failed != 0 {
		t.Fatalf("same-host sync succeeded=%d failed=%d err=%v", succeeded, failed, err)
	}
	if server.count("/api/graphql", "shared-token") != 1 {
		t.Fatalf("GraphQL batch requests=%+v", server.requests)
	}
}

func TestPersistSyncResultDoesNotCountTerminalPartialFailure(t *testing.T) {
	ctx := context.Background()
	service, database, snapshot, taskFeatures := newSyncService(t)
	current := addSyncPullRequest(
		t, database, &snapshot, taskFeatures, "github.com", "acme", "api", 42,
	)
	current.State = domain.PullRequestStateOpen
	current.SyncError = "previous error"
	current.Stale = false
	if _, err := database.UpsertPullRequest(ctx, current); err != nil {
		t.Fatal(err)
	}
	partial := current
	partial.State = domain.PullRequestStateMerged
	partial.NodeID = "merged-node"
	succeeded, failed, err := service.persistSyncResult(ctx, []domain.PullRequest{current}, repositorySyncResult{
		partials: map[string]domain.PullRequest{current.TaskID: partial},
		failures: map[string]error{current.TaskID: errors.New("review metadata unavailable")},
	})
	if err != nil || succeeded != 0 || failed != 0 {
		t.Fatalf("persist result=(%d, %d, %v)", succeeded, failed, err)
	}
	stored, err := database.GetPullRequest(ctx, current.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != domain.PullRequestStateMerged || stored.NodeID != "merged-node" ||
		stored.SyncError != "" || !stored.Stale || stored.LastSyncedAt == nil {
		t.Fatalf("stored terminal partial=%+v", stored)
	}
}

// A chunked fetch stops at its first failing chunk, so the pull requests from
// that position on were never attempted and must be retried rather than
// recorded as refreshed.
func TestUnresolvedIndexMarksWhereAChunkedFetchStopped(t *testing.T) {
	values := []domain.PullRequest{
		{TaskID: "first"}, {TaskID: "second"}, {TaskID: "third"},
	}
	for name, test := range map[string]struct {
		batch githubprovider.BatchResult
		want  int
	}{
		"nothing attempted": {
			batch: githubprovider.BatchResult{},
			want:  0,
		},
		"first chunk succeeded": {
			batch: githubprovider.BatchResult{
				PullRequests: map[string]domain.PullRequest{"first": {TaskID: "first"}},
			},
			want: 1,
		},
		"item error still counts as attempted": {
			batch: githubprovider.BatchResult{
				PullRequests: map[string]domain.PullRequest{"first": {TaskID: "first"}},
				Errors:       map[string]error{"second": fmt.Errorf("not found")},
			},
			want: 2,
		},
		"everything attempted": {
			batch: githubprovider.BatchResult{
				PullRequests: map[string]domain.PullRequest{
					"first": {TaskID: "first"}, "second": {TaskID: "second"}, "third": {TaskID: "third"},
				},
			},
			want: 3,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := unresolvedIndex(values, test.batch); got != test.want {
				t.Fatalf("unresolvedIndex=%d, want %d", got, test.want)
			}
		})
	}
}
