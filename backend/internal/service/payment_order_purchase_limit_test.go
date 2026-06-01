package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestCheckBalanceProductPurchaseLimitCountsActiveAndSuccessfulOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}

	user, err := client.User.Create().
		SetEmail("purchase-limit@example.com").
		SetPasswordHash("hash").
		SetUsername("purchase-limit-user").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	product := createTestBalanceProduct(t, ctx, client, "Limited", 10)
	otherProduct := createTestBalanceProduct(t, ctx, client, "Other", 20)

	createPurchaseLimitOrder(t, ctx, client, user.ID, product.ID, OrderStatusCompleted)
	createPurchaseLimitOrder(t, ctx, client, user.ID, product.ID, OrderStatusPending)
	createPurchaseLimitOrder(t, ctx, client, user.ID, product.ID, OrderStatusCancelled)
	createPurchaseLimitOrder(t, ctx, client, user.ID, otherProduct.ID, OrderStatusCompleted)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = svc.checkBalanceProductPurchaseLimit(ctx, tx, user.ID, &BalanceProduct{ID: product.ID, PurchaseLimit: 2})
	if !infraerrors.IsTooManyRequests(err) {
		t.Fatalf("checkBalanceProductPurchaseLimit error = %v, want too many requests", err)
	}
}

func TestCheckBalanceProductPurchaseLimitAllowsUnlimited(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client}

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := svc.checkBalanceProductPurchaseLimit(ctx, tx, 1, &BalanceProduct{ID: 1, PurchaseLimit: 0}); err != nil {
		t.Fatalf("checkBalanceProductPurchaseLimit returned error: %v", err)
	}
}

func createPurchaseLimitOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID, productID int64, status string) {
	t.Helper()
	now := time.Now()
	creator := client.PaymentOrder.Create().
		SetUserID(userID).
		SetUserEmail("purchase-limit@example.com").
		SetUserName("purchase-limit-user").
		SetAmount(10).
		SetPayAmount(10).
		SetRechargeCode("code-" + status).
		SetOutTradeNo("trade-" + status + "-" + now.Format("150405.000000000")).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("payment-" + status).
		SetOrderType(payment.OrderTypeBalance).
		SetBalanceProductID(productID).
		SetStatus(status).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		SetExpiresAt(now.Add(time.Hour))
	if status != OrderStatusPending {
		creator.SetPaidAt(now)
	}
	if _, err := creator.Save(ctx); err != nil {
		t.Fatalf("create payment order: %v", err)
	}
}
