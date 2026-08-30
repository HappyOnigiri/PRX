package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
)

const graphQLBatchSize = 20

type graphQLError struct {
	Message    string         `json:"message"`
	Path       []any          `json:"path"`
	Extensions map[string]any `json:"extensions"`
}

type graphQLResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []graphQLError             `json:"errors"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphQLPullRequest struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Merged    bool   `json:"merged"`
	IsDraft   bool   `json:"isDraft"`
	Mergeable string `json:"mergeable"`
	UpdatedAt string `json:"updatedAt"`
	Author    *struct {
		Login string `json:"login"`
	} `json:"author"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
		PageInfo graphQLPageInfo `json:"pageInfo"`
	} `json:"assignees"`
	LatestReviews struct {
		Nodes []struct {
			State  string `json:"state"`
			Author *struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"nodes"`
		PageInfo graphQLPageInfo `json:"pageInfo"`
	} `json:"latestReviews"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer map[string]any `json:"requestedReviewer"`
		} `json:"nodes"`
		PageInfo graphQLPageInfo `json:"pageInfo"`
	} `json:"reviewRequests"`
}

type graphQLItem struct {
	current         domain.PullRequest
	repositoryAlias string
	pullAlias       string
}

func (p *LiveProvider) FetchBatch(
	ctx context.Context,
	current []domain.PullRequest,
) (BatchResult, error) {
	result := BatchResult{
		PullRequests: make(map[string]domain.PullRequest, len(current)),
		Errors:       make(map[string]error),
	}
	for start := 0; start < len(current); start += graphQLBatchSize {
		end := min(start+graphQLBatchSize, len(current))
		chunk, err := p.fetchGraphQLChunk(ctx, current[start:end])
		if err != nil {
			// A host may reject the GraphQL endpoint with any client or server
			// status, not just 404, while REST still works. Retrying the whole
			// set over REST is cheaper than losing the host entirely.
			if StatusCodeOf(err) >= http.StatusBadRequest {
				return p.fetchRESTBatch(ctx, current)
			}
			return result, err
		}
		for taskID, value := range chunk.PullRequests {
			result.PullRequests[taskID] = value
		}
		for taskID, itemErr := range chunk.Errors {
			result.Errors[taskID] = itemErr
		}
	}
	return result, nil
}

func (p *LiveProvider) fetchRESTBatch(
	ctx context.Context,
	current []domain.PullRequest,
) (BatchResult, error) {
	result := BatchResult{
		PullRequests: make(map[string]domain.PullRequest, len(current)),
		Errors:       make(map[string]error),
	}
	for _, value := range current {
		updated, err := p.Fetch(ctx, value)
		if err != nil {
			result.Errors[value.TaskID] = err
			continue
		}
		result.PullRequests[value.TaskID] = updated
	}
	return result, nil
}

func (p *LiveProvider) fetchGraphQLChunk(
	ctx context.Context,
	current []domain.PullRequest,
) (BatchResult, error) {
	query, variables, items := buildGraphQLQuery(current)
	var response graphQLResponse
	status, err := p.doGraphQL(ctx, query, variables, &response)
	if err != nil {
		return BatchResult{}, graphQLHTTPError(status, err)
	}
	result := BatchResult{
		PullRequests: make(map[string]domain.PullRequest, len(items)),
		Errors:       make(map[string]error),
	}
	itemErrors, globalError := mapGraphQLErrors(response.Errors, items)
	if globalError != nil {
		return result, globalError
	}
	for _, item := range items {
		if itemErr := itemErrors[item.current.TaskID]; itemErr != nil {
			result.Errors[item.current.TaskID] = itemErr
			continue
		}
		repositoryBody := response.Data[item.repositoryAlias]
		if len(repositoryBody) == 0 || bytes.Equal(repositoryBody, []byte("null")) {
			result.Errors[item.current.TaskID] = providerGraphQLError(
				ErrorClassNotFound,
				"repository was not found or is not visible",
			)
			continue
		}
		var repository map[string]json.RawMessage
		if err := json.Unmarshal(repositoryBody, &repository); err != nil {
			return result, fmt.Errorf("decode GraphQL repository: %w", err)
		}
		pullBody := repository[item.pullAlias]
		if len(pullBody) == 0 || bytes.Equal(pullBody, []byte("null")) {
			result.Errors[item.current.TaskID] = providerGraphQLError(
				ErrorClassOther,
				fmt.Sprintf(
					"pull request %s/%s#%d was not found",
					item.current.Owner,
					item.current.Repository,
					item.current.Number,
				),
			)
			continue
		}
		var pullRequest graphQLPullRequest
		if err := json.Unmarshal(pullBody, &pullRequest); err != nil {
			return result, fmt.Errorf("decode GraphQL pull request: %w", err)
		}
		if err := p.completeGraphQLConnections(ctx, &pullRequest); err != nil {
			result.Errors[item.current.TaskID] = err
			continue
		}
		updated, err := domainPullRequestFromGraphQL(item.current, pullRequest)
		if err != nil {
			result.Errors[item.current.TaskID] = err
			continue
		}
		result.PullRequests[item.current.TaskID] = updated
	}
	return result, nil
}

