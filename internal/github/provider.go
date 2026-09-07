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
	// Fetch は最新の PR レコードを返す。後続のメタデータ取得が失敗した場合、
	// 実装は部分的に更新したレコードをエラーと共に返してよい。呼び出し元は
	// その部分状態を保存し、扱いを自分で判断する。
	Fetch(ctx context.Context, current domain.PullRequest) (domain.PullRequest, error)
}

type BatchResult struct {
	PullRequests map[string]domain.PullRequest
	// PartialPullRequests は、項目単位のエラーで更新が止まる直前まで判明していた
	// 最新のフィールドを保持する。
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

// NewLiveProvider は解決済みの資格情報からプロバイダを構築する。資格情報の探索は
// Resolver の担当で、どの候補も設定されたホストの範囲に留まるようにしている。
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
		// 終了した pull request ではレビュー中の判定に使わないので、追加のリクエストを
		// かけずに前回同期の値を落とす。
		current.ReviewRequestPending = false
		current.ChangesRequestedAt = nil
		current.LastPushedAt = nil
		return markPullRequestSynced(current), nil
	}
	review, err := p.fetchReviewSummary(ctx, current)
	if err != nil {
		return current, err
	}
	// レビューの結果はコミット取得より先に反映する。項目単位の失敗が無関係な
	// 成功を巻き添えにしないため。docs/design/github-sync.md を参照。
	current.State = state
	current.ReviewState = review.state
	current.ReviewRequestPending = review.requestPending
	current.ChangesRequestedAt = review.changesRequestedAt
	pushedAt, err := p.lastPushedAt(ctx, current)
	if err != nil {
		// LastPushedAt は前回成功した値のまま残す。
		return current, err
	}
	current.LastPushedAt = pushedAt
	return markPullRequestSynced(current), nil
}

// reviewSummary は REST 経路が組み立てるレビューの要約。
type reviewSummary struct {
	state              domain.ReviewState
	requestPending     bool
	changesRequestedAt *time.Time
}

// fetchReviewSummary はレビューとレビュー依頼を全ページたどって要約する。
func (p *LiveProvider) fetchReviewSummary(
	ctx context.Context,
	current domain.PullRequest,
) (reviewSummary, error) {
	reviews, err := allPages(
		ctx,
		func(ctx context.Context, options *gh.ListOptions) ([]*gh.PullRequestReview, *gh.Response, error) {
			return p.client.PullRequests.ListReviews(
				ctx, current.Owner, current.Repository, int(current.Number), options,
			)
		},
	)
	if err != nil {
		return reviewSummary{}, wrapProviderError("fetch reviews", err, nil)
	}
	requestedPages, err := allPages(
		ctx,
		func(ctx context.Context, options *gh.ListOptions) ([]*gh.Reviewers, *gh.Response, error) {
			value, response, err := p.client.PullRequests.ListReviewers(
				ctx, current.Owner, current.Repository, int(current.Number), options,
			)
			if err != nil {
				return nil, response, err
			}
			return []*gh.Reviewers{value}, response, nil
		},
	)
	if err != nil {
		return reviewSummary{}, wrapProviderError("fetch requested reviewers", err, nil)
	}
	requested := &gh.Reviewers{}
	for _, page := range requestedPages {
		requested.Users = append(requested.Users, page.Users...)
		requested.Teams = append(requested.Teams, page.Teams...)
	}
	latest := map[string]restReview{}
	for _, review := range reviews {
		state := strings.ToUpper(review.GetState())
		if state != "APPROVED" && state != "CHANGES_REQUESTED" {
			continue
		}
		latest[review.GetUser().GetLogin()] = restReview{
			state: state, submittedAt: review.GetSubmittedAt().UTC(),
		}
	}
	reviewState := domain.ReviewStateNone
	for _, review := range latest {
		if review.state == "CHANGES_REQUESTED" {
			reviewState = domain.ReviewStateChangesRequested
			break
		}
		if review.state == "APPROVED" {
			reviewState = domain.ReviewStateApproved
		}
	}
	pending := len(requested.Users) > 0 || len(requested.Teams) > 0
	if reviewState == domain.ReviewStateNone && pending {
		reviewState = domain.ReviewStateRequired
	}
	return reviewSummary{
		state:              reviewState,
		requestPending:     pending,
		changesRequestedAt: latestChangesRequestedAtFromREST(latest),
	}, nil
}

