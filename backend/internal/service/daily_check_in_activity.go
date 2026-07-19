package service

import (
	"context"
	"math"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	DailyCheckInActivityType = "daily_check_in"
	DailyCheckInTimezone     = "Asia/Shanghai"

	maxDailyCheckInReward = 1_000_000
)

var (
	ErrDailyCheckInConfigInvalid = infraerrors.BadRequest("DAILY_CHECK_IN_CONFIG_INVALID", "daily check-in config is invalid")
	ErrDailyCheckInUnavailable   = infraerrors.Forbidden("DAILY_CHECK_IN_UNAVAILABLE", "daily check-in activity is unavailable")
	ErrDailyCheckInAlreadyDone   = infraerrors.Conflict("DAILY_CHECK_IN_ALREADY_DONE", "daily check-in has already been completed")
)

type DailyCheckInConfig struct {
	Enabled      bool      `json:"enabled"`
	RewardAmount float64   `json:"reward_amount"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DailyCheckInUserState struct {
	ViewedAt        *time.Time `json:"viewed_at,omitempty"`
	CheckedInToday  bool       `json:"checked_in_today"`
	TotalCheckIns   int64      `json:"total_check_ins"`
	LastCheckedInAt *time.Time `json:"last_checked_in_at,omitempty"`
}

type DailyCheckInStatus struct {
	Enabled         bool       `json:"enabled"`
	CheckedInToday  bool       `json:"checked_in_today"`
	RewardAmount    float64    `json:"reward_amount"`
	Timezone        string     `json:"timezone"`
	TotalCheckIns   int64      `json:"total_check_ins"`
	LastCheckedInAt *time.Time `json:"last_checked_in_at,omitempty"`
	ViewedAt        *time.Time `json:"viewed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DailyCheckInClaim struct {
	RewardAmount float64   `json:"reward_amount"`
	BalanceAfter float64   `json:"balance_after"`
	CheckInDate  string    `json:"check_in_date"`
	CheckedInAt  time.Time `json:"checked_in_at"`
}

type DailyCheckInRepository interface {
	GetConfig(ctx context.Context) (*DailyCheckInConfig, error)
	SaveConfig(ctx context.Context, enabled bool, rewardAmount float64) (*DailyCheckInConfig, error)
	GetUserState(ctx context.Context, userID int64, checkInDate string) (*DailyCheckInUserState, error)
	MarkViewed(ctx context.Context, userID int64, viewedAt time.Time) error
	Claim(ctx context.Context, userID int64, checkInDate string) (*DailyCheckInClaim, error)
}

type DailyCheckInActivityService struct {
	repo                 DailyCheckInRepository
	billingCache         BillingCache
	authCacheInvalidator APIKeyAuthCacheInvalidator
	now                  func() time.Time
}

func NewDailyCheckInActivityService(
	repo DailyCheckInRepository,
	billingCache BillingCache,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *DailyCheckInActivityService {
	return &DailyCheckInActivityService{
		repo:                 repo,
		billingCache:         billingCache,
		authCacheInvalidator: authCacheInvalidator,
		now:                  time.Now,
	}
}

func (s *DailyCheckInActivityService) GetAdminConfig(ctx context.Context) (*DailyCheckInConfig, error) {
	return s.repo.GetConfig(ctx)
}

func (s *DailyCheckInActivityService) UpdateAdminConfig(ctx context.Context, enabled bool, rewardAmount float64) (*DailyCheckInConfig, error) {
	rewardAmount = normalizeDailyCheckInReward(rewardAmount)
	if rewardAmount <= 0 || rewardAmount > maxDailyCheckInReward {
		return nil, ErrDailyCheckInConfigInvalid
	}
	return s.repo.SaveConfig(ctx, enabled, rewardAmount)
}

func (s *DailyCheckInActivityService) GetStatus(ctx context.Context, userID int64) (*DailyCheckInStatus, error) {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	status := &DailyCheckInStatus{
		Enabled:      config.Enabled,
		RewardAmount: config.RewardAmount,
		Timezone:     DailyCheckInTimezone,
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	}
	if !config.Enabled {
		return status, nil
	}

	state, err := s.repo.GetUserState(ctx, userID, s.currentCheckInDate())
	if err != nil {
		return nil, err
	}
	if state != nil {
		status.CheckedInToday = state.CheckedInToday
		status.TotalCheckIns = state.TotalCheckIns
		status.LastCheckedInAt = state.LastCheckedInAt
		status.ViewedAt = state.ViewedAt
	}
	return status, nil
}

func (s *DailyCheckInActivityService) MarkViewed(ctx context.Context, userID int64) error {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	return s.repo.MarkViewed(ctx, userID, s.now())
}

func (s *DailyCheckInActivityService) CheckIn(ctx context.Context, userID int64) (*DailyCheckInClaim, error) {
	claim, err := s.repo.Claim(ctx, userID, s.currentCheckInDate())
	if err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCache != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.billingCache.InvalidateUserBalance(cacheCtx, userID)
	}
	return claim, nil
}

func (s *DailyCheckInActivityService) currentCheckInDate() string {
	now := s.now()
	location, err := time.LoadLocation(DailyCheckInTimezone)
	if err == nil {
		now = now.In(location)
	}
	return now.Format(time.DateOnly)
}

func normalizeDailyCheckInReward(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*1e8) / 1e8
}
