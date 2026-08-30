package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HappyOnigiri/PRX/internal/domain"
	githubprovider "github.com/HappyOnigiri/PRX/internal/github"
)

type repositoryKey struct {
	host       string
	owner      string
	repository string
}

type repositorySyncResult struct {
	successes map[string]domain.PullRequest
	partials  map[string]domain.PullRequest
	failures  map[string]error
	authID    string
}

type candidateAttempt struct {
	successes map[string]domain.PullRequest
	partials  map[string]domain.PullRequest
	failures  map[string]error
	failedAt  int
	err       error
	class     githubprovider.ErrorClass
}

func (s *Service) syncLive(
	ctx context.Context,
	snapshot domain.Snapshot,
	taskFeature map[string]string,
	featureID string,
	taskID string,
	resolver *githubprovider.Resolver,
) (succeeded, failed int, err error) {
	groups, keys := groupPullRequests(snapshot.PullRequests, taskFeature, featureID, taskID)
	cache, _ := s.repository.(GitHubAuthCache)
	processed, succeeded, failed, err := s.syncUncachedHostGroups(ctx, keys, groups, resolver, cache)
	if err != nil {
		return succeeded, failed, err
	}
	remainingSucceeded, remainingFailed, err := s.syncRepositoryGroups(
		ctx, keys, groups, processed, resolver, cache,
	)
	return succeeded + remainingSucceeded, failed + remainingFailed, err
}

func groupPullRequests(
	pullRequests []domain.PullRequest,
	taskFeature map[string]string,
	featureID, taskID string,
) (map[repositoryKey][]domain.PullRequest, []repositoryKey) {
	groups := make(map[repositoryKey][]domain.PullRequest)
	for _, value := range pullRequests {
		if taskID != "" && value.TaskID != taskID {
			continue
		}
		if featureID != "" && taskFeature[value.TaskID] != featureID {
			continue
		}
		host := strings.ToLower(strings.TrimSpace(value.Host))
		if host == "" {
			host = "github.com"
		}
		key := repositoryKey{
			host:       host,
			owner:      strings.ToLower(value.Owner),
			repository: strings.ToLower(value.Repository),
		}
		groups[key] = append(groups[key], value)
	}
	keys := make([]repositoryKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].host != keys[j].host {
			return keys[i].host < keys[j].host
		}
		if keys[i].owner != keys[j].owner {
			return keys[i].owner < keys[j].owner
		}
		return keys[i].repository < keys[j].repository
	})
	return groups, keys
}

func (s *Service) syncUncachedHostGroups(
	ctx context.Context,
	keys []repositoryKey,
	groups map[repositoryKey][]domain.PullRequest,
	resolver *githubprovider.Resolver,
	cache GitHubAuthCache,
) (processed map[repositoryKey]bool, succeeded, failed int, err error) {
	processed = make(map[repositoryKey]bool)
	keysByHost := make(map[string][]repositoryKey)
	// keys is already sorted, and hosts are visited in that same order so a
	// failure stops after the same hosts on every run.
	hosts := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, seen := keysByHost[key.host]; !seen {
			hosts = append(hosts, key.host)
		}
		keysByHost[key.host] = append(keysByHost[key.host], key)
	}
	for _, host := range hosts {
		hostKeys := keysByHost[host]
		if len(hostKeys) < 2 || repositoriesHaveCachedAuth(ctx, cache, hostKeys) {
			continue
		}
		hostSucceeded, hostFailed, hostErr := s.syncHostBatch(ctx, hostKeys, groups, resolver, cache)
		if hostErr != nil {
			return processed, succeeded, failed, hostErr
		}
		succeeded += hostSucceeded
		failed += hostFailed
		for _, key := range hostKeys {
			processed[key] = true
		}
	}
	return processed, succeeded, failed, nil
}

