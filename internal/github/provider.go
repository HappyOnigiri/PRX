package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	gh "github.com/google/go-github/v80/github"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

type Provider interface {
	// Fetch returns the newest pull-request record. Implementations may return a
	// partially updated record together with an error when the pull-request body
	// was fetched but a later metadata request failed; callers persist that
	// partial state and decide whether the known state still needs attention.
	Fetch(ctx context.Context, current domain.PullRequest) (domain.PullRequest, error)
}

type BatchResult struct {
	PullRequests map[string]domain.PullRequest
	// PartialPullRequests contains the newest fields known before an item-level
	// error stopped the rest of its refresh.
	PartialPullRequests map[string]domain.PullRequest
	Errors              map[string]error
}

type BatchProvider interface {
	FetchBatch(ctx context.Context, current []domain.PullRequest) (BatchResult, error)
}

type LiveProvider struct {
	client     *gh.Client
	httpClient *http.Client
	token      string
	graphqlURL string
}

type LiveProviderOptions struct {
	Token      string
	APIURL     string
	UploadURL  string
	GraphQLURL string
	HTTPClient *http.Client
}

// NewLiveProvider constructs a provider from already-resolved credentials.
// Credential discovery belongs to Resolver so every candidate remains scoped
// to its configured host.
func NewLiveProvider(ctx context.Context, options LiveProviderOptions) (*LiveProvider, error) {
	return NewConfiguredLiveProvider(
		ctx, options.Token, options.APIURL, options.UploadURL, options.GraphQLURL, options.HTTPClient,
	)
}

func NewConfiguredLiveProvider(
	ctx context.Context,
	token, apiURL, uploadURL, graphqlURL string,
	httpClient *http.Client,
) (*LiveProvider, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, unavailableError("GitHub credential is empty")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	} else {
		clientCopy := *httpClient
		httpClient = &clientCopy
		if httpClient.Timeout == 0 {
			httpClient.Timeout = 30 * time.Second
		}
	}
	redirectPolicy := httpClient.CheckRedirect
	httpClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if err := rejectCrossOriginRedirect(request, via); err != nil {
			return err
		}
		if redirectPolicy != nil {
			return redirectPolicy(request, via)
		}
		return nil
	}
	client := gh.NewClient(httpClient).WithAuthToken(token)
	if apiURL != "" || uploadURL != "" {
		if apiURL == "" {
			apiURL = "https://api.github.com/"
		}
		if uploadURL == "" {
			uploadURL = "https://uploads.github.com/"
		}
		var err error
		client, err = client.WithEnterpriseURLs(apiURL, uploadURL)
		if err != nil {
			return nil, fmt.Errorf("configure GitHub URLs: %w", err)
		}
	}
	client.UserAgent = "prx/0.1"
	if graphqlURL == "" {
		graphqlURL = defaultGraphQLURL(apiURL)
	}
	return &LiveProvider{
		client: client, httpClient: httpClient, token: token, graphqlURL: graphqlURL,
	}, nil
}

