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
	failures  map[string]error
	authID    string
}

type candidateAttempt struct {
	successes map[string]domain.PullRequest
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
	groups := make(map[repositoryKey][]domain.PullRequest)
	for _, value := range snapshot.PullRequests {
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

	cache, _ := s.repository.(GitHubAuthCache)
	for _, key := range keys {
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
		for _, current := range values {
			if updated, ok := result.successes[current.TaskID]; ok {
				if _, persistErr := s.repository.UpsertPullRequest(ctx, updated); persistErr != nil {
					return succeeded, failed, persistErr
				}
				succeeded++
				continue
			}
			failure := result.failures[current.TaskID]
			if failure == nil {
				failure = fmt.Errorf("GitHub synchronization did not produce a result")
			}
			current.LastSyncedAt = timePointer(time.Now().UTC())
			current.SyncError = failure.Error()
			current.Stale = true
			if _, persistErr := s.repository.UpsertPullRequest(ctx, current); persistErr != nil {
				return succeeded, failed, persistErr
			}
			failed++
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
	for index, current := range values {
		updated, fetchErr := provider.Fetch(ctx, current)
		if fetchErr == nil {
			result.successes[current.TaskID] = updated
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
		result.failedAt = index
		result.err = fetchErr
		result.class = class
		return result, nil
	}
	return result, nil
}

func mergeCandidateAttempt(destination *repositorySyncResult, attempt candidateAttempt) {
	for taskID, value := range attempt.successes {
		destination.successes[taskID] = value
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

func timePointer(value time.Time) *time.Time { return &value }