func (s *Service) syncRepositoryGroups(
	ctx context.Context,
	keys []repositoryKey,
	groups map[repositoryKey][]domain.PullRequest,
	processed map[repositoryKey]bool,
	resolver *githubprovider.Resolver,
	cache GitHubAuthCache,
) (succeeded, failed int, err error) {
	for _, key := range keys {
		if processed[key] {
			continue
		}
		values := groups[key]
		result, resultErr := s.syncRepository(ctx, key, values, resolver, cache)
		if resultErr != nil {
			return succeeded, failed, resultErr
		}
		if cache != nil && result.authID != "" {
			if cacheErr := cache.UpsertGitHubRepositoryAuthCache(
				ctx,
				key.host,
				key.owner,
				key.repository,
				result.authID,
			); cacheErr != nil {
				return succeeded, failed, cacheErr
			}
		}
		groupSucceeded, groupFailed, persistErr := s.persistSyncResult(ctx, values, result)
		succeeded += groupSucceeded
		failed += groupFailed
		if persistErr != nil {
			return succeeded, failed, persistErr
		}
	}
	return succeeded, failed, nil
}

// persistSyncResult stores every refreshed pull request and records the rest as
// stale with the failure that kept them from refreshing. A known terminal pull
// request is retained as stale without becoming an actionable failure. Both
// synchronization paths share this fallback for items in neither map.
func (s *Service) persistSyncResult(
	ctx context.Context,
	values []domain.PullRequest,
	result repositorySyncResult,
) (succeeded, failed int, err error) {
	for _, current := range values {
		if updated, ok := result.successes[current.TaskID]; ok {
			if _, persistErr := s.repository.UpsertPullRequest(ctx, updated); persistErr != nil {
				return succeeded, failed, persistErr
			}
			succeeded++
			continue
		}
		failure := result.failures[current.TaskID]
		partial, hasPartial := result.partials[current.TaskID]
		if !hasPartial {
			partial = current
		}
		value, needsAttention := syncFailureValue(
			current,
			partial,
			failure,
			s.currentTime(),
		)
		if _, persistErr := s.repository.UpsertPullRequest(ctx, value); persistErr != nil {
			return succeeded, failed, persistErr
		}
		if needsAttention {
			failed++
		}
	}
	return succeeded, failed, nil
}

func syncFailureValue(
	current, partial domain.PullRequest,
	failure error,
	attemptedAt time.Time,
) (domain.PullRequest, bool) {
	value := partial
	value.TaskID = current.TaskID
	value.LastSyncedAt = timePointer(attemptedAt)
	value.Stale = true
	state := value.State
	if state == "" || state == domain.PullRequestStateUnknown {
		state = current.State
		value.State = state
	}
	if terminalPullRequestState(state) {
		value.SyncError = ""
		return value, false
	}
	if failure == nil {
		failure = fmt.Errorf("GitHub synchronization did not produce a result")
	}
	value.SyncError = failure.Error()
	return value, true
}

func terminalPullRequestState(state domain.PullRequestState) bool {
	return state == domain.PullRequestStateClosed || state == domain.PullRequestStateMerged
}

func repositoriesHaveCachedAuth(
	ctx context.Context,
	cache GitHubAuthCache,
	keys []repositoryKey,
) bool {
	if cache == nil {
		return false
	}
	for _, key := range keys {
		_, found, err := cache.GetGitHubRepositoryAuthCache(ctx, key.host, key.owner, key.repository)
		if err != nil || found {
			return true
		}
	}
	return false
}

