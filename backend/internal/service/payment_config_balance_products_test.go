package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestBalanceProductPurchaseLimitPersists(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	product, err := svc.CreateBalanceProduct(ctx, CreateBalanceProductRequest{
		Name:          "Limited",
		Price:         9.9,
		Amount:        10,
		ForSale:       true,
		PurchaseLimit: 2,
	})
	if err != nil {
		t.Fatalf("CreateBalanceProduct returned error: %v", err)
	}
	if product.PurchaseLimit != 2 {
		t.Fatalf("PurchaseLimit = %d, want 2", product.PurchaseLimit)
	}

	nextLimit := 3
	updated, err := svc.UpdateBalanceProduct(ctx, product.ID, UpdateBalanceProductRequest{
		PurchaseLimit: &nextLimit,
	})
	if err != nil {
		t.Fatalf("UpdateBalanceProduct returned error: %v", err)
	}
	if updated.PurchaseLimit != 3 {
		t.Fatalf("updated PurchaseLimit = %d, want 3", updated.PurchaseLimit)
	}
}

func TestBalanceProductPurchaseLimitRejectsNegative(t *testing.T) {
	if err := validateBalanceProductRequired(CreateBalanceProductRequest{
		Name:          "Invalid",
		Price:         9.9,
		Amount:        10,
		PurchaseLimit: -1,
	}); !infraerrors.IsBadRequest(err) {
		t.Fatalf("validateBalanceProductRequired error = %v, want bad request", err)
	}

	negativeLimit := -1
	if err := validateBalanceProductPatch(UpdateBalanceProductRequest{PurchaseLimit: &negativeLimit}); !infraerrors.IsBadRequest(err) {
		t.Fatalf("validateBalanceProductPatch error = %v, want bad request", err)
	}
}
