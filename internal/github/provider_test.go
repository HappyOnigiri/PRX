package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	gh "github.com/google/go-github/v80/github"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestParsePullRequestURL(t *testing.T) {
	_, _, _, _, err := ParsePullRequestURL("https://github.com/Acme/API/pull/42/files")
	if err == nil {
		t.Fatal("suffix must be rejected")
	}
	owner, repo, number, canonical, err := ParsePullRequestURL("https://github.com/Acme/API/pull/42")
	if err != nil || owner != "Acme" || repo != "API" || number != 42 ||
		canonical != "https://github.com/Acme/API/pull/42" {
		t.Fatalf("got %s/%s #%d %s err=%v", owner, repo, number, canonical, err)
	}
}

func TestLiveProviderMapsStates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"number":7,"state":"open","draft":false,"mergeable":false,"node_id":"PR_7",` +
					`"user":{"login":"octocat"},"assignees":[{"login":"mona"}],` +
					`"updated_at":"2026-01-01T00:00:00Z"}`,
			),
		)
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
	if got.ReviewState != domain.ReviewStateChangesRequested || got.Mergeability != domain.MergeabilityConflicting ||
		got.Author != "octocat" ||
		got.Stale {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestLiveProviderReturnsCoreStateWhenReviewFetchFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"number":7,"state":"open","draft":true,"mergeable":false,"node_id":"PR_7",` +
				`"user":{"login":"octocat"},"assignees":[{"login":"mona"}],` +
				`"updated_at":"2026-01-01T00:00:00Z"}`,
		))
	})
	mux.HandleFunc("/repos/acme/api/pulls/7/reviews", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "review service unavailable", http.StatusServiceUnavailable)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := gh.NewClient(nil)
	base, _ := client.BaseURL.Parse(server.URL + "/")
	client.BaseURL = base
	provider := &LiveProvider{client: client}
	got, err := provider.Fetch(context.Background(), domain.PullRequest{
		Owner: "acme", Repository: "api", Number: 7,
		State: domain.PullRequestStateUnknown, ReviewState: domain.ReviewStateApproved,
		Mergeability: domain.MergeabilityMergeable, Stale: true,
	})
	if err == nil {
		t.Fatal("expected review fetch error")
	}
	if got.State != domain.PullRequestStateOpen || !got.Draft || got.Mergeability != domain.MergeabilityConflicting ||
		got.NodeID != "PR_7" || got.Author != "octocat" || len(got.Assignees) != 1 ||
		got.ReviewState != domain.ReviewStateApproved || got.GitHubUpdatedAt == nil {
		t.Fatalf("partial core result=%+v", got)
	}
}

func TestFixturePreservesError(t *testing.T) {
	provider, _ := NewFixtureProvider("demo")
	got, err := provider.Fetch(
		context.Background(),
		domain.PullRequest{Number: 3, State: domain.PullRequestStateUnknown, Stale: true},
	)
	if err != nil || got.State != domain.PullRequestStateMerged || got.Stale {
		t.Fatalf("fixture got=%+v err=%v", got, err)
	}
}

func newReviewServer(t *testing.T, reviewsJSON string) *LiveProvider {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"number":7,"state":"open","draft":false,"mergeable":true,"node_id":"PR_7","user":{"login":"octocat"},` +
					`"updated_at":"2026-01-01T00:00:00Z"}`,
			),
		)
	})
	mux.HandleFunc("/repos/acme/api/pulls/7/reviews", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reviewsJSON))
	})
	mux.HandleFunc("/repos/acme/api/pulls/7/requested_reviewers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"users":[],"teams":[]}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := gh.NewClient(nil)
	base, _ := client.BaseURL.Parse(server.URL + "/")
	client.BaseURL = base
	return &LiveProvider{client: client}
}

func TestLiveProviderIgnoresNonDecisionReviews(t *testing.T) {
	cases := []struct {
		name    string
		reviews string
		want    domain.ReviewState
	}{
		{
			name:    "comment after approval keeps approved",
			reviews: `[{"state":"APPROVED","user":{"login":"reviewer"}},{"state":"COMMENTED","user":{"login":"reviewer"}}]`,
			want:    domain.ReviewStateApproved,
		},
		{
			name:    "dismissed review does not request changes",
			reviews: `[{"state":"DISMISSED","user":{"login":"reviewer"}}]`,
			want:    domain.ReviewStateNone,
		},
		{
			name:    "pending review is ignored",
			reviews: `[{"state":"APPROVED","user":{"login":"reviewer"}},{"state":"PENDING","user":{"login":"reviewer"}}]`,
			want:    domain.ReviewStateApproved,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := newReviewServer(t, tc.reviews)
			got, err := provider.Fetch(
				context.Background(),
				domain.PullRequest{Owner: "acme", Repository: "api", Number: 7},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReviewState != tc.want {
				t.Fatalf("review state = %q, want %q", got.ReviewState, tc.want)
			}
		})
	}
}

func TestLiveProviderTreatsTerminalPullRequestsAsSuccessfulWithoutReviewMetadata(t *testing.T) {
	for name, state := range map[string]struct {
		state  string
		merged bool
	}{
		"closed": {state: "closed"},
		"merged": {state: "closed", merged: true},
	} {
		t.Run(name, func(t *testing.T) {
			var metadataRequests atomic.Int32
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"number":7,"state":%q,"merged":%t,"draft":false,`+
					`"mergeable":true,"node_id":"PR_7","user":{"login":"octocat"},`+
					`"updated_at":"2026-01-01T00:00:00Z"}`, state.state, state.merged)
			})
			metadataHandler := func(w http.ResponseWriter, _ *http.Request) {
				metadataRequests.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}
			mux.HandleFunc("/repos/acme/api/pulls/7/reviews", metadataHandler)
			mux.HandleFunc("/repos/acme/api/pulls/7/requested_reviewers", metadataHandler)
			server := httptest.NewServer(mux)
			defer server.Close()
			client := gh.NewClient(nil)
			base, _ := client.BaseURL.Parse(server.URL + "/")
			client.BaseURL = base
			provider := &LiveProvider{client: client}

			got, err := provider.Fetch(context.Background(), domain.PullRequest{
				Owner: "acme", Repository: "api", Number: 7,
			})
			if err != nil {
				t.Fatalf("terminal fetch=%+v err=%v", got, err)
			}
			if state.merged && got.State != domain.PullRequestStateMerged {
				t.Fatalf("merged state=%q", got.State)
			}
			if !state.merged && got.State != domain.PullRequestStateClosed {
				t.Fatalf("closed state=%q", got.State)
			}
			if got.Stale || got.SyncError != "" || got.LastSyncedAt == nil {
				t.Fatalf("terminal result=%+v", got)
			}
			if metadataRequests.Load() != 0 {
				t.Fatalf("terminal pull request fetched metadata %d times", metadataRequests.Load())
			}
		})
	}
}

