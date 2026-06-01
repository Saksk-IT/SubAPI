package service

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
)

type balanceProductSalesCountRow struct {
	BalanceProductID int64 `json:"balance_product_id"`
	Count            int64 `json:"count"`
}

type planSalesCountRow struct {
	PlanID int64 `json:"plan_id"`
	Count  int64 `json:"count"`
}

func productSalesCountStatuses() []string {
	return []string{OrderStatusCompleted, OrderStatusPaid, OrderStatusRecharging}
}

func collectBalanceProductIDs(products []*BalanceProduct) []int64 {
	ids := make([]int64, 0, len(products))
	seen := make(map[int64]struct{}, len(products))
	for _, product := range products {
		if product == nil || product.ID <= 0 {
			continue
		}
		if _, ok := seen[product.ID]; ok {
			continue
		}
		seen[product.ID] = struct{}{}
		ids = append(ids, product.ID)
	}
	return ids
}

func collectPlanIDs(plans []*dbent.SubscriptionPlan) []int64 {
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]struct{}, len(plans))
	for _, plan := range plans {
		if plan == nil || plan.ID <= 0 {
			continue
		}
		if _, ok := seen[plan.ID]; ok {
			continue
		}
		seen[plan.ID] = struct{}{}
		ids = append(ids, plan.ID)
	}
	return ids
}

func (s *PaymentConfigService) attachBalanceProductSalesCounts(ctx context.Context, products []*BalanceProduct) error {
	ids := collectBalanceProductIDs(products)
	if len(ids) == 0 {
		return nil
	}
	counts, err := s.GetBalanceProductSalesCountMap(ctx, ids)
	if err != nil {
		return fmt.Errorf("count balance product sales: %w", err)
	}
	for _, product := range products {
		if product != nil {
			product.SalesCount = counts[product.ID]
		}
	}
	return nil
}

func (s *PaymentConfigService) GetBalanceProductSalesCountMap(ctx context.Context, ids []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []balanceProductSalesCountRow
	err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(productSalesCountStatuses()...),
			paymentorder.BalanceProductIDIn(ids...),
		).
		GroupBy(paymentorder.FieldBalanceProductID).
		Aggregate(dbent.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.BalanceProductID] = row.Count
	}
	return result, nil
}

func (s *PaymentConfigService) GetPlanSalesCountMap(ctx context.Context, plans []*dbent.SubscriptionPlan) (map[int64]int64, error) {
	ids := collectPlanIDs(plans)
	result := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []planSalesCountRow
	err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(productSalesCountStatuses()...),
			paymentorder.PlanIDIn(ids...),
		).
		GroupBy(paymentorder.FieldPlanID).
		Aggregate(dbent.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PlanID] = row.Count
	}
	return result, nil
}
