package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPaymentDashboardStatsFiltersFirstRechargeAndBuildsOfferStats(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := createPaymentStatsUser(t, ctx, client)
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	repo := newFirstRechargeMemoryRepo()
	repo.offers = []FirstRechargeOffer{
		{ID: 101, Name: "starter", Price: 9.9, Amount: 19.9, Enabled: true, SortOrder: 10},
		{ID: 102, Name: "pro", Price: 19.9, Amount: 49.9, Enabled: true, SortOrder: 20},
	}
	svc := &PaymentService{entClient: client}
	svc.SetFirstRechargeActivityService(NewFirstRechargeActivityService(repo, &firstRechargeUserRepoFake{}))

	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusCompleted, PayAmount: 9.9,
		PaidAt: &now, ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(101),
		OutTradeNo: "first-recharge-today-1",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusPaid, PayAmount: 19.9,
		PaidAt: &now, ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(102),
		OutTradeNo: "first-recharge-today-2",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusRecharging, PayAmount: 9.9,
		PaidAt: &yesterday, ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(101),
		OutTradeNo: "first-recharge-yesterday",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusCompleted, PayAmount: 99,
		PaidAt: &now, OutTradeNo: "normal-order-today",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusCompleted, PayAmount: 29.9,
		ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(102),
		OutTradeNo: "first-recharge-without-paid-at",
	})

	stats, err := svc.GetDashboardStatsWithParams(ctx, DashboardStatsParams{
		Days:         7,
		ActivityType: FirstRechargeActivityType,
	})
	require.NoError(t, err)

	require.InDelta(t, 39.7, stats.TotalAmount, 0.00001)
	require.Equal(t, 3, stats.TotalCount)
	require.InDelta(t, 29.8, stats.TodayAmount, 0.00001)
	require.Equal(t, 2, stats.TodayCount)
	require.Len(t, stats.OfferStats, 2)
	require.Equal(t, OfferStat{OfferID: 101, Name: "starter", Price: 9.9, Amount: 19.9, Count: 2, Revenue: 19.8}, stats.OfferStats[0])
	require.Equal(t, OfferStat{OfferID: 102, Name: "pro", Price: 19.9, Amount: 49.9, Count: 1, Revenue: 19.9}, stats.OfferStats[1])
}

func TestAdminListOrdersFiltersFirstRechargeWithOtherFilters(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := createPaymentStatsUser(t, ctx, client)
	now := time.Now()
	svc := &PaymentService{entClient: client}

	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusCompleted, PaymentType: payment.TypeAlipay,
		PaidAt: &now, ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(101),
		OutTradeNo: "first-recharge-alipay-needle",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusPending, PaymentType: payment.TypeWxpay,
		ActivityType: FirstRechargeActivityType, FirstRechargeOfferID: paymentStatsInt64Ptr(101),
		OutTradeNo: "first-recharge-wxpay-needle",
	})
	createPaymentStatsOrder(t, ctx, client, paymentStatsOrderInput{
		UserID: user.ID, Email: user.Email, Status: OrderStatusCompleted, PaymentType: payment.TypeAlipay,
		PaidAt: &now, OutTradeNo: "normal-alipay-needle",
	})

	orders, total, err := svc.AdminListOrders(ctx, 0, OrderListParams{
		Page:         1,
		PageSize:     20,
		Status:       OrderStatusCompleted,
		PaymentType:  payment.TypeAlipay,
		ActivityType: FirstRechargeActivityType,
		Keyword:      "needle",
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, orders, 1)
	require.Equal(t, FirstRechargeActivityType, orders[0].ActivityType)
	require.Equal(t, "first-recharge-alipay-needle", orders[0].OutTradeNo)
}

type paymentStatsOrderInput struct {
	UserID               int64
	Email                string
	Status               string
	PaymentType          string
	PayAmount            float64
	PaidAt               *time.Time
	ActivityType         string
	FirstRechargeOfferID *int64
	OutTradeNo           string
}

func createPaymentStatsUser(t *testing.T, ctx context.Context, client *dbent.Client) *dbent.User {
	t.Helper()
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("payment-stats-%d@example.test", time.Now().UnixNano())).
		SetPasswordHash("hash").
		SetUsername("payment-stats-user").
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createPaymentStatsOrder(t *testing.T, ctx context.Context, client *dbent.Client, input paymentStatsOrderInput) *dbent.PaymentOrder {
	t.Helper()
	paymentType := input.PaymentType
	if paymentType == "" {
		paymentType = payment.TypeAlipay
	}
	payAmount := input.PayAmount
	if payAmount <= 0 {
		payAmount = 9.9
	}
	status := input.Status
	if status == "" {
		status = OrderStatusCompleted
	}
	creator := client.PaymentOrder.Create().
		SetUserID(input.UserID).
		SetUserEmail(input.Email).
		SetUserName("payment-stats-user").
		SetAmount(payAmount).
		SetPayAmount(payAmount).
		SetFeeRate(0).
		SetRechargeCode("code-" + input.OutTradeNo).
		SetOutTradeNo(input.OutTradeNo).
		SetPaymentType(paymentType).
		SetPaymentTradeNo("trade-" + input.OutTradeNo).
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(status).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.test")
	if input.PaidAt != nil {
		creator.SetPaidAt(*input.PaidAt)
	}
	if input.ActivityType != "" {
		creator.SetActivityType(input.ActivityType)
	}
	if input.FirstRechargeOfferID != nil {
		creator.SetFirstRechargeOfferID(*input.FirstRechargeOfferID)
	}
	order, err := creator.Save(ctx)
	require.NoError(t, err)
	return order
}

func paymentStatsInt64Ptr(value int64) *int64 {
	return &value
}
