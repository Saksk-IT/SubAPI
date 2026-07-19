package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUserActivityServiceListsVisibleFirstRechargeAndTracksView(t *testing.T) {
	repo := newFirstRechargeMemoryRepo()
	repo.config = FirstRechargeConfig{
		Enabled:          true,
		EligibilityScope: FirstRechargeEligibilityAllUsers,
		PurchaseMode:     FirstRechargePurchaseModeProductLink,
		ProductURL:       "https://shop.example.test/first-recharge",
		CreatedAt:        time.Now().Add(-time.Hour),
		UpdatedAt:        time.Now(),
	}
	firstRechargeService := NewFirstRechargeActivityService(
		repo,
		&firstRechargeUserRepoFake{users: map[int64]*User{}},
		newFirstRechargePaymentConfig(false),
	)
	svc := NewUserActivityService(firstRechargeService)

	activities, err := svc.ListForUser(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, activities, 1)
	require.Equal(t, UserActivityFirstRecharge, activities[0].ID)
	require.Nil(t, activities[0].ViewedAt)
	require.Equal(t, "https://shop.example.test/first-recharge", activities[0].FirstRecharge.ProductURL)

	require.NoError(t, svc.MarkViewed(context.Background(), 7, UserActivityFirstRecharge))
	activities, err = svc.ListForUser(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, activities, 1)
	require.NotNil(t, activities[0].ViewedAt)
	require.True(t, activities[0].FirstRecharge.PopupDismissed)
}

func TestUserActivityServiceHidesUnavailableActivities(t *testing.T) {
	repo := newFirstRechargeMemoryRepo()
	repo.config.Enabled = false
	svc := NewUserActivityService(NewFirstRechargeActivityService(
		repo,
		&firstRechargeUserRepoFake{users: map[int64]*User{}},
		newFirstRechargePaymentConfig(true),
	))

	activities, err := svc.ListForUser(context.Background(), 8)
	require.NoError(t, err)
	require.Empty(t, activities)

	err = svc.MarkViewed(context.Background(), 8, "unknown")
	require.Equal(t, "USER_ACTIVITY_NOT_FOUND", infraerrors.Reason(err))
}
