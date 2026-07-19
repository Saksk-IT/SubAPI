package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDailyCheckInActivityServiceDelegatesCalendarDayToRepository(t *testing.T) {
	repo := newDailyCheckInMemoryRepo()
	repo.config.Enabled = true
	repo.claim = &DailyCheckInClaim{RewardAmount: 2.5, BalanceAfter: 12.5, CheckInDate: "2026-07-20"}
	svc := NewDailyCheckInActivityService(repo, nil, nil)

	claim, err := svc.CheckIn(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, repo.claimCalled)
	require.Equal(t, "2026-07-20", claim.CheckInDate)
	require.Equal(t, 2.5, claim.RewardAmount)
}

func TestDailyCheckInActivityServiceValidatesAndRoundsReward(t *testing.T) {
	repo := newDailyCheckInMemoryRepo()
	svc := NewDailyCheckInActivityService(repo, nil, nil)

	_, err := svc.UpdateAdminConfig(context.Background(), true, 0)
	require.Equal(t, "DAILY_CHECK_IN_CONFIG_INVALID", infraerrors.Reason(err))

	config, err := svc.UpdateAdminConfig(context.Background(), true, 1.123456789)
	require.NoError(t, err)
	require.Equal(t, 1.12345679, config.RewardAmount)
}

func TestDailyCheckInActivityServiceBuildsStatus(t *testing.T) {
	repo := newDailyCheckInMemoryRepo()
	repo.config.Enabled = true
	checkedAt := time.Date(2026, 7, 19, 1, 2, 3, 0, time.UTC)
	repo.state = &DailyCheckInUserState{
		CheckedInToday:  true,
		TotalCheckIns:   3,
		LastCheckedInAt: &checkedAt,
	}
	svc := NewDailyCheckInActivityService(repo, nil, nil)

	status, err := svc.GetStatus(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, status.Enabled)
	require.True(t, status.CheckedInToday)
	require.Equal(t, int64(3), status.TotalCheckIns)
	require.Equal(t, DailyCheckInTimezone, status.Timezone)
}

type dailyCheckInMemoryRepo struct {
	config      DailyCheckInConfig
	state       *DailyCheckInUserState
	claim       *DailyCheckInClaim
	claimCalled bool
}

func newDailyCheckInMemoryRepo() *dailyCheckInMemoryRepo {
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	return &dailyCheckInMemoryRepo{
		config: DailyCheckInConfig{
			RewardAmount: 1,
			Timezone:     DailyCheckInTimezone,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
}

func (r *dailyCheckInMemoryRepo) GetConfig(context.Context) (*DailyCheckInConfig, error) {
	config := r.config
	return &config, nil
}

func (r *dailyCheckInMemoryRepo) SaveConfig(_ context.Context, enabled bool, rewardAmount float64) (*DailyCheckInConfig, error) {
	r.config.Enabled = enabled
	r.config.RewardAmount = rewardAmount
	config := r.config
	return &config, nil
}

func (r *dailyCheckInMemoryRepo) GetUserState(context.Context, int64) (*DailyCheckInUserState, error) {
	if r.state == nil {
		return &DailyCheckInUserState{}, nil
	}
	state := *r.state
	return &state, nil
}

func (r *dailyCheckInMemoryRepo) MarkViewed(_ context.Context, _ int64, viewedAt time.Time) error {
	if r.state == nil {
		r.state = &DailyCheckInUserState{}
	}
	r.state.ViewedAt = &viewedAt
	return nil
}

func (r *dailyCheckInMemoryRepo) Claim(_ context.Context, _ int64) (*DailyCheckInClaim, error) {
	r.claimCalled = true
	if r.claim == nil {
		return nil, ErrDailyCheckInUnavailable
	}
	claim := *r.claim
	return &claim, nil
}
