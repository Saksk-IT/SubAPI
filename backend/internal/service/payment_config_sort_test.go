package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balanceproduct"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
)

func TestCompactProductSortUpdatesKeepsLastSortOrder(t *testing.T) {
	ids, sortOrderByID := compactProductSortUpdates([]ProductSortOrderUpdate{
		{ID: 1, SortOrder: 10},
		{ID: 0, SortOrder: 20},
		{ID: 1, SortOrder: 30},
		{ID: 2, SortOrder: 40},
	})

	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids = %v, want [1 2]", ids)
	}
	if got := sortOrderByID[1]; got != 30 {
		t.Fatalf("sortOrderByID[1] = %d, want 30", got)
	}
	if got := sortOrderByID[2]; got != 40 {
		t.Fatalf("sortOrderByID[2] = %d, want 40", got)
	}
}

func TestUpdatePlanSortOrdersPersistsOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	first := createTestPlan(t, ctx, client, "Basic", 10)
	second := createTestPlan(t, ctx, client, "Pro", 20)

	err := svc.UpdatePlanSortOrders(ctx, []ProductSortOrderUpdate{
		{ID: second.ID, SortOrder: 0},
		{ID: first.ID, SortOrder: 10},
	})
	if err != nil {
		t.Fatalf("UpdatePlanSortOrders returned error: %v", err)
	}

	plans, err := client.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if got := []int64{plans[0].ID, plans[1].ID}; got[0] != second.ID || got[1] != first.ID {
		t.Fatalf("ordered plan ids = %v, want [%d %d]", got, second.ID, first.ID)
	}
}

func TestUpdateBalanceProductSortOrdersPersistsOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	first := createTestBalanceProduct(t, ctx, client, "Small", 10)
	second := createTestBalanceProduct(t, ctx, client, "Large", 20)

	err := svc.UpdateBalanceProductSortOrders(ctx, []ProductSortOrderUpdate{
		{ID: second.ID, SortOrder: 0},
		{ID: first.ID, SortOrder: 10},
	})
	if err != nil {
		t.Fatalf("UpdateBalanceProductSortOrders returned error: %v", err)
	}

	products, err := client.BalanceProduct.Query().Order(balanceproduct.BySortOrder()).All(ctx)
	if err != nil {
		t.Fatalf("list balance products: %v", err)
	}
	if got := []int64{products[0].ID, products[1].ID}; got[0] != second.ID || got[1] != first.ID {
		t.Fatalf("ordered balance product ids = %v, want [%d %d]", got, second.ID, first.ID)
	}
}

func createTestPlan(t *testing.T, ctx context.Context, client *dbent.Client, name string, sortOrder int) *dbent.SubscriptionPlan {
	t.Helper()
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(1).
		SetName(name).
		SetPrice(9.9).
		SetValidityDays(30).
		SetValidityUnit("days").
		SetSortOrder(sortOrder).
		Save(ctx)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	return plan
}

func createTestBalanceProduct(t *testing.T, ctx context.Context, client *dbent.Client, name string, sortOrder int) *dbent.BalanceProduct {
	t.Helper()
	product, err := client.BalanceProduct.Create().
		SetName(name).
		SetPrice(9.9).
		SetAmount(10).
		SetSortOrder(sortOrder).
		Save(ctx)
	if err != nil {
		t.Fatalf("create balance product: %v", err)
	}
	return product
}
