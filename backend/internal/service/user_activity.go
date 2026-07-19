package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const UserActivityFirstRecharge = "first_recharge"

var ErrUserActivityNotFound = infraerrors.NotFound("USER_ACTIVITY_NOT_FOUND", "user activity not found")

type UserActivity struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	ViewedAt      *time.Time           `json:"viewed_at,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	FirstRecharge *FirstRechargeStatus `json:"first_recharge,omitempty"`
}

type UserActivityService struct {
	firstRechargeService *FirstRechargeActivityService
}

func NewUserActivityService(firstRechargeService *FirstRechargeActivityService) *UserActivityService {
	return &UserActivityService{firstRechargeService: firstRechargeService}
}

func (s *UserActivityService) ListForUser(ctx context.Context, userID int64) ([]UserActivity, error) {
	activities := make([]UserActivity, 0, 1)
	if s == nil || s.firstRechargeService == nil {
		return activities, nil
	}

	status, err := s.firstRechargeService.GetStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if status == nil || !status.Enabled || !status.Eligible || status.Completed {
		return activities, nil
	}
	if status.PurchaseMode == FirstRechargePurchaseModeInternalPayment && len(status.Offers) == 0 {
		return activities, nil
	}

	activities = append(activities, UserActivity{
		ID:            UserActivityFirstRecharge,
		Type:          UserActivityFirstRecharge,
		ViewedAt:      status.ViewedAt,
		CreatedAt:     status.CreatedAt,
		UpdatedAt:     status.UpdatedAt,
		FirstRecharge: status,
	})
	return activities, nil
}

func (s *UserActivityService) MarkViewed(ctx context.Context, userID int64, activityID string) error {
	if s == nil || s.firstRechargeService == nil || activityID != UserActivityFirstRecharge {
		return ErrUserActivityNotFound
	}
	return s.firstRechargeService.MarkViewed(ctx, userID)
}