type graphQLConnection struct {
	name     string
	pageInfo *graphQLPageInfo
	fields   string
	append   func(json.RawMessage) error
}

func graphQLConnections(value *graphQLPullRequest) []graphQLConnection {
	return []graphQLConnection{
		{
			name: "assignees", pageInfo: &value.Assignees.PageInfo,
			fields: "nodes{login} pageInfo{hasNextPage endCursor}",
			append: func(body json.RawMessage) error {
				var page struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
					PageInfo graphQLPageInfo `json:"pageInfo"`
				}
				if err := json.Unmarshal(body, &page); err != nil {
					return err
				}
				value.Assignees.Nodes = append(value.Assignees.Nodes, page.Nodes...)
				value.Assignees.PageInfo = page.PageInfo
				return nil
			},
		},
		{
			name: "latestReviews", pageInfo: &value.LatestReviews.PageInfo,
			fields: "nodes{author{login} state} pageInfo{hasNextPage endCursor}",
			append: func(body json.RawMessage) error {
				var page struct {
					Nodes []struct {
						State  string `json:"state"`
						Author *struct {
							Login string `json:"login"`
						} `json:"author"`
					} `json:"nodes"`
					PageInfo graphQLPageInfo `json:"pageInfo"`
				}
				if err := json.Unmarshal(body, &page); err != nil {
					return err
				}
				value.LatestReviews.Nodes = append(value.LatestReviews.Nodes, page.Nodes...)
				value.LatestReviews.PageInfo = page.PageInfo
				return nil
			},
		},
		{
			name: "reviewRequests", pageInfo: &value.ReviewRequests.PageInfo,
			fields: "nodes{requestedReviewer{... on User{login} ... on Team{slug}}} " +
				"pageInfo{hasNextPage endCursor}",
			append: func(body json.RawMessage) error {
				var page struct {
					Nodes []struct {
						RequestedReviewer map[string]any `json:"requestedReviewer"`
					} `json:"nodes"`
					PageInfo graphQLPageInfo `json:"pageInfo"`
				}
				if err := json.Unmarshal(body, &page); err != nil {
					return err
				}
				value.ReviewRequests.Nodes = append(value.ReviewRequests.Nodes, page.Nodes...)
				value.ReviewRequests.PageInfo = page.PageInfo
				return nil
			},
		},
	}
}

func (p *LiveProvider) completeGraphQLConnections(
	ctx context.Context,
	value *graphQLPullRequest,
) error {
	connections := graphQLConnections(value)
	for _, connection := range connections {
		for connection.pageInfo.HasNextPage {
			cursor := connection.pageInfo.EndCursor
			if cursor == "" {
				return providerGraphQLError(ErrorClassOther, "GitHub returned an empty pagination cursor")
			}
			query := fmt.Sprintf(
				"query($id:ID!,$cursor:String!){node(id:$id){... on PullRequest{%s(first:100,after:$cursor){%s}}}}",
				connection.name,
				connection.fields,
			)
			var response struct {
				Data struct {
					Node map[string]json.RawMessage `json:"node"`
				} `json:"data"`
				Errors []graphQLError `json:"errors"`
			}
			status, err := p.doGraphQL(
				ctx,
				query,
				map[string]any{"id": value.ID, "cursor": cursor},
				&response,
			)
			if err != nil {
				return graphQLHTTPError(status, err)
			}
			if len(response.Errors) > 0 {
				return providerGraphQLError(classifyGraphQLError(response.Errors[0]), response.Errors[0].Message)
			}
			body := response.Data.Node[connection.name]
			if len(body) == 0 {
				return providerGraphQLError(ErrorClassOther, "GitHub omitted a paginated connection")
			}
			if err := connection.append(body); err != nil {
				return fmt.Errorf("decode GraphQL %s page: %w", connection.name, err)
			}
		}
	}
	return nil
}

