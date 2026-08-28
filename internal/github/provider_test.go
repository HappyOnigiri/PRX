package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func newReviewServer(t *testing.T, reviewsJSON string) *LiveProvider {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/api/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":7,"state":"open","draft":false,"mergeable":true,"node_id":"PR_7","user":{"login":"octocat"},"updated_at":"2026-01-01T00:00:00Z"}`))
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
		want    string
	}{
		{
			name:    "comment after approval keeps approved",
			reviews: `[{"state":"APPROVED","user":{"login":"reviewer"}},{"state":"COMMENTED","user":{"login":"reviewer"}}]`,
			want:    "approved",
		},
		{
			name:    "dismissed review does not request changes",
			reviews: `[{"state":"DISMISSED","user":{"login":"reviewer"}}]`,
			want:    "none",
		},
		{
			name:    "pending review is ignored",
			reviews: `[{"state":"APPROVED","user":{"login":"reviewer"}},{"state":"PENDING","user":{"login":"reviewer"}}]`,
			want:    "approved",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := newReviewServer(t, tc.reviews)
			got, err := provider.Fetch(context.Background(), domain.PullRequest{Owner: "acme", Repository: "api", Number: 7})
			if err != nil {
				t.Fatal(err)
			}
			if got.ReviewState != tc.want {
				t.Fatalf("review state = %q, want %q", got.ReviewState, tc.want)
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
	valid := `{"https://github.com/acme/api/pull/42":{"state":"open","review_state":"approved","mergeability":"mergeable"}}`
	if _, err := NewFixtureProvider(write(valid)); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
	cases := map[string]string{
		"missing state":        `{"https://github.com/acme/api/pull/42":{"review_state":"approved","mergeability":"mergeable"}}`,
		"typo in review_state": `{"https://github.com/acme/api/pull/42":{"state":"open","review_state":"aproved","mergeability":"mergeable"}}`,
		"typo in mergeability": `{"https://github.com/acme/api/pull/42":{"state":"open","review_state":"approved","mergeability":"merged"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewFixtureProvider(write(body)); err == nil {
				t.Fatal("expected the fixture to be rejected")
			}
		})
	}
}