func defaultGraphQLURL(apiURL string) string {
	if apiURL == "" {
		return "https://api.github.com/graphql"
	}
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Host == "" {
		return "https://api.github.com/graphql"
	}
	if strings.EqualFold(parsed.Host, "api.github.com") {
		parsed.Path = "/graphql"
	} else {
		parsed.Path = "/api/graphql"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func rejectCrossOriginRedirect(request *http.Request, via []*http.Request) error {
	if len(via) == 0 || sameOrigin(via[len(via)-1].URL, request.URL) {
		return nil
	}
	return fmt.Errorf("refusing redirect to a different origin")
}

func sameOrigin(first, second *url.URL) bool {
	if first == nil || second == nil {
		return false
	}
	return first.Scheme == second.Scheme && strings.EqualFold(first.Host, second.Host)
}

func (p *LiveProvider) Fetch(ctx context.Context, current domain.PullRequest) (domain.PullRequest, error) {
	value, response, err := p.client.PullRequests.Get(ctx, current.Owner, current.Repository, int(current.Number))
	if err != nil {
		return current, wrapProviderError("fetch pull request", err, response)
	}
	state := domain.PullRequestState(value.GetState())
	if value.GetMerged() {
		state = domain.PullRequestStateMerged
	}
	mergeability := domain.MergeabilityUnknown
	if value.Mergeable != nil {
		if value.GetMergeable() {
			mergeability = domain.MergeabilityMergeable
		} else {
			mergeability = domain.MergeabilityConflicting
		}
	}
	assignees := make([]string, 0, len(value.Assignees))
	for _, assignee := range value.Assignees {
		assignees = append(assignees, assignee.GetLogin())
	}
	updated := value.GetUpdatedAt().UTC()
	current.NodeID = value.GetNodeID()
	current.Author = value.GetUser().GetLogin()
	current.Assignees = assignees
	current.State = state
	current.Draft = value.GetDraft()
	current.Mergeability = mergeability
	current.GitHubUpdatedAt = &updated
	if state == domain.PullRequestStateClosed || state == domain.PullRequestStateMerged {
		return markPullRequestSynced(current), nil
	}
	reviews, err := allPages(
		ctx,
		func(ctx context.Context, options *gh.ListOptions) ([]*gh.PullRequestReview, *gh.Response, error) {
			return p.client.PullRequests.ListReviews(
				ctx,
				current.Owner,
				current.Repository,
				int(current.Number),
				options,
			)
		},
	)
	if err != nil {
		return current, wrapProviderError("fetch reviews", err, nil)
	}
	requestedPages, err := allPages(
		ctx,
		func(ctx context.Context, options *gh.ListOptions) ([]*gh.Reviewers, *gh.Response, error) {
			value, response, err := p.client.PullRequests.ListReviewers(
				ctx,
				current.Owner,
				current.Repository,
				int(current.Number),
				options,
			)
			if err != nil {
				return nil, response, err
			}
			return []*gh.Reviewers{value}, response, nil
		},
	)
	if err != nil {
		return current, wrapProviderError("fetch requested reviewers", err, nil)
	}
	requested := &gh.Reviewers{}
	for _, page := range requestedPages {
		requested.Users = append(requested.Users, page.Users...)
		requested.Teams = append(requested.Teams, page.Teams...)
	}
	reviewState := domain.ReviewStateNone
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
			reviewState = domain.ReviewStateChangesRequested
			break
		}
		if state == "APPROVED" {
			reviewState = domain.ReviewStateApproved
		}
	}
	if reviewState == domain.ReviewStateNone && (len(requested.Users) > 0 || len(requested.Teams) > 0) {
		reviewState = domain.ReviewStateRequired
	}
	current.State = state
	current.ReviewState = reviewState
	return markPullRequestSynced(current), nil
}

func markPullRequestSynced(value domain.PullRequest) domain.PullRequest {
	now := time.Now().UTC()
	value.LastSyncedAt = &now
	value.SyncError = ""
	value.Stale = false
	return value
}

func (p *LiveProvider) Probe(ctx context.Context, owner, repository string) error {
	_, response, err := p.client.PullRequests.List(ctx, owner, repository, &gh.PullRequestListOptions{
		State:       "all",
		ListOptions: gh.ListOptions{PerPage: 1},
	})
	if err != nil {
		return wrapProviderError("probe pull request access", err, response)
	}
	return nil
}