func (s *Service) syncHostBatch(
	ctx context.Context,
	keys []repositoryKey,
	groups map[repositoryKey][]domain.PullRequest,
	resolver *githubprovider.Resolver,
	cache GitHubAuthCache,
) (succeeded, failed int, err error) {
	work := make([]domain.PullRequest, 0)
	keyByTask := make(map[string]repositoryKey)
	for _, key := range keys {
		for _, value := range groups[key] {
			work = append(work, value)
			keyByTask[value.TaskID] = key
		}
	}
	result := repositorySyncResult{
		successes: map[string]domain.PullRequest{},
		partials:  map[string]domain.PullRequest{},
		failures:  map[string]error{},
	}
	for _, candidate := range resolver.Candidates(keys[0].host) {
		if len(work) == 0 {
			break
		}
		provider, openErr := resolver.Open(ctx, candidate)
		if openErr != nil {
			if githubprovider.ClassOf(openErr) == githubprovider.ErrorClassUnauthorized {
				resolver.MarkUnauthorized(candidate.ID)
			}
			if isFallbackClass(githubprovider.ClassOf(openErr)) {
				continue
			}
			addRemainingFailures(result.failures, work, 0, openErr)
			work = nil
			break
		}
		batch, batchErr := provider.FetchBatch(ctx, work)
		// A chunked fetch reports what earlier chunks already produced alongside
		// the failure, so take those results before deciding what the failure
		// means for the pull requests it never reached.
		next := make([]domain.PullRequest, 0)
		for _, current := range work {
			itemErr := batch.Errors[current.TaskID]
			if updated, ok := batch.PullRequests[current.TaskID]; ok && itemErr == nil {
				result.successes[current.TaskID] = updated
				if cache != nil {
					key := keyByTask[current.TaskID]
					if cacheErr := cache.UpsertGitHubRepositoryAuthCache(
						ctx, key.host, key.owner, key.repository, candidate.ID,
					); cacheErr != nil {
						return succeeded, failed, cacheErr
					}
				}
				continue
			}
			if updated, ok := batch.PullRequests[current.TaskID]; ok && itemErr != nil {
				result.partials[current.TaskID] = updated
			}
			if updated, ok := batch.PartialPullRequests[current.TaskID]; ok {
				result.partials[current.TaskID] = updated
			}
			if itemErr == nil && batchErr == nil {
				itemErr = fmt.Errorf("GitHub synchronization did not produce a result")
			}
			if itemErr == nil || isFallbackClass(githubprovider.ClassOf(itemErr)) {
				next = append(next, current)
				continue
			}
			result.failures[current.TaskID] = itemErr
		}
		if batchErr != nil {
			class := githubprovider.ClassOf(batchErr)
			if class == githubprovider.ErrorClassUnauthorized {
				resolver.MarkUnauthorized(candidate.ID)
			}
			if !isFallbackClass(class) {
				addRemainingFailures(result.failures, next, 0, batchErr)
				work = nil
				break
			}
		}
		work = next
	}
	addUnavailableFailures(result.failures, work, keys[0].host)
	for _, key := range keys {
		groupSucceeded, groupFailed, persistErr := s.persistSyncResult(ctx, groups[key], result)
		succeeded += groupSucceeded
		failed += groupFailed
		if persistErr != nil {
			return succeeded, failed, persistErr
		}
	}
	return succeeded, failed, nil
}