func TestFixtureRejectsValuesOutsideTheSchema(t *testing.T) {
	write := func(body string) string {
		path := filepath.Join(t.TempDir(), "fixture.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	valid := `{"https://github.com/acme/api/pull/42":{"state":"open","review_state":"approved",` +
		`"mergeability":"mergeable"}}`
	if _, err := NewFixtureProvider(write(valid)); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
	cases := map[string]string{
		"missing state": `{"https://github.com/acme/api/pull/42":{"review_state":"approved",` +
			`"mergeability":"mergeable"}}`,
		"typo in review_state": `{"https://github.com/acme/api/pull/42":{"state":"open",` +
			`"review_state":"aproved","mergeability":"mergeable"}}`,
		"typo in mergeability": `{"https://github.com/acme/api/pull/42":{"state":"open","review_state":"approved",` +
			`"mergeability":"merged"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewFixtureProvider(write(body)); err == nil {
				t.Fatal("expected the fixture to be rejected")
			}
		})
	}
}

func TestLiveProviderBoundsRequestsAndKeepsSyncing(t *testing.T) {
	provider, err := NewLiveProvider(context.Background(), LiveProviderOptions{Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	if timeout := provider.client.Client().Timeout; timeout == 0 {
		t.Fatal("the GitHub HTTP client has no timeout; an unresponsive endpoint would stall the sync")
	}

	// A stalled endpoint must fail that pull request rather than hang forever, so
	// the caller can record the failure and move on to the next one.
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-block
	}))
	defer func() {
		close(block)
		server.Close()
	}()
	stalled := gh.NewClient(&http.Client{Timeout: 100 * time.Millisecond})
	base, _ := stalled.BaseURL.Parse(server.URL + "/")
	stalled.BaseURL = base
	slow := &LiveProvider{client: stalled}
	if _, err := slow.Fetch(
		context.Background(),
		domain.PullRequest{Owner: "acme", Repository: "api", Number: 7},
	); err == nil {
		t.Fatal("expected the stalled request to fail")
	}

	// The next pull request still syncs through a healthy endpoint.
	healthy := newReviewServer(t, `[{"state":"APPROVED","user":{"login":"reviewer"}}]`)
	got, err := healthy.Fetch(context.Background(), domain.PullRequest{Owner: "acme", Repository: "api", Number: 7})
	if err != nil || got.Stale {
		t.Fatalf("healthy fetch after a stalled one: stale=%v err=%v", got.Stale, err)
	}
}

func TestLiveProviderFollowsReviewPagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"number":7,"state":"open","draft":false,"mergeable":true,"node_id":"PR_7","user":{"login":"octocat"},` +
					`"updated_at":"2026-01-01T00:00:00Z"}`,
			),
		)
	})
	var reviewPages int
	mux.HandleFunc("/repos/acme/api/pulls/7/reviews", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reviewPages++
		if r.URL.Query().Get("page") == "" {
			// The decisive review lives on the second page.
			w.Header().Set("Link", `<`+r.URL.Path+`?page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"state":"COMMENTED","user":{"login":"reviewer"}}]`))
			return
		}
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
	if reviewPages != 2 {
		t.Fatalf("requested %d review pages, want 2", reviewPages)
	}
	if got.ReviewState != domain.ReviewStateChangesRequested {
		t.Fatalf("review state = %q, want changes_requested from the second page", got.ReviewState)
	}
}
