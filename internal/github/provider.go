package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	gh "github.com/google/go-github/v80/github"
	"golang.org/x/oauth2"
)

type Provider interface {
	Fetch(context.Context, domain.PullRequest) (domain.PullRequest, error)
}

type LiveProvider struct{ client *gh.Client }

func NewLiveProvider(ctx context.Context) (*LiveProvider, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GH_TOKEN"))
	}
	if token == "" {
		command := exec.CommandContext(ctx, "gh", "auth", "token")
		output, err := command.Output()
		if err != nil {
			return nil, fmt.Errorf("GitHub authentication is unavailable; set GITHUB_TOKEN or run gh auth login")
		}
		token = strings.TrimSpace(string(output))
	}
	// oauth2.NewClient returns a client with no timeout, so an unresponsive
	// endpoint would stall the whole sync instead of failing that one PR.
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	httpClient.Timeout = 30 * time.Second
	client := gh.NewClient(httpClient)
	client.UserAgent = "prx/0.1"
	return &LiveProvider{client: client}, nil
}

func (p *LiveProvider) Fetch(ctx context.Context, current domain.PullRequest) (domain.PullRequest, error) {
	value, _, err := p.client.PullRequests.Get(ctx, current.Owner, current.Repository, int(current.Number))
	if err != nil {
		return current, fmt.Errorf("fetch pull request: %w", err)
	}
	reviews, _, err := p.client.PullRequests.ListReviews(ctx, current.Owner, current.Repository, int(current.Number), &gh.ListOptions{PerPage: 100})
	if err != nil {
		return current, fmt.Errorf("fetch reviews: %w", err)
	}
	requested, _, err := p.client.PullRequests.ListReviewers(ctx, current.Owner, current.Repository, int(current.Number), &gh.ListOptions{PerPage: 100})
	if err != nil {
		return current, fmt.Errorf("fetch requested reviewers: %w", err)
	}
	reviewState := "none"
	latest := map[string]string{}
	for _, review := range reviews {
		state := strings.ToUpper(review.GetState())
		if state != "APPROVED" && state != "CHANGES_REQUESTED" {
			continue
		}
		latest[review.GetUser().GetLogin()] = state
	}
	for _, state := range latest {
		if state == "CHANGES_REQUESTED" {
			reviewState = "changes_requested"
			break
		}
		if state == "APPROVED" {
			reviewState = "approved"
		}
	}
	if reviewState == "none" && (len(requested.Users) > 0 || len(requested.Teams) > 0) {
		reviewState = "required"
	}
	state := value.GetState()
	if value.GetMerged() {
		state = "merged"
	}
	mergeability := "unknown"
	if value.Mergeable != nil {
		if value.GetMergeable() {
			mergeability = "mergeable"
		} else {
			mergeability = "conflicting"
		}
	}
	assignees := make([]string, 0, len(value.Assignees))
	for _, assignee := range value.Assignees {
		assignees = append(assignees, assignee.GetLogin())
	}
	now := time.Now().UTC()
	updated := value.GetUpdatedAt().UTC()
	current.NodeID = value.GetNodeID()
	current.Author = value.GetUser().GetLogin()
	current.Assignees = assignees
	current.State = state
	current.Draft = value.GetDraft()
	current.ReviewState = reviewState
	current.Mergeability = mergeability
	current.GitHubUpdatedAt = &updated
	current.LastSyncedAt = &now
	current.SyncError = ""
	current.Stale = false
	return current, nil
}

type FixtureProvider struct{ values map[string]fixture }

type fixture struct {
	State        string   `json:"state"`
	Draft        bool     `json:"draft"`
	ReviewState  string   `json:"review_state"`
	Mergeability string   `json:"mergeability"`
	Author       string   `json:"author"`
	Assignees    []string `json:"assignees"`
	Error        string   `json:"error"`
}

// The persisted columns carry CHECK constraints, so a typo in a hand-written
// fixture has to fail while reading the file rather than as a raw SQLite error
// halfway through a sync.
var fixtureFields = []struct {
	name    string
	value   func(fixture) string
	allowed []string
}{
	{"state", func(f fixture) string { return f.State }, []string{"open", "closed", "merged", "unknown"}},
	{"review_state", func(f fixture) string { return f.ReviewState }, []string{"none", "required", "approved", "changes_requested", "unknown"}},
	{"mergeability", func(f fixture) string { return f.Mergeability }, []string{"mergeable", "conflicting", "unknown"}},
}

func NewFixtureProvider(path string) (*FixtureProvider, error) {
	if path == "demo" {
		return &FixtureProvider{values: map[string]fixture{}}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]fixture{}
	if err := json.Unmarshal(body, &values); err != nil {
		return nil, fmt.Errorf("decode GitHub fixture: %w", err)
	}
	urls := make([]string, 0, len(values))
	for url := range values {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	for _, url := range urls {
		value := values[url]
		if value.Error != "" {
			continue
		}
		for _, field := range fixtureFields {
			current := field.value(value)
			if !slices.Contains(field.allowed, current) {
				return nil, fmt.Errorf("GitHub fixture %s: %s is %q; expected one of %s",
					url, field.name, current, strings.Join(field.allowed, ", "))
			}
		}
	}
	return &FixtureProvider{values: values}, nil
}

func (p *FixtureProvider) Fetch(_ context.Context, current domain.PullRequest) (domain.PullRequest, error) {
	value, ok := p.values[current.URL]
	if !ok {
		states := []fixture{
			{State: "open", ReviewState: "required", Mergeability: "mergeable", Author: "octocat"},
			{State: "open", ReviewState: "approved", Mergeability: "mergeable", Author: "hubot"},
			{State: "open", ReviewState: "changes_requested", Mergeability: "conflicting", Author: "monalisa"},
			{State: "merged", ReviewState: "approved", Mergeability: "mergeable", Author: "octocat"},
		}
		value = states[int(current.Number)%len(states)]
	}
	if value.Error != "" {
		return current, fmt.Errorf("%s", value.Error)
	}
	now := time.Now().UTC()
	current.State = value.State
	current.Draft = value.Draft
	current.ReviewState = value.ReviewState
	current.Mergeability = value.Mergeability
	current.Author = value.Author
	current.Assignees = value.Assignees
	current.NodeID = "fixture:" + strconv.FormatInt(current.Number, 10)
	current.GitHubUpdatedAt = &now
	current.LastSyncedAt = &now
	current.SyncError = ""
	current.Stale = false
	return current, nil
}

func ParsePullRequestURL(value string) (owner, repository string, number int64, canonical string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", "", 0, "", fmt.Errorf("expected an https://github.com/OWNER/REPO/pull/NUMBER URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" || parts[0] == "" || parts[1] == "" {
		return "", "", 0, "", fmt.Errorf("expected an https://github.com/OWNER/REPO/pull/NUMBER URL")
	}
	number, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil || number < 1 {
		return "", "", 0, "", fmt.Errorf("pull request number must be positive")
	}
	canonical = fmt.Sprintf("https://github.com/%s/%s/pull/%d", parts[0], parts[1], number)
	return parts[0], parts[1], number, canonical, nil
}