func buildGraphQLQuery(current []domain.PullRequest) (string, map[string]any, []graphQLItem) {
	ordered := append([]domain.PullRequest(nil), current...)
	sort.SliceStable(ordered, func(i, j int) bool {
		first := strings.ToLower(ordered[i].Owner + "/" + ordered[i].Repository)
		second := strings.ToLower(ordered[j].Owner + "/" + ordered[j].Repository)
		if first != second {
			return first < second
		}
		return ordered[i].Number < ordered[j].Number
	})
	variables := make(map[string]any, len(ordered)*3)
	items := make([]graphQLItem, 0, len(ordered))
	repositories := make(map[string]int)
	repositoryItems := make(map[int][]graphQLItem)
	for index, value := range ordered {
		key := strings.ToLower(value.Owner + "/" + value.Repository)
		repositoryIndex, ok := repositories[key]
		if !ok {
			repositoryIndex = len(repositories)
			repositories[key] = repositoryIndex
			variables["owner"+strconv.Itoa(repositoryIndex)] = value.Owner
			variables["name"+strconv.Itoa(repositoryIndex)] = value.Repository
		}
		variables["number"+strconv.Itoa(index)] = value.Number
		item := graphQLItem{
			current: value, repositoryAlias: "r" + strconv.Itoa(repositoryIndex),
			pullAlias: "p" + strconv.Itoa(index),
		}
		items = append(items, item)
		repositoryItems[repositoryIndex] = append(repositoryItems[repositoryIndex], item)
	}
	var declarations, body strings.Builder
	declarations.WriteString("query(")
	for index := 0; index < len(repositories); index++ {
		if index > 0 {
			declarations.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&declarations, "$owner%d:String!,$name%d:String!", index, index)
	}
	for index := range ordered {
		if len(repositories) > 0 || index > 0 {
			declarations.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&declarations, "$number%d:Int!", index)
	}
	declarations.WriteString("){")
	for index := 0; index < len(repositories); index++ {
		_, _ = fmt.Fprintf(
			&body,
			"r%d:repository(owner:$owner%d,name:$name%d){",
			index,
			index,
			index,
		)
		for _, item := range repositoryItems[index] {
			itemIndex, _ := strconv.Atoi(strings.TrimPrefix(item.pullAlias, "p"))
			_, _ = fmt.Fprintf(&body, "%s:pullRequest(number:$number%d){%s}", item.pullAlias, itemIndex, graphQLFields)
		}
		body.WriteByte('}')
	}
	body.WriteByte('}')
	return declarations.String() + body.String(), variables, items
}

const graphQLFields = "id author{login} assignees(first:100){nodes{login} pageInfo{hasNextPage endCursor}} " +
	"state merged isDraft mergeable updatedAt " +
	"latestReviews(first:100){nodes{author{login} state} pageInfo{hasNextPage endCursor}} " +
	"reviewRequests(first:100){nodes{requestedReviewer{... on User{login} ... on Team{slug}}} " +
	"pageInfo{hasNextPage endCursor}}"

func (p *LiveProvider) doGraphQL(
	ctx context.Context,
	query string,
	variables map[string]any,
	destination any,
) (int, error) {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.graphqlURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "prx/0.1")
	response, err := p.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		return response.StatusCode, errors.New(strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(destination); err != nil {
		return response.StatusCode, err
	}
	return response.StatusCode, nil
}

