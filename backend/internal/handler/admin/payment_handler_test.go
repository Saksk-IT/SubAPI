package admin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestAdminPlanResultFromPlanIncludesCurrency(t *testing.T) {
	plan := &dbent.SubscriptionPlan{
		ID:              1,
		GroupID:         2,
		Name:            "NZ Plan",
		Price:           12.5,
		PriceMultiplier: 1.25,
		Currency:        "NZD",
	}

	got := adminPlanResultFromPlan(plan, service.PlanDisplayInfo{}, 3)
	if got.Currency != "NZD" {
		t.Fatalf("expected currency NZD, got %q", got.Currency)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal admin plan result: %v", err)
	}
	if !strings.Contains(string(body), `"currency":"NZD"`) {
		t.Fatalf("expected currency in admin plan response, got %s", string(body))
	}
}

func TestSanitizeAdminPaymentOrderForResponseAddsCurrency(t *testing.T) {
	now := time.Now()
	order := &dbent.PaymentOrder{
		ID:          1,
		UserID:      2,
		Amount:      100,
		PayAmount:   108,
		FeeRate:     8,
		OutTradeNo:  "sub2_202606250001",
		PaymentType: "stripe",
		OrderType:   "subscription",
		Status:      "COMPLETED",
		ExpiresAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
		ProviderSnapshot: map[string]any{
			"schema_version": 2,
			"currency":       "USD",
		},
	}

	got := sanitizeAdminPaymentOrderForResponse(order)
	if got == nil {
		t.Fatal("expected sanitized order")
	}
	if got.Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", got.Currency)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal sanitized order: %v", err)
	}
	if strings.Contains(string(body), "provider_snapshot") {
		t.Fatalf("expected provider_snapshot to be omitted, got %s", string(body))
	}
}