func (s *Service) syncRepository(
	ctx context.Context,
	key repositoryKey,
	values []domain.PullRequest,
	resolver *githubprovider.Resolver,
	cache GitHubAuthCache,
) (repositorySyncResult, error) {
	result := repositorySyncResult{
		successes: map[string]domain.PullRequest{},
		partials:  map[string]domain.PullRequest{},
		failures:  map[string]error{},
	}
	candidates := resolver.Candidates(key.host)
	if len(candidates) == 0 {
		for _, value := range values {
			result.failures[value.TaskID] = fmt.Errorf("GitHub host %q has no authentication method", key.host)
		}
		return result, nil
	}

	work := values
	if cache != nil {
		var stop bool
		var err error
		work, stop, err = s.tryCachedRepository(ctx, key, work, candidates, resolver, cache, &result)
		if err != nil {
			return result, err
		}
		if stop {
			return result, nil
		}
	}

	for _, candidate := range candidates {
		if len(work) == 0 {
			break
		}
		attempt, openErr := s.attemptCandidate(ctx, resolver, candidate, work, true)
		if openErr != nil {
			if githubprovider.ClassOf(openErr) == githubprovider.ErrorClassUnauthorized {
				resolver.MarkUnauthorized(candidate.ID)
			}
			if githubprovider.ClassOf(openErr) == githubprovider.ErrorClassAuthUnavailable {
				continue
			}
			if isFallbackClass(githubprovider.ClassOf(openErr)) {
				continue
			}
			addRemainingFailures(result.failures, work, 0, openErr)
			return result, nil
		}
		mergeCandidateAttempt(&result, attempt)
		if attempt.err == nil {
			result.authID = candidate.ID
			return result, nil
		}
		if attempt.class == githubprovider.ErrorClassUnauthorized {
			resolver.MarkUnauthorized(candidate.ID)
		}
		if !isFallbackClass(attempt.class) {
			addRemainingFailures(result.failures, work, attempt.failedAt, attempt.err)
			return result, nil
		}
		if attempt.failedAt >= 0 && attempt.failedAt < len(work) {
			work = work[attempt.failedAt:]
		} else {
			work = nil
		}
	}
	addUnavailableFailures(result.failures, work, key.host)
	return result, nil
}

func (s *Service) tryCachedRepository(
	ctx context.Context,
	key repositoryKey,
	work []domain.PullRequest,
	candidates []githubprovider.Candidate,
	resolver *githubprovider.Resolver,
	cache GitHubAuthCache,
	result *repositorySyncResult,
) ([]domain.PullRequest, bool, error) {
	cachedID, found, err := cache.GetGitHubRepositoryAuthCache(ctx, key.host, key.owner, key.repository)
	if err != nil || !found {
		return work, false, err
	}
	candidate, ok := candidateByID(candidates, cachedID)
	if !ok {
		_ = cache.DeleteGitHubRepositoryAuthCache(ctx, key.host, key.owner, key.repository)
		return work, false, nil
	}
	attempt, openErr := s.attemptCandidate(ctx, resolver, candidate, work, false)
	if openErr != nil {
		attempt = candidateAttempt{
			failedAt: 0,
			err:      openErr,
			class:    githubprovider.ClassOf(openErr),
		}
	}
	mergeCandidateAttempt(result, attempt)
	if attempt.err == nil {
		result.authID = candidate.ID
		return work, true, nil
	}
	if !isFallbackClass(attempt.class) {
		addRemainingFailures(result.failures, work, attempt.failedAt, attempt.err)
		return work, true, nil
	}
	if attempt.class == githubprovider.ErrorClassUnauthorized {
		resolver.MarkUnauthorized(candidate.ID)
	}
	_ = cache.DeleteGitHubRepositoryAuthCache(ctx, key.host, key.owner, key.repository)
	return workAfterFailure(work, attempt.failedAt), false, nil
}

func workAfterFailure(work []domain.PullRequest, failedAt int) []domain.PullRequest {
	if failedAt >= 0 && failedAt < len(work) {
		return work[failedAt:]
	}
	return nil
}

func addUnavailableFailures(
	failures map[string]error,
	work []domain.PullRequest,
	host string,
) {
	for _, value := range work {
		if _, ok := failures[value.TaskID]; !ok {
			failures[value.TaskID] = fmt.Errorf(
				"no usable GitHub authentication method for host %q",
				host,
			)
		}
	}
}

