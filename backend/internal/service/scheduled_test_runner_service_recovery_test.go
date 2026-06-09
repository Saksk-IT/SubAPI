//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestScheduledTestRunner_TryRecoverAccount_UsesErrorCodeRecoveryScope(t *testing.T) {
	repo := &rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:           43,
			Status:       StatusError,
			Schedulable:  false,
			ErrorMessage: "Custom error code 500: upstream unavailable",
		},
	}
	blocker := &runtimeBlockRecorder{}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitSvc.SetAccountRuntimeBlocker(blocker)
	runner := &ScheduledTestRunnerService{rateLimitSvc: rateLimitSvc}

	recovery := runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                       1,
		AccountID:                43,
		AutoRecoverErrorCodeStop: true,
	})

	require.NotNil(t, recovery)
	require.True(t, recovery.ClearedError)
	require.False(t, recovery.ClearedRateLimit)
	require.True(t, recovery.RestoredScheduling)
	require.Equal(t, 1, repo.clearErrorCalls)
	require.Equal(t, 1, repo.setSchedulableCalls)
	require.Equal(t, []bool{true}, repo.setSchedulableValues)
	require.Equal(t, []int64{43}, blocker.clearedIDs)
}

func TestScheduledTestRunner_TryRecoverAccount_UsesManualStopRecoveryScope(t *testing.T) {
	repo := &rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:          44,
			Status:      StatusActive,
			Schedulable: false,
		},
	}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	runner := &ScheduledTestRunnerService{rateLimitSvc: rateLimitSvc}

	recovery := runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                    2,
		AccountID:             44,
		AutoRecoverManualStop: true,
	})

	require.NotNil(t, recovery)
	require.False(t, recovery.ClearedError)
	require.False(t, recovery.ClearedRateLimit)
	require.True(t, recovery.RestoredScheduling)
	require.Equal(t, 0, repo.clearErrorCalls)
	require.Equal(t, 0, repo.clearRateLimitCalls)
	require.Equal(t, 1, repo.setSchedulableCalls)
	require.Equal(t, []bool{true}, repo.setSchedulableValues)
}

func TestScheduledTestRunner_TryRecoverAccount_RuntimeScopeDoesNotRestoreManualStop(t *testing.T) {
	now := time.Now()
	repo := &rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:               45,
			Status:           StatusActive,
			Schedulable:      false,
			RateLimitedAt:    &now,
			RateLimitResetAt: &now,
		},
	}
	cache := &tempUnschedCacheRecorder{}
	blocker := &runtimeBlockRecorder{}
	rateLimitSvc := NewRateLimitService(repo, nil, &config.Config{}, nil, cache)
	rateLimitSvc.SetAccountRuntimeBlocker(blocker)
	runner := &ScheduledTestRunnerService{rateLimitSvc: rateLimitSvc}

	recovery := runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                      3,
		AccountID:               45,
		AutoRecoverRuntimeState: true,
	})

	require.NotNil(t, recovery)
	require.False(t, recovery.ClearedError)
	require.True(t, recovery.ClearedRateLimit)
	require.False(t, recovery.RestoredScheduling)
	require.Equal(t, 1, repo.clearRateLimitCalls)
	require.Equal(t, 1, repo.clearTempUnschedCalls)
	require.Equal(t, 0, repo.setSchedulableCalls)
	require.Equal(t, []int64{45}, cache.deletedIDs)
	require.Equal(t, []int64{45}, blocker.clearedIDs)
}

func TestScheduledTestRunner_RunOnePlan_RecordsRecoveryActionsOnSuccessfulResult(t *testing.T) {
	now := time.Now()
	accountRepo := &rateLimitClearRepoStub{
		getByIDAccount: &Account{
			ID:               46,
			Status:           StatusError,
			Schedulable:      false,
			ErrorMessage:     "Custom error code 500: upstream unavailable",
			RateLimitedAt:    &now,
			RateLimitResetAt: &now,
		},
	}
	cache := &tempUnschedCacheRecorder{}
	blocker := &runtimeBlockRecorder{}
	rateLimitSvc := NewRateLimitService(accountRepo, nil, &config.Config{}, nil, cache)
	rateLimitSvc.SetAccountRuntimeBlocker(blocker)

	resultRepo := &scheduledTestResultRepoRecorder{}
	runner := &ScheduledTestRunnerService{
		planRepo: &scheduledTestPlanRepoRecorder{},
		scheduledSvc: &ScheduledTestService{
			resultRepo: resultRepo,
		},
		accountTestSvc: &scheduledAccountTesterStub{
			result: &ScheduledTestResult{
				Status:       "success",
				ResponseText: "ok",
				LatencyMs:    1234,
				StartedAt:    now,
				FinishedAt:   now.Add(time.Second),
			},
		},
		rateLimitSvc: rateLimitSvc,
	}

	runner.runOnePlan(context.Background(), &ScheduledTestPlan{
		ID:                       4,
		AccountID:                46,
		ModelID:                  "gpt-4o-mini",
		CronExpression:           "*/30 * * * *",
		MaxResults:               20,
		AutoRecover:              true,
		AutoRecoverErrorCodeStop: true,
		AutoRecoverRuntimeState:  true,
	})

	require.Len(t, resultRepo.created, 1)
	saved := resultRepo.created[0]
	require.True(t, saved.RecoveryClearedError)
	require.True(t, saved.RecoveryClearedRuntimeState)
	require.True(t, saved.RecoveryRestoredScheduling)
	require.Equal(t, int64(4), saved.PlanID)
	require.Equal(t, 1, accountRepo.clearErrorCalls)
	require.Equal(t, 1, accountRepo.clearRateLimitCalls)
	require.Equal(t, 1, accountRepo.setSchedulableCalls)
	require.Equal(t, []int64{46}, blocker.clearedIDs)
}

type scheduledAccountTesterStub struct {
	result *ScheduledTestResult
	err    error
}

func (s *scheduledAccountTesterStub) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type scheduledTestResultRepoRecorder struct {
	created []*ScheduledTestResult
}

func (r *scheduledTestResultRepoRecorder) Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	if result == nil {
		return nil, errors.New("nil scheduled test result")
	}
	saved := *result
	if saved.ID == 0 {
		saved.ID = int64(len(r.created) + 1)
	}
	r.created = append(r.created, &saved)
	return &saved, nil
}

func (r *scheduledTestResultRepoRecorder) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestResultRepoRecorder) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	return nil
}

type scheduledTestPlanRepoRecorder struct {
	updatedAfterRun []int64
}

func (r *scheduledTestPlanRepoRecorder) Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return nil, errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func (r *scheduledTestPlanRepoRecorder) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	r.updatedAfterRun = append(r.updatedAfterRun, id)
	return nil
}