// restReview はレビュアーごとの最新レビューを、状態と提出時刻の組で保つ。
type restReview struct {
	state       string
	submittedAt time.Time
}

// latestChangesRequestedAtFromREST は変更要求レビューだけを見る。GraphQL 経路と
// 同じく、コメントだけのレビューでは時刻を返さない。
func latestChangesRequestedAtFromREST(latest map[string]restReview) *time.Time {
	var result *time.Time
	for _, review := range latest {
		if review.state != "CHANGES_REQUESTED" || review.submittedAt.IsZero() {
			continue
		}
		if result == nil || review.submittedAt.After(*result) {
			submitted := review.submittedAt
			result = &submitted
		}
	}
	return result
}

// lastPushedAt は最新コミットの時刻を返す。REST の commit には push 時刻がないので
// committer の日時を使う。GraphQL 経路の pushedAt が null のときと同じ値になる。
func (p *LiveProvider) lastPushedAt(
	ctx context.Context,
	current domain.PullRequest,
) (*time.Time, error) {
	// 使うのは最後の 1 件だけなので、1 件ずつ引いて最終ページ番号を得てから
	// そのページだけを取り直す。全ページを取得して捨てるとリクエスト量が増える。
	page, response, err := p.client.PullRequests.ListCommits(
		ctx, current.Owner, current.Repository, int(current.Number), &gh.ListOptions{PerPage: 1},
	)
	if err != nil {
		return nil, wrapProviderError("fetch commits", err, response)
	}
	if response != nil && response.LastPage != 0 {
		page, response, err = p.client.PullRequests.ListCommits(
			ctx, current.Owner, current.Repository, int(current.Number),
			&gh.ListOptions{PerPage: 1, Page: response.LastPage},
		)
		if err != nil {
			return nil, wrapProviderError("fetch commits", err, response)
		}
	}
	if len(page) == 0 {
		return nil, nil
	}
	pushed := page[len(page)-1].GetCommit().GetCommitter().GetDate().UTC()
	if pushed.IsZero() {
		return nil, nil
	}
	return &pushed, nil
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

// allPages は GitHub のリストエンドポイントの全ページをたどる。1 ページ目で
// 止めると長寿命の PR を黙って切り捨て、その結果のレビュー状態が最新の
// ものとして保存されてしまう。
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

// Fixture はフィクスチャファイルが URL ごとに記録する PR の状態 1 件。
type Fixture struct {
	State        domain.PullRequestState `json:"state"`
	Draft        bool                    `json:"draft"`
	ReviewState  domain.ReviewState      `json:"review_state"`
	Mergeability domain.Mergeability     `json:"mergeability"`
	Author       string                  `json:"author"`
	Assignees    []string                `json:"assignees"`
	Error        string                  `json:"error"`
	// ReviewRequestPending 以下はレビュー中の判定に使う。CHECK 制約のある列ではない
	// ので fixtureFields の検証対象には含めない。
	ReviewRequestPending bool       `json:"review_request_pending"`
	ChangesRequestedAt   *time.Time `json:"changes_requested_at"`
	LastPushedAt         *time.Time `json:"last_pushed_at"`
}

// 保存先のカラムには CHECK 制約があるため、手書きフィクスチャの誤記は同期の
// 途中で生の SQLite エラーになるのではなく、ファイル読み込み時に失敗させる。
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
		// 4 パターンで実装済み・承認済み・レビュー中・マージ済みを 1 つずつ出す。
		// デモは WebUI の見た目を確かめる唯一の手段なので、色の軸を網羅させる。
		reviewedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		pushedAt := reviewedAt.Add(time.Hour)
		states := []Fixture{
			{
				State:        domain.PullRequestStateOpen,
				ReviewState:  domain.ReviewStateNone,
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
				State:              domain.PullRequestStateOpen,
				ReviewState:        domain.ReviewStateChangesRequested,
				Mergeability:       domain.MergeabilityConflicting,
				Author:             "monalisa",
				ChangesRequestedAt: &reviewedAt,
				LastPushedAt:       &pushedAt,
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
	current.ReviewRequestPending = value.ReviewRequestPending
	current.ChangesRequestedAt = value.ChangesRequestedAt
	current.LastPushedAt = value.LastPushedAt
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