func (s *Service) attemptCandidate(
	ctx context.Context,
	resolver *githubprovider.Resolver,
	candidate githubprovider.Candidate,
	values []domain.PullRequest,
	probe bool,
) (candidateAttempt, error) {
	result := candidateAttempt{
		successes: map[string]domain.PullRequest{},
		partials:  map[string]domain.PullRequest{},
		failures:  map[string]error{},
		failedAt:  -1,
	}
	provider, err := resolver.Open(ctx, candidate)
	if err != nil {
		return result, err
	}
	if probe {
		if err := provider.Probe(ctx, values[0].Owner, values[0].Repository); err != nil {
			result.failedAt = 0
			result.err = err
			result.class = githubprovider.ClassOf(err)
			return result, nil
		}
	}
	batchResult, batchErr := provider.FetchBatch(ctx, values)
	for taskID, value := range batchResult.PullRequests {
		if itemErr := batchResult.Errors[taskID]; itemErr != nil {
			result.partials[taskID] = value
			continue
		}
		result.successes[taskID] = value
	}
	for taskID, value := range batchResult.PartialPullRequests {
		result.partials[taskID] = value
	}
	// A chunked fetch stops at the first failing chunk and reports the earlier
	// chunks with the failure, so keep what it produced and let the caller retry
	// only from the first pull request the failure reached.
	resolved := values
	if batchErr != nil {
		resolved = values[:unresolvedIndex(values, batchResult)]
	}
	for index, current := range resolved {
		fetchErr := batchResult.Errors[current.TaskID]
		if fetchErr == nil {
			continue
		}
		class := githubprovider.ClassOf(fetchErr)
		if class == githubprovider.ErrorClassNotFound && !probe {
			if probeErr := provider.Probe(ctx, current.Owner, current.Repository); probeErr == nil {
				result.failures[current.TaskID] = fmt.Errorf(
					"pull request %s/%s#%d was not found",
					current.Owner,
					current.Repository,
					current.Number,
				)
				probe = true
				continue
			} else {
				fetchErr = probeErr
				class = githubprovider.ClassOf(probeErr)
			}
		}
		if class == githubprovider.ErrorClassNotFound && probe {
			result.failures[current.TaskID] = fmt.Errorf(
				"pull request %s/%s#%d was not found",
				current.Owner,
				current.Repository,
				current.Number,
			)
			continue
		}
		if isFallbackClass(class) {
			result.failedAt = index
			result.err = fetchErr
			result.class = class
			return result, nil
		}
		result.failures[current.TaskID] = fetchErr
	}
	if batchErr != nil {
		result.failedAt = len(resolved)
		result.err = batchErr
		result.class = githubprovider.ClassOf(batchErr)
	}
	return result, nil
}

// unresolvedIndex reports the position of the first pull request the batch
// neither updated nor reported an item error for. A chunked fetch stops at its
// first failing chunk, so nothing from that position on was attempted.
func unresolvedIndex(values []domain.PullRequest, batch githubprovider.BatchResult) int {
	for index, value := range values {
		if _, ok := batch.PullRequests[value.TaskID]; ok {
			continue
		}
		if _, ok := batch.PartialPullRequests[value.TaskID]; ok {
			continue
		}
		if _, ok := batch.Errors[value.TaskID]; ok {
			continue
		}
		return index
	}
	return len(values)
}

func mergeCandidateAttempt(destination *repositorySyncResult, attempt candidateAttempt) {
	for taskID, value := range attempt.successes {
		destination.successes[taskID] = value
	}
	for taskID, value := range attempt.partials {
		destination.partials[taskID] = value
	}
	for taskID, err := range attempt.failures {
		destination.failures[taskID] = err
	}
}

func candidateByID(candidates []githubprovider.Candidate, id string) (githubprovider.Candidate, bool) {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return githubprovider.Candidate{}, false
}

func isFallbackClass(class githubprovider.ErrorClass) bool {
	return class == githubprovider.ErrorClassAuthUnavailable ||
		class == githubprovider.ErrorClassUnauthorized ||
		class == githubprovider.ErrorClassPermission ||
		class == githubprovider.ErrorClassNotFound
}

func addRemainingFailures(destination map[string]error, values []domain.PullRequest, failedAt int, err error) {
	if failedAt < 0 || failedAt >= len(values) {
		failedAt = 0
	}
	for _, value := range values[failedAt:] {
		destination[value.TaskID] = err
	}
}

func (s *Service) currentTime() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func timePointer(value time.Time) *time.Time { return &value }
