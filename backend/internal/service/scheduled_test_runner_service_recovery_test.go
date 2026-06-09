//go:build unit

package service

import (
	"context"
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

	runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                       1,
		AccountID:                43,
		AutoRecoverErrorCodeStop: true,
	})

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

	runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                    2,
		AccountID:             44,
		AutoRecoverManualStop: true,
	})

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

	runner.tryRecoverAccount(context.Background(), &ScheduledTestPlan{
		ID:                      3,
		AccountID:               45,
		AutoRecoverRuntimeState: true,
	})

	require.Equal(t, 1, repo.clearRateLimitCalls)
	require.Equal(t, 1, repo.clearTempUnschedCalls)
	require.Equal(t, 0, repo.setSchedulableCalls)
	require.Equal(t, []int64{45}, cache.deletedIDs)
	require.Equal(t, []int64{45}, blocker.clearedIDs)
}
