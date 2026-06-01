package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

var productSalesOrderSeq int64

func TestListBalanceProductsIncludesSuccessfulSalesCount(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	user := createProductSalesTestUser(t, ctx, client)
	product := createTestBalanceProduct(t, ctx, client, "Small", 10)
	otherProduct := createTestBalanceProduct(t, ctx, client, "Large", 20)

	createProductSalesBalanceOrder(t, ctx, client, user.ID, product.ID, OrderStatusCompleted)
	createProductSalesBalanceOrder(t, ctx, client, user.ID, product.ID, OrderStatusPaid)
	createProductSalesBalanceOrder(t, ctx, client, user.ID, product.ID, OrderStatusRecharging)
	createProductSalesBalanceOrder(t, ctx, client, user.ID, product.ID, OrderStatusPending)
	createProductSalesBalanceOrder(t, ctx, client, user.ID, product.ID, OrderStatusCancelled)
	createProductSalesBalanceOrder(t, ctx, client, user.ID, otherProduct.ID, OrderStatusCompleted)

	products, err := svc.ListBalanceProducts(ctx)
	if err != nil {
		t.Fatalf("ListBalanceProducts returned error: %v", err)
	}

	got := balanceProductSalesCountByID(products)
	if got[product.ID] != 3 {
		t.Fatalf("sales count for product = %d, want 3", got[product.ID])
	}
	if got[otherProduct.ID] != 1 {
		t.Fatalf("sales count for other product = %d, want 1", got[otherProduct.ID])
	}
}

func TestGetPlanSalesCountMapCountsSuccessfulOrdersOnly(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	user := createProductSalesTestUser(t, ctx, client)
	plan := createTestPlan(t, ctx, client, "Basic", 10)
	otherPlan := createTestPlan(t, ctx, client, "Pro", 20)

	createProductSalesPlanOrder(t, ctx, client, user.ID, plan.ID, OrderStatusCompleted)
	createProductSalesPlanOrder(t, ctx, client, user.ID, plan.ID, OrderStatusPaid)
	createProductSalesPlanOrder(t, ctx, client, user.ID, plan.ID, OrderStatusRecharging)
	createProductSalesPlanOrder(t, ctx, client, user.ID, plan.ID, OrderStatusPending)
	createProductSalesPlanOrder(t, ctx, client, user.ID, plan.ID, OrderStatusFailed)
	createProductSalesPlanOrder(t, ctx, client, user.ID, otherPlan.ID, OrderStatusCompleted)

	counts, err := svc.GetPlanSalesCountMap(ctx, []*dbent.SubscriptionPlan{plan, otherPlan})
	if err != nil {
		t.Fatalf("GetPlanSalesCountMap returned error: %v", err)
	}
	if counts[plan.ID] != 3 {
		t.Fatalf("sales count for plan = %d, want 3", counts[plan.ID])
	}
	if counts[otherPlan.ID] != 1 {
		t.Fatalf("sales count for other plan = %d, want 1", counts[otherPlan.ID])
	}
}

func createProductSalesTestUser(t *testing.T, ctx context.Context, client *dbent.Client) *dbent.User {
	t.Helper()
	user, err := client.User.Create().
		SetEmail("product-sales@example.com").
		SetPasswordHash("hash").
		SetUsername("product-sales-user").
		Save(ctx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func balanceProductSalesCountByID(products []*BalanceProduct) map[int64]int64 {
	out := make(map[int64]int64, len(products))
	for _, product := range products {
		if product != nil {
			out[product.ID] = product.SalesCount
		}
	}
	return out
}

func createProductSalesBalanceOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID, productID int64, status string) {
	t.Helper()
	creator := createProductSalesOrderBase(t, ctx, client, userID, status).
		SetOrderType(payment.OrderTypeBalance).
		SetBalanceProductID(productID)
	saveProductSalesOrder(t, ctx, creator)
}

func createProductSalesPlanOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID, planID int64, status string) {
	t.Helper()
	creator := createProductSalesOrderBase(t, ctx, client, userID, status).
		SetOrderType(payment.OrderTypeSubscription).
		SetPlanID(planID)
	saveProductSalesOrder(t, ctx, creator)
}

func createProductSalesOrderBase(t *testing.T, ctx context.Context, client *dbent.Client, userID int64, status string) *dbent.PaymentOrderCreate {
	t.Helper()
	now := time.Now()
	seq := atomic.AddInt64(&productSalesOrderSeq, 1)
	creator := client.PaymentOrder.Create().
		SetUserID(userID).
		SetUserEmail("product-sales@example.com").
		SetUserName("product-sales-user").
		SetAmount(10).
		SetPayAmount(10).
		SetRechargeCode(fmt.Sprintf("sales-%s-%d", status, seq)).
		SetOutTradeNo(fmt.Sprintf("sales-trade-%s-%d", status, seq)).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("sales-payment-" + status).
		SetStatus(status).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		SetExpiresAt(now.Add(time.Hour))
	if status != OrderStatusPending {
		creator.SetPaidAt(now)
	}
	return creator
}

func saveProductSalesOrder(t *testing.T, ctx context.Context, creator *dbent.PaymentOrderCreate) {
	t.Helper()
	if _, err := creator.Save(ctx); err != nil {
		t.Fatalf("create payment order: %v", err)
	}
}
