package app

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"

	"github.com/HappyOnigiri/PRX/internal/config"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func (s *Service) SyncStatus(ctx context.Context) (domain.GitHubSyncStatus, error) {
	if s.configStore == nil {
		return domain.GitHubSyncStatus{}, domain.NewError(
			domain.DomainErrorCodeInvalidConfig,
			"GitHub configuration is unavailable",
		)
	}
	repository, ok := s.repository.(GitHubSyncStateRepository)
	if !ok {
		return domain.GitHubSyncStatus{}, domain.NewError(
			domain.DomainErrorCodeInternal,
			"GitHub sync state is unavailable",
		)
	}
	state, err := repository.GitHubSyncState(ctx)
	if err != nil {
		return domain.GitHubSyncStatus{}, err
	}
	interval := config.DefaultAutoSyncIntervalSeconds
	settings, loadErr := s.configStore.Load()
	if loadErr == nil {
		interval = settings.GitHub.AutoSyncIntervalSeconds
	} else if state.Error == "" {
		state.Error = configDomainError(loadErr).Error()
	}
	return syncStatus(interval, state), nil
}

func (s *Service) SyncIfDue(ctx context.Context) (bool, domain.GitHubSyncStatus, error) {
	if s.configStore == nil {
		return false, domain.GitHubSyncStatus{}, domain.NewError(
			domain.DomainErrorCodeInvalidConfig,
			"GitHub configuration is unavailable",
		)
	}
	repository, ok := s.repository.(GitHubSyncStateRepository)
	if !ok {
		return false, domain.GitHubSyncStatus{}, domain.NewError(
			domain.DomainErrorCodeInternal,
			"GitHub sync state is unavailable",
		)
	}
	interval := config.DefaultAutoSyncIntervalSeconds
	settings, loadErr := s.configStore.Load()
	if loadErr == nil {
		interval = settings.GitHub.AutoSyncIntervalSeconds
	}
	now := s.now().UTC()
	runID := uuid.NewString()
	acquired, err := repository.AcquireGitHubAutoSync(
		ctx,
		runID,
		now,
		saturatedSubtract(now.Unix(), interval),
	)
	if err != nil {
		return false, domain.GitHubSyncStatus{}, err
	}
	if !acquired {
		status, statusErr := repository.GitHubSyncState(ctx)
		return false, syncStatus(interval, status), statusErr
	}
	if loadErr != nil {
		runError := configDomainError(loadErr).Error()
		if completeErr := repository.CompleteGitHubSync(ctx, runID, s.now().UTC(), 0, 0, runError); completeErr != nil {
			return true, domain.GitHubSyncStatus{}, completeErr
		}
		status, statusErr := repository.GitHubSyncState(ctx)
		return true, syncStatus(interval, status), statusErr
	}

	succeeded, failed, syncErr := s.syncSelected(ctx, "", "", true)
	runError := ""
	if syncErr != nil {
		runError = syncErr.Error()
	}
	completeErr := repository.CompleteGitHubSync(ctx, runID, s.now().UTC(), succeeded, failed, runError)
	status, statusErr := repository.GitHubSyncState(ctx)
	if completeErr != nil {
		return true, syncStatus(interval, status), completeErr
	}
	if statusErr != nil {
		return true, domain.GitHubSyncStatus{}, statusErr
	}
	// The automatic path records errors but deliberately does not fail the CLI
	// command or page load that happened to notice the expired interval.
	return true, syncStatus(interval, status), nil
}

func (s *Service) Sync(ctx context.Context, featureID, taskID string) (succeeded, failed int, err error) {
	repository, recordsState := s.repository.(GitHubSyncStateRepository)
	// A refresh that covers only one feature or task says nothing about the
	// pull requests it skipped, so it must not reset the shared interval or
	// overwrite the counts the last full refresh recorded. Seeding reaches this
	// function with a feature, which keeps fixture results out of the status.
	recordsState = recordsState && featureID == "" && taskID == ""
	runID := uuid.NewString()
	if recordsState {
		if startErr := repository.StartGitHubSync(ctx, runID, s.now().UTC()); startErr != nil {
			return 0, 0, startErr
		}
	}
	succeeded, failed, err = s.syncSelected(ctx, featureID, taskID, false)
	if recordsState {
		runError := ""
		if err != nil {
			runError = err.Error()
		}
		if completeErr := repository.CompleteGitHubSync(
			ctx, runID, s.now().UTC(), succeeded, failed, runError,
		); completeErr != nil {
			return succeeded, failed, errors.Join(err, completeErr)
		}
	}
	return succeeded, failed, err
}

func syncStatus(interval int64, state domain.GitHubSyncState) domain.GitHubSyncStatus {
	return domain.GitHubSyncStatus{
		IntervalSeconds: interval,
		LastAttemptAt:   state.LastAttemptAt,
		LastUpdatedAt:   state.LastCompletedAt,
		Succeeded:       state.Succeeded,
		Failed:          state.Failed,
		Error:           state.Error,
	}
}

func saturatedSubtract(value, amount int64) int64 {
	if value < 0 && amount > value-math.MinInt64 {
		return math.MinInt64
	}
	return value - amount
}