// allPages walks every page of a GitHub list endpoint. Stopping at the first
// page silently truncates long-lived pull requests, and the resulting review
// state is stored as if it were fresh.
func allPages[T any](
	ctx context.Context,
	fetch func(context.Context, *gh.ListOptions) ([]T, *gh.Response, error),
) ([]T, error) {
	options := &gh.ListOptions{PerPage: 100}
	var all []T
	for {
		page, response, err := fetch(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if response == nil || response.NextPage == 0 {
			return all, nil
		}
		options.Page = response.NextPage
	}
}

type FixtureProvider struct{ values map[string]Fixture }

// Fixture is one pull-request state a fixture file records for a URL.
type Fixture struct {
	State        domain.PullRequestState `json:"state"`
	Draft        bool                    `json:"draft"`
	ReviewState  domain.ReviewState      `json:"review_state"`
	Mergeability domain.Mergeability     `json:"mergeability"`
	Author       string                  `json:"author"`
	Assignees    []string                `json:"assignees"`
	Error        string                  `json:"error"`
}

// The persisted columns carry CHECK constraints, so a typo in a hand-written
// fixture has to fail while reading the file rather than as a raw SQLite error
// halfway through a sync.
var fixtureFields = []struct {
	name    string
	value   func(Fixture) string
	allowed []string
}{
	{
		"state",
		func(f Fixture) string { return string(f.State) },
		[]string{
			string(domain.PullRequestStateOpen),
			string(domain.PullRequestStateClosed),
			string(domain.PullRequestStateMerged),
			string(domain.PullRequestStateUnknown),
		},
	},
	{
		"review_state",
		func(f Fixture) string { return string(f.ReviewState) },
		[]string{
			string(domain.ReviewStateNone),
			string(domain.ReviewStateRequired),
			string(domain.ReviewStateApproved),
			string(domain.ReviewStateChangesRequested),
			string(domain.ReviewStateUnknown),
		},
	},
	{
		"mergeability",
		func(f Fixture) string { return string(f.Mergeability) },
		[]string{
			string(domain.MergeabilityMergeable),
			string(domain.MergeabilityConflicting),
			string(domain.MergeabilityUnknown),
		},
	},
}

func NewFixtureProvider(path string) (*FixtureProvider, error) {
	if path == "demo" {
		return &FixtureProvider{values: map[string]Fixture{}}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]Fixture{}
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

func (p *FixtureProvider) Fetch(ctx context.Context, current domain.PullRequest) (domain.PullRequest, error) {
	value, ok := p.values[current.URL]
	if !ok {
		states := []Fixture{
			{
				State:        domain.PullRequestStateOpen,
				ReviewState:  domain.ReviewStateRequired,
				Mergeability: domain.MergeabilityMergeable,
				Author:       "octocat",
			},
			{
				State:        domain.PullRequestStateOpen,
				ReviewState:  domain.ReviewStateApproved,
				Mergeability: domain.MergeabilityMergeable,
				Author:       "hubot",
			},
			{
				State:        domain.PullRequestStateOpen,
				ReviewState:  domain.ReviewStateChangesRequested,
				Mergeability: domain.MergeabilityConflicting,
				Author:       "monalisa",
			},
			{
				State:        domain.PullRequestStateMerged,
				ReviewState:  domain.ReviewStateApproved,
				Mergeability: domain.MergeabilityMergeable,
				Author:       "octocat",
			},
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
	_, owner, repository, number, canonical, err = ParsePullRequestURLDetails(value)
	return owner, repository, number, canonical, err
}

func ParsePullRequestURLDetails(
	value string,
) (host, owner, repository string, number int64, canonical string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		parsed.Path == "" ||
		strings.HasSuffix(parsed.Path, "/") {
		return "", "", "", 0, "", fmt.Errorf("expected an https://HOST/OWNER/REPO/pull/NUMBER URL")
	}
	parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" || parts[0] == "" || parts[1] == "" {
		return "", "", "", 0, "", fmt.Errorf("expected an https://HOST/OWNER/REPO/pull/NUMBER URL")
	}
	number, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil || number < 1 {
		return "", "", "", 0, "", fmt.Errorf("pull request number must be positive")
	}
	host = strings.ToLower(parsed.Host)
	canonical = fmt.Sprintf("https://%s/%s/%s/pull/%d", host, parts[0], parts[1], number)
	return host, parts[0], parts[1], number, canonical, nil
}
