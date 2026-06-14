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

func TestBulkUpdateBalanceProductsUpdatesSelectedDisplayFields(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	first, err := svc.CreateBalanceProduct(ctx, CreateBalanceProductRequest{
		Name:    "First",
		Price:   10,
		Amount:  100,
		ForSale: true,
	})
	if err != nil {
		t.Fatalf("CreateBalanceProduct first returned error: %v", err)
	}
	second, err := svc.CreateBalanceProduct(ctx, CreateBalanceProductRequest{
		Name:    "Second",
		Price:   20,
		Amount:  200,
		ForSale: true,
	})
	if err != nil {
		t.Fatalf("CreateBalanceProduct second returned error: %v", err)
	}

	description := "批量描述"
	features := "特性 A\n特性 B"
	tags := "标签 A\n标签 B"
	updated, err := svc.BulkUpdateBalanceProducts(ctx, BulkUpdateBalanceProductsRequest{
		ProductIDs: []int64{first.ID, second.ID, first.ID},
		Fields: UpdateBalanceProductRequest{
			Description: &description,
			Features:    &features,
			Tags:        &tags,
		},
	})
	if err != nil {
		t.Fatalf("BulkUpdateBalanceProducts returned error: %v", err)
	}
	if updated != 2 {
		t.Fatalf("updated = %d, want 2", updated)
	}

	got, err := svc.GetBalanceProduct(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetBalanceProduct returned error: %v", err)
	}
	if got.Description != description || got.Features != features || got.Tags != tags {
		t.Fatalf("display fields = (%q, %q, %q), want (%q, %q, %q)", got.Description, got.Features, got.Tags, description, features, tags)
	}
	if got.Name != "First" || got.Price != 10 || got.Amount != 100 {
		t.Fatalf("non-bulk fields changed: name=%q price=%v amount=%v", got.Name, got.Price, got.Amount)
	}
}

func TestBulkUpdateBalanceProductsRejectsNoFields(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentConfigService{entClient: client}

	_, err := svc.BulkUpdateBalanceProducts(ctx, BulkUpdateBalanceProductsRequest{
		ProductIDs: []int64{1},
		Fields:     UpdateBalanceProductRequest{},
	})
	if !infraerrors.IsBadRequest(err) {
		t.Fatalf("BulkUpdateBalanceProducts error = %v, want bad request", err)
	}
}

func TestBulkUpdateBalanceProductsRejectsUnsupportedFields(t *testing.T) {
	name := "new name"
	err := validateBulkBalanceProductPatch(UpdateBalanceProductRequest{Name: &name})
	if !infraerrors.IsBadRequest(err) {
		t.Fatalf("validateBulkBalanceProductPatch error = %v, want bad request", err)
	}
}
