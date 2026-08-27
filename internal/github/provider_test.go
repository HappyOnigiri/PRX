package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/domain"
	gh "github.com/google/go-github/v80/github"
)

func TestParsePullRequestURL(t *testing.T) {
	_, _, _, _, err := ParsePullRequestURL("https://github.com/Acme/API/pull/42/files")
	if err == nil {
		t.Fatal("suffix must be rejected")
	}
	owner, repo, number, canonical, err := ParsePullRequestURL("https://github.com/Acme/API/pull/42")
	if err != nil || owner != "Acme" || repo != "API" || number != 42 || canonical != "https://github.com/Acme/API/pull/42" {
		t.Fatalf("got %s/%s #%d %s err=%v", owner, repo, number, canonical, err)
	}
}

func TestLiveProviderMapsStates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":7,"state":"open","draft":false,"mergeable":false,"node_id":"PR_7","user":{"login":"octocat"},"assignees":[{"login":"mona"}],"updated_at":"2026-01-01T00:00:00Z"}`))
	})
	mux.HandleFunc("/repos/acme/api/pulls/7/reviews", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"state":"CHANGES_REQUESTED","user":{"login":"reviewer"}}]`))
	})
	mux.HandleFunc("/repos/acme/api/pulls/7/requested_reviewers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"users":[],"teams":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := gh.NewClient(nil)
	base, _ := client.BaseURL.Parse(server.URL + "/")
	client.BaseURL = base
	provider := &LiveProvider{client: client}
	got, err := provider.Fetch(context.Background(), domain.PullRequest{Owner: "acme", Repository: "api", Number: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewState != "changes_requested" || got.Mergeability != "conflicting" || got.Author != "octocat" || got.Stale {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestFixturePreservesError(t *testing.T) {
	provider, _ := NewFixtureProvider("demo")
	got, err := provider.Fetch(context.Background(), domain.PullRequest{Number: 3, State: "unknown", Stale: true})
	if err != nil || got.State != "merged" || got.Stale {
		t.Fatalf("fixture got=%+v err=%v", got, err)
	}
}