func mapGraphQLErrors(
	errorsFound []graphQLError,
	items []graphQLItem,
) (map[string]error, error) {
	result := make(map[string]error)
	byRepository := make(map[string][]graphQLItem)
	byPull := make(map[string]graphQLItem)
	for _, item := range items {
		byRepository[item.repositoryAlias] = append(byRepository[item.repositoryAlias], item)
		byPull[item.repositoryAlias+"/"+item.pullAlias] = item
	}
	for _, graphErr := range errorsFound {
		class := classifyGraphQLError(graphErr)
		err := providerGraphQLError(class, graphErr.Message)
		if len(graphErr.Path) == 0 {
			return result, err
		}
		repositoryAlias, _ := graphErr.Path[0].(string)
		if len(graphErr.Path) > 1 {
			pullAlias, _ := graphErr.Path[1].(string)
			if item, ok := byPull[repositoryAlias+"/"+pullAlias]; ok {
				result[item.current.TaskID] = err
				continue
			}
		}
		for _, item := range byRepository[repositoryAlias] {
			result[item.current.TaskID] = err
		}
	}
	return result, nil
}

func classifyGraphQLError(value graphQLError) ErrorClass {
	message := strings.ToLower(value.Message)
	typeName, _ := value.Extensions["type"].(string)
	joined := message + " " + strings.ToLower(typeName)
	switch {
	case strings.Contains(joined, "rate limit"):
		return ErrorClassRateLimit
	case strings.Contains(joined, "unauthorized"), strings.Contains(joined, "bad credentials"):
		return ErrorClassUnauthorized
	case strings.Contains(joined, "forbidden"), strings.Contains(joined, "permission"),
		strings.Contains(joined, "resource not accessible"):
		return ErrorClassPermission
	case strings.Contains(joined, "could not resolve"), strings.Contains(joined, "not found"):
		return ErrorClassNotFound
	default:
		return ErrorClassOther
	}
}

func graphQLHTTPError(status int, err error) error {
	return &ProviderError{
		Class: classifyError(err, status, nil), StatusCode: status,
		Operation: "fetch GraphQL pull requests", Err: err,
	}
}

func providerGraphQLError(class ErrorClass, message string) error {
	return &ProviderError{
		Class: class, Operation: "fetch GraphQL pull requests", Err: errors.New(message),
	}
}

func domainPullRequestFromGraphQL(
	current domain.PullRequest,
	value graphQLPullRequest,
) (domain.PullRequest, error) {
	updatedAt, err := time.Parse(time.RFC3339, value.UpdatedAt)
	if err != nil {
		return current, fmt.Errorf("parse GitHub updatedAt: %w", err)
	}
	state := domain.PullRequestState(strings.ToLower(value.State))
	if value.Merged {
		state = domain.PullRequestStateMerged
	}
	if state != domain.PullRequestStateOpen && state != domain.PullRequestStateClosed &&
		state != domain.PullRequestStateMerged {
		state = domain.PullRequestStateUnknown
	}
	mergeability := domain.Mergeability(strings.ToLower(value.Mergeable))
	if mergeability != domain.MergeabilityMergeable && mergeability != domain.MergeabilityConflicting {
		mergeability = domain.MergeabilityUnknown
	}
	assignees := make([]string, 0, len(value.Assignees.Nodes))
	for _, assignee := range value.Assignees.Nodes {
		assignees = append(assignees, assignee.Login)
	}
	reviewState := domain.ReviewStateNone
	for _, review := range value.LatestReviews.Nodes {
		switch strings.ToUpper(review.State) {
		case "CHANGES_REQUESTED":
			reviewState = domain.ReviewStateChangesRequested
		case "APPROVED":
			if reviewState != domain.ReviewStateChangesRequested {
				reviewState = domain.ReviewStateApproved
			}
		}
	}
	if reviewState == domain.ReviewStateNone && len(value.ReviewRequests.Nodes) > 0 {
		reviewState = domain.ReviewStateRequired
	}
	now := time.Now().UTC()
	current.NodeID = value.ID
	if value.Author != nil {
		current.Author = value.Author.Login
	}
	current.Assignees = assignees
	current.State = state
	current.Draft = value.IsDraft
	current.Mergeability = mergeability
	current.ReviewState = reviewState
	githubUpdatedAt := updatedAt.UTC()
	current.GitHubUpdatedAt = &githubUpdatedAt
	current.LastSyncedAt = &now
	current.SyncError = ""
	current.Stale = false
	return current, nil
}
