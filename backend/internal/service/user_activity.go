package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UserActivityFirstRecharge = "first_recharge"
	UserActivityDailyCheckIn  = "daily_check_in"
)

var ErrUserActivityNotFound = infraerrors.NotFound("USER_ACTIVITY_NOT_FOUND", "user activity not found")

type UserActivity struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	ViewedAt      *time.Time           `json:"viewed_at,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	FirstRecharge *FirstRechargeStatus `json:"first_recharge,omitempty"`
	DailyCheckIn  *DailyCheckInStatus  `json:"daily_check_in,omitempty"`
}

type UserActivityService struct {
	firstRechargeService *FirstRechargeActivityService
	dailyCheckInService  *DailyCheckInActivityService
}

func NewUserActivityService(firstRechargeService *FirstRechargeActivityService, dailyCheckInService *DailyCheckInActivityService) *UserActivityService {
	return &UserActivityService{
		firstRechargeService: firstRechargeService,
		dailyCheckInService:  dailyCheckInService,
	}
}

func (s *UserActivityService) ListForUser(ctx context.Context, userID int64) ([]UserActivity, error) {
	activities := make([]UserActivity, 0, 2)
	if s == nil {
		return activities, nil
	}

	if s.dailyCheckInService != nil {
		status, err := s.dailyCheckInService.GetStatus(ctx, userID)
		if err != nil {
			return nil, err
		}
		if status != nil && status.Enabled {
			activities = append(activities, UserActivity{
				ID:           UserActivityDailyCheckIn,
				Type:         UserActivityDailyCheckIn,
				ViewedAt:     status.ViewedAt,
				CreatedAt:    status.CreatedAt,
				UpdatedAt:    status.UpdatedAt,
				DailyCheckIn: status,
			})
		}
	}

	if s.firstRechargeService != nil {
		status, err := s.firstRechargeService.GetStatus(ctx, userID)
		if err != nil {
			return nil, err
		}
		if status != nil && status.Enabled && status.Eligible && !status.Completed &&
			(status.PurchaseMode != FirstRechargePurchaseModeInternalPayment || len(status.Offers) > 0) {
			activities = append(activities, UserActivity{
				ID:            UserActivityFirstRecharge,
				Type:          UserActivityFirstRecharge,
				ViewedAt:      status.ViewedAt,
				CreatedAt:     status.CreatedAt,
				UpdatedAt:     status.UpdatedAt,
				FirstRecharge: status,
			})
		}
	}
	return activities, nil
}

func (s *UserActivityService) MarkViewed(ctx context.Context, userID int64, activityID string) error {
	if s == nil {
		return ErrUserActivityNotFound
	}
	switch activityID {
	case UserActivityFirstRecharge:
		if s.firstRechargeService != nil {
			return s.firstRechargeService.MarkViewed(ctx, userID)
		}
	case UserActivityDailyCheckIn:
		if s.dailyCheckInService != nil {
			return s.dailyCheckInService.MarkViewed(ctx, userID)
		}
	}
	return ErrUserActivityNotFound
}
