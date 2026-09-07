package app

import (
	"context"
	"errors"
	"math"
	"time"

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
		recordContext, cancel := recordingContext(ctx)
		defer cancel()
		recorded, completeErr := repository.CompleteGitHubSync(
			recordContext, runID, s.now().UTC(), 0, 0, runError,
		)
		if completeErr != nil {
			return true, domain.GitHubSyncStatus{}, completeErr
		}
		status, statusErr := repository.GitHubSyncState(recordContext)
		return recorded, syncStatus(interval, status), statusErr
	}

	succeeded, failed, syncErr := s.syncSelected(ctx, "", "", true)
	runError := ""
	if syncErr != nil {
		runError = syncErr.Error()
	}
	// 呼び出し側が消えたことで refresh が終わった可能性があるが、結果は必ず記録する。
	// さもないと取得済みの試行が、何に止められたかの説明もないまま interval を握る。
	recordContext, cancel := recordingContext(ctx)
	defer cancel()
	recorded, completeErr := repository.CompleteGitHubSync(
		recordContext, runID, s.now().UTC(), succeeded, failed, runError,
	)
	status, statusErr := repository.GitHubSyncState(recordContext)
	if completeErr != nil {
		return true, syncStatus(interval, status), completeErr
	}
	if statusErr != nil {
		return true, domain.GitHubSyncStatus{}, statusErr
	}
	// 自動経路はエラーを記録するが、interval 切れにたまたま気づいた CLI コマンドや
	// ページ読み込みを意図的に失敗させない。
	// 上書きされた記録は、interval を取得しなかった場合と同じ扱いで報告する。
	return recorded, syncStatus(interval, status), nil
}

// recordingContext は実行状態の書き込みを、refresh 自体を終わらせたかもしれない
// キャンセルから切り離しつつ、書き込みの待ち時間には上限を設ける。
func recordingContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), syncRecordTimeout)
}

const syncRecordTimeout = 5 * time.Second

func (s *Service) Sync(ctx context.Context, featureID, taskID string) (succeeded, failed int, err error) {
	repository, recordsState := s.repository.(GitHubSyncStateRepository)
	// 単一の feature や task だけを対象にした refresh は、飛ばした pull request について
	// 何も語らない。よって共有の interval をリセットしたり、
	// 直近の全体 refresh が記録した件数を上書きしたりしてはならない。
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
		recordContext, cancel := recordingContext(ctx)
		defer cancel()
		if _, completeErr := repository.CompleteGitHubSync(
			recordContext, runID, s.now().UTC(), succeeded, failed, runError,
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
